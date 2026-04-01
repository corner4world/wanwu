package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	model_service "github.com/UnicomAI/wanwu/api/proto/model-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	ag_ui_util "github.com/UnicomAI/wanwu/pkg/ag-ui-util"
	"github.com/UnicomAI/wanwu/pkg/constant"
	http_client "github.com/UnicomAI/wanwu/pkg/http-client"
	"github.com/UnicomAI/wanwu/pkg/log"
	mp "github.com/UnicomAI/wanwu/pkg/model-provider"
	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/UnicomAI/wanwu/pkg/wga"
	wga_persistent "github.com/UnicomAI/wanwu/pkg/wga-persistent"
	wga_option "github.com/UnicomAI/wanwu/pkg/wga/wga-option"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GeneralAgentCopilotRuntimeInfo(_ *gin.Context) *response.GeneralAgentCopilotRuntimeInfoResp {
	return &response.GeneralAgentCopilotRuntimeInfoResp{
		Version: "0.0.1",
		Agents: map[string]response.GeneralAgentCopilotRuntimeInfoAgent{
			"default": {
				Name:        "万悟通用智能体",
				Description: "万悟通用智能体",
				ClassName:   "WGA",
			},
		},
		Mode:                          "sse",
		AudioFileTranscriptionEnabled: false,
		A2UIEnabled:                   false,
	}
}

func GeneralAgentConversationChat(ctx *gin.Context, userId, orgId string, req request.GeneralAgentConversationChatReq) error {
	// 验证 threadId 是否存在
	existsResp, err := assistant.WgaConversationExists(ctx.Request.Context(), &assistant_service.WgaConversationExistsReq{
		ThreadId: req.ThreadID,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return err
	}
	if !existsResp.Exists {
		return fmt.Errorf("threadId not found: %s", req.ThreadID)
	}

	runID := uuid.NewString()

	// 获取 WGA Conversation 配置
	wgaConversationConfigResp, err := assistant.GetWgaConversationConfig(ctx.Request.Context(), &assistant_service.GetWgaConversationConfigReq{
		ThreadId: req.ThreadID,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})

	// 获取 WGA 配置
	wgaConfigResp, err := assistant.GetWgaConfig(ctx.Request.Context(), &assistant_service.GetWgaConfigReq{
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})

	//if err != nil {
	//	return err
	//}
	//wgaUserConfig := wgaConfigResp.Config

	// TODO测试用
	var wgaConversationConfig *assistant_service.WgaConversationConfig
	var wgaConfig *assistant_service.WgaConfig
	if err != nil {
		log.Warnf("get WGA config failed: %v", err)
	} else {
		wgaConversationConfig = wgaConversationConfigResp.Config
		wgaConfig = wgaConfigResp.Config
	}

	// 获取最后一条用户提问，并预处理 content
	var currentUserMessage *request.GeneralAgentConversationMessage
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			msg := req.Messages[i]
			msg.Content = formatMessageContent(msg.Content)
			currentUserMessage = &msg
			break
		}
	}

	// 从 ES 获取历史记录
	historyMessages := getHistoryMessages(ctx, userId, orgId, req.ThreadID)

	// 拼接历史记录和用户提问
	allMessages := append(historyMessages, *currentUserMessage)

	// 过滤消息
	filteredMessages := filterMessages(allMessages)

	// ---- 测试用 ---
	//agentID := config.WgaCfg().AgentID
	////构建 WGA 选项
	//opts := buildWgaOptions(ctx, config.WgaCfg(), req.ThreadID, runID, filteredMessages)
	////运行 WGA 对话
	//_, iter, err := wga.Run(ctx.Request.Context(), agentID, opts...)

	// ---- 用户配置 ---
	// 构建 WGA 选项
	opts, err := buildWgaOptionsFromUserConfig(ctx, wgaConversationConfig, wgaConfig, req.ThreadID, runID, filteredMessages, currentUserMessage)
	if err != nil {
		log.Errorf("[GeneralAgentConversationChat] threadId=%s, runId=%s, buildWgaOptionsFromUserConfig error: %v", req.ThreadID, runID, err)
		return err
	}

	_, iter, err := wga.Run(ctx.Request.Context(), config.WgaCfg().AgentID, opts...)
	if err != nil {
		log.Errorf("[GeneralAgentConversationChat] threadId=%s, runId=%s, wga.Run error: %v", req.ThreadID, runID, err)
		return err
	}

	tr := ag_ui_util.NewEinoTranslator(req.ThreadID, runID)
	eventCh := tr.TranslateStream(ctx.Request.Context(), iter)

	processorConfig := &ag_ui_util.ProcessorConfig{
		ToolNameMapper: map[string]string{
			"transfer_to_agent": "正在交给专业智能体",
		},
		ExcludedAgentNames: []string{
			"default",
			"Supervisor Agent",
		},
		ResultFormatters: map[string]func(string) string{
			"bochaWebSearch":      WgaFormatBochaWebSearchResult,
			"tavily_basic_search": WgaFormatTavilySearchResult,
			"tavily_deep_search":  WgaFormatTavilySearchResult,
			"tavily_day_search":   WgaFormatTavilySearchResult,
			"tavily_week_search":  WgaFormatTavilySearchResult,
			"tavily_image_search": WgaFormatTavilySearchResult,
			"tavily_date_search":  WgaFormatTavilySearchResult,
		},
	}

	processor := ag_ui_util.NewStreamProcessor(processorConfig)
	processedEventCh, historyCh := processor.Process(ctx.Request.Context(), eventCh)

	// 保存智能体返回的消息
	go saveWgaChatHistory(context.Background(), historyCh, *currentUserMessage, userId, orgId, req.ThreadID, runID)

	outputCh := injectWgaWorkspaceActivity(
		ctx.Request.Context(),
		processedEventCh,
		req.ThreadID,
		runID,
		config.WgaCfg().Persistent.BaseDir,
		config.WgaCfg().Persistent.Enabled,
	)

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	ctx.Stream(func(w io.Writer) bool {
		select {
		case line, ok := <-outputCh:
			if !ok {
				log.Infof("[GeneralAgentConversationChat] threadId=%s, runId=%s, outputCh closed", req.ThreadID, runID)
				return false
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", line)
			return true
		case <-ctx.Request.Context().Done():
			return false
		}
	})
	return nil
}

func buildWgaOptions(ctx *gin.Context, cfg *config.WgaConfig, threadID, runID string, messages []request.GeneralAgentConversationMessage) []wga_option.Option {
	opts := []wga_option.Option{
		wga_option.WithModelConfig(wga_option.ModelConfig{
			Provider:     cfg.Model.Provider,
			ProviderName: cfg.Model.ProviderName,
			BaseURL:      cfg.Model.BaseURL,
			APIKey:       cfg.Model.APIKey,
			Model:        cfg.Model.Model,
			ModelName:    cfg.Model.ModelName,
		}),
		wga_option.WithRunSession(wga_option.RunSession{
			ThreadID: threadID,
			RunID:    runID,
		}),
	}

	if cfg.Persistent.Enabled {
		// 持久化存储
		mode := wga_persistent.ModeVersioned
		if cfg.Persistent.Mode == string(wga_persistent.ModeOverwrite) {
			mode = wga_persistent.ModeOverwrite
		}
		store, err := wga_persistent.NewStore(mode, cfg.Persistent.BaseDir, threadID)
		if err == nil {
			// 创建目录并从上一次输出复制
			_, info, err := store.GetRunDir(runID, wga_persistent.WithMkdir(true))
			if err == nil {
				opts = append(opts,
					wga_option.WithInputDir(filepath.Clean(info.Dir)+"/."),
					wga_option.WithOutputDir(info.Dir),
				)
			}
		}
	}

	// 传递历史消息
	if len(messages) > 0 {
		msgs := make([]adk.Message, len(messages))
		for i, msg := range messages {
			msgs[i] = convertWgaMessage(ctx, msg.Role, msg.Content)
		}
		opts = append(opts, wga_option.WithMessages(msgs))
	}

	for _, tool := range cfg.Tools {
		opts = append(opts, wga_option.WithToolConfig(wga_option.ToolConfig{
			Title:   tool.Title,
			APIAuth: &tool.APIAuth,
		}))
	}

	return opts
}

func buildWgaOptionsFromUserConfig(ctx *gin.Context, wgaConversationConfig *assistant_service.WgaConversationConfig, wgaConfig *assistant_service.WgaConfig, threadID, runID string, messages []request.GeneralAgentConversationMessage, currentUserMessage *request.GeneralAgentConversationMessage) ([]wga_option.Option, error) {
	opts := []wga_option.Option{
		wga_option.WithRunSession(wga_option.RunSession{
			ThreadID: threadID,
			RunID:    runID,
		}),
	}

	if wgaConversationConfig != nil && wgaConversationConfig.ModelConfig != nil && wgaConversationConfig.ModelConfig.ModelId != "" {
		modelInfo, err := model.GetModel(ctx.Request.Context(), &model_service.GetModelReq{ModelId: wgaConversationConfig.ModelConfig.ModelId})
		if err != nil {
			return nil, err
		}
		endpoint := mp.ToModelEndpoint(wgaConversationConfig.ModelConfig.ModelId, wgaConversationConfig.ModelConfig.Model)
		modelURL, _ := endpoint["model_url"].(string)
		var APIKey string
		modelConfig, err := mp.ToModelConfig(modelInfo.Provider, modelInfo.ModelType, modelInfo.ProviderConfig)
		if err == nil {
			cfg := make(map[string]any)
			if b, err := json.Marshal(modelConfig); err == nil {
				if err = json.Unmarshal(b, &cfg); err == nil {
					if apiKey, ok := cfg["apiKey"].(string); ok {
						APIKey = apiKey
					}
				}
			}
		}
		var modelParams *mp_common.LLMParams
		if wgaConversationConfig.ModelConfig.Config != "" {
			llmParams, _, err := mp.ToModelParams(wgaConversationConfig.ModelConfig.Provider, wgaConversationConfig.ModelConfig.ModelType, wgaConversationConfig.ModelConfig.Config)
			if err == nil && llmParams != nil {
				modelParams = llmParams.(*mp_common.LLMParams)
			}
		}
		opts = append(opts, wga_option.WithModelConfig(wga_option.ModelConfig{
			Provider:     wgaConversationConfig.ModelConfig.Provider,
			ProviderName: wgaConversationConfig.ModelConfig.Provider,
			BaseURL:      modelURL,
			APIKey:       APIKey,
			Model:        wgaConversationConfig.ModelConfig.Model,
			ModelName:    wgaConversationConfig.ModelConfig.Model,
			Params:       modelParams,
		}))
		log.Infof("[modelConfig] modelConfig=%v,modelURL=%s,apiKey=%s,modelParams=%v", wgaConversationConfig.ModelConfig, modelURL, APIKey, modelParams)
	} else {
		opts = []wga_option.Option{
			wga_option.WithModelConfig(wga_option.ModelConfig{
				Provider:     config.WgaCfg().Model.Provider,
				ProviderName: config.WgaCfg().Model.ProviderName,
				BaseURL:      config.WgaCfg().Model.BaseURL,
				APIKey:       config.WgaCfg().Model.APIKey,
				Model:        config.WgaCfg().Model.Model,
				ModelName:    config.WgaCfg().Model.Model,
			}),
			wga_option.WithRunSession(wga_option.RunSession{
				ThreadID: threadID,
				RunID:    runID,
			}),
		}
	}

	// 下载用户消息中的URL文件到inputDir
	if config.WgaCfg().Persistent.Enabled && currentUserMessage != nil {
		urls := extractURLsFromContent(currentUserMessage.Content)
		if len(urls) > 0 {
			mode := wga_persistent.ModeVersioned
			if config.WgaCfg().Persistent.Mode == string(wga_persistent.ModeOverwrite) {
				mode = wga_persistent.ModeOverwrite
			}
			store, err := wga_persistent.NewStore(mode, config.WgaCfg().Persistent.BaseDir, threadID)
			if err == nil {
				_, info, err := store.GetRunDir(runID, wga_persistent.WithMkdir(true))
				if err == nil {
					//TODO 测试是否正确
					inputDir := filepath.Clean(info.Dir)
					if err := downloadURLsToDir(urls, inputDir); err != nil {
						log.Errorf("[GeneralAgentConversationChat] download URLs to inputDir failed: %v", err)
					}
					opts = append(opts,
						wga_option.WithInputDir(inputDir),
						wga_option.WithOutputDir(info.Dir),
					)
				}
			}
		}
	}

	// 持久化存储（无URL下载时）
	if config.WgaCfg().Persistent.Enabled {
		// 持久化存储
		mode := wga_persistent.ModeVersioned
		if config.WgaCfg().Persistent.Mode == string(wga_persistent.ModeOverwrite) {
			mode = wga_persistent.ModeOverwrite
		}
		store, err := wga_persistent.NewStore(mode, config.WgaCfg().Persistent.BaseDir, threadID)
		if err == nil {
			// 创建目录并从上一次输出复制
			_, info, err := store.GetRunDir(runID, wga_persistent.WithMkdir(true))
			if err == nil {
				opts = append(opts,
					wga_option.WithInputDir(filepath.Clean(info.Dir)+"/."),
					wga_option.WithOutputDir(info.Dir),
				)
			}
		}
	}

	// 传递历史消息
	if len(messages) > 0 {
		msgs := make([]adk.Message, len(messages))
		for i, msg := range messages {
			msgs[i] = convertWgaMessage(ctx, msg.Role, msg.Content)
		}
		opts = append(opts, wga_option.WithMessages(msgs))
	}

	// 工具信息使用config
	for _, tool := range config.WgaCfg().Tools {
		opts = append(opts, wga_option.WithToolConfig(wga_option.ToolConfig{
			Title:   tool.Title,
			APIAuth: &tool.APIAuth,
		}))
	}

	// 工具信息
	//for _, tool := range wgaConfig.ToolList {
	//	switch tool.ToolType {
	//	case constant.ToolTypeBuiltIn:
	//		toolResp, err := mcp.GetSquareTool(ctx.Request.Context(), &mcp_service.GetSquareToolReq{
	//			ToolSquareId: tool.ToolId,
	//			Identity: &mcp_service.Identity{
	//				UserId: wgaConfig.UserId,
	//				OrgId:  wgaConfig.OrgId,
	//			},
	//		})
	//		if err != nil {
	//			log.Warnf("[wga] failed to get tool: %v", err)
	//			continue
	//		}
	//		toolDetail := toToolSquareDetail(ctx, toolResp)
	//
	//		authType := toolDetail.ApiAuth.AuthType
	//		if authType == "" {
	//			authType = util.AuthTypeNone
	//		}
	//		apiAuth := &util.ApiAuthWebRequest{
	//			AuthType:           authType,
	//			ApiKeyHeaderPrefix: toolDetail.ApiAuth.ApiKeyHeaderPrefix,
	//			ApiKeyHeader:       toolDetail.ApiAuth.ApiKeyHeader,
	//			ApiKeyQueryParam:   toolDetail.ApiAuth.ApiKeyQueryParam,
	//			ApiKeyValue:        toolDetail.ApiAuth.ApiKeyValue,
	//		}
	//
	//		opts = append(opts, wga_option.WithToolConfig(wga_option.ToolConfig{
	//			Title:   toolDetail.ToolSquareInfo.Name,
	//			APIAuth: apiAuth,
	//		}))
	//	}
	//}

	// TODO 智能体配置

	return opts, nil
}

func isURLString(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}

func extractURLsFromContent(content interface{}) []string {
	var urls []string
	switch v := content.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "binary" {
					if urlStr, ok := m["url"].(string); ok {
						urls = append(urls, urlStr)
					}
				}
			}
		}
	}
	return urls
}

func downloadURLsToDir(urls []string, dir string) error {
	log.Infof("[downloadURLsToDir] start downloading %d URLs to dir: %s", len(urls), dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create input dir failed: %w", err)
	}
	for _, urlStr := range urls {
		log.Infof("[downloadURLsToDir] downloading: %s", urlStr)
		u, err := url.Parse(urlStr)
		if err != nil {
			log.Errorf("parse URL failed: %v", err)
			continue
		}
		body, err := http_client.Default().Get(context.Background(), &http_client.HttpRequestParams{
			Url: urlStr,
		})
		if err != nil {
			log.Errorf("download URL %s failed: %v", urlStr, err)
			continue
		}
		filename := filepath.Base(u.Path)
		if filename == "" || strings.Contains(filename, "?") {
			filename = uuid.NewString()
		}
		filePath := filepath.Join(dir, filename)
		if err := os.WriteFile(filePath, body, 0644); err != nil {
			log.Errorf("save file failed: %v", err)
			continue
		}
		log.Infof("[downloadURLsToDir] downloaded %s to %s, size: %d bytes", urlStr, filePath, len(body))
	}
	return nil
}

func injectWgaWorkspaceActivity(
	ctx context.Context,
	eventCh <-chan aguievents.Event,
	threadID, runID, baseDir string,
	persistentEnabled bool,
) <-chan string {
	out := make(chan string, 1024)

	go func() {
		defer util.PrintPanicStack()
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-eventCh:
				if !ok {
					return
				}

				if evt.Type() == aguievents.EventTypeRunFinished {
					if wsEvent := buildWgaWorkspaceEvent(threadID, runID, baseDir, persistentEnabled); wsEvent != nil {
						if data, err := json.Marshal(wsEvent); err == nil {
							select {
							case out <- string(data):
							case <-ctx.Done():
								return
							}
						}
					}
				}

				if data, err := json.Marshal(evt); err == nil {
					select {
					case out <- string(data):
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out
}

func buildWgaWorkspaceEvent(threadID, runID, baseDir string, persistentEnabled bool) aguievents.Event {
	if !persistentEnabled {
		return nil
	}

	store, err := wga_persistent.NewStore(wga_persistent.ModeVersioned, baseDir, threadID)
	if err != nil {
		return nil
	}

	ok, info, err := store.GetRunDir(runID)
	if err != nil || !ok {
		return nil
	}

	statInfo, err := os.Stat(info.Dir)
	if err != nil || !statInfo.IsDir() {
		return nil
	}

	totalSize, fileCount, err := getWgaWorkspaceInfo(info.Dir, info.Dir)
	if err != nil || fileCount == 0 {
		return nil
	}

	return aguievents.NewActivitySnapshotEvent(
		aguievents.GenerateStepID(),
		ag_ui_util.ActivityTypeWorkspace,
		&ag_ui_util.WorkspaceActivityContent{
			RunID:     runID,
			ThreadID:  threadID,
			FileCount: fileCount,
			TotalSize: totalSize,
			Timestamp: time.Now().UnixMilli(),
		},
	)
}

func getWgaWorkspaceInfo(rootDir, currentDir string) (int64, int, error) {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return 0, 0, err
	}

	var totalSize int64
	var fileCount int

	for _, entry := range entries {
		// 跳过隐藏文件
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fullPath := filepath.Join(currentDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			log.Warnf("[wga] failed to get file info: %s: %v", fullPath, err)
			continue
		}

		if entry.IsDir() {
			dirSize, dirFileCount, err := getWgaWorkspaceInfo(rootDir, fullPath)
			if err != nil {
				log.Warnf("[wga] failed to build file tree for dir: %s: %v", fullPath, err)
				continue
			}
			totalSize += dirSize
			fileCount += dirFileCount
		} else {
			totalSize += info.Size()
			fileCount++
		}
	}

	return totalSize, fileCount, nil
}

func convertWgaMessage(ctx *gin.Context, role string, content interface{}) *schema.Message {
	switch v := content.(type) {
	case string:
		if isURLString(v) {
			return nil
		}
		return &schema.Message{
			Role:    schema.RoleType(role),
			Content: v,
		}
	case []interface{}:
		parts := make([]schema.MessageInputPart, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				parts = append(parts, convertWgaMessageInputPart(ctx, m))
			}
		}
		if len(parts) == 0 {
			return &schema.Message{Role: schema.RoleType(role)}
		}
		return &schema.Message{
			Role:                  schema.RoleType(role),
			UserInputMultiContent: parts,
		}
	default:
		return &schema.Message{Role: schema.RoleType(role)}
	}
}

func convertWgaMessageInputPart(ctx *gin.Context, m map[string]interface{}) schema.MessageInputPart {
	part := schema.MessageInputPart{}

	typ, _ := m["type"].(string)
	switch typ {
	case "text":
		part.Type = schema.ChatMessagePartTypeText
		if text, ok := m["text"].(string); ok {
			part.Text = text
		}
	case "binary":
		mimeType, _ := m["mimeType"].(string)
		url, _ := m["url"].(string)
		base64Data, _ := FileUrlConvertBase64(ctx, &request.FileUrlConvertBase64Req{
			FileUrl: url,
		})
		switch {
		case strings.HasPrefix(mimeType, "image/"):
			part.Type = schema.ChatMessagePartTypeImageURL
			part.Image = &schema.MessageInputImage{
				MessagePartCommon: schema.MessagePartCommon{
					Base64Data: &base64Data,
					MIMEType:   mimeType,
				},
			}
		case strings.HasPrefix(mimeType, "audio/"):
			part.Type = schema.ChatMessagePartTypeAudioURL
			part.Audio = &schema.MessageInputAudio{
				MessagePartCommon: schema.MessagePartCommon{
					Base64Data: &base64Data,
					MIMEType:   mimeType,
				},
			}
		case strings.HasPrefix(mimeType, "video/"):
			part.Type = schema.ChatMessagePartTypeVideoURL
			part.Video = &schema.MessageInputVideo{
				MessagePartCommon: schema.MessagePartCommon{
					Base64Data: &base64Data,
					MIMEType:   mimeType,
				},
			}
		default:
			part.Type = schema.ChatMessagePartTypeFileURL
			part.File = &schema.MessageInputFile{
				MessagePartCommon: schema.MessagePartCommon{
					Base64Data: &base64Data,
					MIMEType:   mimeType,
				},
			}
		}
	default:
		part.Type = schema.ChatMessagePartType(typ)
		if text, ok := m["text"].(string); ok {
			part.Text = text
		}
	}
	return part
}

// 对话过滤
// 过滤规则：
// 1. 过滤掉 role 为 tool 的消息
// 2. 过滤掉 role 为 reasoning 的消息
// 3. 过滤掉 role 为 assistant 且 content 为空的消息
// 4. 若 role 为 assistant 且 content 中有 toolCalls，则将 toolCalls 置空
func filterMessages(messages []request.GeneralAgentConversationMessage) []request.GeneralAgentConversationMessage {
	filtered := make([]request.GeneralAgentConversationMessage, 0, len(messages))
	for _, msg := range messages {
		// 过滤掉 tool 消息
		if msg.Role == "tool" {
			continue
		}
		// 过滤掉 reasoning 消息
		if msg.Role == "reasoning" {
			continue
		}
		// 过滤掉空角色或空内容的消息
		if msg.Role == "" || msg.Role == "assistant" {
			content := formatMessageContent(msg.Content)
			if content == nil || content == "" {
				continue
			}
		}
		// 检查 content 中是否有 toolCalls，如果有则过滤掉 toolCalls
		if toolCallsContent, ok := msg.Content.(map[string]interface{}); ok {
			if _, hasToolCalls := toolCallsContent["toolCalls"]; hasToolCalls {
				newContent := map[string]interface{}{}
				if text, ok := toolCallsContent["text"]; ok {
					newContent["text"] = text
				}
				filtered = append(filtered, request.GeneralAgentConversationMessage{
					Role:    msg.Role,
					Content: newContent,
				})
				continue
			}
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

// formatMessageContent 格式化消息内容，参考 convertWgaMessage 处理多模态内容
func formatMessageContent(content interface{}) interface{} {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		parts := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				part := make(map[string]interface{})
				typ, _ := m["type"].(string)
				part["type"] = typ
				switch typ {
				case "text":
					if text, ok := m["text"].(string); ok {
						part["text"] = text
					}
				case "binary":
					if url, ok := m["url"].(string); ok {
						part["url"] = url
					}
					if mimeType, ok := m["mimeType"].(string); ok {
						part["mimeType"] = mimeType
					}
				}
				parts = append(parts, part)
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return parts
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			return text
		}
		return nil
	default:
		return nil
	}
}

// normalizeMessageForES 规范化消息内容，确保 content 字段是 ES 可以接受的格式
func normalizeMessageForES(item map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range item {
		if k == "content" {
			result[k] = formatMessageContent(v)
		} else if k == "messages" {
			if msgs, ok := v.([]interface{}); ok {
				normalizedMsgs := make([]interface{}, 0, len(msgs))
				for _, msg := range msgs {
					if msgMap, ok := msg.(map[string]interface{}); ok {
						normalizedMsgs = append(normalizedMsgs, normalizeMessageForES(msgMap))
					} else {
						normalizedMsgs = append(normalizedMsgs, msg)
					}
				}
				result[k] = normalizedMsgs
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}
	return result
}

// getHistoryMessages 从 ES 获取历史记录
func getHistoryMessages(ctx *gin.Context, userId, orgId, threadId string) []request.GeneralAgentConversationMessage {
	resp, err := assistant.SearchFromES(ctx.Request.Context(), &assistant_service.SearchFromESReq{
		IndexName:  constant.ESIndexWgaChatHistory,
		Conditions: map[string]string{"threadId": threadId, "userId": userId, "orgId": orgId},
		PageNo:     1,
		PageSize:   100,
		SortOrder:  "asc",
	})
	if err != nil {
		log.Warnf("[wga] failed to get history messages: %v", err)
		return nil
	}

	var messages []request.GeneralAgentConversationMessage
	for _, docJson := range resp.DocJsonList {
		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(docJson), &doc); err != nil {
			continue
		}

		if msgs, ok := doc["messages"].([]interface{}); ok {
			for _, msg := range msgs {
				if msgMap, ok := msg.(map[string]interface{}); ok {
					role, _ := msgMap["role"].(string)
					content := msgMap["content"]
					messages = append(messages, request.GeneralAgentConversationMessage{
						Role:    role,
						Content: content,
					})
				}
			}
		}
	}

	return messages
}

// buildUserHistoryItem 构建用户消息的历史记录项
func buildUserHistoryItem(msg request.GeneralAgentConversationMessage, userId, orgId, threadId, runId string) map[string]interface{} {
	return map[string]interface{}{
		"id":          fmt.Sprintf("%s_user_%d", threadId, time.Now().UnixNano()),
		"threadId":    threadId,
		"runId":       runId,
		"userId":      userId,
		"orgId":       orgId,
		"messageType": "text",
		"messageId":   fmt.Sprintf("user_%d", time.Now().UnixNano()),
		"role":        "user",
		"content":     msg.Content,
		"createdAt":   time.Now().UnixMilli(),
	}
}

// saveChatHistoryItem 保存单条历史记录到 ES
func saveChatHistoryItem(ctx context.Context, doc map[string]interface{}) {
	docJson, err := json.Marshal(doc)
	if err != nil {
		log.Warnf("[wga] marshal history doc failed: %v", err)
		return
	}

	_, err = assistant.SaveToES(ctx, &assistant_service.SaveToESReq{
		IndexName: constant.ESIndexWgaChatHistory,
		DocJson:   string(docJson),
	})
	if err != nil {
		log.Warnf("[wga] save history to ES failed: %v", err)
	}
}

// saveWgaChatHistory 保存智能体返回的聊天历史到 ES
func saveWgaChatHistory(ctx context.Context, historyCh <-chan interface{}, userMessage request.GeneralAgentConversationMessage, userId, orgId, threadId, runId string) {
	defer util.PrintPanicStack()

	var messages []map[string]interface{}
	for msg := range historyCh {
		data, err := json.Marshal(msg)
		if err != nil {
			log.Warnf("[wga] marshal history msg failed: %v", err)
			continue
		}

		var item map[string]interface{}
		if err := json.Unmarshal(data, &item); err != nil {
			log.Warnf("[wga] unmarshal history msg failed: %v", err)
			continue
		}

		messages = append(messages, normalizeMessageForES(item))
	}

	createdAt := time.Now().UnixMilli()

	userMsgItem := map[string]interface{}{
		"role":    "user",
		"content": userMessage.Content,
	}

	allMessages := append([]map[string]interface{}{userMsgItem}, messages...)

	doc := map[string]interface{}{
		"id":        util.GenUUID(),
		"threadId":  threadId,
		"runId":     runId,
		"userId":    userId,
		"orgId":     orgId,
		"createdAt": createdAt,
		"messages":  allMessages,
	}

	cfg := config.WgaCfg()
	if cfg.Persistent.Enabled {
		store, err := wga_persistent.NewStore(wga_persistent.ModeVersioned, cfg.Persistent.BaseDir, threadId)
		if err == nil {
			ok, info, err := store.GetRunDir(runId)
			if err == nil && ok {
				totalSize, fileCount, _ := getWgaWorkspaceInfo(info.Dir, info.Dir)
				doc["workspace"] = map[string]interface{}{
					"fileCount":    fileCount,
					"totalSize":    totalSize,
					"workspaceDir": info.Dir,
				}
			}
		}
	}

	saveChatHistoryItem(ctx, doc)
}
