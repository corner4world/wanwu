package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/UnicomAI/wanwu/internal/channel-service/adapter/types"
	"github.com/UnicomAI/wanwu/pkg/log"
)

const (
	// DefaultBaseURL 微信 openclaw 默认地址
	DefaultBaseURL         = "https://ilinkai.weixin.qq.com"
	SessionExpiredErrCode  = -14
	MaxTextChunkSize       = 2000
	DefaultLongPollTimeout = 40
)

// WeChatAdapter 微信平台适配器
// 使用 openclaw 协议通过 Long Poll 接收消息
type WeChatAdapter struct {
	mu        sync.RWMutex
	config    types.AdapterConfig
	connected bool
	handler   types.MessageHandler

	// openclaw 客户端状态
	baseURL   string
	token     string
	accountID string
	status    string

	// 轮询控制
	polling  bool
	pollBuf  string
	stopChan chan struct{}

	// 消息去重：记录最近处理过的消息指纹
	seenMsgs map[string]struct{}

	// HTTP 客户端
	httpClient *http.Client
}

// NewWeChatAdapter 创建微信适配器
func NewWeChatAdapter() *WeChatAdapter {
	return &WeChatAdapter{
		status:     "offline",
		httpClient: &http.Client{Timeout: 60 * time.Second},
		stopChan:   make(chan struct{}),
		seenMsgs:   make(map[string]struct{}),
	}
}

// Connect 连接到微信平台
func (w *WeChatAdapter) Connect(config types.AdapterConfig) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.config = config
	w.baseURL = config.BaseUrl
	if w.baseURL == "" {
		w.baseURL = DefaultBaseURL
	}
	w.token = config.Token
	w.accountID = config.AccountId

	// 如果有 token，验证会话并开始轮询
	if w.token != "" {
		w.status = "logged_in"
		w.connected = true
		w.polling = true
		go w.pollLoop(context.Background())
		log.Infof("[WeChat] Adapter connected for channel %s (with token, polling started)", config.ChannelID)
		return nil
	}

	// 没有 token，设置为等待登录状态
	w.status = "waiting_login"
	w.connected = true
	log.Infof("[WeChat] Adapter connected for channel %s (waiting for login)", config.ChannelID)
	return nil
}

// Disconnect 断开微信连接
func (w *WeChatAdapter) Disconnect() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.polling = false
	w.status = "offline"
	w.connected = false

	select {
	case <-w.stopChan:
		// already closed
	default:
		close(w.stopChan)
	}

	log.Infof("[WeChat] Adapter disconnected for channel %s", w.config.ChannelID)
	return nil
}

// IsConnected 检查是否已连接
func (w *WeChatAdapter) IsConnected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.connected
}

// GetAccountInfo 获取微信账号信息
func (w *WeChatAdapter) GetAccountInfo() (accountId, nickname, avatar string, err error) {
	return w.accountID, "", "", nil
}

// SendMessage 向微信用户发送文本消息
func (w *WeChatAdapter) SendMessage(ctx context.Context, userID, content string, extra map[string]string) error {
	w.mu.RLock()
	token := w.token
	baseURL := w.baseURL
	w.mu.RUnlock()

	if token == "" {
		return fmt.Errorf("wechat not logged in")
	}

	// 提取 contextToken
	contextToken := ""
	if extra != nil {
		contextToken = extra["contextToken"]
	}

	// 分块发送（微信消息长度限制）
	chunks := chunkText(content, MaxTextChunkSize)
	for i, chunk := range chunks {
		if err := w.sendTextMessage(ctx, baseURL, token, userID, chunk, contextToken); err != nil {
			return fmt.Errorf("send chunk %d failed: %w", i, err)
		}
	}

	return nil
}

// OnMessage 注册消息回调
func (w *WeChatAdapter) OnMessage(handler types.MessageHandler) {
	w.handler = handler
}

// CreateStreamSender 创建流式回复发送器
// 微信不支持流式卡片，返回 nil 降级为非流式 SendMessage
func (w *WeChatAdapter) CreateStreamSender(ctx context.Context, userID string, extra map[string]string) types.StreamSender {
	return nil
}

// SetLoginInfo 设置登录信息（扫码登录成功后调用）
func (w *WeChatAdapter) SetLoginInfo(token, baseURL, accountID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.token = token
	if baseURL != "" {
		w.baseURL = baseURL
	}
	w.accountID = accountID
	w.status = "logged_in"

	// 开始轮询
	if !w.polling {
		w.polling = true
		w.stopChan = make(chan struct{})
		go w.pollLoop(context.Background())
	}

	log.Infof("[WeChat] Login info set for account %s", accountID)
}

// GetStatus 获取状态
func (w *WeChatAdapter) GetStatus() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

// GetQRCode 获取登录二维码
func (w *WeChatAdapter) GetQRCode(ctx context.Context) (*QRCodeResult, error) {
	w.mu.RLock()
	baseURL := w.baseURL
	w.mu.RUnlock()

	apiURL := baseURL + "/ilink/bot/get_bot_qrcode?bot_type=3"

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("iLink-App-ClientVersion", "1")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get QR code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get QR code failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		QRCode           string `json:"qrcode"`
		QRCodeImgContent string `json:"qrcode_img_content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse QR code response failed: %w", err)
	}

	return &QRCodeResult{
		QRCodeID:  result.QRCode,
		QRCodeURL: result.QRCodeImgContent,
	}, nil
}

// CheckQRStatus 检查二维码状态
func (w *WeChatAdapter) CheckQRStatus(ctx context.Context, qrcodeID string) (*QRStatusResult, error) {
	w.mu.RLock()
	baseURL := w.baseURL
	w.mu.RUnlock()

	apiURL := fmt.Sprintf("%s/ilink/bot/get_qrcode_status?qrcode=%s", baseURL, qrcodeID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("iLink-App-ClientVersion", "1")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check QR status failed: status=%d", resp.StatusCode)
	}

	var result struct {
		Status      string `json:"status"`
		BotToken    string `json:"bot_token"`
		BaseURL     string `json:"base_url"`
		ILinkBotID  string `json:"ilink_bot_id"`
		ILinkUserID string `json:"ilink_user_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &QRStatusResult{
		Status:      result.Status,
		BotToken:    result.BotToken,
		BaseURL:     result.BaseURL,
		ILinkBotID:  result.ILinkBotID,
		ILinkUserID: result.ILinkUserID,
	}, nil
}

// --- 内部方法 ---

// pollLoop 消息轮询
func (w *WeChatAdapter) pollLoop(ctx context.Context) {
	log.Infof("[WeChat] Starting poll loop for account %s", w.accountID)
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for w.polling {
		resp, err := w.getUpdates(ctx, w.pollBuf, DefaultLongPollTimeout*time.Second)
		if err != nil {
			log.Errorf("[WeChat] getUpdates error: %v", err)
			time.Sleep(backoff)
			if backoff*2 > maxBackoff {
				backoff = maxBackoff
			} else {
				backoff = backoff * 2
			}
			continue
		}

		if resp.ErrCode == SessionExpiredErrCode {
			log.Errorf("[WeChat] Session expired for account %s", w.accountID)
			w.mu.Lock()
			w.token = ""
			w.status = "session_expired"
			w.polling = false
			w.mu.Unlock()
			return
		}

		backoff = 1 * time.Second
		w.pollBuf = resp.GetUpdatesBuf

		for _, msg := range resp.Msgs {
			// 消息去重：用 from_user_id + context_token + create_time_ms 组合指纹
			msgFingerprint := fmt.Sprintf("%s:%s:%d", msg.FromUserID, msg.ContextToken, msg.CreateTimeMs)
			w.mu.Lock()
			if _, seen := w.seenMsgs[msgFingerprint]; seen {
				w.mu.Unlock()
				continue
			}
			w.seenMsgs[msgFingerprint] = struct{}{}
			// 限制去重缓存大小，避免内存泄漏
			if len(w.seenMsgs) > 1000 {
				w.seenMsgs = make(map[string]struct{})
			}
			w.mu.Unlock()

			platformMsg := w.convertMessage(&msg)
			if w.handler != nil {
				go func() {
					if err := w.handler(ctx, platformMsg); err != nil {
						log.Errorf("[WeChat] Message handler error: %v", err)
					}
				}()
			}
		}
	}

	log.Infof("[WeChat] Poll loop stopped for account %s", w.accountID)
}

// getUpdates 获取更新
func (w *WeChatAdapter) getUpdates(ctx context.Context, pollBuf string, timeout time.Duration) (*GetUpdatesResponse, error) {
	w.mu.RLock()
	token := w.token
	baseURL := w.baseURL
	w.mu.RUnlock()

	reqBody := map[string]interface{}{
		"get_updates_buf": pollBuf,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/ilink/bot/getupdates", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	w.setHeaders(httpReq, token)

	client := &http.Client{Timeout: timeout + 5*time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	var result GetUpdatesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// sendTextMessage 发送文本消息
func (w *WeChatAdapter) sendTextMessage(ctx context.Context, baseURL, token, toUserID, content, contextToken string) error {
	req := &SendRequest{
		Msg: SendMsg{
			ToUserID:     toUserID,
			ClientID:     generateClientID(),
			MessageType:  2, // BOT
			MessageState: 2, // FINISH
			ContextToken: contextToken,
			ItemList: []MessageItem{{
				Type:     1, // Text
				TextItem: &TextItem{Text: content},
			}},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/ilink/bot/sendmessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	w.setHeaders(httpReq, token)

	resp, err := w.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("send message failed: %s", string(respBody))
	}

	var sendResp SendResponse
	if err := json.Unmarshal(respBody, &sendResp); err == nil {
		if sendResp.Ret != 0 {
			return fmt.Errorf("send message failed: ret=%d, errmsg=%s", sendResp.Ret, sendResp.ErrMsg)
		}
	}

	return nil
}

// setHeaders 设置请求头
func (w *WeChatAdapter) setHeaders(req *http.Request, token string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AuthorizationType", "ilink_bot_token")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("iLink-App-ClientVersion", "1")
}

// convertMessage 转换 openclaw 消息为平台消息
func (w *WeChatAdapter) convertMessage(msg *WeixinMessage) *types.PlatformMessage {
	result := &types.PlatformMessage{
		ChannelID:   w.config.ChannelID,
		UserID:      msg.FromUserID,
		ChannelType: types.ChannelTypeWeChat,
		MsgType:     "text",
		Extra:       make(map[string]string),
	}

	// 提取文本内容
	for _, item := range msg.ItemList {
		switch item.Type {
		case 1: // Text
			if item.TextItem != nil {
				result.Content = item.TextItem.Text
			}
		case 2: // Image
			result.Content = "[图片]"
			result.MsgType = "image"
		case 3: // Voice
			if item.VoiceItem != nil && item.VoiceItem.Text != "" {
				result.Content = item.VoiceItem.Text
			} else {
				result.Content = "[语音]"
			}
			result.MsgType = "voice"
		case 4: // File
			if item.FileItem != nil {
				result.Content = item.FileItem.FileName
			}
			result.MsgType = "file"
		case 5: // Video
			result.Content = "[视频]"
			result.MsgType = "video"
		}
	}

	if result.Content == "" {
		result.Content = "[未知消息]"
	}

	// 保存上下文 token（用于回复同一会话）
	result.Extra["contextToken"] = msg.ContextToken
	result.Extra["fromUserId"] = msg.FromUserID

	return result
}

// chunkText 分块文本
func chunkText(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxSize {
			chunks = append(chunks, text)
			break
		}
		chunks = append(chunks, text[:maxSize])
		text = text[maxSize:]
	}
	return chunks
}

func generateClientID() string {
	return fmt.Sprintf("wanwu-%d", time.Now().UnixNano())
}

// --- openclaw 协议类型定义 ---

// QRCodeResult 二维码结果
type QRCodeResult struct {
	QRCodeID  string // 二维码 ID
	QRCodeURL string // 二维码图片 URL
	Base64    string // base64 编码的二维码图片
}

// QRStatusResult 二维码扫描状态
type QRStatusResult struct {
	Status      string // waiting/scanned/confirmed/expired
	BotToken    string
	BaseURL     string
	ILinkBotID  string
	ILinkUserID string
}

// WeixinMessage openclaw 微信消息格式
type WeixinMessage struct {
	FromUserID   string        `json:"from_user_id"`
	ToUserID     string        `json:"to_user_id"`
	MessageType  int           `json:"msg_type"`
	MessageState int           `json:"msg_state"`
	ContextToken string        `json:"context_token"`
	CreateTimeMs int64         `json:"create_time_ms"`
	ItemList     []MessageItem `json:"item_list"`
}

// MessageItem 消息项
type MessageItem struct {
	Type      int        `json:"type"`
	TextItem  *TextItem  `json:"text_item,omitempty"`
	ImageItem *ImageItem `json:"image_item,omitempty"`
	VoiceItem *VoiceItem `json:"voice_item,omitempty"`
	FileItem  *FileItem  `json:"file_item,omitempty"`
	VideoItem *VideoItem `json:"video_item,omitempty"`
}

// TextItem 文本项
type TextItem struct {
	Text string `json:"text"`
}

// ImageItem 图片项
type ImageItem struct {
	Media  *CDNMedia `json:"media,omitempty"`
	AESKey string    `json:"aes_key,omitempty"`
}

// VoiceItem 语音项
type VoiceItem struct {
	Media    *CDNMedia `json:"media,omitempty"`
	Playtime int       `json:"playtime,omitempty"`
	Text     string    `json:"text,omitempty"`
}

// FileItem 文件项
type FileItem struct {
	Media    *CDNMedia `json:"media,omitempty"`
	FileName string    `json:"file_name,omitempty"`
	Len      string    `json:"len,omitempty"`
	MD5      string    `json:"md5,omitempty"`
}

// VideoItem 视频项
type VideoItem struct {
	Media *CDNMedia `json:"media,omitempty"`
}

// CDNMedia CDN 媒体信息
type CDNMedia struct {
	MediaID string `json:"media_id"`
	AESKey  string `json:"aes_key"`
	URL     string `json:"url,omitempty"`
}

// SendRequest 发送消息请求
type SendRequest struct {
	Msg SendMsg `json:"msg"`
}

// SendMsg 发送消息
type SendMsg struct {
	FromUserID   string        `json:"from_user_id,omitempty"`
	ToUserID     string        `json:"to_user_id"`
	ClientID     string        `json:"client_id"`
	MessageType  int           `json:"msg_type"`
	MessageState int           `json:"msg_state"`
	ContextToken string        `json:"context_token,omitempty"`
	ItemList     []MessageItem `json:"item_list"`
}

// SendResponse 发送消息响应
type SendResponse struct {
	Ret    int    `json:"ret"`
	ErrMsg string `json:"errmsg,omitempty"`
}

// GetUpdatesResponse 获取更新响应
type GetUpdatesResponse struct {
	Ret           int             `json:"ret"`
	ErrCode       int             `json:"errcode"`
	ErrMsg        string          `json:"errmsg"`
	GetUpdatesBuf string          `json:"get_updates_buf"`
	Msgs          []WeixinMessage `json:"msgs"`
}
