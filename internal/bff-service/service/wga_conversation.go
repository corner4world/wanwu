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
	err_code "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	ag_ui_util "github.com/UnicomAI/wanwu/pkg/ag-ui-util"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	http_client "github.com/UnicomAI/wanwu/pkg/http-client"
	"github.com/UnicomAI/wanwu/pkg/log"
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

const (
	wgaConversationHistoryEventESIndexName = "wga_chat_history_event" // 通用智能体聊天历史ES索引
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
		return grpc_util.ErrorStatus(err_code.Code_BFFGeneral, fmt.Sprintf("threadId %s not found", req.ThreadID))
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
	if err != nil {
		return err
	}

	// 获取 WGA 配置
	wgaConfigResp, err := assistant.GetWgaConfig(ctx.Request.Context(), &assistant_service.GetWgaConfigReq{
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return err
	}
	wgaConversationConfig := wgaConversationConfigResp.Config
	wgaConfig := wgaConfigResp.Config

	// 过滤消息 FIXME 历史信息应从后端获取
	messages := filterWgaMessages(req.Messages)

	var currentUserMessage *request.GeneralAgentConversationMessage
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == ag_ui_util.RoleUser {
			currentUserMessage = &req.Messages[i]
			break
		}
	}

	// 构建 WGA 选项
	opts, err := buildWgaRunOptions(ctx, req.ThreadID, runID, wgaConversationConfig, wgaConfig, messages, currentUserMessage)
	if err != nil {
		return grpc_util.ErrorStatus(err_code.Code_BFFGeneral, err.Error())
	}

	// 运行 WGA
	_, iter, err := wga.Run(ctx.Request.Context(), config.WgaCfg().AgentID, opts...)
	if err != nil {
		return err
	}

	// 将 WGA 事件流转换为 AG-UI 事件流
	tr := ag_ui_util.NewEinoTranslator(req.ThreadID, runID)
	eventCh := tr.TranslateStream(ctx.Request.Context(), iter)

	processorConfig := &ag_ui_util.ProcessorConfig{
		ToolNameMapper: map[string]string{
			"transfer_to_agent": "正在交给智能体",
		},
		// ExcludedAgentNames: []string{
		// 	"Supervisor Agent",
		// },
		ResultFormatters: map[string]func(string) string{
			"bocha_comprehensive_search": WgaFormatBochaWebSearchResult,
			"bocha_web_search_only":      WgaFormatBochaWebSearchResult,
			"bocha_search_structured":    WgaFormatBochaWebSearchResult,
			"bocha_search_day":           WgaFormatBochaWebSearchResult,
			"bocha_search_last_week":     WgaFormatBochaWebSearchResult,
			"tavily_basic_search":        WgaFormatTavilySearchResult,
			"tavily_deep_search":         WgaFormatTavilySearchResult,
			"tavily_day_search":          WgaFormatTavilySearchResult,
			"tavily_week_search":         WgaFormatTavilySearchResult,
			"tavily_image_search":        WgaFormatTavilySearchResult,
			"tavily_date_search":         WgaFormatTavilySearchResult,
		},
	}

	processor := ag_ui_util.NewStreamProcessor(processorConfig)
	processedEventCh, historyEventCh := processor.Process(ctx.Request.Context(), eventCh, map[string]interface{}{
		"threadId":       req.ThreadID,
		"runId":          runID,
		"messages":       []interface{}{currentUserMessage},
		"state":          map[string]interface{}{},
		"tools":          []interface{}{},
		"context":        []interface{}{},
		"forwardedProps": map[string]interface{}{},
	})

	// 保存智能体返回的消息
	go saveWgaChatHistoryEvent(context.Background(), historyEventCh, userId, orgId, req.ThreadID, runID,
		config.WgaCfg().Persistent.BaseDir,
		config.WgaCfg().Persistent.Enabled,
	)

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

func buildWgaRunOptions(ctx *gin.Context, threadID, runID string, wgaConversationConfig *assistant_service.WgaConversationConfig, wgaConfig *assistant_service.WgaConfig, messages []request.GeneralAgentConversationMessage, currentUserMessage *request.GeneralAgentConversationMessage) ([]wga_option.Option, error) {
	opts := []wga_option.Option{
		wga_option.WithRunSession(wga_option.RunSession{
			ThreadID: threadID,
			RunID:    runID,
		}),
	}

	// 校验并构建模型配置选项
	modelConfigValid := false
	if wgaConversationConfig != nil && wgaConversationConfig.ModelConfig != nil && wgaConversationConfig.ModelConfig.ModelId != "" {
		if err := checkModelConfigFromProto(ctx, wgaConversationConfig.GetModelConfig()); err == nil {
			modelOpt, err := buildWgaModelOption(ctx, wgaConversationConfig.ModelConfig)
			if err == nil {
				opts = append(opts, modelOpt)
				modelConfigValid = true
			}
		}
	}

	// 用户模型配置无效，尝试使用默认配置
	if !modelConfigValid {
		defaultModelConfig := config.WgaCfg().Model
		if defaultModelConfig.Model != "" && defaultModelConfig.BaseURL != "" {
			opts = append(opts, wga_option.WithModelConfig(wga_option.ModelConfig{
				Provider:     defaultModelConfig.Provider,
				ProviderName: defaultModelConfig.ProviderName,
				BaseURL:      defaultModelConfig.BaseURL,
				APIKey:       defaultModelConfig.APIKey,
				Model:        defaultModelConfig.Model,
				ModelName:    defaultModelConfig.ModelName,
			}))
		} else {
			return nil, fmt.Errorf("model config is required: user config invalid and default config not set")
		}
	}

	// 校验并构建工具配置选项
	toolConfigValid := false
	if wgaConfig != nil && len(wgaConfig.ToolList) > 0 {
		if err := checkWgaToolConfig(ctx, wgaConfig.UserId, wgaConfig.OrgId, wgaConfig.ToolList); err == nil {
			toolOpts, err := buildWgaToolOptions(ctx, wgaConfig.UserId, wgaConfig.OrgId, wgaConfig.ToolList)
			if err == nil {
				opts = append(opts, toolOpts...)
				toolConfigValid = true
			}
		}
	}

	// 用户工具配置无效，尝试使用默认配置
	if !toolConfigValid {
		defaultTools := config.WgaCfg().Tools
		if len(defaultTools) > 0 {
			for _, tool := range defaultTools {
				opts = append(opts, wga_option.WithToolConfig(wga_option.ToolConfig{
					Title:   tool.Title,
					APIAuth: &tool.APIAuth,
				}))
			}
		} else {
			return nil, fmt.Errorf("tool list is required: user config invalid and default config not set")
		}
	}

	// TODO 智能体配置

	// 持久化存储
	if config.WgaCfg().Persistent.Enabled {
		var inputDir string
		mode := wga_persistent.ModeVersioned
		if config.WgaCfg().Persistent.Mode == string(wga_persistent.ModeOverwrite) {
			mode = wga_persistent.ModeOverwrite
		}
		store, err := wga_persistent.NewStore(mode, config.WgaCfg().Persistent.BaseDir, threadID)
		if err == nil {
			// 创建目录并从上一次输出复制
			_, info, err := store.GetRunDir(runID, wga_persistent.WithMkdir(true))
			if err == nil {
				inputDir = info.Dir
				opts = append(opts,
					wga_option.WithInputDir(filepath.Clean(info.Dir)+"/."),
					wga_option.WithOutputDir(info.Dir),
				)
			}
		}
		// 下载用户消息中的URL文件到inputDir
		if currentUserMessage != nil {
			if urls := currentUserMessage.GetURLs(); len(urls) > 0 {
				if err := downloadURLsToDir(urls, inputDir); err != nil {
					log.Errorf("download URLs %+v to inputDir %v failed: %v", urls, inputDir, err)
				}
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

	return opts, nil
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
// 1. 过滤掉 role 非 user/assistant 的消息
// 2. 过滤掉 role 为 assistant 且 content 为空的消息
func filterWgaMessages(messages []request.GeneralAgentConversationMessage) []request.GeneralAgentConversationMessage {
	filtered := make([]request.GeneralAgentConversationMessage, 0, len(messages))
	for _, msg := range messages {
		// 过滤掉 tool 消息
		if msg.Role != ag_ui_util.RoleUser && msg.Role != ag_ui_util.RoleAssistant {
			continue
		}
		// 过滤掉空内容的消息
		if msg.Content == nil {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

// saveWgaChatHistoryEvent 保存智能体返回的聊天历史到 ES
func saveWgaChatHistoryEvent(ctx context.Context, historyEventCh <-chan aguievents.Event, userId, orgId, threadId, runId, baseDir string, persistentEnabled bool) {
	defer util.PrintPanicStack()

	var events []aguievents.Event
	for event := range historyEventCh {
		if event.Type() == aguievents.EventTypeRunFinished {
			if wsEvent := buildWgaWorkspaceEvent(threadId, runId, baseDir, persistentEnabled); wsEvent != nil {
				events = append(events, wsEvent)
			}
		}
		events = append(events, event)
	}
	b, _ := json.Marshal(events)

	doc := map[string]interface{}{
		"id":        util.GenUUID(),
		"threadId":  threadId,
		"runId":     runId,
		"userId":    userId,
		"orgId":     orgId,
		"createdAt": time.Now().UnixMilli(),
		"events":    string(b),
	}
	docJson, err := json.Marshal(doc)
	if err != nil {
		log.Warnf("[wga] marshal history doc failed: %v", err)
		return
	}

	_, err = assistant.SaveToES(ctx, &assistant_service.SaveToESReq{
		IndexName: wgaConversationHistoryEventESIndexName,
		DocJson:   string(docJson),
	})
	if err != nil {
		log.Warnf("[wga] save history to ES failed: %v", err)
	}
}

func downloadURLsToDir(urls []string, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("[downloadURLsToDir] create dir %s failed: %w", dir, err)
	}
	for _, urlStr := range urls {
		log.Infof("[downloadURLsToDir] downloading URL %s to %s", urlStr, dir)
		u, err := url.Parse(urlStr)
		if err != nil {
			log.Errorf("[downloadURLsToDir] parse URL %s failed: %v", urlStr, err)
			continue
		}
		body, err := http_client.Default().Get(context.Background(), &http_client.HttpRequestParams{
			Url: urlStr,
		})
		if err != nil {
			log.Errorf("[downloadURLsToDir] download URL %s failed: %v", urlStr, err)
			continue
		}
		filename := filepath.Base(u.Path)
		if filename == "" || strings.Contains(filename, "?") {
			filename = uuid.NewString()
		}
		filePath := filepath.Join(dir, filename)
		if err := os.WriteFile(filePath, body, 0644); err != nil {
			log.Errorf("[downloadURLsToDir] save file %s failed: %v", filePath, err)
			continue
		}
		log.Infof("[downloadURLsToDir] downloaded URL %s to %s, size: %d bytes", urlStr, filePath, len(body))
	}
	return nil
}
