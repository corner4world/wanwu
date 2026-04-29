package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	err_code "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
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

func wgaConversationHistoryEventESIndexNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "index_not_found_exception") && strings.Contains(err.Error(), wgaConversationHistoryEventESIndexName)
}

func GeneralAgentConversationChat(ctx *gin.Context, userId, orgId string, req request.GeneralAgentConversationChatReq) error {
	// 过滤出当前用户消息（最后一条 User 消息）
	var userInputMessage *request.GeneralAgentConversationMessage
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == ag_ui_util.RoleUser {
			userInputMessage = &req.Messages[i]
			break
		}
	}
	if userInputMessage == nil {
		return grpc_util.ErrorStatus(err_code.Code_BFFGeneral, "no user message found in input messages")
	}

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

	// 获取上一次输出的目录信息（如果存在）
	var lastWorkspaceTotalSize int64
	var lastWorkspaceFileCount int
	if config.WgaCfg().Persistent.Enabled {
		store, err := wga_persistent.NewStore(wga_persistent.Mode(config.WgaCfg().Persistent.Mode), config.WgaCfg().Persistent.BaseDir, req.ThreadID)
		if err != nil {
			log.Errorf("[wga] thread %v failed to create persistent store: %v", req.ThreadID, err)
		} else {
			ok, info, err := store.GetLastRunDir()
			if err != nil {
				log.Errorf("[wga] thread %v failed to get last run dir: %v", req.ThreadID, err)
			}
			if ok {
				lastWorkspaceTotalSize, lastWorkspaceFileCount, err = getWgaWorkspaceInfo(info.Dir)
				if err != nil {
					log.Errorf("[wga] thread %v failed to get last workspace info: %v", req.ThreadID, err)
				}
			}
		}
	}

	runID := uuid.NewString()

	// 构建 WGA 选项
	opts, err := buildWgaRunOptions(ctx, userId, orgId, req.AgentID, req.ThreadID, runID, userInputMessage)
	if err != nil {
		return err
	}

	// 运行 WGA
	agentID := req.AgentID
	if agentID == "" {
		agentID = config.WgaCfg().AgentID
	}
	_, iter, err := wga.Run(ctx.Request.Context(), agentID, opts...)
	if err != nil {
		return err
	}

	// 将 WGA 事件流转换为 AG-UI 事件流
	tr := ag_ui_util.NewEinoTranslator(req.ThreadID, runID)
	eventCh := tr.TranslateStream(ctx.Request.Context(), iter)

	processorConfig := &ag_ui_util.ProcessorConfig{
		ToolNameMapper: map[string]string{
			"transfer_to_agent": "正在交给专业智能体",
		},
		ExcludedAgentNames: []string{
			config.WgaCfg().AgentID,     // 排除主智能体，避免生成冗余的切换智能体提示
			ag_ui_util.DefaultAgentName, // 排除default，避免生成冗余的切换智能体提示
		},
		// ResultFormatters: map[string]func(string) string{
		// 	"bocha_comprehensive_search": WgaFormatBochaWebSearchResult,
		// 	"bocha_web_search_only":      WgaFormatBochaWebSearchResult,
		// 	"bocha_search_structured":    WgaFormatBochaWebSearchResult,
		// 	"bocha_search_day":           WgaFormatBochaWebSearchResult,
		// 	"bocha_search_last_week":     WgaFormatBochaWebSearchResult,
		// 	"tavily_basic_search":        WgaFormatTavilySearchResult,
		// 	"tavily_deep_search":         WgaFormatTavilySearchResult,
		// 	"tavily_day_search":          WgaFormatTavilySearchResult,
		// 	"tavily_week_search":         WgaFormatTavilySearchResult,
		// 	"tavily_image_search":        WgaFormatTavilySearchResult,
		// 	"tavily_date_search":         WgaFormatTavilySearchResult,
		// },
	}

	processor := ag_ui_util.NewStreamProcessor(processorConfig)
	processedEventCh, historyEventCh := processor.Process(ctx.Request.Context(), eventCh, map[string]interface{}{
		"threadId":       req.ThreadID,
		"runId":          runID,
		"messages":       []interface{}{userInputMessage},
		"state":          map[string]interface{}{},
		"tools":          []interface{}{},
		"context":        []interface{}{},
		"forwardedProps": map[string]interface{}{},
	})

	// 保存智能体返回的消息
	go saveWgaChatHistoryEvent(context.Background(), historyEventCh, userId, orgId, req.ThreadID, runID,
		config.WgaCfg().Persistent.BaseDir,
		config.WgaCfg().Persistent.Enabled,
		lastWorkspaceTotalSize,
		lastWorkspaceFileCount,
	)

	outputCh := processWgaEvent2OutputCh(
		ctx.Request.Context(),
		processedEventCh,
		req.ThreadID,
		runID,
		config.WgaCfg().Persistent.BaseDir,
		config.WgaCfg().Persistent.Enabled,
		lastWorkspaceTotalSize,
		lastWorkspaceFileCount,
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

func filterWgaHistoryMessages(ctx *gin.Context, userId, orgId, threadId string) ([]*schema.Message, error) {
	resp, err := assistant.SearchFromES(ctx.Request.Context(), &assistant_service.SearchFromESReq{
		IndexName: wgaConversationHistoryEventESIndexName,
		Conditions: map[string]string{
			"threadId": threadId,
			"userId":   userId,
			"orgId":    orgId,
		},
		SortOrder: "asc",
		PageNo:    1,
		PageSize:  1000,
	})
	if err != nil {
		if wgaConversationHistoryEventESIndexNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var messages []adk.Message
	for _, docJson := range resp.DocJsonList {
		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(docJson), &doc); err != nil {
			continue
		}

		if eventsStr, ok := doc["events"].(string); ok {
			var events []map[string]interface{}
			if err := json.Unmarshal([]byte(eventsStr), &events); err != nil {
				log.Errorf("[wga] thread %v unmarshal events err: %v", threadId, err)
				continue
			}
			for _, event := range events {
				eventType, ok := event["type"].(string)
				if !ok {
					continue
				}
				var message *schema.Message
				switch aguievents.EventType(eventType) {
				case aguievents.EventTypeRunStarted:
					if input := event["input"].(map[string]interface{}); input != nil {
						if msgs, ok := input["messages"].([]interface{}); ok {
							for _, msg := range msgs { // messages中只有一个用户消息
								if m, ok := msg.(map[string]interface{}); ok {
									message = convertWgaMessage(m["role"].(string), m["content"])
								}
							}
						}
					}

				case aguievents.EventTypeReasoningMessageContent:
					if delta, ok := event["delta"].(string); ok {
						message = &schema.Message{
							Role:             ag_ui_util.RoleAssistant,
							ReasoningContent: delta,
						}
					}
				case aguievents.EventTypeTextMessageContent:
					if delta, ok := event["delta"].(string); ok {
						message = &schema.Message{
							Role:    ag_ui_util.RoleAssistant,
							Content: delta,
						}
					}
				default:
					// do nothing
				}
				if message != nil {
					messages = append(messages, message)
				}
			}
		}
	}
	return messages, nil
}

func buildWgaRunOptions(ctx *gin.Context, userID, orgID, agentID, threadID, runID string, userInputMessage *request.GeneralAgentConversationMessage) ([]wga_option.Option, error) {
	// 获取 WGA 配置
	wgaConfigResp, err := assistant.GetWgaConfig(ctx.Request.Context(), &assistant_service.GetWgaConfigReq{
		Identity: &assistant_service.Identity{
			UserId: userID,
			OrgId:  orgID,
		},
	})
	if err != nil {
		return nil, err
	}
	wgaConfig := wgaConfigResp.Config

	// 解析用户消息中的 @提及资源
	var mentionResources *wgaMentionResources
	if userInputMessage != nil {
		mentionNames := parseWgaResourceMentions(userInputMessage.Content)
		mentionResources = fetchWgaMentionResources(ctx, userID, orgID, mentionNames)
	}

	// 获取 WGA Conversation 配置
	wgaConversationConfigResp, err := assistant.GetWgaConversationConfig(ctx.Request.Context(), &assistant_service.GetWgaConversationConfigReq{
		ThreadId: threadID,
		Identity: &assistant_service.Identity{
			UserId: userID,
			OrgId:  orgID,
		},
	})
	if err != nil {
		return nil, err
	}
	wgaConversationConfig := wgaConversationConfigResp.Config

	opts := []wga_option.Option{
		wga_option.WithRunSession(wga_option.RunSession{
			ThreadID: threadID,
			RunID:    runID,
		}),
	}

	// 校验并构建模型配置选项
	if wgaConversationConfig != nil && wgaConversationConfig.ModelConfig != nil && wgaConversationConfig.ModelConfig.ModelId != "" {
		if err := checkModelConfigFromProto(ctx, wgaConversationConfig.GetModelConfig()); err != nil {
			return nil, err
		}
		modelOpt, err := buildWgaModelOption(ctx, wgaConversationConfig.ModelConfig)
		if err != nil {
			return nil, err
		}
		opts = append(opts, modelOpt)
	}

	// 校验并构建工具配置选项
	if wgaConfig != nil && len(wgaConfig.ToolList) > 0 {
		toolOpts, err := buildWgaToolOptions(ctx, userID, orgID, wgaConfig.ToolList)
		if err != nil {
			return nil, err
		}
		if agentID != "" {
			checkResult, err := wga.CheckToolOptions(ctx.Request.Context(), agentID, toolOpts...)
			if err != nil {
				return nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, fmt.Sprintf("tool options check failed: %v", err))
			}
			for _, tc := range checkResult.ToolCategories {
				if !tc.Meet {
					return nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, fmt.Sprintf("tool category (%s) condition (%s) not meet", tc.Category, tc.Condition))
				}
			}
		}
		opts = append(opts, toolOpts...)
	}

	// 校验并构建工作流配置选项（追加@提及的工作流）
	workflowList := make([]*assistant_service.WgaConfigWorkflow, 0)
	if wgaConfig != nil {
		workflowList = append(workflowList, wgaConfig.WorkflowList...)
	}
	if mentionResources != nil {
		workflowList = append(workflowList, mentionResources.WorkflowList...)
	}
	if len(workflowList) > 0 {
		// 去重
		seen := make(map[string]bool)
		dedupedList := make([]*assistant_service.WgaConfigWorkflow, 0, len(workflowList))
		for _, w := range workflowList {
			if !seen[w.WorkflowId] {
				seen[w.WorkflowId] = true
				dedupedList = append(dedupedList, w)
			}
		}
		if err := checkWgaWorkflowConfig(ctx, userID, orgID, dedupedList); err != nil {
			return nil, err
		}
		workflowOpts, err := buildWgaWorkflowOptions(ctx, userID, orgID, dedupedList)
		if err != nil {
			return nil, err
		}
		opts = append(opts, workflowOpts...)
	}

	// 校验并构建MCP配置选项（追加@提及的MCP）
	mcpList := make([]*assistant_service.WgaConfigMcp, 0)
	if wgaConfig != nil {
		mcpList = append(mcpList, wgaConfig.McpList...)
	}
	if mentionResources != nil {
		mcpList = append(mcpList, mentionResources.McpList...)
	}
	if len(mcpList) > 0 {
		// 去重
		seen := make(map[string]bool)
		dedupedList := make([]*assistant_service.WgaConfigMcp, 0, len(mcpList))
		for _, m := range mcpList {
			if !seen[m.McpId] {
				seen[m.McpId] = true
				dedupedList = append(dedupedList, m)
			}
		}
		if err := checkWgaMCPConfig(ctx, userID, orgID, dedupedList); err != nil {
			return nil, err
		}
		mcpOpts, err := buildWgaMCPOptions(ctx, userID, orgID, dedupedList)
		if err != nil {
			return nil, err
		}
		opts = append(opts, mcpOpts...)
	}

	// 校验并构建智能体配置选项（追加@提及的智能体）
	assistantList := make([]*assistant_service.WgaConfigAssistant, 0)
	if wgaConfig != nil {
		assistantList = append(assistantList, wgaConfig.AssistantList...)
	}
	if mentionResources != nil {
		assistantList = append(assistantList, mentionResources.AssistantList...)
	}
	if len(assistantList) > 0 {
		// 去重
		seen := make(map[string]bool)
		dedupedList := make([]*assistant_service.WgaConfigAssistant, 0, len(assistantList))
		for _, a := range assistantList {
			if !seen[a.AssistantId] {
				seen[a.AssistantId] = true
				dedupedList = append(dedupedList, a)
			}
		}
		if err := checkWgaAssistantConfig(ctx, userID, orgID, dedupedList); err != nil {
			return nil, err
		}
		assistantOpts, err := buildWgaAssistantOptions(ctx, userID, orgID, dedupedList)
		if err != nil {
			return nil, err
		}
		opts = append(opts, assistantOpts...)
	}

	// 校验并构建Skills配置选项（追加@提及的Skills）
	skillList := make([]*assistant_service.WgaConfigSkill, 0)
	if wgaConfig != nil {
		skillList = append(skillList, wgaConfig.SkillList...)
	}
	if mentionResources != nil {
		skillList = append(skillList, mentionResources.SkillList...)
	}
	if len(skillList) > 0 {
		// 去重
		seen := make(map[string]bool)
		dedupedList := make([]*assistant_service.WgaConfigSkill, 0, len(skillList))
		for _, s := range skillList {
			if !seen[s.SkillId] {
				seen[s.SkillId] = true
				dedupedList = append(dedupedList, s)
			}
		}
		if err := checkWgaSkillConfig(ctx, userID, orgID, dedupedList); err != nil {
			return nil, err
		}
		skillOpts, err := buildWgaSkillOptions(ctx, userID, orgID, threadID, runID, dedupedList)
		if err != nil {
			return nil, err
		}
		opts = append(opts, skillOpts...)
	}

	// 校验并构建Knowledge配置选项（追加@提及的Knowledge）
	knowledgeList := make([]*assistant_service.WgaConfigKnowledge, 0)
	if wgaConfig != nil {
		knowledgeList = append(knowledgeList, wgaConfig.KnowledgeList...)
	}
	if mentionResources != nil {
		knowledgeList = append(knowledgeList, mentionResources.KnowledgeList...)
	}
	if len(knowledgeList) > 0 {
		// 去重
		seen := make(map[string]bool)
		dedupedList := make([]*assistant_service.WgaConfigKnowledge, 0, len(knowledgeList))
		for _, k := range knowledgeList {
			if !seen[k.KnowledgeId] {
				seen[k.KnowledgeId] = true
				dedupedList = append(dedupedList, k)
			}
		}
		if err := checkWgaKnowledgeConfig(ctx, userID, orgID, dedupedList); err != nil {
			return nil, err
		}
		knowledgeOpts, err := buildWgaKnowledgeOptions(ctx, userID, orgID, threadID, runID, dedupedList)
		if err != nil {
			return nil, err
		}
		opts = append(opts, knowledgeOpts...)
	}

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
		if urls := userInputMessage.GetURLs(); len(urls) > 0 {
			if err := downloadURLsToDir(urls, inputDir); err != nil {
				log.Errorf("download URLs %+v to inputDir %v failed: %v", urls, inputDir, err)
			}
		}
	}

	// 历史消息 + @资源提示消息 + 当前用户消息
	messages, err := filterWgaHistoryMessages(ctx, userID, orgID, threadID)
	if err != nil {
		return nil, err
	}
	if mentionResources != nil && mentionResources.hasResources() {
		messages = append(messages, buildWgaMentionResourcesMessage(mentionResources))
	}
	messages = append(messages, convertWgaMessage(userInputMessage.Role, userInputMessage.Content))
	opts = append(opts, wga_option.WithMessages(messages))

	return opts, nil
}

// saveWgaChatHistoryEvent 保存智能体返回的聊天历史到 ES
func saveWgaChatHistoryEvent(
	ctx context.Context,
	historyEventCh <-chan aguievents.Event,
	userId, orgId, threadId, runId, baseDir string,
	persistentEnabled bool,
	lastWorkspaceTotalSize int64,
	lastWorkspaceFileCount int) {
	defer util.PrintPanicStack()

	var events []aguievents.Event
	for event := range historyEventCh {
		if persistentEnabled && event.Type() == aguievents.EventTypeRunFinished {
			if wsEvent := buildWgaWorkspaceEvent(threadId, runId, baseDir, lastWorkspaceTotalSize, lastWorkspaceFileCount); wsEvent != nil {
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

func processWgaEvent2OutputCh(
	ctx context.Context,
	eventCh <-chan aguievents.Event,
	threadID, runID, baseDir string,
	persistentEnabled bool,
	lastWorkspaceTotalSize int64,
	lastWorkspaceFileCount int,
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

				// inject workspace activity event when run finished, to trigger AG-UI to fetch workspace info and update workspace activity card
				if persistentEnabled && evt.Type() == aguievents.EventTypeRunFinished {
					if wsEvent := buildWgaWorkspaceEvent(threadID, runID, baseDir, lastWorkspaceTotalSize, lastWorkspaceFileCount); wsEvent != nil {
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

func buildWgaWorkspaceEvent(threadID, runID, baseDir string, lastWorkspaceTotalSize int64, lastWorkspaceFileCount int) aguievents.Event {
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

	totalSize, fileCount, err := getWgaWorkspaceInfo(info.Dir)
	if err != nil || fileCount == 0 {
		return nil
	}

	if totalSize == lastWorkspaceTotalSize && fileCount == lastWorkspaceFileCount {
		// 工作空间内容未变化，不发送事件
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

func getWgaWorkspaceInfo(currentDir string) (int64, int, error) {
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

		currentPath := filepath.Join(currentDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			log.Warnf("[wga] failed to get file info: %s: %v", currentPath, err)
			continue
		}

		if entry.IsDir() {
			dirSize, dirFileCount, err := getWgaWorkspaceInfo(currentPath)
			if err != nil {
				log.Warnf("[wga] failed to build file tree for dir: %s: %v", currentPath, err)
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

// buildWgaMentionResourcesMessage 构建 @ 资源提示消息
func buildWgaMentionResourcesMessage(resources *wgaMentionResources) *schema.Message {
	var lines []string
	lines = append(lines, "本次对话已加载用户 @ 提及的资源：")

	for _, item := range resources.McpItems {
		lines = append(lines, fmt.Sprintf("- @%s (mcp) %s", item.Name, item.Desc))
	}
	for _, item := range resources.WorkflowItems {
		lines = append(lines, fmt.Sprintf("- @%s (workflow) %s", item.Name, item.Desc))
	}
	for _, item := range resources.SkillItems {
		lines = append(lines, fmt.Sprintf("- @%s (skill) %s", item.Name, item.Desc))
	}
	for _, item := range resources.AssistantItems {
		lines = append(lines, fmt.Sprintf("- @%s (assistant) %s", item.Name, item.Desc))
	}
	for _, item := range resources.KnowledgeItems {
		lines = append(lines, fmt.Sprintf("- @%s (knowledge) %s", item.Name, item.Desc))
	}

	return &schema.Message{
		Role:    schema.System,
		Content: strings.Join(lines, "\n"),
	}
}

func convertWgaMessage(role string, content interface{}) *schema.Message {
	switch v := content.(type) {
	case string:
		return &schema.Message{
			Role:    schema.RoleType(role),
			Content: v,
		}
	case []interface{}:
		var parts []schema.MessageInputPart
		for _, msg := range v {
			if m, ok := msg.(map[string]interface{}); ok {
				parts = append(parts, convertWgaMessageInputPart(m))
			}
		}
		return &schema.Message{
			Role:                  schema.RoleType(role),
			UserInputMultiContent: parts,
		}
	default:
		return &schema.Message{Role: schema.RoleType(role)}
	}
}

func convertWgaMessageInputPart(m map[string]interface{}) schema.MessageInputPart {
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
		switch {
		case strings.HasPrefix(mimeType, "image/"):
			part.Type = schema.ChatMessagePartTypeImageURL
			part.Image = &schema.MessageInputImage{
				MessagePartCommon: schema.MessagePartCommon{
					URL:      &url,
					MIMEType: mimeType,
				},
			}
		case strings.HasPrefix(mimeType, "audio/"):
			part.Type = schema.ChatMessagePartTypeAudioURL
			part.Audio = &schema.MessageInputAudio{
				MessagePartCommon: schema.MessagePartCommon{
					URL:      &url,
					MIMEType: mimeType,
				},
			}
		case strings.HasPrefix(mimeType, "video/"):
			part.Type = schema.ChatMessagePartTypeVideoURL
			part.Video = &schema.MessageInputVideo{
				MessagePartCommon: schema.MessagePartCommon{
					URL:      &url,
					MIMEType: mimeType,
				},
			}
		default:
			// github.com/cloudwego/eino-ext/libs/acl/openai/chat_model.go:485 v0.1.1
			// 该版本暂未支持 schema.ChatMessagePartTypeFileURL 处理，暂用 schema.ChatMessagePartTypeText 替代
			// part.Type = schema.ChatMessagePartTypeFileURL
			// part.File = &schema.MessageInputFile{
			// 	MessagePartCommon: schema.MessagePartCommon{
			// 		URL:      &url,
			// 		MIMEType: mimeType,
			// 	},
			// }
			part.Type = schema.ChatMessagePartTypeText
			if fileName, _ := m["fileName"].(string); fileName != "" {
				part.Text = fmt.Sprintf("[%s](%s)", fileName, url)
			} else {
				part.Text = url
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

func downloadURLsToDir(urls map[string]string, dir string) error {
	if len(urls) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("[downloadURLsToDir] create dir %s failed: %w", dir, err)
	}
	for fileName, urlStr := range urls {
		log.Infof("[downloadURLsToDir] downloading URL %s to %s", urlStr, dir)
		body, err := http_client.Default().Get(context.Background(), &http_client.HttpRequestParams{
			Url: urlStr,
		})
		if err != nil {
			log.Errorf("[downloadURLsToDir] download URL %s failed: %v", urlStr, err)
			continue
		}
		filePath := filepath.Join(dir, fileName)
		if err := os.WriteFile(filePath, body, 0644); err != nil {
			log.Errorf("[downloadURLsToDir] save file %s failed: %v", filePath, err)
			continue
		}
		log.Infof("[downloadURLsToDir] downloaded URL %s to %s, size: %d bytes", urlStr, filePath, len(body))
	}
	return nil
}
