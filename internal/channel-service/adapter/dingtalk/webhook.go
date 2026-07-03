package dingtalk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/UnicomAI/wanwu/pkg/log"
)

// WebhookClient 钉钉 Webhook 模式客户端
// 用于接收钉钉 HTTP 回调推送，通过 sessionWebhook 或 API 回复消息
type WebhookClient struct {
	appKey    string
	appSecret string
	channelID string // 渠道 ID

	accessToken     string
	tokenExpiry     time.Time
	accessTokenLock sync.RWMutex

	// sessionWebhook 缓存
	sessionWebhooks map[string]*sessionWebhookInfo
	webhookMu       sync.RWMutex

	// 消息处理器
	messageHandler MessageHandler

	// HTTP 客户端
	httpClient *http.Client

	// 状态
	status string
}

type sessionWebhookInfo struct {
	webhookURL  string
	expiredTime time.Time
}

// WebhookConfig Webhook 客户端配置
type WebhookConfig struct {
	AppKey    string
	AppSecret string
	ChannelID string
}

// NewWebhookClient 创建 Webhook 客户端
func NewWebhookClient(config *WebhookConfig) *WebhookClient {
	return &WebhookClient{
		appKey:          config.AppKey,
		appSecret:       config.AppSecret,
		channelID:       config.ChannelID,
		sessionWebhooks: make(map[string]*sessionWebhookInfo),
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		status:          "offline",
	}
}

// OnMessage 注册消息处理器
func (c *WebhookClient) OnMessage(handler MessageHandler) {
	c.messageHandler = handler
}

// Start 启动（Webhook 模式不需要建立连接，标记为在线即可）
func (c *WebhookClient) Start(_ context.Context) error {
	c.status = "online"
	log.Infof("[DingTalk Webhook] Client started for channel %s", c.channelID)
	return nil
}

// Stop 停止
func (c *WebhookClient) Stop() {
	c.status = "offline"
	log.Infof("[DingTalk Webhook] Client stopped for channel %s", c.channelID)
}

// GetStatus 获取状态
func (c *WebhookClient) GetStatus() string {
	return c.status
}

// HandleWebhook 处理钉钉 Webhook 推送消息
// 此方法应在 HTTP callback 路由中调用
func (c *WebhookClient) HandleWebhook(ctx context.Context, body []byte, timestamp, sign string) error {
	// 1. 验证签名
	if err := c.verifySign(timestamp, sign); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// 2. 解析消息
	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return fmt.Errorf("failed to parse webhook message: %w", err)
	}

	// 3. 转换为统一消息格式
	msg := c.convertToMessage(&req)
	msg.ChannelID = c.channelID
	msg.Receiver = req.ChatbotUserID

	// 4. 缓存 sessionWebhook 用于回复
	if req.SessionWebhook != "" && req.Sender != "" {
		c.webhookMu.Lock()
		c.sessionWebhooks[req.Sender] = &sessionWebhookInfo{
			webhookURL:  req.SessionWebhook,
			expiredTime: time.Now().Add(time.Duration(req.SessionWebhookExpiredTime) * time.Second),
		}
		c.webhookMu.Unlock()
	}

	// 5. 更新状态
	c.status = "online"

	// 6. 调用消息处理器
	if c.messageHandler != nil {
		go func() {
			if err := c.messageHandler(ctx, msg); err != nil {
				log.Errorf("[DingTalk Webhook] Message handler error: %v", err)
			}
		}()
	}

	return nil
}

// convertToMessage 转换 Webhook 请求为统一消息格式
func (c *WebhookClient) convertToMessage(req *WebhookRequest) *Message {
	msg := &Message{
		MsgType:        MessageType(req.Msgtype),
		Conversation:   req.ConversationID,
		Sender:         req.Sender,
		SenderNick:     req.SenderNick,
		Timestamp:      time.UnixMilli(req.CreateTime),
		IsInGroup:      req.ConversationType == "2",
		SessionWebhook: req.SessionWebhook,
		Raw:            make(map[string]interface{}),
	}

	// 解析 @ 用户
	for _, atUser := range req.AtUsers {
		msg.AtUserIDs = append(msg.AtUserIDs, atUser.DingtalkID)
	}

	// 根据消息类型设置内容
	switch req.Msgtype {
	case "text":
		msg.Content = req.Text.Content
	case "picture":
		msg.Content = req.Picture.PicURL
		msg.MsgType = MessageTypeImage
	case "voice":
		msg.Content = req.Voice.Content
		msg.MsgType = MessageTypeVoice
	case "file":
		msg.Content = req.File.FileName
		msg.MsgType = MessageTypeFile
	case "richText":
		msg.Content = req.RichText.Content
		msg.MsgType = MessageTypeMarkdown
	}

	return msg
}

// verifySign 验证钉钉 Webhook 签名
func (c *WebhookClient) verifySign(timestamp, sign string) error {
	if c.appSecret == "" {
		return nil // 没有配置 secret 则跳过验证
	}

	// 构造签名字符串: timestamp + "\n" + appSecret
	stringToSign := timestamp + "\n" + c.appSecret

	// 计算 HMAC-SHA256
	h := hmac.New(sha256.New, []byte(c.appSecret))
	h.Write([]byte(stringToSign))
	expectedSign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	if sign != expectedSign {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// GetAccessToken 获取 access token（自动缓存和刷新）
func (c *WebhookClient) GetAccessToken(ctx context.Context) (string, error) {
	c.accessTokenLock.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.accessTokenLock.RUnlock()
		return token, nil
	}
	c.accessTokenLock.RUnlock()

	apiURL := "https://api.dingtalk.com/v1.0/oauth2/accessToken"
	reqBody := map[string]string{
		"appKey":    c.appKey,
		"appSecret": c.appSecret,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var result AccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("get access token failed: errcode=%d, errmsg=%s", result.ErrCode, result.ErrMsg)
	}

	c.accessTokenLock.Lock()
	c.accessToken = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	c.accessTokenLock.Unlock()

	return c.accessToken, nil
}

// GetUserInfo 获取用户信息
func (c *WebhookClient) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.dingtalk.com/v1.0/contact/users/%s", userID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		ErrCode int       `json:"errcode"`
		ErrMsg  string    `json:"errmsg"`
		User    *UserInfo `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("get user info failed: %s", result.ErrMsg)
	}

	return result.User, nil
}

// SendText 发送文本消息
// 优先使用 sessionWebhook 回复，否则使用 API 发送
func (c *WebhookClient) SendText(ctx context.Context, receiver string, content string, atUserIDs []string) error {
	// 优先使用 sessionWebhook
	if webhookURL := c.getSessionWebhook(receiver); webhookURL != "" {
		return c.sendViaWebhook(ctx, webhookURL, &TextMessage{
			MsgType: "text",
			Text: struct {
				Content string `json:"content"`
			}{Content: content},
		})
	}

	// 使用 API 发送
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"
	reqBody := map[string]interface{}{
		"robotCode": c.appKey,
		"userIds":   []string{receiver},
		"msgKey":    "sampleText",
		"msgParam": map[string]string{
			"content": content,
		},
	}

	return c.sendAPI(ctx, token, apiURL, reqBody)
}

// SendGroupText 发送文本消息到群聊（通过 API）
func (c *WebhookClient) SendGroupText(ctx context.Context, conversationID string, content string, atUserIDs []string) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := "https://api.dingtalk.com/v1.0/robot/groupMessages/send"
	reqBody := map[string]interface{}{
		"robotCode":          c.appKey,
		"openConversationId": conversationID,
		"msgKey":             "sampleText",
		"msgParam": map[string]string{
			"content": content,
		},
	}

	return c.sendAPI(ctx, token, apiURL, reqBody)
}

// SendMarkdown 发送 Markdown 消息
func (c *WebhookClient) SendMarkdown(ctx context.Context, receiver string, title, content string, atUserIDs []string) error {
	// 优先使用 sessionWebhook
	if webhookURL := c.getSessionWebhook(receiver); webhookURL != "" {
		return c.sendViaWebhook(ctx, webhookURL, &MarkdownMessage{
			MsgType: "markdown",
			Markdown: struct {
				Title string `json:"title"`
				Text  string `json:"text"`
			}{Title: title, Text: content},
		})
	}

	// 使用 API 发送
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	apiURL := "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"
	reqBody := map[string]interface{}{
		"robotCode": c.appKey,
		"userIds":   []string{receiver},
		"msgKey":    "sampleMarkdown",
		"msgParam": map[string]string{
			"title": title,
			"text":  content,
		},
	}

	return c.sendAPI(ctx, token, apiURL, reqBody)
}

// SendLink 发送链接消息
func (c *WebhookClient) SendLink(ctx context.Context, receiver string, title, content, picURL, messageURL string) error {
	// 优先使用 sessionWebhook
	if webhookURL := c.getSessionWebhook(receiver); webhookURL != "" {
		return c.sendViaWebhook(ctx, webhookURL, &LinkMessage{
			MsgType: "link",
			Link: struct {
				Title      string `json:"title"`
				Text       string `json:"text"`
				MessageURL string `json:"messageUrl"`
				PicURL     string `json:"picUrl,omitempty"`
			}{Title: title, Text: content, PicURL: picURL, MessageURL: messageURL},
		})
	}

	return fmt.Errorf("link message via API not implemented")
}

// SendImage Webhook 模式不支持图片上传，降级为 Markdown
func (c *WebhookClient) SendImage(ctx context.Context, receiver string, imageData []byte, atUserIDs []string) error {
	return fmt.Errorf("image message not supported in webhook mode, use StreamClient instead")
}

// getSessionWebhook 获取缓存的 sessionWebhook URL
func (c *WebhookClient) getSessionWebhook(userID string) string {
	c.webhookMu.RLock()
	defer c.webhookMu.RUnlock()

	info, ok := c.sessionWebhooks[userID]
	if !ok || time.Now().After(info.expiredTime) {
		return ""
	}
	return info.webhookURL
}

// sendViaWebhook 通过 sessionWebhook 发送消息
func (c *WebhookClient) sendViaWebhook(ctx context.Context, webhookURL string, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	log.Debugf("[DingTalk Webhook] sendViaWebhook: url=%s, body=%s", webhookURL, truncate(string(body), 500))

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk webhook request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	log.Debugf("[DingTalk Webhook] sendViaWebhook response: status=%d, body=%s", resp.StatusCode, truncate(string(respBody), 500))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk webhook returned status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var result SendMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Warnf("[DingTalk Webhook] sendViaWebhook: failed to parse response as SendMessageResponse: %v, raw=%s", err, truncate(string(respBody), 200))
		return nil // HTTP 200 但 JSON 格式不是预期的，sessionWebhook 回复可能不返回 errcode
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("send message via webhook failed: errcode=%d, errmsg=%s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// sendAPI 通过钉钉 API 发送消息
func (c *WebhookClient) sendAPI(ctx context.Context, token, apiURL string, reqBody interface{}) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	log.Debugf("[DingTalk Webhook] sendAPI: url=%s, body=%s", apiURL, truncate(string(body), 500))

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	log.Debugf("[DingTalk Webhook] sendAPI response: status=%d, body=%s", resp.StatusCode, truncate(string(respBody), 500))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk API returned status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var result SendMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Warnf("[DingTalk Webhook] sendAPI: failed to parse response as SendMessageResponse: %v, raw=%s", err, truncate(string(respBody), 200))
		return nil // HTTP 200 但 JSON 格式不是预期的
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("send message via API failed: errcode=%d, errmsg=%s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// SignURL 生成带签名的 Webhook URL（用于自定义机器人）
func SignURL(webhookURL, secret string) (string, error) {
	if secret == "" {
		return webhookURL, nil
	}

	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	stringToSign := timestamp + "\n" + secret

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))

	if strings.Contains(webhookURL, "?") {
		return fmt.Sprintf("%s&timestamp=%s&sign=%s", webhookURL, timestamp, sign), nil
	}
	return fmt.Sprintf("%s?timestamp=%s&sign=%s", webhookURL, timestamp, sign), nil
}
