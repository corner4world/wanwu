package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/UnicomAI/wanwu/internal/channel-service/adapter"
	"github.com/UnicomAI/wanwu/internal/channel-service/adapter/types"
	"github.com/UnicomAI/wanwu/internal/channel-service/config"
	"github.com/UnicomAI/wanwu/pkg/log"
)

// CallbackServer HTTP 回调服务器
// 用于接收钉钉 Webhook、微信等平台的 HTTP 回调推送
type CallbackServer struct {
	cfg     *config.Config
	manager *adapter.Manager
	server  *http.Server
}

// NewCallbackServer 创建回调服务器
func NewCallbackServer(cfg *config.Config, mgr *adapter.Manager) *CallbackServer {
	s := &CallbackServer{
		cfg:     cfg,
		manager: mgr,
	}
	return s
}

// Start 启动 HTTP 回调服务器
func (s *CallbackServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// 钉钉 Webhook 回调
	mux.HandleFunc("/callback/v1/channel/dingtalk/", s.handleDingTalkWebhook)

	// 微信回调（阶段四实现）
	mux.HandleFunc("/callback/v1/channel/wechat/", s.handleWeChatWebhook)

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := s.cfg.Callback.Endpoint
	if addr == "" {
		addr = ":8090"
	}

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Errorf("callback server error: %v", err)
		}
	}()

	log.Infof("channel-service callback server started at %s", addr)
	return nil
}

// Stop 停止回调服务器
func (s *CallbackServer) Stop(ctx context.Context) {
	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			log.Errorf("failed to shutdown callback server: %v", err)
		}
	}
}

// handleDingTalkWebhook 处理钉钉 Webhook 回调
// URL 格式: /callback/v1/channel/dingtalk/{channelId}
func (s *CallbackServer) handleDingTalkWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从 URL 中提取 channelId
	path := strings.TrimPrefix(r.URL.Path, "/callback/v1/channel/dingtalk/")
	channelID := strings.TrimSuffix(path, "/")
	if channelID == "" {
		http.Error(w, "missing channel id", http.StatusBadRequest)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("[Callback] Failed to read dingtalk webhook body: %v", err)
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	defer func() { _ = r.Body.Close() }()

	// 获取签名头
	timestamp := r.Header.Get("timestamp")
	sign := r.Header.Get("sign")

	// 获取适配器
	adapterInst, ok := s.manager.GetAdapter(channelID)
	if !ok {
		log.Errorf("[Callback] Adapter not found for channel %s", channelID)
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}

	// 检查适配器是否支持 Webhook
	handler, ok := adapterInst.(types.WebhookHandler)
	if !ok {
		log.Errorf("[Callback] Adapter for channel %s does not support webhook", channelID)
		http.Error(w, "channel does not support webhook", http.StatusBadRequest)
		return
	}

	// 处理 Webhook
	if err := handler.HandleWebhook(r.Context(), body, timestamp, sign); err != nil {
		log.Errorf("[Callback] Failed to handle dingtalk webhook for channel %s: %v", channelID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"errcode": 0,
		"errmsg":  "ok",
	})
}

// handleWeChatWebhook 处理微信回调（阶段四实现）
func (s *CallbackServer) handleWeChatWebhook(w http.ResponseWriter, r *http.Request) {
	// TODO: 微信回调处理
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
