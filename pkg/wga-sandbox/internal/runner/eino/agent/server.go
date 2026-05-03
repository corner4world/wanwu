package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	chatmodel "github.com/UnicomAI/wanwu/pkg/wga-sandbox/internal/runner/eino/agent/chat-model"
	"github.com/UnicomAI/wanwu/pkg/wga-sandbox/internal/runner/eino/agent/shared"
	"github.com/cloudwego/eino/adk"
	"github.com/joho/godotenv"
)

const (
	maxRequestBodyBytes = 10 << 20 // 10MB
	requestTimeout      = 30 * time.Minute
)

type httpServer struct{}

type chatRequest struct {
	Messages []adk.Message `json:"messages"`
}

func newHTTPServer() *httpServer {
	return &httpServer{}
}

func (s *httpServer) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/chat", s.handleChat)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func loadEnvFromFile(path string) map[string]string {
	envMap, err := godotenv.Read(path)
	if err != nil {
		log.Printf("[Env] Failed to read .env from %s: %v", shared.SanitizeForLog(path), err)
		return make(map[string]string)
	}
	return envMap
}

func getEnvValue(envMap map[string]string, key string) string {
	if val, ok := envMap[key]; ok {
		return val
	}
	return os.Getenv(key)
}

func (s *httpServer) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspace := r.URL.Query().Get("workspace")
	if workspace == "" {
		http.Error(w, "workspace parameter is required", http.StatusBadRequest)
		return
	}

	agentType := r.URL.Query().Get("agent_type")

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Chat] Failed to decode request body: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		log.Printf("[Chat] Empty messages received")
		http.Error(w, "messages is required", http.StatusBadRequest)
		return
	}

	log.Printf("[Chat] Workspace: %s, Messages count: %d", shared.SanitizeForLog(workspace), len(req.Messages))

	envPath := filepath.Join(workspace, ".env")
	envMap := loadEnvFromFile(envPath)

	if agentType == "" {
		agentType = getEnvValue(envMap, "EINO_AGENT_TYPE")
	}
	if agentType == "" {
		agentType = "chat-model"
	}

	log.Printf("[Chat] Agent type: %s", shared.SanitizeForLog(agentType))

	cfg := shared.AppConfig{
		Workspace: workspace,
		APIKey:    getEnvValue(envMap, "OPENAI_API_KEY"),
		BaseURL:   getEnvValue(envMap, "OPENAI_BASE_URL"),
		ModelID:   getEnvValue(envMap, "OPENAI_MODEL_ID"),
	}

	var app shared.AgentApp
	var err error

	switch agentType {
	case "chat-model":
		app, err = chatmodel.NewApp(context.Background(), cfg)
	default:
		http.Error(w, "invalid agent_type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("[Chat] Failed to create app: %v", err)
		http.Error(w, fmt.Sprintf("failed to create app: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("[Chat] Failed to close app: %v", err)
		}
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("[Chat] Streaming not supported")
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	sessionID := fmt.Sprintf("session-%s", uuid.New().String())
	log.Printf("[Chat] Starting runner - SessionID: %s", sessionID)

	iter := app.Query(ctx, req.Messages)
	if iter == nil {
		log.Printf("[Chat] Query returned nil iterator")
		return
	}
	sseWriter := shared.NewHTTPSSEWriter(w, flusher)
	shared.ProcessEvents(iter, sseWriter)
	log.Printf("[Chat] Request completed - SessionID: %s", sessionID)
}
