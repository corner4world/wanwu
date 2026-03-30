package main

import (
	"log"
	"os"
	"sync"

	ccb "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"
)

var (
	client  cozeloop.Client
	handler callbacks.Handler
	mu      sync.Mutex
	initErr error
	inited  bool
)

// Init initializes the CozeLoop client and registers the global handler
// Environment variables required:
// - COZELOOP_WORKSPACE_ID: your workspace id
// - COZELOOP_API_TOKEN: your token
// - COZELOOP_ENABLED: set to "true" to enable (optional, default: false)
// - COZELOOP_API_BASE_URL: API base URL for local deployment (optional, e.g., http://localhost:8082)
func Init() error {
	mu.Lock()
	defer mu.Unlock()

	if inited {
		return initErr
	}

	enabled := os.Getenv("COZELOOP_ENABLED")
	if enabled != "true" {
		log.Println("[CozeLoop] CozeLoop is disabled (set COZELOOP_ENABLED=true to enable)")
		inited = true
		return nil
	}

	workspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
	apiToken := os.Getenv("COZELOOP_API_TOKEN")
	apiBaseURL := os.Getenv("COZELOOP_API_BASE_URL")

	if workspaceID == "" || apiToken == "" {
		log.Println("[CozeLoop] Warning: COZELOOP_WORKSPACE_ID or COZELOOP_API_TOKEN not set, skipping initialization")
		inited = true
		return nil
	}

	opts := []cozeloop.Option{
		cozeloop.WithWorkspaceID(workspaceID),
		cozeloop.WithAPIToken(apiToken),
	}

	// Add API base URL for local deployment if specified
	if apiBaseURL != "" {
		opts = append(opts, cozeloop.WithAPIBaseURL(apiBaseURL))
		log.Printf("[CozeLoop] Using custom API base URL: %s", apiBaseURL)
	}

	var err error
	client, err = cozeloop.NewClient(opts...)
	if err != nil {
		log.Printf("[CozeLoop] Failed to create cozeloop client: %v", err)
		initErr = err
		inited = true
		return err
	}

	handler = ccb.NewLoopHandler(client)
	callbacks.AppendGlobalHandlers(handler)

	inited = true
	log.Println("[CozeLoop] CozeLoop initialized successfully")

	return nil
}
