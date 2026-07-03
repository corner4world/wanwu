package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/UnicomAI/wanwu/internal/channel-service/adapter/types"
	"github.com/UnicomAI/wanwu/internal/channel-service/client"
	"github.com/UnicomAI/wanwu/internal/channel-service/client/model"
	"github.com/UnicomAI/wanwu/internal/channel-service/config"
	"github.com/UnicomAI/wanwu/internal/channel-service/wanwu"
	"github.com/UnicomAI/wanwu/pkg/log"
)

// Handler 消息对话处理器
// 接收平台消息 → 查找通道配置 → 获取 API Key → 调用万悟 OpenAPI → 流式回传
type Handler struct {
	cfg         config.Config
	cli         client.IClient
	manager     adapterManager
	convManager *wanwu.ConversationManager
}

// adapterManager 适配器管理接口（避免循环依赖）
type adapterManager interface {
	GetAdapter(channelID string) (types.Adapter, bool)
	SendMessage(ctx context.Context, channelID, userID, content string, extra map[string]string) error
	CreateStreamSender(ctx context.Context, channelID, userID string, extra map[string]string) types.StreamSender
}

// NewHandler 创建消息处理器
func NewHandler(cfg config.Config, cli client.IClient, manager adapterManager) *Handler {
	return &Handler{
		cfg:         cfg,
		cli:         cli,
		manager:     manager,
		convManager: wanwu.NewConversationManager(),
	}
}

// HandlePlatformMessage 处理来自平台的消息
func (h *Handler) HandlePlatformMessage(ctx context.Context, msg *types.PlatformMessage) error {
	log.Infof("received platform message: channel=%s, user=%s, type=%s, content=%s",
		msg.ChannelID, msg.UserID, msg.MsgType, truncate(msg.Content, 100))

	// 1. 查找通道配置
	ch, err := h.cli.GetChannel(ctx, msg.ChannelID)
	if err != nil {
		return fmt.Errorf("channel not found %s: %w", msg.ChannelID, err)
	}

	// 2. 检查通道状态
	if !ch.Enabled || ch.Status != "loggedIn" {
		return fmt.Errorf("channel %s is not active (enabled=%v, status=%s)", ch.ChannelID, ch.Enabled, ch.Status)
	}

	// 3. 获取 API Key
	if !ch.HasApiKey() {
		if ch.ApiKeyID != "" {
			return fmt.Errorf("channel %s has api_key_id (%s) but api_key value is empty, please rebind the API Key", ch.ChannelID, ch.ApiKeyID)
		}
		return fmt.Errorf("channel %s has no api key bound, please bind an API Key in channel settings", ch.ChannelID)
	}

	// 4. 按 appType 分发
	switch ch.AppType {
	case "wga":
		return h.handleWGAMessage(ctx, ch, msg)
	default: // "agent"
		return h.handleAgentMessage(ctx, ch, msg)
	}
}

// handleAgentMessage 处理普通智能体消息
func (h *Handler) handleAgentMessage(ctx context.Context, ch *model.Channel, msg *types.PlatformMessage) error {
	apiKey := ch.ApiKey

	// 获取或创建万悟会话 ID（同一用户同一通道复用同一会话，保持上下文记忆）
	wanwuClient := wanwu.NewClient(h.cfg.BFF.BaseURL)
	conversationID, ok := h.convManager.GetConversationID(msg.ChannelID, msg.UserID, "agent")
	if !ok {
		// 首次对话，创建会话
		convResp, err := wanwuClient.CreateConversation(ctx, apiKey, &wanwu.CreateConversationRequest{
			UUID:  ch.AppID,
			Title: truncate(msg.Content, 50),
		})
		if err != nil {
			log.Warnf("failed to create conversation for channel %s user %s: %v, will chat without conversation_id", msg.ChannelID, msg.UserID, err)
			// 创建会话失败时不阻断对话，不传 conversation_id 让 BFF 自动处理
		} else {
			conversationID = convResp.ConversationID
			h.convManager.SetConversationID(msg.ChannelID, msg.UserID, "agent", conversationID)
			log.Infof("created conversation %s for channel %s user %s", conversationID, msg.ChannelID, msg.UserID)
		}
	}

	// 调用万悟 OpenAPI 智能体对话
	chatReq := &wanwu.ChatRequest{
		UUID:           ch.AppID,
		ConversationID: conversationID,
		Query:          msg.Content,
		Stream:         true,
	}

	resp, err := wanwuClient.ChatWithAgent(ctx, apiKey, chatReq)
	if err != nil {
		return fmt.Errorf("failed to call wanwu chat api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 处理 SSE 流式响应
	return h.handleAgentSSEResponse(ctx, ch, msg, resp)
}

// handleWGAMessage 处理通用智能体（WGA）消息
func (h *Handler) handleWGAMessage(ctx context.Context, ch *model.Channel, msg *types.PlatformMessage) error {
	apiKey := ch.ApiKey

	// 检查 modelUuid
	if ch.ModelUuid == "" {
		return fmt.Errorf("channel %s is wga type but has no model_uuid configured", ch.ChannelID)
	}

	// 获取或创建 WGA 会话（threadId）
	wanwuClient := wanwu.NewClient(h.cfg.BFF.BaseURL)
	threadID, ok := h.convManager.GetConversationID(msg.ChannelID, msg.UserID, "wga")
	if !ok {
		// 首次对话，创建 WGA 会话
		convResp, err := wanwuClient.CreateWGAConversation(ctx, apiKey, &wanwu.WGACreateConversationRequest{
			Title:     truncate(msg.Content, 50),
			ModelUuid: ch.ModelUuid,
		})
		if err != nil {
			return fmt.Errorf("failed to create wga conversation for channel %s user %s: %w", msg.ChannelID, msg.UserID, err)
		}
		threadID = convResp.ThreadID
		h.convManager.SetConversationID(msg.ChannelID, msg.UserID, "wga", threadID)
		log.Infof("created wga conversation %s for channel %s user %s", threadID, msg.ChannelID, msg.UserID)
	}

	// 调用 WGA 对话接口
	chatReq := &wanwu.WGAChatRequest{
		ThreadID: threadID,
		Messages: []wanwu.WGAMessage{
			{
				Role:    "user",
				Content: msg.Content,
			},
		},
		ModelUuid: ch.ModelUuid,
	}

	resp, err := wanwuClient.ChatWithWGA(ctx, apiKey, chatReq)
	if err != nil {
		return fmt.Errorf("failed to call wga chat api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 处理 WGA AG-UI SSE 流式响应
	return h.handleWGASSEResponse(ctx, ch, msg, resp)
}

// handleAgentSSEResponse 处理普通智能体 SSE 流式响应
func (h *Handler) handleAgentSSEResponse(ctx context.Context, ch *model.Channel, msg *types.PlatformMessage, resp *http.Response) error {
	// 尝试创建流式发送器（支持流式卡片的平台会返回非 nil）
	streamSender := h.manager.CreateStreamSender(ctx, msg.ChannelID, msg.UserID, msg.Extra)

	var fullContent strings.Builder
	reader := bufio.NewReader(resp.Body)
	chunkCount := 0

	log.Infof("[AgentSSE] channel=%s user=%s start streaming from agent %s (streamSender=%v)",
		ch.ChannelID, msg.UserID, ch.AppID, streamSender != nil)

	for {
		select {
		case <-ctx.Done():
			log.Infof("[AgentSSE] channel=%s user=%s context cancelled after %d chunks", ch.ChannelID, msg.UserID, chunkCount)
			closeStreamSender(streamSender, ctx, ctx.Err())
			return ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			closeStreamSender(streamSender, ctx, err)
			return fmt.Errorf("error reading SSE stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析 SSE 数据行
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)

		if data == "[DONE]" {
			log.Infof("[AgentSSE] channel=%s user=%s received [DONE] signal after %d chunks", ch.ChannelID, msg.UserID, chunkCount)
			break
		}

		// 跳过空数据行
		if data == "" {
			continue
		}

		// 解析 SSE 数据
		// BFF OpenAPI agent chat 返回格式: data:{"response":"...","msg_id":"...","eventType":...}
		var sseData struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal([]byte(data), &sseData); err != nil {
			log.Errorf("[AgentSSE] channel=%s user=%s failed to parse SSE data: %v, raw: %s", ch.ChannelID, msg.UserID, err, data)
			continue
		}

		if sseData.Response != "" {
			// 流式路径：逐 chunk 更新卡片
			if streamSender != nil {
				if err := streamSender.SendChunk(ctx, sseData.Response, false); err != nil {
					log.Errorf("[AgentSSE] channel=%s user=%s stream sender chunk failed, falling back to non-streaming: %v",
						ch.ChannelID, msg.UserID, err)
					// 流式发送失败，收尾卡片（置 failed）后降级为非流式
					closeStreamSender(streamSender, ctx, fmt.Errorf("stream chunk failed: %w", err))
					streamSender = nil
				}
			}
			fullContent.WriteString(sseData.Response)
			chunkCount++
			// log.Debugf("[AgentSSE] channel=%s user=%s chunk #%d: %q", ch.ChannelID, msg.UserID, chunkCount, truncate(sseData.Response, 100))
		}
	}

	replyContent := fullContent.String()

	// 流式路径：发送最终 chunk 标记完成
	if streamSender != nil {
		if err := streamSender.SendChunk(ctx, "", true); err != nil {
			log.Errorf("[AgentSSE] channel=%s user=%s stream sender final chunk failed: %v", ch.ChannelID, msg.UserID, err)
			// 最终 chunk 失败，收尾卡片（置 failed）后降级为非流式
			closeStreamSender(streamSender, ctx, fmt.Errorf("final chunk failed: %w", err))
			streamSender = nil
		} else {
			// 正常完成：SendChunk(isFinal) 已将卡片置为 finished，Close 此时为幂等 no-op
			closeStreamSender(streamSender, ctx, nil)
			log.Infof("[AgentSSE] channel=%s user=%s stream completed via card, total %d chunks, %d chars",
				ch.ChannelID, msg.UserID, chunkCount, len(replyContent))
			return nil
		}
	}

	// 非流式路径：将完整回复发送给平台用户
	if replyContent == "" {
		log.Warnf("[AgentSSE] channel=%s user=%s empty reply from wanwu agent after %d chunks", ch.ChannelID, msg.UserID, chunkCount)
		return nil
	}

	log.Infof("[AgentSSE] channel=%s user=%s stream completed, total %d chunks, reply length=%d, content: %s",
		ch.ChannelID, msg.UserID, chunkCount, len(replyContent), truncate(replyContent, 200))

	if err := h.manager.SendMessage(ctx, msg.ChannelID, msg.UserID, replyContent, msg.Extra); err != nil {
		return fmt.Errorf("failed to send reply to platform: %w", err)
	}

	log.Infof("[AgentSSE] channel=%s user=%s reply sent to platform successfully", ch.ChannelID, msg.UserID)
	return nil
}

// handleWGASSEResponse 处理 WGA AG-UI SSE 流式响应
// AG-UI 事件格式：
//
//	data: {"type":"TEXT_MESSAGE_START","messageId":"msg-1","role":"assistant"}
//	data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"msg-1","delta":"你好"}
//	data: {"type":"TEXT_MESSAGE_END","messageId":"msg-1"}
//	data: {"type":"RUN_FINISHED","threadId":"xxx","runId":"run-1"}
func (h *Handler) handleWGASSEResponse(ctx context.Context, ch *model.Channel, msg *types.PlatformMessage, resp *http.Response) error {
	// 尝试创建流式发送器（支持流式卡片的平台会返回非 nil）
	streamSender := h.manager.CreateStreamSender(ctx, msg.ChannelID, msg.UserID, msg.Extra)

	var fullContent strings.Builder
	reader := bufio.NewReader(resp.Body)

	log.Infof("[WGA-SSE] channel=%s user=%s start streaming from WGA model %s (streamSender=%v)",
		ch.ChannelID, msg.UserID, ch.ModelUuid, streamSender != nil)

	for {
		select {
		case <-ctx.Done():
			closeStreamSender(streamSender, ctx, ctx.Err())
			return ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			closeStreamSender(streamSender, ctx, err)
			return fmt.Errorf("error reading WGA SSE stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析 SSE 数据行
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)

		if data == "[DONE]" {
			break
		}

		// 跳过空数据行
		if data == "" {
			continue
		}

		// 解析 AG-UI 事件
		var event struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			log.Errorf("failed to parse WGA SSE data: %v, raw: %s", err, data)
			continue
		}

		switch event.Type {
		case "TEXT_MESSAGE_CONTENT":
			if event.Delta != "" {
				// 流式路径：逐 chunk 更新卡片
				if streamSender != nil {
					if err := streamSender.SendChunk(ctx, event.Delta, false); err != nil {
						log.Errorf("[WGA-SSE] channel=%s user=%s stream sender chunk failed, falling back to non-streaming: %v",
							ch.ChannelID, msg.UserID, err)
						// 流式发送失败，收尾卡片（置 failed）后降级为非流式
						closeStreamSender(streamSender, ctx, fmt.Errorf("stream chunk failed: %w", err))
						streamSender = nil
					}
				}
				fullContent.WriteString(event.Delta)
			}
		case "RUN_FINISHED":
			// 对话结束，跳出循环
			goto wgaDone
		}
	}

wgaDone:

	replyContent := fullContent.String()

	// 流式路径：发送最终 chunk 标记完成
	if streamSender != nil {
		if err := streamSender.SendChunk(ctx, "", true); err != nil {
			log.Errorf("[WGA-SSE] channel=%s user=%s stream sender final chunk failed: %v", ch.ChannelID, msg.UserID, err)
			// 最终 chunk 失败，收尾卡片（置 failed）后降级为非流式
			closeStreamSender(streamSender, ctx, fmt.Errorf("final chunk failed: %w", err))
			streamSender = nil
		} else {
			// 正常完成：SendChunk(isFinal) 已将卡片置为 finished，Close 此时为幂等 no-op
			closeStreamSender(streamSender, ctx, nil)
			log.Infof("[WGA-SSE] channel=%s user=%s stream completed via card, %d chars",
				ch.ChannelID, msg.UserID, len(replyContent))
			return nil
		}
	}

	// 非流式路径：将完整回复发送给平台用户
	if replyContent == "" {
		log.Warnf("empty reply from wanwu wga for channel %s", ch.ChannelID)
		return nil
	}

	if err := h.manager.SendMessage(ctx, msg.ChannelID, msg.UserID, replyContent, msg.Extra); err != nil {
		return fmt.Errorf("failed to send reply to platform: %w", err)
	}

	log.Infof("replied to user %s on channel %s (wga): %s", msg.UserID, ch.ChannelID, truncate(replyContent, 50))
	return nil
}

// closeStreamSender 收尾流式发送器（置卡片为 finished/failed），忽略 nil 与 Close 自身的错误。
// 用于所有离开流式路径的出口（成功/失败/取消），避免卡片卡在 processing 状态。
func closeStreamSender(s types.StreamSender, ctx context.Context, err error) {
	if s == nil {
		return
	}
	if closeErr := s.Close(ctx, err); closeErr != nil {
		log.Warnf("close stream sender failed: %v (original err: %v)", closeErr, err)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
