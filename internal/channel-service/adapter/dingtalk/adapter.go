package dingtalk

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/UnicomAI/wanwu/internal/channel-service/adapter/types"
	"github.com/UnicomAI/wanwu/pkg/log"
)

// DingTalkStreamSender 钉钉流式回复发送器
// 实现钉钉 AI 卡片的流式输出（打字机效果）
type DingTalkStreamSender struct {
	client         *StreamClient
	cardInstanceID string
	accumulated    strings.Builder
	chunkCount     int

	mu        sync.Mutex
	closed    bool // Close 已调用，保证幂等
	finalized bool // SendChunk(isFinal=true) 已成功将卡片置为 finished
}

// SendChunk 发送一个流式内容块
// content: 本次增量内容
// isFinal: 是否为最后一个块
func (s *DingTalkStreamSender) SendChunk(ctx context.Context, content string, isFinal bool) error {
	s.accumulated.WriteString(content)
	s.chunkCount++

	// 每 8 个 chunk 或 isFinal 时更新卡片
	if (s.chunkCount-1)%8 == 0 || isFinal {
		fullContent := s.accumulated.String()
		err := s.client.StreamingCard(ctx, s.cardInstanceID, "content", fullContent, true, isFinal, false)
		if err != nil {
			log.Errorf("[DingTalk] Streaming card chunk failed: cardInstanceID=%s, chunkCount=%d, err=%v",
				s.cardInstanceID, s.chunkCount, err)
			return err
		}
		log.Debugf("[DingTalk] Streaming card chunk: cardInstanceID=%s, chunkCount=%d, isFinal=%v",
			s.cardInstanceID, s.chunkCount, isFinal)
		// 正常完成的最终 chunk 已将卡片置为 finished，Close 时无需重复收尾
		if isFinal {
			s.mu.Lock()
			s.finalized = true
			s.mu.Unlock()
		}
	}
	return nil
}

// Close 收尾流式发送器，把卡片置为终态（finished/failed），必须在所有退出路径上调用。
// err==nil 走完成态，err!=nil 走失败态；幂等，重复调用无副作用。
func (s *DingTalkStreamSender) Close(ctx context.Context, err error) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	finalized := s.finalized
	s.mu.Unlock()

	// SendChunk(isFinal=true) 已成功收尾，无需再调 API
	if finalized {
		return nil
	}

	// 流式更新失败或未发送最终 chunk，显式收尾卡片，避免卡片卡在 processing 状态
	cardData := map[string]string{"content": s.accumulated.String()}
	if err != nil {
		log.Infof("[DingTalk] Failing streaming card: cardInstanceID=%s, err=%v", s.cardInstanceID, err)
		if failErr := s.client.FailCard(ctx, s.cardInstanceID, cardData); failErr != nil {
			log.Errorf("[DingTalk] FailCard failed: cardInstanceID=%s, err=%v", s.cardInstanceID, failErr)
			return failErr
		}
		return nil
	}

	log.Infof("[DingTalk] Finishing streaming card: cardInstanceID=%s", s.cardInstanceID)
	if finishErr := s.client.FinishCard(ctx, s.cardInstanceID, cardData); finishErr != nil {
		log.Errorf("[DingTalk] FinishCard failed: cardInstanceID=%s, err=%v", s.cardInstanceID, finishErr)
		return finishErr
	}
	return nil
}

// DingTalkAdapter 钉钉平台适配器
// 支持两种模式：Stream（推荐，WebSocket 长连接）和 Webhook（HTTP 回调）
type DingTalkAdapter struct {
	mu        sync.RWMutex
	config    types.AdapterConfig
	connected bool
	handler   types.MessageHandler

	// Stream 模式客户端
	stream *StreamClient
	// Webhook 模式客户端
	webhook *WebhookClient
}

// NewDingTalkAdapter 创建钉钉适配器
func NewDingTalkAdapter() *DingTalkAdapter {
	return &DingTalkAdapter{}
}

// Connect 连接到钉钉平台
// 根据配置选择 Stream 或 Webhook 模式
func (d *DingTalkAdapter) Connect(config types.AdapterConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.config = config

	// 判断模式：默认使用 Stream，如果配置了 connectionMode=webhook 则使用 Webhook
	mode := config.Extra["connectionMode"]
	if mode == "" {
		mode = "stream"
	}

	switch mode {
	case "stream":
		streamConfig := &StreamConfig{
			AppKey:    config.AppKey,
			AppSecret: config.AppSecret,
			ChannelID: config.ChannelID,
		}
		streamClient := NewStreamClient(streamConfig)

		// 注册消息回调
		if d.handler != nil {
			streamClient.OnMessage(func(ctx context.Context, msg *Message) error {
				platformMsg := &types.PlatformMessage{
					ChannelID:      config.ChannelID,
					ConversationID: msg.Conversation,
					UserID:         msg.Sender,
					Content:        msg.Content,
					MsgType:        string(msg.MsgType),
					ChannelType:    types.ChannelTypeDingTalk,
					Extra: map[string]string{
						"sessionWebhook": msg.SessionWebhook,
						"senderNick":     msg.SenderNick,
						"isInGroup":      fmt.Sprintf("%v", msg.IsInGroup),
						"conversationID": msg.Conversation,
						"messageID":      msg.MessageID,
					},
				}
				if err := d.handler(ctx, platformMsg); err != nil {
					log.Errorf("[DingTalk] Failed to handle message: %v", err)
				}
				return nil
			})
		}

		if err := streamClient.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start dingtalk stream: %w", err)
		}

		d.stream = streamClient
		d.connected = true
		log.Infof("[DingTalk] Adapter connected for channel %s (stream mode)", config.ChannelID)

	case "webhook":
		webhookConfig := &WebhookConfig{
			AppKey:    config.AppKey,
			AppSecret: config.AppSecret,
			ChannelID: config.ChannelID,
		}
		webhookClient := NewWebhookClient(webhookConfig)

		// 注册消息回调
		if d.handler != nil {
			webhookClient.OnMessage(func(ctx context.Context, msg *Message) error {
				platformMsg := &types.PlatformMessage{
					ChannelID:      config.ChannelID,
					ConversationID: msg.Conversation,
					UserID:         msg.Sender,
					Content:        msg.Content,
					MsgType:        string(msg.MsgType),
					ChannelType:    types.ChannelTypeDingTalk,
					Extra: map[string]string{
						"sessionWebhook": msg.SessionWebhook,
						"senderNick":     msg.SenderNick,
						"isInGroup":      fmt.Sprintf("%v", msg.IsInGroup),
						"conversationID": msg.Conversation,
						"messageID":      msg.MessageID,
					},
				}
				if err := d.handler(ctx, platformMsg); err != nil {
					log.Errorf("[DingTalk] Failed to handle message: %v", err)
				}
				return nil
			})
		}

		if err := webhookClient.Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start dingtalk webhook: %w", err)
		}

		d.webhook = webhookClient
		d.connected = true
		log.Infof("[DingTalk] Adapter connected for channel %s (webhook mode)", config.ChannelID)

	default:
		return fmt.Errorf("unsupported dingtalk connection mode: %s", mode)
	}

	return nil
}

// Disconnect 断开钉钉连接
func (d *DingTalkAdapter) Disconnect() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stream != nil {
		d.stream.Stop()
		d.stream = nil
	}
	if d.webhook != nil {
		d.webhook.Stop()
		d.webhook = nil
	}
	d.connected = false
	log.Infof("[DingTalk] Adapter disconnected for channel %s", d.config.ChannelID)
	return nil
}

// IsConnected 检查是否已连接
func (d *DingTalkAdapter) IsConnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connected
}

// GetAccountInfo 获取钉钉账号信息
func (d *DingTalkAdapter) GetAccountInfo() (accountId, nickname, avatar string, err error) {
	// 钉钉的 accountId 即 appKey
	return d.config.AppKey, "", "", nil
}

// SendMessage 向钉钉用户发送消息
// 优先使用 sessionWebhook 回复（与钉钉官方 SDK 行为一致，最可靠），
// 其次根据群聊/单聊选择对应 API 降级发送
func (d *DingTalkAdapter) SendMessage(ctx context.Context, userID, content string, extra map[string]string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 1. 优先使用 sessionWebhook 回复（最可靠，钉钉官方推荐方式）
	// sessionWebhook 由钉钉在用户发消息时提供，有效期通常为 1 小时
	if extra != nil {
		if webhookURL := extra["sessionWebhook"]; webhookURL != "" {
			if d.stream != nil {
				log.Infof("[DingTalk] SendMessage via sessionWebhook (stream mode), userID=%s", userID)
				return d.stream.ReplyWithWebhook(ctx, webhookURL, content, nil)
			}
			if d.webhook != nil {
				log.Infof("[DingTalk] SendMessage via sessionWebhook (webhook mode), userID=%s", userID)
				return d.webhook.sendViaWebhook(ctx, webhookURL, &TextMessage{
					MsgType: "text",
					Text: struct {
						Content string `json:"content"`
					}{Content: content},
				})
			}
		}
	}

	// 2. 降级：通过 API 发送（需要区分群聊和单聊）
	isInGroup := false
	if extra != nil {
		isInGroup = extra["isInGroup"] == "true"
	}

	if d.stream != nil {
		if isInGroup {
			conversationID := ""
			if extra != nil {
				conversationID = extra["conversationID"]
			}
			log.Infof("[DingTalk] SendMessage via groupMessages API, userID=%s, conversationID=%s", userID, conversationID)
			return d.stream.SendGroupText(ctx, conversationID, content, nil)
		}
		log.Infof("[DingTalk] SendMessage via oToMessages API, userID=%s", userID)
		return d.stream.SendText(ctx, userID, content, nil)
	}

	if d.webhook != nil {
		if isInGroup {
			conversationID := ""
			if extra != nil {
				conversationID = extra["conversationID"]
			}
			log.Infof("[DingTalk] SendMessage via webhook groupMessages API, conversationID=%s", conversationID)
			return d.webhook.SendGroupText(ctx, conversationID, content, nil)
		}
		log.Infof("[DingTalk] SendMessage via webhook oToMessages API, userID=%s", userID)
		return d.webhook.SendText(ctx, userID, content, nil)
	}

	return fmt.Errorf("dingtalk adapter not connected")
}

// SendMessageWithWebhook 使用 sessionWebhook 回复消息（更高效的回复方式）
func (d *DingTalkAdapter) SendMessageWithWebhook(ctx context.Context, webhookURL, content string) error {
	d.mu.RLock()
	stream := d.stream
	d.mu.RUnlock()

	if stream != nil {
		return stream.ReplyWithWebhook(ctx, webhookURL, content, nil)
	}

	return fmt.Errorf("stream client not available for webhook reply")
}

// SendMarkdownMessage 发送 Markdown 格式消息
func (d *DingTalkAdapter) SendMarkdownMessage(ctx context.Context, userID, title, content string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.stream != nil {
		return d.stream.SendMarkdown(ctx, userID, title, content, nil)
	}
	if d.webhook != nil {
		return d.webhook.SendMarkdown(ctx, userID, title, content, nil)
	}

	return fmt.Errorf("dingtalk adapter not connected")
}

// OnMessage 注册消息回调
func (d *DingTalkAdapter) OnMessage(handler types.MessageHandler) {
	d.handler = handler
}

// CreateStreamSender 创建钉钉流式回复发送器
// Stream 模式下默认启用流式卡片回复，配置 streamReply=false 可禁用，否则返回 nil（降级为非流式）
func (d *DingTalkAdapter) CreateStreamSender(ctx context.Context, userID string, extra map[string]string) types.StreamSender {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 检查是否禁用了流式回复（Stream 模式下默认启用）
	if d.config.Extra["streamReply"] == "false" {
		return nil
	}

	// 仅 Stream 模式支持流式卡片
	if d.stream == nil {
		log.Debugf("[DingTalk] Stream client not available, falling back to non-streaming reply")
		return nil
	}

	// 获取必要的消息上下文
	conversationID := extra["conversationID"]
	isInGroup := extra["isInGroup"] == "true"

	if conversationID == "" {
		log.Warnf("[DingTalk] Cannot create stream sender: missing conversationID")
		return nil
	}

	// 获取卡片模板 ID（默认使用钉钉 AI Markdown 模板）
	cardTemplateID := d.config.Extra["cardTemplateId"]
	if cardTemplateID == "" {
		cardTemplateID = DefaultCardTemplateID
	}

	// 创建并投放 AI 卡片
	cardData := map[string]string{
		"content":    "",
		"flowStatus": string(AICardStatusProcessing),
	}

	cardInstanceID, err := d.stream.CreateAndDeliverCard(
		ctx,
		cardTemplateID,
		cardData,
		userID,
		conversationID,
		isInGroup,
	)
	if err != nil {
		log.Errorf("[DingTalk] Failed to create streaming card, falling back to non-streaming: %v", err)
		return nil
	}

	log.Infof("[DingTalk] Created streaming card: cardInstanceID=%s, userID=%s, isInGroup=%v",
		cardInstanceID, userID, isInGroup)

	return &DingTalkStreamSender{
		client:         d.stream,
		cardInstanceID: cardInstanceID,
	}
}

// HandleWebhook 处理 Webhook 回调请求（Webhook 模式专用）
// 此方法应在 HTTP callback 路由中调用
func (d *DingTalkAdapter) HandleWebhook(ctx context.Context, body []byte, timestamp, sign string) error {
	d.mu.RLock()
	webhook := d.webhook
	d.mu.RUnlock()

	if webhook == nil {
		return fmt.Errorf("webhook client not initialized")
	}

	return webhook.HandleWebhook(ctx, body, timestamp, sign)
}
