package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	app_service "github.com/UnicomAI/wanwu/api/proto/app-service"
	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	"github.com/UnicomAI/wanwu/api/proto/common"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	model_service "github.com/UnicomAI/wanwu/api/proto/model-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	minio_util "github.com/UnicomAI/wanwu/internal/bff-service/pkg/minio-util"
	"github.com/UnicomAI/wanwu/pkg/constant"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	mp "github.com/UnicomAI/wanwu/pkg/model-provider"
	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
	openapi3_util "github.com/UnicomAI/wanwu/pkg/openapi3-util"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/UnicomAI/wanwu/pkg/wga"
	wga_persistent "github.com/UnicomAI/wanwu/pkg/wga-persistent"
	wga_option "github.com/UnicomAI/wanwu/pkg/wga/wga-option"
	"github.com/gin-gonic/gin"
)

var resourceNameRegex = regexp.MustCompile(`@([\p{Han}a-zA-Z0-9_-]+)`)

func GetGeneralAgentToolSelect(ctx *gin.Context, userId, orgId, agentId string) (*response.ListResult, error) {
	toolResp, err := mcp.GetToolSelect(ctx.Request.Context(), &mcp_service.GetToolSelectReq{
		Identity: &mcp_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}

	toolNameToInfo := make(map[string]*mcp_service.GetToolItem)
	for _, item := range toolResp.List {
		if item.ToolType == constant.ToolTypeBuiltIn {
			toolNameToInfo[item.ToolName] = item
		}
	}

	// 获取全量工具列表
	toolCategories, err := wga.GetAgentToolCategories(config.WgaCfg().AgentID)
	if err != nil {
		return nil, err
	}
	// 对全量工具列表进行条件覆盖，默认不限制工具选择
	for _, toolCategory := range toolCategories {
		toolCategory.Condition = "none"
	}
	// 如果agentId不为空，则根据agentId获取工具选择条件进行覆盖，限制工具选择
	if agentId != "" {
		agentToolCategories, err := wga.GetAgentToolCategories(agentId)
		if err != nil {
			return nil, err
		}
		for _, toolCategory := range toolCategories {
			for _, agentToolCategory := range agentToolCategories {
				if toolCategory.Category == agentToolCategory.Category {
					toolCategory.Condition = agentToolCategory.Condition
					break
				}
			}
		}
	}

	result := make([]response.GetGeneralAgentToolSelectResp, 0, len(toolCategories))
	for _, tc := range toolCategories {
		categoryResp := response.GetGeneralAgentToolSelectResp{
			Category:  gin_util.I18nKey(ctx, string(tc.Category)),
			Condition: string(tc.Condition),
			ToolList:  []response.ToolInfo{},
		}

		for _, t := range tc.Tools {
			if item, ok := toolNameToInfo[t.Doc.Info.Title]; ok {
				categoryResp.ToolList = append(categoryResp.ToolList, response.ToolInfo{
					ToolId:          item.ToolId,
					ToolName:        item.ToolName,
					ToolType:        item.ToolType,
					Desc:            item.Desc,
					NeedApiKeyInput: item.NeedApiKeyInput,
					APIKey:          item.ApiKey,
					Avatar:          cacheToolAvatar(ctx, constant.ToolTypeBuiltIn, item.AvatarPath),
				})
			}
		}

		result = append(result, categoryResp)
	}

	return &response.ListResult{
		List:  result,
		Total: int64(len(result)),
	}, nil

}

func GetGeneralAgentToolInfo(ctx *gin.Context, userId, orgId, toolId, toolType string) (*response.GeneralAgentToolInfoResp, error) {
	resp, err := mcp.GetSquareTool(ctx.Request.Context(), &mcp_service.GetSquareToolReq{
		ToolSquareId: toolId,
		Identity: &mcp_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("tool not found: %s", toolId))
	}

	var actions []*protocol.Tool
	if resp.BuiltInTools != nil {
		for _, tool := range resp.BuiltInTools.Tools {
			actions = append(actions, toToolAction(tool))
		}
	}

	return &response.GeneralAgentToolInfoResp{
		Actions: actions,
		ToolInfo: response.ToolInfo{
			ToolId:          resp.Info.ToolSquareId,
			ToolName:        resp.Info.Name,
			ToolType:        constant.ToolTypeBuiltIn,
			Desc:            resp.Info.Desc,
			NeedApiKeyInput: resp.BuiltInTools.NeedApiKeyInput,
			APIKey:          resp.BuiltInTools.ApiAuth.ApiKeyValue,
			Avatar:          cacheToolAvatar(ctx, constant.ToolTypeBuiltIn, resp.Info.AvatarPath),
		},
	}, nil
}

func GetGeneralAgentUploadLimit(ctx *gin.Context, userId, orgId string) (*response.GeneralAgentUploadLimitResp, error) {
	cfg := config.WgaCfg()
	uploadLimit := cfg.UploadLimit

	retList := make([]*response.GeneralAgentUploadLimit, 0)
	retList = append(retList, buildGeneralAgentUploadLimit("image", uploadLimit.ImageTypes, uploadLimit.MaxImageSize))
	retList = append(retList, buildGeneralAgentUploadLimit("file", uploadLimit.FileTypes, uploadLimit.MaxFileSize))

	return &response.GeneralAgentUploadLimitResp{
		UploadLimitList: retList,
	}, nil
}

func buildGeneralAgentUploadLimit(fileType, extStr string, maxSize int) *response.GeneralAgentUploadLimit {
	var extList []string
	if extStr != "" {
		extList = strings.Split(extStr, ";")
	}
	return &response.GeneralAgentUploadLimit{
		FileType: fileType,
		MaxSize:  maxSize,
		ExtList:  extList,
	}
}

func UpdateGeneralAgentConfig(ctx *gin.Context, userId, orgId string, req request.UpdateGeneralAgentConfigReq) error {
	// 校验 assistant 配置
	assistantList := make([]*assistant_service.WgaConfigAssistant, 0, len(req.AssistantList))
	for _, a := range req.AssistantList {
		assistantList = append(assistantList, &assistant_service.WgaConfigAssistant{
			AssistantId:   a.AssistantID,
			AssistantType: util.Int2Str(constant.AgentCategorySingle), // 默认单智能体
		})
	}
	if err := checkWgaAssistantConfig(ctx, userId, orgId, assistantList); err != nil {
		return err
	}

	// 校验 tool 配置
	toolList := make([]*assistant_service.WgaConfigTool, 0, len(req.ToolList))
	toolIds := make([]string, 0, len(req.ToolList))
	for _, t := range req.ToolList {
		toolList = append(toolList, &assistant_service.WgaConfigTool{
			ToolId:   t.ToolID,
			ToolType: t.ToolType,
		})
		if t.ToolType == constant.ToolTypeBuiltIn {
			toolIds = append(toolIds, t.ToolID)
		}
	}
	validToolIds, _ := getValidToolIds(ctx, userId, orgId, toolIds)
	for _, t := range req.ToolList {
		if !validToolIds[t.ToolID] {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("tool not found or invalid: %s", t.ToolID))
		}
	}

	// 校验 mcp 配置
	mcpList := make([]*assistant_service.WgaConfigMcp, 0, len(req.MCPList))
	for _, m := range req.MCPList {
		mcpList = append(mcpList, &assistant_service.WgaConfigMcp{
			McpId:   m.MCPID,
			McpType: m.MCPType,
		})
	}
	if err := checkWgaMCPConfig(ctx, userId, orgId, mcpList); err != nil {
		return err
	}

	// 校验 workflow 配置
	workflowList := make([]*assistant_service.WgaConfigWorkflow, 0, len(req.WorkflowList))
	for _, w := range req.WorkflowList {
		workflowList = append(workflowList, &assistant_service.WgaConfigWorkflow{
			WorkflowId: w.WorkflowID,
		})
	}
	if err := checkWgaWorkflowConfig(ctx, userId, orgId, workflowList); err != nil {
		return err
	}

	// 校验 skill 配置
	skillList := make([]*assistant_service.WgaConfigSkill, 0, len(req.SkillList))
	for _, s := range req.SkillList {
		skillList = append(skillList, &assistant_service.WgaConfigSkill{
			SkillId:   s.SkillID,
			SkillType: constant.SkillTypeCustom, // 固定为自定义技能
		})
	}
	if err := checkWgaSkillConfig(ctx, userId, orgId, skillList); err != nil {
		return err
	}

	_, err := assistant.UpdateWgaConfig(ctx.Request.Context(), &assistant_service.UpdateWgaConfigReq{
		ToolList:      toolList,
		AssistantList: assistantList,
		McpList:       mcpList,
		WorkflowList:  workflowList,
		SkillList:     skillList,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	return err
}

func GetGeneralAgentConfig(ctx *gin.Context, userId, orgId string) (*response.GetGeneralAgentConfigResp, error) {
	resp, err := assistant.GetWgaConfig(ctx.Request.Context(), &assistant_service.GetWgaConfigReq{
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}

	result := &response.GetGeneralAgentConfigResp{}

	// 过滤存在的 tool
	toolIds := make([]string, 0, len(resp.Config.ToolList))
	for _, t := range resp.Config.ToolList {
		if t.ToolType == constant.ToolTypeBuiltIn {
			toolIds = append(toolIds, t.ToolId)
		}
	}
	validToolIds, _ := getValidToolIds(ctx, userId, orgId, toolIds)
	for _, t := range resp.Config.ToolList {
		if t.ToolType == constant.ToolTypeBuiltIn {
			if validToolIds[t.ToolId] {
				result.ToolList = append(result.ToolList, request.ToolSelected{
					ToolID:   t.ToolId,
					ToolType: t.ToolType,
				})
			}
		}
	}

	// 过滤存在的 assistant
	assistantIds := make([]string, 0, len(resp.Config.AssistantList))
	for _, a := range resp.Config.AssistantList {
		assistantIds = append(assistantIds, a.AssistantId)
	}
	validAssistantIds, _, _ := getValidAssistantIds(ctx, userId, orgId, assistantIds)
	for _, a := range resp.Config.AssistantList {
		if validAssistantIds[a.AssistantId] {
			result.AssistantList = append(result.AssistantList, request.AssistantSelected{
				AssistantID: a.AssistantId,
			})
		}
	}

	// 过滤存在的 mcp
	var mcpCustomIds, mcpServerIds []string
	for _, m := range resp.Config.McpList {
		switch m.McpType {
		case constant.MCPTypeMCP:
			mcpCustomIds = append(mcpCustomIds, m.McpId)
		case constant.MCPTypeMCPServer:
			mcpServerIds = append(mcpServerIds, m.McpId)
		}
	}
	validMcpIds, mcpTypes, _ := getValidMcpIds(ctx, mcpCustomIds, mcpServerIds)
	for _, m := range resp.Config.McpList {
		// 验证 MCP 存在且类型匹配
		if validMcpIds[m.McpId] && mcpTypes[m.McpId] == m.McpType {
			result.MCPList = append(result.MCPList, request.MCPSelected{
				MCPID:   m.McpId,
				MCPType: m.McpType,
			})
		}
	}

	// 过滤存在的 workflow
	workflowIds := make([]string, 0, len(resp.Config.WorkflowList))
	for _, w := range resp.Config.WorkflowList {
		workflowIds = append(workflowIds, w.WorkflowId)
	}
	validWorkflowIds, _ := getValidWorkflowIds(ctx, workflowIds)
	for _, w := range resp.Config.WorkflowList {
		if validWorkflowIds[w.WorkflowId] {
			result.WorkflowList = append(result.WorkflowList, request.WorkflowSelected{
				WorkflowID: w.WorkflowId,
			})
		}
	}

	// 过滤存在的 skill
	var customSkillIds []string
	for _, s := range resp.Config.SkillList {
		customSkillIds = append(customSkillIds, s.SkillId)
	}
	validSkillIds, _ := getValidSkillIds(ctx, customSkillIds)
	for _, s := range resp.Config.SkillList {
		if validSkillIds[s.SkillId] {
			result.SkillList = append(result.SkillList, request.SkillSelected{
				SkillID: s.SkillId,
			})
		}
	}

	return result, nil
}

func GetGeneralAgentSubList(ctx *gin.Context) (*response.GetGeneralAgentSubListResp, error) {
	result := &response.GetGeneralAgentSubListResp{}

	// 获取wga所支持的子智能体
	for _, agent := range config.WgaCfg().SubAgents {
		result.WgaAgentList = append(result.WgaAgentList, response.WgaAgentInfo{
			AgentID:     agent.AgentID,
			AgentName:   agent.AgentName,
			Avatar:      request.Avatar{Path: agent.AvatarPath},
			Placeholder: agent.Placeholder,
		})
	}

	return result, nil
}

func CreateGeneralAgentConversation(ctx *gin.Context, userId, orgId string, req request.CreateGeneralAgentConversationReq) (*response.CreateGeneralAgentConversationResp, error) {
	if err := checkModelConfig(ctx, req.ModelConfig); err != nil {
		return nil, err
	}

	resp, err := assistant.WgaConversationCreate(ctx.Request.Context(), &assistant_service.WgaConversationCreateReq{
		Prompt: req.Title,
		ModelConfig: &common.AppModelConfig{
			ModelId:   req.ModelConfig.ModelId,
			Provider:  req.ModelConfig.Provider,
			Model:     req.ModelConfig.Model,
			ModelType: req.ModelConfig.ModelType,
		},
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}
	return &response.CreateGeneralAgentConversationResp{ThreadID: resp.ThreadId}, nil
}

func DeleteGeneralAgentConversation(ctx *gin.Context, userId, orgId string, req request.DeleteGeneralAgentConversationReq) error {
	// 删除对话记录
	_, err := assistant.WgaConversationDelete(ctx.Request.Context(), &assistant_service.WgaConversationDeleteReq{
		ThreadId: req.ThreadID,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return err
	}

	// 同步删除 workspace
	cfg := config.WgaCfg()
	store, err := wga_persistent.NewStore(wga_persistent.Mode(cfg.Persistent.Mode), cfg.Persistent.BaseDir, req.ThreadID)
	if err != nil {
		log.Errorf("[wga] thread %v delete persistent store err: %v", req.ThreadID, err)
	} else {
		if threadDir := store.GetThreadDir().Dir; threadDir != "" {
			if err := util.DeleteDir(threadDir); err != nil {
				log.Errorf("[wga] thread %v delete persistent dir err: %v", req.ThreadID, err)
			}
		}
	}

	// 同步删除 ES 中的聊天历史
	_, err = assistant.DeleteFromES(ctx.Request.Context(), &assistant_service.DeleteFromESReq{
		IndexName: wgaConversationHistoryEventESIndexName,
		Conditions: map[string]string{
			"threadId": req.ThreadID,
			"userId":   userId,
			"orgId":    orgId,
		},
	})
	if err != nil && !wgaConversationHistoryEventESIndexNotFound(err) {
		log.Errorf("[wga] thread %v delete chat history from ES err: %v", req.ThreadID, err)
	}

	return nil
}

func GetGeneralAgentConversationList(ctx *gin.Context, userId, orgId string, req request.GetGeneralAgentConversationListReq) (*response.ListResult, error) {
	resp, err := assistant.WgaConversationList(ctx.Request.Context(), &assistant_service.WgaConversationListReq{
		PageSize: int32(req.PageSize),
		PageNo:   int32(req.PageNo),
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}
	var result []response.GeneralAgentConversationInfo
	for _, info := range resp.Data {
		result = append(result, response.GeneralAgentConversationInfo{
			ThreadID:  info.ThreadId,
			Title:     info.Title,
			CreatedAt: util.Time2Str(info.CreatedAt),
		})
	}
	return &response.ListResult{List: result, Total: resp.Total}, nil
}

func GetGeneralAgentConversationDetail(ctx *gin.Context, userId, orgId, threadId string) (*response.ListResult, error) {
	exist, err := assistant.WgaConversationExists(ctx.Request.Context(), &assistant_service.WgaConversationExistsReq{
		ThreadId: threadId,
		Identity: &assistant_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		return nil, err
	}
	if !exist.Exists {
		return &response.ListResult{}, nil
	}

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
			return &response.ListResult{}, nil
		}
		return nil, err
	}

	result := make([]response.GeneralAgentConversationDetailInfo, 0, len(resp.DocJsonList))
	for _, docJson := range resp.DocJsonList {
		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(docJson), &doc); err != nil {
			continue
		}

		createdAt, _ := doc["createdAt"].(int64)
		runId, _ := doc["runId"].(string)

		info := response.GeneralAgentConversationDetailInfo{
			ThreadID:  threadId,
			RunID:     runId,
			CreatedAt: createdAt,
		}
		if eventsStr, ok := doc["events"].(string); ok {
			var events []interface{}
			if err := json.Unmarshal([]byte(eventsStr), &events); err != nil {
				log.Errorf("[wga] thread %v unmarshal events err: %v", threadId, err)
				continue
			}
			info.Events = events
		}
		result = append(result, info)
	}

	return &response.ListResult{List: result, Total: int64(len(result))}, nil
}

// GetGeneralAgentConversationConfig 获取WGA对话配置
func GetGeneralAgentConversationConfig(ctx *gin.Context, userId, orgId string, req request.GetGeneralAgentConversationConfigReq) (*response.GetGeneralAgentConversationConfigResp, error) {
	resp, err := assistant.GetWgaConversationConfig(ctx.Request.Context(), &assistant_service.GetWgaConversationConfigReq{
		ThreadId: req.ThreadID,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}

	wgaConfig := resp.Config
	result := &response.GetGeneralAgentConversationConfigResp{
		ThreadID:    wgaConfig.ThreadId,
		ModelConfig: request.AppModelConfig{},
	}

	// 处理模型配置 - 需要验证模型是否存在且已启用
	if wgaConfig.ModelConfig != nil && wgaConfig.ModelConfig.ModelId != "" {
		// 验证模型是否存在
		modelInfo, err := model.GetModel(ctx.Request.Context(), &model_service.GetModelReq{ModelId: wgaConfig.ModelConfig.ModelId})
		if err == nil && modelInfo != nil && modelInfo.IsActive {
			// 模型存在且已启用，返回配置
			result.ModelConfig = request.AppModelConfig{
				Provider:    wgaConfig.ModelConfig.Provider,
				Model:       wgaConfig.ModelConfig.Model,
				ModelId:     wgaConfig.ModelConfig.ModelId,
				ModelType:   wgaConfig.ModelConfig.ModelType,
				DisplayName: modelInfo.DisplayName,
			}
		}
		// 如果模型不存在或未启用，返回空的 ModelConfig
	}

	return result, nil
}

func UpdateGeneralAgentConversationConfig(ctx *gin.Context, userId, orgId string, req request.UpdateGeneralAgentConversationConfigReq) error {
	if err := checkModelConfig(ctx, req.ModelConfig); err != nil {
		return err
	}
	_, err := assistant.UpdateWgaConversationConfig(ctx.Request.Context(), &assistant_service.UpdateWgaConversationConfigReq{
		ThreadId: req.ThreadID,
		ModelConfig: &common.AppModelConfig{
			ModelId:   req.ModelConfig.ModelId,
			Provider:  req.ModelConfig.Provider,
			Model:     req.ModelConfig.Model,
			ModelType: req.ModelConfig.ModelType,
		},
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	return err
}

// CheckGeneralAgentConversationConfig 检查WGA对话配置
func CheckGeneralAgentConversationConfig(ctx *gin.Context, userId, orgId string, req request.GeneralAgentConfigCheckRequest) (*response.GeneralAgentConfigCheckResponse, error) {
	// 查询对话配置
	conversationConfigResp, err := assistant.GetWgaConversationConfig(ctx.Request.Context(), &assistant_service.GetWgaConversationConfigReq{
		ThreadId: req.ThreadID,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}
	wgaConversationConfig := conversationConfigResp.Config

	// 查询配置
	configResp, err := assistant.GetWgaConfig(ctx.Request.Context(), &assistant_service.GetWgaConfigReq{
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}
	wgaConfig := configResp.Config

	var opts []wga_option.Option

	// 构建模型配置选项
	if wgaConversationConfig.GetModelConfig() != nil && wgaConversationConfig.ModelConfig.ModelId != "" {
		modelOpt, err := buildWgaModelOption(ctx, wgaConversationConfig.ModelConfig)
		if err != nil {
			return nil, err
		}
		opts = append(opts, modelOpt)
	}

	if req.AgentID == "" {
		// 如果agentId为空，则不限制工具配置，直接返回模型检查结果
		return &response.GeneralAgentConfigCheckResponse{
			Meet:      true,
			ModelMeet: true,
		}, nil
	}

	// 构建工具配置选项
	toolOpts, err := buildWgaToolOptions(ctx, userId, orgId, wgaConfig.ToolList)
	if err != nil {
		return nil, err
	}
	opts = append(opts, toolOpts...)

	// 检查工具配置
	checkResult, err := wga.CheckToolOptions(ctx.Request.Context(), req.AgentID, opts...)
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("check tool options err: %v", err))
	}

	result := &response.GeneralAgentConfigCheckResponse{
		ModelMeet: true,
		ToolsMeet: make([]response.GeneralAgentToolCategories, 0, len(checkResult.ToolCategories)),
	}

	// 处理工具检查结果
	for _, tc := range checkResult.ToolCategories {
		category := response.GeneralAgentToolCategories{
			Category:  tc.Category,
			Condition: tc.Condition,
			Meet:      tc.Meet,
			Tools:     make([]response.GeneralAgentCheckTool, 0, len(tc.Tools)),
		}
		for _, t := range tc.Tools {
			category.Tools = append(category.Tools, response.GeneralAgentCheckTool{
				ToolID: t.Title,
				Meet:   t.Meet,
			})
		}
		result.ToolsMeet = append(result.ToolsMeet, category)
	}

	result.Meet = result.ModelMeet
	for _, tc := range result.ToolsMeet {
		if !tc.Meet {
			result.Meet = false
			break
		}
	}

	return result, nil
}

func GeneralAgentWorkspaceDownload(ctx *gin.Context, userId, orgId string, req request.GeneralAgentWorkspaceDownloadReq) (string, []byte, error) {
	cfg := config.WgaCfg()
	if !cfg.Persistent.Enabled {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "persistent not enabled")
	}

	store, err := wga_persistent.NewStore(wga_persistent.Mode(cfg.Persistent.Mode), cfg.Persistent.BaseDir, req.ThreadID)
	if err != nil {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}

	ok, info, err := store.GetRunDir(req.RunID)
	if err != nil {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if !ok {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "run directory not found")
	}

	workDir := info.Dir
	targetPath := workDir
	if req.Path != "" {
		targetPath = filepath.Join(workDir, req.Path)
	}

	fi, err := os.Stat(targetPath)
	if err != nil {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("path not found: %v", err))
	}

	if fi.IsDir() {
		zipName := fmt.Sprintf("workspace_%s_%s.zip", req.RunID, filepath.Base(req.Path))
		zipData, err := util.ZipDir(targetPath + "/.")
		if err != nil {
			return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to create zip: %v", err))
		}
		return zipName, zipData, nil
	}

	fileName := filepath.Base(req.Path)
	fileData, err := os.ReadFile(targetPath)
	if err != nil {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to read file: %v", err))
	}
	return fileName, fileData, nil
}

func GeneralAgentWorkspacePreview(ctx *gin.Context, userId, orgId string, req request.GeneralAgentWorkspacePreviewReq) (string, []byte, string, error) {
	cfg := config.WgaCfg()
	if !cfg.Persistent.Enabled {
		return "", nil, "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, "persistent not enabled")
	}

	store, err := wga_persistent.NewStore(wga_persistent.Mode(cfg.Persistent.Mode), cfg.Persistent.BaseDir, req.ThreadID)
	if err != nil {
		return "", nil, "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}

	ok, info, err := store.GetRunDir(req.RunID)
	if err != nil {
		return "", nil, "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if !ok {
		return "", nil, "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, "run directory not found")
	}

	workDir := info.Dir
	targetPath := filepath.Join(workDir, req.Path)

	fi, err := os.Stat(targetPath)
	if err != nil {
		return "", nil, "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("path not found: %v", err))
	}
	if fi.IsDir() {
		return "", nil, "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, "path is a directory, not a file")
	}

	fileName := filepath.Base(req.Path)
	fileData, err := os.ReadFile(targetPath)
	if err != nil {
		return "", nil, "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to read file: %v", err))
	}

	contentType := http.DetectContentType(fileData)
	return fileName, fileData, contentType, nil
}

func GeneralAgentWorkspaceInfo(ctx *gin.Context, userId, orgId string, req request.GeneralAgentWorkspaceReq) (*response.GeneralAgentWorkspaceResp, error) {
	cfg := config.WgaCfg()
	if !cfg.Persistent.Enabled {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "persistent not enabled")
	}

	store, err := wga_persistent.NewStore(wga_persistent.Mode(cfg.Persistent.Mode), cfg.Persistent.BaseDir, req.ThreadID)
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}

	ok, info, err := store.GetRunDir(req.RunID)
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, err.Error())
	}
	if !ok {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "run directory not found")
	}

	workDir := info.Dir
	files, err := buildWgaFileTree(workDir, "")
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to read directory: %v", err))
	}

	return &response.GeneralAgentWorkspaceResp{
		GeneralAgentConversationWorkspaceInfo: response.GeneralAgentConversationWorkspaceInfo{
			ThreadID:  req.ThreadID,
			RunID:     req.RunID,
			FileCount: int32(len(files)),
			TotalSize: calculateWgaFileTreeTotalSize(files),
			IsDisplay: true,
		},
		Path:  "",
		Files: files,
	}, nil
}

// --- internal wga model ---

// checkModelConfig 校验请求中的模型配置（用于创建/更新对话配置）
func checkModelConfig(ctx *gin.Context, modelConfig *request.AppModelConfig) error {
	if modelConfig == nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "modelConfig is required")
	}
	if modelConfig.ModelId == "" {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "modelId is required")
	}
	if modelConfig.Model == "" {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "model is required")
	}
	// 校验模型是否存在
	modelInfo, err := model.GetModel(ctx.Request.Context(), &model_service.GetModelReq{ModelId: modelConfig.ModelId})
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("model not found: %s", modelConfig.ModelId))
	}
	// 校验模型是否已启用
	if !modelInfo.IsActive {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("model is not active: %s", modelConfig.ModelId))
	}
	// 校验 model 名称是否匹配
	if modelInfo.Model != modelConfig.Model {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("model name mismatch: expected %s, got %s", modelInfo.Model, modelConfig.Model))
	}
	return nil
}

// checkModelConfigFromProto 校验proto类型的模型配置（用于运行时检查）
func checkModelConfigFromProto(ctx *gin.Context, modelConfig *common.AppModelConfig) error {
	if modelConfig == nil || modelConfig.ModelId == "" {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "modelConfig is required for conversation")
	}
	modelInfo, err := model.GetModel(ctx.Request.Context(), &model_service.GetModelReq{ModelId: modelConfig.ModelId})
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("model not found: %s", modelConfig.ModelId))
	}
	// 校验模型是否已启用
	if !modelInfo.IsActive {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("model is not active: %s", modelConfig.ModelId))
	}
	if modelInfo.Model != modelConfig.Model {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("model name mismatch: expected %s, got %s", modelInfo.Model, modelConfig.Model))
	}
	return nil
}

// buildWgaModelOption 构建模型配置选项
func buildWgaModelOption(ctx *gin.Context, modelConfig *common.AppModelConfig) (wga_option.Option, error) {
	if modelConfig == nil || modelConfig.ModelId == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "modelConfig is required")
	}

	modelInfo, err := model.GetModel(ctx.Request.Context(), &model_service.GetModelReq{ModelId: modelConfig.ModelId})
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("model not found: %s", modelConfig.ModelId))
	}
	// 校验模型是否已启用
	if !modelInfo.IsActive {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("model is not active: %s", modelConfig.ModelId))
	}

	endpoint := mp.ToModelEndpoint(modelConfig.ModelId, modelConfig.Model)
	modelURL, _ := endpoint["model_url"].(string)

	var apiKey string
	cfg, err := mp.ToModelConfig(modelInfo.Provider, modelInfo.ModelType, modelInfo.ProviderConfig)
	if err == nil {
		var cfgMap map[string]any
		if b, err := json.Marshal(cfg); err == nil {
			if err = json.Unmarshal(b, &cfgMap); err == nil {
				if k, ok := cfgMap["apiKey"].(string); ok {
					apiKey = k
				}
			}
		}
	}

	var modelParams *mp_common.LLMParams
	if modelConfig.Config != "" {
		llmParams, _, err := mp.ToModelParams(modelConfig.Provider, modelConfig.ModelType, modelConfig.Config)
		if err == nil && llmParams != nil {
			modelParams = llmParams.(*mp_common.LLMParams)
		}
	}

	return wga_option.WithModelConfig(wga_option.ModelConfig{
		Provider:     modelConfig.Provider,
		ProviderName: modelConfig.Provider,
		BaseURL:      modelURL,
		APIKey:       apiKey,
		Model:        modelConfig.Model,
		ModelName:    modelConfig.Model,
		Params:       modelParams,
	}), nil
}

// --- internal wga tool ---

// buildWgaToolOptions 构建工具配置选项（复用逻辑）
func buildWgaToolOptions(ctx *gin.Context, userId, orgId string, toolList []*assistant_service.WgaConfigTool) ([]wga_option.Option, error) {
	var opts []wga_option.Option
	for _, tool := range toolList {
		switch tool.ToolType {
		case constant.ToolTypeBuiltIn:
			toolResp, err := mcp.GetSquareTool(ctx.Request.Context(), &mcp_service.GetSquareToolReq{
				ToolSquareId: tool.ToolId,
				Identity: &mcp_service.Identity{
					UserId: userId,
					OrgId:  orgId,
				},
			})
			if err != nil {
				// 工具不存在时跳过，不阻断运行
				log.Warnf("[wga] tool %s not found, skip: %v", tool.ToolId, err)
				continue
			}
			toolDetail := toToolSquareDetail(ctx, toolResp)

			authType := toolDetail.ApiAuth.AuthType
			if authType == "" {
				authType = util.AuthTypeNone
			}
			apiAuth := &util.ApiAuthWebRequest{
				AuthType:           authType,
				ApiKeyHeaderPrefix: toolDetail.ApiAuth.ApiKeyHeaderPrefix,
				ApiKeyHeader:       toolDetail.ApiAuth.ApiKeyHeader,
				ApiKeyQueryParam:   toolDetail.ApiAuth.ApiKeyQueryParam,
				ApiKeyValue:        toolDetail.ApiAuth.ApiKeyValue,
			}

			opts = append(opts, wga_option.WithToolConfig(wga_option.ToolConfig{
				Title:   toolDetail.Name,
				APIAuth: apiAuth,
			}))
		}
	}
	return opts, nil
}

// getValidToolIds 批量获取有效的Tool ID映射
func getValidToolIds(ctx *gin.Context, userId, orgId string, toolIds []string) (map[string]bool, error) {
	if len(toolIds) == 0 {
		return make(map[string]bool), nil
	}
	validIds := make(map[string]bool)
	for _, toolId := range toolIds {
		_, err := mcp.GetSquareTool(ctx.Request.Context(), &mcp_service.GetSquareToolReq{
			ToolSquareId: toolId,
			Identity: &mcp_service.Identity{
				UserId: userId,
				OrgId:  orgId,
			},
		})
		if err == nil {
			validIds[toolId] = true
		}
	}
	return validIds, nil
}

// --- internal wga assistant ---

// checkWgaAssistantConfig 校验wga智能体配置（用于更新配置）
// 通用智能体配置只支持单智能体
func checkWgaAssistantConfig(ctx *gin.Context, userId, orgId string, assistantList []*assistant_service.WgaConfigAssistant) error {
	if len(assistantList) == 0 {
		return nil
	}
	assistantIds := make([]string, 0, len(assistantList))
	for _, a := range assistantList {
		assistantIds = append(assistantIds, a.AssistantId)
	}
	validIds, assistantInfos, err := getValidAssistantIds(ctx, userId, orgId, assistantIds)
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "assistant not found")
	}

	// 校验所有智能体
	for _, a := range assistantList {
		// 校验智能体是否存在
		if !validIds[a.AssistantId] {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("assistant not found: %s", a.AssistantId))
		}

		// 校验智能体是否已发布
		appInfo, err := app.GetAppInfo(ctx.Request.Context(), &app_service.GetAppInfoReq{
			AppId:   a.AssistantId,
			AppType: constant.AppTypeAgent,
		})
		if err != nil || appInfo.PublishType == "" {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("assistant not published: %s", a.AssistantId))
		}

		// 校验智能体类型：通用智能体只支持单智能体
		info := assistantInfos[a.AssistantId]
		if info != nil && info.Category != constant.AgentCategorySingle {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("assistant must be single agent: %s", a.AssistantId))
		}
	}
	return nil
}

func buildWgaAssistantOptions(ctx *gin.Context, userId, orgId string, assistantList []*assistant_service.WgaConfigAssistant) ([]wga_option.Option, error) {
	if len(assistantList) == 0 {
		return nil, nil
	}

	var assistantIds []string
	for _, a := range assistantList {
		assistantIds = append(assistantIds, a.AssistantId)
	}
	resp, err := assistant.GetAssistantByIds(ctx.Request.Context(), &assistant_service.GetAssistantByIdsReq{
		AssistantIdList: assistantIds,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}

	var opts []wga_option.Option
	for _, a := range resp.AssistantInfos {
		if a.Info == nil {
			continue
		}
		schemaData, err := renderAgentChatProxySchema(a.Info.AppId, a.Info.Name, a.Info.Desc)
		if err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("render assistant(%s) openapi schema err: %v", a.Info.AppId, err))
		}
		doc, err := openapi3_util.LoadFromData(ctx.Request.Context(), schemaData)
		if err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("load assistant(%s) openapi schema err: %v", a.Info.AppId, err))
		}
		opts = append(opts, wga_option.WithExtraTool(wga_option.ExtraTool{
			OpenAPI3Schema: doc,
		}))
	}

	return opts, nil
}

// getValidAssistantIds 批量获取有效的智能体ID映射
// 返回: validIds - 有效ID映射, assistantInfos - 智能体信息映射, error
func getValidAssistantIds(ctx *gin.Context, userId, orgId string, assistantIds []string) (map[string]bool, map[string]*assistant_service.AssistantBrief, error) {
	if len(assistantIds) == 0 {
		return make(map[string]bool), make(map[string]*assistant_service.AssistantBrief), nil
	}
	assistantResp, err := assistant.GetAssistantByIds(ctx.Request.Context(), &assistant_service.GetAssistantByIdsReq{
		AssistantIdList: assistantIds,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	validIds := make(map[string]bool)
	assistantInfos := make(map[string]*assistant_service.AssistantBrief)
	for _, info := range assistantResp.AssistantInfos {
		validIds[info.Info.AppId] = true
		assistantInfos[info.Info.AppId] = info
	}
	return validIds, assistantInfos, nil
}

// --- internal wga mcp ---

// checkWgaMCPConfig 校验wga MCP配置（用于更新配置）
func checkWgaMCPConfig(ctx *gin.Context, userId, orgId string, mcpList []*assistant_service.WgaConfigMcp) error {
	if len(mcpList) == 0 {
		return nil
	}

	var mcpCustomIds, mcpServerIds []string
	for _, m := range mcpList {
		switch m.McpType {
		case constant.MCPTypeMCP:
			mcpCustomIds = append(mcpCustomIds, m.McpId)
		case constant.MCPTypeMCPServer:
			mcpServerIds = append(mcpServerIds, m.McpId)
		default:
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("invalid mcp type: %s", m.McpType))
		}
	}

	validIds, mcpTypes, err := getValidMcpIds(ctx, mcpCustomIds, mcpServerIds)
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "mcp not found")
	}

	for _, m := range mcpList {
		// 校验 MCP 是否存在
		if !validIds[m.McpId] {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("mcp not found: %s", m.McpId))
		}
		// 校验 McpType 与 ID 是否匹配
		if actualType, ok := mcpTypes[m.McpId]; !ok || actualType != m.McpType {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("mcp type mismatch: %s (expected %s, got %s)", m.McpId, m.McpType, actualType))
		}
	}
	return nil
}

func buildWgaMCPOptions(ctx *gin.Context, userId, orgId string, mcpList []*assistant_service.WgaConfigMcp) ([]wga_option.Option, error) {
	if len(mcpList) == 0 {
		return nil, nil
	}

	var mcpCustomIds, mcpServerIds []string
	for _, m := range mcpList {
		switch m.McpType {
		case constant.MCPTypeMCP:
			mcpCustomIds = append(mcpCustomIds, m.McpId)
		case constant.MCPTypeMCPServer:
			mcpServerIds = append(mcpServerIds, m.McpId)
		default:
			return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("invalid mcp type: %s", m.McpType))
		}
	}

	mcpResp, err := mcp.GetMCPByMCPIdList(ctx.Request.Context(), &mcp_service.GetMCPByMCPIdListReq{
		McpIdList:       mcpCustomIds,
		McpServerIdList: mcpServerIds,
	})
	if err != nil {
		return nil, err
	}

	var opts []wga_option.Option
	for _, item := range mcpResp.Infos {
		opts = append(opts, wga_option.WithMCP(wga_option.MCP{
			Name: item.Info.GetName(),
			URL:  util.IfElse(item.Transport == constant.MCPTransportStreamable, item.StreamableUrl, item.SseUrl),
		}))
	}
	for _, item := range mcpResp.Servers {
		opts = append(opts, wga_option.WithMCP(wga_option.MCP{
			Name: item.Name,
			URL:  util.IfElse(item.Transport == constant.MCPTransportStreamable, item.StreamableUrl, item.SseUrl),
		}))
	}
	return opts, nil
}

// getValidMcpIds 批量获取有效的MCP ID映射
// 返回: validIds - 有效ID映射, mcpTypes - ID对应的类型映射(mcp/mcpserver), error
func getValidMcpIds(ctx *gin.Context, mcpCustomIds, mcpServerIds []string) (map[string]bool, map[string]string, error) {
	if len(mcpCustomIds) == 0 && len(mcpServerIds) == 0 {
		return make(map[string]bool), make(map[string]string), nil
	}
	mcpResp, err := mcp.GetMCPByMCPIdList(ctx.Request.Context(), &mcp_service.GetMCPByMCPIdListReq{
		McpIdList:       mcpCustomIds,
		McpServerIdList: mcpServerIds,
	})
	if err != nil {
		return nil, nil, err
	}
	validIds := make(map[string]bool)
	mcpTypes := make(map[string]string)
	for _, item := range mcpResp.Infos {
		validIds[item.McpId] = true
		mcpTypes[item.McpId] = constant.MCPTypeMCP
	}
	for _, item := range mcpResp.Servers {
		validIds[item.McpServerId] = true
		mcpTypes[item.McpServerId] = constant.MCPTypeMCPServer
	}
	return validIds, mcpTypes, nil
}

// --- internal wga workflow ---

// checkWgaWorkflowConfig 校验wga Workflow配置（用于更新配置）
func checkWgaWorkflowConfig(ctx *gin.Context, userId, orgId string, workflowList []*assistant_service.WgaConfigWorkflow) error {
	if len(workflowList) == 0 {
		return nil
	}

	workflowIds := make([]string, 0, len(workflowList))
	for _, w := range workflowList {
		workflowIds = append(workflowIds, w.WorkflowId)
	}

	validIds, err := getValidWorkflowIds(ctx, workflowIds)
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "workflow not found")
	}

	for _, w := range workflowList {
		if !validIds[w.WorkflowId] {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("workflow not found: %s", w.WorkflowId))
		}
	}
	return nil
}

func buildWgaWorkflowOptions(ctx *gin.Context, userId, orgId string, workflowList []*assistant_service.WgaConfigWorkflow) ([]wga_option.Option, error) {
	if len(workflowList) == 0 {
		return nil, nil
	}

	var workflowIDs []string
	for _, wf := range workflowList {
		workflowIDs = append(workflowIDs, wf.WorkflowId)
	}
	workflowSchemas, err := GetWorkflowSchemas(ctx, workflowIDs)
	if err != nil {
		return nil, err
	}
	var opts []wga_option.Option
	for _, schema := range workflowSchemas {
		opts = append(opts, wga_option.WithExtraTool(wga_option.ExtraTool{OpenAPI3Schema: schema}))
	}
	return opts, nil
}

// getValidWorkflowIds 批量获取有效的Workflow ID映射
func getValidWorkflowIds(ctx *gin.Context, workflowIds []string) (map[string]bool, error) {
	if len(workflowIds) == 0 {
		return make(map[string]bool), nil
	}
	workflowResp, err := ListWorkflowByIDs(ctx, "", workflowIds)
	if err != nil {
		return nil, err
	}
	validIds := make(map[string]bool)
	for _, w := range workflowResp.Workflows {
		validIds[w.WorkflowId] = true
	}
	return validIds, nil
}

// --- internal wga skill ---

// checkWgaSkillConfig 校验wga Skill配置（用于更新配置）
func checkWgaSkillConfig(ctx *gin.Context, userId, orgId string, skillList []*assistant_service.WgaConfigSkill) error {
	if len(skillList) == 0 {
		return nil
	}

	var customSkillIds []string
	for _, s := range skillList {
		switch s.SkillType {
		case constant.SkillTypeCustom:
			customSkillIds = append(customSkillIds, s.SkillId)
		default:
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("invalid skill type: %s", s.SkillType))
		}
	}

	validIds, err := getValidSkillIds(ctx, customSkillIds)
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "skill not found")
	}

	for _, s := range skillList {
		if !validIds[s.SkillId] {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("skill not found: %s", s.SkillId))
		}
	}
	return nil
}

func buildWgaSkillOptions(ctx *gin.Context, userId, orgId, threadId, runId string, skillList []*assistant_service.WgaConfigSkill) ([]wga_option.Option, error) {
	if len(skillList) == 0 {
		return nil, nil
	}

	var customSkillIds []string
	for _, s := range skillList {
		switch s.SkillType {
		case constant.SkillTypeCustom:
			customSkillIds = append(customSkillIds, s.SkillId)
		default:
			return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("invalid skill type: %s", s.SkillType))
		}
	}

	resp, err := mcp.GetCustomSkillDetailByIdList(ctx.Request.Context(), &mcp_service.CustomSkillDetailByIdListReq{
		SkillIds: customSkillIds,
	})
	if err != nil {
		return nil, err
	}

	var opts []wga_option.Option
	for _, s := range resp.SkillDetails {
		skillUrl, _ := url.JoinPath("http://", config.Cfg().Minio.Endpoint, s.ObjectPath)

		b, skillZipName, err := minio_util.DownloadFile(ctx.Request.Context(), skillUrl)
		if err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to download skill file from %s: %v", skillUrl, err))
		}
		skillTempDir := filepath.Join(os.TempDir(), "wga", threadId, runId, "skills", s.SkillId)
		if err := os.MkdirAll(skillTempDir, 0755); err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to create skill temp dir %s: %v", skillTempDir, err))
		}
		skillZipPath := filepath.Join(skillTempDir, skillZipName)
		if err := os.WriteFile(skillZipPath, b, 0644); err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to write skill zip %s: %v", skillZipPath, err))
		}
		if _, err := util.UnzipDir(ctx.Request.Context(), skillZipPath, skillTempDir); err != nil {
			return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to unzip skill %s: %v", skillZipPath, err))
		}
		if err := util.DeleteFile(skillZipPath); err != nil {
			log.Warnf("failed to delete skill zip file %s: %v", skillZipPath, err)
		}
		opts = append(opts, wga_option.WithSkill(wga_option.Skill{Dir: skillTempDir}))
	}

	return opts, nil
}

// getValidSkillIds 批量获取有效的Skill ID映射
func getValidSkillIds(ctx *gin.Context, skillIds []string) (map[string]bool, error) {
	if len(skillIds) == 0 {
		return make(map[string]bool), nil
	}
	resp, err := mcp.GetCustomSkillDetailByIdList(ctx.Request.Context(), &mcp_service.CustomSkillDetailByIdListReq{
		SkillIds: skillIds,
	})
	if err != nil {
		return nil, err
	}
	validIds := make(map[string]bool)
	for _, s := range resp.SkillDetails {
		validIds[s.SkillId] = true
	}
	return validIds, nil
}

// --- internal wga workspace ---

func buildWgaFileTree(dirPath, parentPath string) ([]response.GeneralAgentFileInfo, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var files []response.GeneralAgentFileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(parentPath, entry.Name())
		fileInfo := response.GeneralAgentFileInfo{
			Name: entry.Name(),
		}

		if entry.IsDir() {
			fileInfo.Type = "directory"
			children, err := buildWgaFileTree(filepath.Join(dirPath, entry.Name()), filePath)
			if err == nil {
				fileInfo.Children = children
			}
		} else {
			fileInfo.Type = "file"
			fileInfo.Size = info.Size()
			fullPath := filepath.Join(dirPath, entry.Name())
			if data, err := os.ReadFile(fullPath); err == nil {
				fileInfo.MimeType = http.DetectContentType(data)
			}
			if fileInfo.MimeType == "" {
				log.Warnf("file %s has empty mime type", filePath)
				fileInfo.MimeType = "application/octet-stream"
			}
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

func calculateWgaFileTreeTotalSize(files []response.GeneralAgentFileInfo) int64 {
	var total int64
	for _, f := range files {
		if f.Type == "directory" {
			total += calculateWgaFileTreeTotalSize(f.Children)
		} else {
			total += f.Size
		}
	}
	return total
}

// --- internal wga resource ---

func GetGeneralAgentResourceSelect(ctx *gin.Context, userId, orgId string, name string) ([]*response.GeneralAgentResourceSelectList, error) {
	result := make([]*response.GeneralAgentResourceSelectList, 0, 4)

	// 并发获取四种资源列表
	var wg sync.WaitGroup
	var mcpErr, workflowErr, skillErr, assistantErr error
	var mcpList []response.MCPSelect
	var workflowList []*response.ExplorationAppInfo
	var skillList []*response.SkillInfo
	var assistantList []*response.ExplorationAppInfo

	// 获取 MCP 列表
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer util.PrintPanicStack()
		resp, err := GetMCPSelect(ctx, userId, orgId, name)
		if err != nil {
			mcpErr = err
			return
		}
		if list, ok := resp.List.([]response.MCPSelect); ok {
			mcpList = list
		}
	}()

	// 获取 Workflow 列表
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer util.PrintPanicStack()
		resp, err := GetWorkflowSelect(ctx, userId, orgId, request.GetExplorationAppListRequest{Name: name})
		if err != nil {
			workflowErr = err
			return
		}
		if list, ok := resp.List.([]*response.ExplorationAppInfo); ok {
			workflowList = list
		}
	}()

	// 获取 Skill 列表
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer util.PrintPanicStack()
		resp, err := GetSkillSelect(ctx, userId, orgId, name, constant.SkillTypeCustom)
		if err != nil {
			skillErr = err
			return
		}
		if list, ok := resp.List.([]*response.SkillInfo); ok {
			skillList = list
		}
	}()

	// 获取 Assistant 列表
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer util.PrintPanicStack()
		resp, err := GetAssistantSelect(ctx, userId, orgId, request.GetExplorationAppListRequest{Name: name})
		if err != nil {
			assistantErr = err
			return
		}
		if list, ok := resp.List.([]*response.ExplorationAppInfo); ok {
			assistantList = list
		}
	}()

	wg.Wait()

	// 检查错误
	if mcpErr != nil {
		return nil, mcpErr
	}
	if workflowErr != nil {
		return nil, workflowErr
	}
	if skillErr != nil {
		return nil, skillErr
	}
	if assistantErr != nil {
		return nil, assistantErr
	}

	// 构建 MCP 列表
	mcpItems := make([]*response.GeneralAgentResourceSelectItem, 0, len(mcpList))
	for _, item := range mcpList {
		mcpItems = append(mcpItems, &response.GeneralAgentResourceSelectItem{
			ID:     item.MCPID,
			Name:   item.Name,
			Desc:   item.Description,
			Avatar: item.Avatar,
			Type:   item.Type,
		})
	}
	result = append(result, &response.GeneralAgentResourceSelectList{
		ListType: "mcp",
		List:     mcpItems,
	})

	// 构建 Workflow 列表
	workflowItems := make([]*response.GeneralAgentResourceSelectItem, 0, len(workflowList))
	for _, item := range workflowList {
		workflowItems = append(workflowItems, &response.GeneralAgentResourceSelectItem{
			ID:     item.AppId,
			Name:   item.Name,
			Desc:   item.Desc,
			Avatar: item.Avatar,
			Type:   item.AppType,
		})
	}
	result = append(result, &response.GeneralAgentResourceSelectList{
		ListType: "workflow",
		List:     workflowItems,
	})

	// 构建 Skill 列表
	skillItems := make([]*response.GeneralAgentResourceSelectItem, 0, len(skillList))
	for _, item := range skillList {
		skillItems = append(skillItems, &response.GeneralAgentResourceSelectItem{
			ID:     item.SkillId,
			Name:   item.SkillName,
			Desc:   item.Desc,
			Avatar: item.Avatar,
			Type:   item.SkillType,
			Author: item.Author,
		})
	}
	result = append(result, &response.GeneralAgentResourceSelectList{
		ListType: "skill",
		List:     skillItems,
	})

	// 构建 Assistant 列表
	assistantItems := make([]*response.GeneralAgentResourceSelectItem, 0, len(assistantList))
	for _, item := range assistantList {
		assistantItems = append(assistantItems, &response.GeneralAgentResourceSelectItem{
			ID:     item.AppId,
			Name:   item.Name,
			Desc:   item.Desc,
			Avatar: item.Avatar,
			Type:   item.AppType,
		})
	}
	result = append(result, &response.GeneralAgentResourceSelectList{
		ListType: "assistant",
		List:     assistantItems,
	})

	return result, nil
}

// parseResourceMentions 从用户消息中解析 @ 提及的资源名称
// 格式: @资源名称 后面跟空格或消息结束
// 支持: "@mcp1 @workflow2 请帮我处理" -> ["mcp1", "workflow2"]
// 返回: 提取到的资源名称列表（去重后）
func parseResourceMentions(content interface{}) []string {
	var text string
	switch v := content.(type) {
	case string:
		text = v
	case []interface{}:
		// 处理多部分消息，提取文本部分
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if typ, _ := m["type"].(string); typ == "text" {
					if t, _ := m["text"].(string); t != "" {
						text += t + " "
					}
				}
			}
		}
	}

	// 使用正则提取 @name 格式，支持中文、英文、数字、下划线、连字符
	matches := resourceNameRegex.FindAllStringSubmatch(text, -1)

	// 去重
	seen := make(map[string]bool)
	var names []string
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}

	return names
}

// MentionResources @提及的资源列表
type MentionResources struct {
	McpList       []*assistant_service.WgaConfigMcp
	WorkflowList  []*assistant_service.WgaConfigWorkflow
	SkillList     []*assistant_service.WgaConfigSkill
	AssistantList []*assistant_service.WgaConfigAssistant
}

// fetchMentionResources 获取@提及的资源列表
// 通过名称搜索下拉资源列表，返回各类型的资源列表
func fetchMentionResources(ctx *gin.Context, userID, orgID string, mentionNames []string) *MentionResources {
	result := &MentionResources{
		McpList:       make([]*assistant_service.WgaConfigMcp, 0),
		WorkflowList:  make([]*assistant_service.WgaConfigWorkflow, 0),
		SkillList:     make([]*assistant_service.WgaConfigSkill, 0),
		AssistantList: make([]*assistant_service.WgaConfigAssistant, 0),
	}

	if len(mentionNames) == 0 {
		return result
	}

	// 对每个名称调用搜索接口
	for _, name := range mentionNames {
		searchResult, err := GetGeneralAgentResourceSelect(ctx, userID, orgID, name)
		if err != nil {
			log.Warnf("[wga] search resources by name '%s' failed: %v", name, err)
			continue
		}

		// 遍历资源列表，按类型收集
		for _, group := range searchResult {
			switch group.ListType {
			case "mcp":
				for _, item := range group.List {
					result.McpList = append(result.McpList, &assistant_service.WgaConfigMcp{
						McpId:   item.ID,
						McpType: item.Type,
					})
				}
			case "workflow":
				for _, item := range group.List {
					result.WorkflowList = append(result.WorkflowList, &assistant_service.WgaConfigWorkflow{
						WorkflowId: item.ID,
					})
				}
			case "skill":
				for _, item := range group.List {
					result.SkillList = append(result.SkillList, &assistant_service.WgaConfigSkill{
						SkillId:   item.ID,
						SkillType: constant.SkillTypeCustom,
					})
				}
			case "assistant":
				for _, item := range group.List {
					result.AssistantList = append(result.AssistantList, &assistant_service.WgaConfigAssistant{
						AssistantId:   item.ID,
						AssistantType: util.Int2Str(constant.AgentCategorySingle),
					})
				}
			}
		}
	}

	return result
}
