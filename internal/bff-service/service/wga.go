package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	net_url "net/url"
	"os"
	"path/filepath"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	"github.com/UnicomAI/wanwu/api/proto/common"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	model_service "github.com/UnicomAI/wanwu/api/proto/model-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/pkg/constant"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	mp "github.com/UnicomAI/wanwu/pkg/model-provider"
	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/UnicomAI/wanwu/pkg/wga"
	wga_persistent "github.com/UnicomAI/wanwu/pkg/wga-persistent"
	wga_option "github.com/UnicomAI/wanwu/pkg/wga/wga-option"
	"github.com/gin-gonic/gin"
)

func GetGeneralAgentAssistantSelect(ctx *gin.Context, userId, orgId string, name string) ([]response.GetGeneralAgentAssistantSelectResp, error) {
	resp, err := assistant.GetAssistantListMyAll(ctx.Request.Context(), &assistant_service.GetAssistantListMyAllReq{
		Name: name,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}
	var result []response.GetGeneralAgentAssistantSelectResp
	for _, assistantInfo := range resp.AssistantInfos {
		// 只展示单智能体
		if assistantInfo.Category != constant.AgentCategorySingle {
			continue
		}
		appBriefInfo := appBriefProto2Model(ctx, assistantInfo.Info, assistantInfo.Category)
		if appBriefInfo.Avatar.Path != "" {
			appBriefInfo.Avatar.Path, _ = net_url.JoinPath(config.Cfg().Server.ApiBaseUrl, appBriefInfo.Avatar.Path)
		}
		result = append(result, response.GetGeneralAgentAssistantSelectResp{
			AppBriefInfo: appBriefInfo,
		})
	}
	return result, nil
}

func GetGeneralAgentToolSelect(ctx *gin.Context, userId, orgId string) ([]response.GetGeneralAgentToolSelectResp, error) {
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

	toolCategories, err := wga.GetAgentToolCategories(config.WgaCfg().AgentID)
	if err != nil {
		return nil, err
	}

	result := make([]response.GetGeneralAgentToolSelectResp, 0, len(toolCategories))
	for _, tc := range toolCategories {
		categoryResp := response.GetGeneralAgentToolSelectResp{
			Category:  gin_util.I18nKey(ctx, tc.Category),
			Condition: string(tc.Condition),
			ToolList:  []response.ToolInfo{},
		}

		for _, t := range tc.Tools {
			toolInfo := response.ToolInfo{}

			if item, ok := toolNameToInfo[t.Title]; ok {
				toolInfo.ToolId = item.ToolId
				toolInfo.ToolName = item.ToolName
				toolInfo.ToolType = item.ToolType
				toolInfo.Desc = item.Desc
				toolInfo.NeedApiKeyInput = item.NeedApiKeyInput
				toolInfo.APIKey = item.ApiKey
				toolInfo.Avatar = cacheToolAvatar(ctx, constant.ToolTypeBuiltIn, item.AvatarPath)
			}

			categoryResp.ToolList = append(categoryResp.ToolList, toolInfo)
		}

		result = append(result, categoryResp)
	}

	return result, nil

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

func UpdateGeneralAgentConfig(ctx *gin.Context, userId, orgId string, req request.UpdateGeneralAgentConfigReq) error {
	// 校验 assistant 配置
	assistantList := make([]*assistant_service.WgaConfigAssistant, 0, len(req.AssistantList))
	for _, a := range req.AssistantList {
		assistantList = append(assistantList, &assistant_service.WgaConfigAssistant{
			AssistantId:   a.AssistantID,
			AssistantType: a.AssistantType,
		})
	}
	if err := checkWgaAssistantConfig(ctx, userId, orgId, assistantList); err != nil {
		return err
	}
	// 校验 tool 配置
	toolList := make([]*assistant_service.WgaConfigTool, 0, len(req.ToolList))
	for _, t := range req.ToolList {
		toolList = append(toolList, &assistant_service.WgaConfigTool{
			ToolId:   t.ToolID,
			ToolType: t.ToolType,
		})
	}
	if err := checkWgaToolConfig(ctx, userId, orgId, toolList); err != nil {
		return err
	}

	_, err := assistant.UpdateWgaConfig(ctx.Request.Context(), &assistant_service.UpdateWgaConfigReq{
		ToolList:      toolList,
		AssistantList: assistantList,
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
	for _, t := range resp.Config.ToolList {
		if t.ToolType == constant.ToolTypeBuiltIn {
			_, err := mcp.GetSquareTool(ctx.Request.Context(), &mcp_service.GetSquareToolReq{
				ToolSquareId: t.ToolId,
				Identity: &mcp_service.Identity{
					UserId: userId,
					OrgId:  orgId,
				},
			})
			if err != nil {
				continue
			}
		}
		result.ToolList = append(result.ToolList, request.ToolSelected{
			ToolID:   t.ToolId,
			ToolType: t.ToolType,
		})
	}

	// 过滤存在的 assistant
	assistantIds := make([]string, 0, len(resp.Config.AssistantList))
	for _, a := range resp.Config.AssistantList {
		assistantIds = append(assistantIds, a.AssistantId)
	}

	if len(assistantIds) > 0 {
		assistantResp, err := assistant.GetAssistantByIds(ctx.Request.Context(), &assistant_service.GetAssistantByIdsReq{
			AssistantIdList: assistantIds,
			Identity: &assistant_service.Identity{
				UserId: userId,
				OrgId:  orgId,
			},
		})
		if err == nil && len(assistantResp.AssistantInfos) > 0 {
			validAssistantIds := make(map[string]bool)
			for _, info := range assistantResp.AssistantInfos {
				validAssistantIds[info.Info.AppId] = true
			}
			for _, a := range resp.Config.AssistantList {
				if validAssistantIds[a.AssistantId] {
					result.AssistantList = append(result.AssistantList, request.AssistantSelected{
						AssistantID:   a.AssistantId,
						AssistantType: a.AssistantType,
					})
				}
			}
		}
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

	// 同步删除 ES 中的聊天历史
	_, err = assistant.DeleteFromES(ctx.Request.Context(), &assistant_service.DeleteFromESReq{
		IndexName: wgaConversationHistoryEventESIndexName,
		Conditions: map[string]string{
			"threadId": req.ThreadID,
			"userId":   userId,
			"orgId":    orgId,
		},
	})
	if err != nil {
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

	// 处理模型配置 - 需要验证模型是否存在
	if wgaConfig.ModelConfig != nil && wgaConfig.ModelConfig.ModelId != "" {
		// 验证模型是否存在
		modelInfo, err := model.GetModel(ctx.Request.Context(), &model_service.GetModelReq{ModelId: wgaConfig.ModelConfig.ModelId})
		if err == nil && modelInfo != nil {
			// 模型存在，返回配置
			result.ModelConfig = request.AppModelConfig{
				Provider:    wgaConfig.ModelConfig.Provider,
				Model:       wgaConfig.ModelConfig.Model,
				ModelId:     wgaConfig.ModelConfig.ModelId,
				ModelType:   wgaConfig.ModelConfig.ModelType,
				DisplayName: modelInfo.DisplayName,
			}
		}
		// 如果模型不存在，返回空的 ModelConfig
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

	// 构建工具配置选项
	toolOpts, err := buildWgaToolOptions(ctx, userId, orgId, wgaConfig.ToolList)
	if err != nil {
		return nil, err
	}
	opts = append(opts, toolOpts...)

	// 检查配置
	checkResult, err := wga.CheckOptions(ctx.Request.Context(), config.WgaCfg().AgentID, opts...)
	if err != nil {
		return nil, err
	}

	result := &response.GeneralAgentConfigCheckResponse{
		ModelMeet: checkResult.Model.Meet,
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

	result.Valid = result.ModelMeet
	for _, tc := range result.ToolsMeet {
		if !tc.Meet {
			result.Valid = false
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

// --- internal ---

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

// checkWgaToolConfig 校验工具配置是否存在且是 WGA 所需的（用于运行前检查）
func checkWgaToolConfig(ctx *gin.Context, userId, orgId string, toolList []*assistant_service.WgaConfigTool) error {
	if len(toolList) == 0 {
		return nil
	}

	// 获取 wga 允许的 tool 名称列表
	toolCategories, err := wga.GetAgentToolCategories(config.WgaCfg().AgentID)
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("get agent tool categories failed: %v", err))
	}

	validToolTitles := make(map[string]bool)
	for _, tc := range toolCategories {
		for _, t := range tc.Tools {
			validToolTitles[t.Title] = true
		}
	}

	for _, t := range toolList {
		switch t.ToolType {
		case constant.ToolTypeBuiltIn:
			// 验证 tool 是否存在
			toolResp, err := mcp.GetSquareTool(ctx.Request.Context(), &mcp_service.GetSquareToolReq{
				ToolSquareId: t.ToolId,
				Identity: &mcp_service.Identity{
					UserId: userId,
					OrgId:  orgId,
				},
			})
			if err != nil {
				return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("tool not found: %s", t.ToolId))
			}

			// 验证 tool 是否在 wga 工具列表中
			if !validToolTitles[toolResp.Info.Name] {
				return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("tool not allowed for wga: %s", toolResp.Info.Name))
			}
		}
	}

	return nil
}

// checkWgaAssistantConfig 校验wga智能体配置（用于更新配置）
func checkWgaAssistantConfig(ctx *gin.Context, userId, orgId string, assistantList []*assistant_service.WgaConfigAssistant) error {
	// 验证 assistant 是否存在且是单智能体
	if len(assistantList) > 0 {
		assistantIds := make([]string, 0, len(assistantList))
		for _, a := range assistantList {
			assistantIds = append(assistantIds, a.AssistantId)
		}
		assistantResp, err := assistant.GetAssistantByIds(ctx.Request.Context(), &assistant_service.GetAssistantByIdsReq{
			AssistantIdList: assistantIds,
			Identity: &assistant_service.Identity{
				UserId: userId,
				OrgId:  orgId,
			},
		})
		if err != nil {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "assistant not found")
		}
		if len(assistantResp.AssistantInfos) != len(assistantList) {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "assistant not found")
		}
		// 验证所有 assistant 都是单智能体
		for _, info := range assistantResp.AssistantInfos {
			if info.Category != constant.AgentCategorySingle {
				return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, fmt.Sprintf("assistant not single agent: %s", info.Info.Name))
			}
		}
	}
	return nil
}

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
