package wanwu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client 万悟平台代理客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建万悟代理客户端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- 数据结构 ---

// wanwuResponse 万悟 BFF 统一响应格式
type wanwuResponse struct {
	Code int64           `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

// ChatRequest 智能体对话请求
type ChatRequest struct {
	UUID           string `json:"uuid"`
	ConversationID string `json:"conversation_id"`
	Query          string `json:"query"`
	Stream         bool   `json:"stream"`
}

// CreateConversationRequest 创建会话请求
type CreateConversationRequest struct {
	UUID  string `json:"uuid"`
	Title string `json:"title"`
}

// CreateConversationResponse 创建会话响应
type CreateConversationResponse struct {
	ConversationID string `json:"conversationId"`
}

// ConversationManager 会话管理器
// 维护 (channelID + platformUserID + appType) -> conversationId 的映射
type ConversationManager struct {
	mu    sync.RWMutex
	cache map[string]string // key: channelID:userID:appType -> value: conversationId/threadId
}

// NewConversationManager 创建会话管理器
func NewConversationManager() *ConversationManager {
	return &ConversationManager{
		cache: make(map[string]string),
	}
}

// cacheKey 生成缓存 key
func (m *ConversationManager) cacheKey(channelID, userID, appType string) string {
	return channelID + ":" + userID + ":" + appType
}

// GetConversationID 获取已有的会话 ID
func (m *ConversationManager) GetConversationID(channelID, userID, appType string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.cache[m.cacheKey(channelID, userID, appType)]
	return id, ok
}

// SetConversationID 设置会话 ID 映射
func (m *ConversationManager) SetConversationID(channelID, userID, appType, conversationID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[m.cacheKey(channelID, userID, appType)] = conversationID
}

// --- API 方法 ---

// CreateConversation 调用智能体创建对话接口
func (c *Client) CreateConversation(ctx context.Context, apiKey string, req *CreateConversationRequest) (*CreateConversationResponse, error) {
	reqURL := fmt.Sprintf("%s/openapi/v1/agent/conversation", c.baseURL)

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create conversation request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call create conversation api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create conversation api returned status %d: %s", resp.StatusCode, string(body))
	}

	var wanwuResp wanwuResponse
	if err := json.Unmarshal(body, &wanwuResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if wanwuResp.Code != 0 {
		return nil, fmt.Errorf("create conversation api error: code=%d, msg=%s", wanwuResp.Code, wanwuResp.Msg)
	}

	var convResp CreateConversationResponse
	if err := json.Unmarshal(wanwuResp.Data, &convResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation data: %w", err)
	}

	return &convResp, nil
}

// ChatWithAgent 调用智能体对话接口（SSE 流式）
func (c *Client) ChatWithAgent(ctx context.Context, apiKey string, chatReq *ChatRequest) (*http.Response, error) {
	reqURL := fmt.Sprintf("%s/openapi/v1/agent/chat", c.baseURL)

	bodyBytes, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	// SSE 流式不设置超时
	req.Header.Set("Accept", "text/event-stream")

	// 使用独立的 HTTP 客户端，不设置超时（用于 SSE 流式响应）
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call chat api: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("chat api returned status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// --- WGA 通用智能体数据结构 ---

// WGACreateConversationRequest WGA 创建对话请求
type WGACreateConversationRequest struct {
	Title     string `json:"title"`
	ModelUuid string `json:"modelUuid"`
}

// WGACreateConversationResponse WGA 创建对话响应
type WGACreateConversationResponse struct {
	ThreadID string `json:"threadId"`
}

// WGAChatRequest WGA 对话请求
type WGAChatRequest struct {
	ThreadID  string       `json:"threadId"`
	Messages  []WGAMessage `json:"messages"`
	ModelUuid string       `json:"modelUuid,omitempty"`
}

// WGAMessage WGA 消息
type WGAMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// --- WGA API 方法 ---

// CreateWGAConversation 调用 WGA 创建对话接口
func (c *Client) CreateWGAConversation(ctx context.Context, apiKey string, req *WGACreateConversationRequest) (*WGACreateConversationResponse, error) {
	reqURL := fmt.Sprintf("%s/openapi/v1/wga/conversation", c.baseURL)

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wga create conversation request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call wga create conversation api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wga create conversation api returned status %d: %s", resp.StatusCode, string(body))
	}

	var wanwuResp wanwuResponse
	if err := json.Unmarshal(body, &wanwuResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if wanwuResp.Code != 0 {
		return nil, fmt.Errorf("wga create conversation api error: code=%d, msg=%s", wanwuResp.Code, wanwuResp.Msg)
	}

	var convResp WGACreateConversationResponse
	if err := json.Unmarshal(wanwuResp.Data, &convResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wga conversation data: %w", err)
	}

	return &convResp, nil
}

// ChatWithWGA 调用 WGA 对话接口（SSE 流式 AG-UI 协议）
func (c *Client) ChatWithWGA(ctx context.Context, apiKey string, chatReq *WGAChatRequest) (*http.Response, error) {
	reqURL := fmt.Sprintf("%s/openapi/v1/wga/conversation/chat", c.baseURL)

	bodyBytes, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wga chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	// 使用独立的 HTTP 客户端，不设置超时（用于 SSE 流式响应）
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call wga chat api: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("wga chat api returned status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}
