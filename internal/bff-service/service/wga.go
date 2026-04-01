package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	model_service "github.com/UnicomAI/wanwu/api/proto/model-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/pkg/constant"
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
		appBriefInfo := appBriefProto2Model(ctx, assistantInfo.Info, assistantInfo.Category)
		if appBriefInfo.Avatar.Path != "" {
			appBriefInfo.Avatar.Path = path.Join(config.Cfg().Server.ApiBaseUrl, appBriefInfo.Avatar.Path)
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

	agentCategories, err := wga.GetAgentToolCategories(config.WgaCfg().AgentID)
	if err != nil {
		return nil, err
	}

	result := make([]response.GetGeneralAgentToolSelectResp, 0, len(agentCategories))
	for _, tc := range agentCategories {
		categoryResp := response.GetGeneralAgentToolSelectResp{
			Category:  tc.Category,
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
				if toolInfo.Avatar.Path != "" {
					toolInfo.Avatar.Path = path.Join(config.Cfg().Server.ApiBaseUrl, toolInfo.Avatar.Path)
				}
			}

			categoryResp.ToolList = append(categoryResp.ToolList, toolInfo)
		}

		result = append(result, categoryResp)
	}

	return result, nil

}

func UpdateGeneralAgentConfig(ctx *gin.Context, userId, orgId string, req request.UpdateGeneralAgentConfigReq) error {
	toolList := make([]*assistant_service.WgaConfigTool, 0, len(req.ToolList))
	for _, t := range req.ToolList {
		toolList = append(toolList, &assistant_service.WgaConfigTool{
			ToolId:   t.ToolID,
			ToolType: t.ToolType,
		})
	}

	assistantList := make([]*assistant_service.WgaConfigAssistant, 0, len(req.AssistantList))
	for _, a := range req.AssistantList {
		assistantList = append(assistantList, &assistant_service.WgaConfigAssistant{
			AssistantId:   a.AssistantID,
			AssistantType: a.AssistantType,
		})
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
	for _, t := range resp.Config.ToolList {
		result.ToolList = append(result.ToolList, request.ToolSelected{
			ToolID:   t.ToolId,
			ToolType: t.ToolType,
		})
	}
	for _, a := range resp.Config.AssistantList {
		result.AssistantList = append(result.AssistantList, request.AssistantSelected{
			AssistantID:   a.AssistantId,
			AssistantType: a.AssistantType,
		})
	}

	return result, nil
}

func GetGeneralAgentToolInfo(ctx *gin.Context, userId, orgId string, toolId, toolType string) (*response.GeneralAgentToolInfoResp, error) {
	resp, err := mcp.GetSquareTool(ctx.Request.Context(), &mcp_service.GetSquareToolReq{
		ToolSquareId: toolId,
		Identity: &mcp_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}

	var actions []*protocol.Tool
	if resp.BuiltInTools != nil {
		for _, tool := range resp.BuiltInTools.Tools {
			actions = append(actions, toToolAction(tool))
		}
	}

	toolAvatar := cacheToolAvatar(ctx, constant.ToolTypeBuiltIn, resp.Info.AvatarPath)
	if toolAvatar.Path != "" {
		toolAvatar.Path = path.Join(config.Cfg().Server.ApiBaseUrl, toolAvatar.Path)
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
			Avatar:          toolAvatar,
		},
	}, nil
}

func CreateGeneralAgentConversation(ctx *gin.Context, userId, orgId string, req request.CreateGeneralAgentConversationReq) (*response.CreateGeneralAgentConversationResp, error) {
	if err := checkCreateConversationConfig(ctx, req); err != nil {
		return nil, err
	}

	var modelConfig *assistant_service.WgaModelConfig
	if req.ModelConfig != nil && req.ModelConfig.ModelId != "" {
		var configJSON string
		if req.ModelConfig.Config != nil {
			configBytes, _ := json.Marshal(req.ModelConfig.Config)
			configJSON = string(configBytes)
		}
		modelConfig = &assistant_service.WgaModelConfig{
			ModelId:   req.ModelConfig.ModelId,
			Provider:  req.ModelConfig.Provider,
			Model:     req.ModelConfig.Model,
			ModelType: req.ModelConfig.ModelType,
			Config:    configJSON,
		}
	}

	resp, err := assistant.WgaConversationCreate(ctx.Request.Context(), &assistant_service.WgaConversationCreateReq{
		Prompt:           req.Title,
		ModelConfig:      modelConfig,
		ConversationType: constant.ConversationTypeWga,
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

func GetGeneralAgentConversationList(ctx *gin.Context, userId, orgId string, req request.GetGeneralAgentConversationListReq) (*response.ListResult, error) {
	resp, err := assistant.WgaConversationList(ctx.Request.Context(), &assistant_service.WgaConversationListReq{
		ConversationType: constant.ConversationTypeWga,
		PageSize:         int32(req.PageSize),
		PageNo:           int32(req.PageNo),
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

func GetGeneralAgentConversationDetail(ctx *gin.Context, userId, orgId string, req request.GetGeneralAgentConversationDetailReq) (*response.ListResult, error) {
	conditions := map[string]string{
		"threadId": req.ThreadID,
		"userId":   userId,
		"orgId":    orgId,
	}

	pageNo := req.PageNo
	if pageNo <= 0 {
		pageNo = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 1000
	}

	resp, err := assistant.SearchFromES(ctx.Request.Context(), &assistant_service.SearchFromESReq{
		IndexName:  constant.ESIndexWgaChatHistory,
		Conditions: conditions,
		SortOrder:  "asc",
		PageNo:     int32(pageNo),
		PageSize:   int32(pageSize),
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

		createdAt, _ := doc["createdAt"].(float64)
		runId, _ := doc["runId"].(string)

		info := response.GeneralAgentConversationDetailInfo{
			ThreadID:     req.ThreadID,
			RunID:        runId,
			CreatedAt:    int64(createdAt),
			RequestFiles: []response.AssistantRequestFile{},
		}

		if messages, ok := doc["messages"].([]interface{}); ok {
			info.Messages = messages
		}

		if workspace, ok := doc["workspace"].(map[string]interface{}); ok {
			info.Workspace = response.GeneralAgentConversationWorkspaceInfo{
				ThreadID:  req.ThreadID,
				RunID:     runId,
				FileCount: int32(getFloat64(workspace["fileCount"])),
				TotalSize: int64(getFloat64(workspace["totalSize"])),
				IsDisplay: true,
			}
		}

		result = append(result, info)
	}

	return &response.ListResult{List: result, Total: int64(len(result))}, nil
}

func getFloat64(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
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
		IndexName: constant.ESIndexWgaChatHistory,
		Conditions: map[string]string{
			"threadId": req.ThreadID,
			"userId":   userId,
			"orgId":    orgId,
		},
	})
	if err != nil {
		log.Warnf("[wga] failed to delete chat history from ES: %v", err)
	}

	return nil
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

	// 处理模型配置
	if wgaConfig.ModelConfig != nil && wgaConfig.ModelConfig.ModelId != "" {
		result.ModelConfig = request.AppModelConfig{
			Provider:    wgaConfig.ModelConfig.Provider,
			Model:       wgaConfig.ModelConfig.Model,
			ModelId:     wgaConfig.ModelConfig.ModelId,
			ModelType:   wgaConfig.ModelConfig.ModelType,
			DisplayName: wgaConfig.ModelConfig.Model,
		}
		if wgaConfig.ModelConfig.Config != "" {
			var modelConfig interface{}
			if err := json.Unmarshal([]byte(wgaConfig.ModelConfig.Config), &modelConfig); err == nil {
				result.ModelConfig.Config = modelConfig
			}
		}
	}

	return result, nil
}

// CheckGeneralAgentConfig 检查WGA对话配置
func CheckGeneralAgentConfig(ctx *gin.Context, userId, orgId string, req request.GeneralAgentConfigCheckRequest) (*response.GeneralAgentConfigCheckResponse, error) {
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

	// 构建参数
	// 模型信息
	var opts []wga_option.Option
	if wgaConversationConfig.ModelConfig != nil && wgaConversationConfig.ModelConfig.ModelId != "" {
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
	}

	// 工具信息
	for _, tool := range wgaConfig.ToolList {
		switch tool.ToolType {
		// 仅处理了内置工具
		case constant.ToolTypeBuiltIn:
			toolResp, err := mcp.GetSquareTool(ctx.Request.Context(), &mcp_service.GetSquareToolReq{
				ToolSquareId: tool.ToolId,
				Identity: &mcp_service.Identity{
					UserId: userId,
					OrgId:  orgId,
				},
			})
			if err != nil {
				return nil, err
			}
			var toolDetail = toToolSquareDetail(ctx, toolResp)

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

			toolConfig := wga_option.ToolConfig{
				Title:   toolDetail.ToolSquareInfo.Name,
				APIAuth: apiAuth,
			}
			opts = append(opts, wga_option.WithToolConfig(toolConfig))
		}
	}

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

func UpdateGeneralAgentConversationConfig(ctx *gin.Context, userId, orgId string, req request.UpdateGeneralAgentConversationConfigReq) error {
	if err := checkUpdateConfig(ctx, userId, orgId, req); err != nil {
		return err
	}

	var modelConfig *assistant_service.WgaModelConfig
	if req.ModelConfig != nil && req.ModelConfig.ModelId != "" {
		var configJSON string
		if req.ModelConfig.Config != nil {
			configBytes, _ := json.Marshal(req.ModelConfig.Config)
			configJSON = string(configBytes)
		}
		modelConfig = &assistant_service.WgaModelConfig{
			ModelId:   req.ModelConfig.ModelId,
			Provider:  req.ModelConfig.Provider,
			Model:     req.ModelConfig.Model,
			ModelType: req.ModelConfig.ModelType,
			Config:    configJSON,
		}
	}

	_, err := assistant.UpdateWgaConversationConfig(ctx.Request.Context(), &assistant_service.UpdateWgaConversationConfigReq{
		ThreadId:    req.ThreadID,
		ModelConfig: modelConfig,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	return err
}

func GeneralAgentWorkspaceDownload(ctx *gin.Context, userId, orgId string, req request.GeneralAgentWorkspaceDownloadReq) (string, []byte, error) {
	cfg := config.WgaCfg()
	if !cfg.Persistent.Enabled {
		return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "persistent not enabled")
	}

	store, err := wga_persistent.NewStore(wga_persistent.ModeVersioned, cfg.Persistent.BaseDir, req.ThreadID)
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
		tarName := fmt.Sprintf("workspace_%s_%s.tar", req.RunID, filepath.Base(req.Path))
		tarData, err := util.TarDir(targetPath+"/.", false)
		if err != nil {
			return "", nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to create tar: %v", err))
		}
		return tarName, tarData, nil
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

	store, err := wga_persistent.NewStore(wga_persistent.ModeVersioned, cfg.Persistent.BaseDir, req.ThreadID)
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

	store, err := wga_persistent.NewStore(wga_persistent.ModeVersioned, cfg.Persistent.BaseDir, req.ThreadID)
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
	files, err := buildFileTree(workDir, "")
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("failed to read directory: %v", err))
	}

	return &response.GeneralAgentWorkspaceResp{
		GeneralAgentConversationWorkspaceInfo: response.GeneralAgentConversationWorkspaceInfo{
			ThreadID:  req.ThreadID,
			RunID:     req.RunID,
			FileCount: int32(len(files)),
			TotalSize: calculateTotalSize(files),
			IsDisplay: true,
		},
		Path:  "",
		Files: files,
	}, nil
}

// --- internal ---

func checkCreateConversationConfig(ctx *gin.Context, req request.CreateGeneralAgentConversationReq) error {
	if req.ModelConfig != nil && req.ModelConfig.ModelId != "" {
		_, err := model.GetModel(ctx.Request.Context(), &model_service.GetModelReq{ModelId: req.ModelConfig.ModelId})
		if err != nil {
			return fmt.Errorf("model not found: %s", req.ModelConfig.ModelId)
		}
	}
	return nil
}

func checkUpdateConfig(ctx *gin.Context, userId, orgId string, req request.UpdateGeneralAgentConversationConfigReq) error {
	if req.ModelConfig != nil && req.ModelConfig.ModelId != "" {
		_, err := model.GetModel(ctx.Request.Context(), &model_service.GetModelReq{ModelId: req.ModelConfig.ModelId})
		if err != nil {
			return fmt.Errorf("model not found: %s", req.ModelConfig.ModelId)
		}
	}

	// if len(req.AssistantList) > 0 {
	// 	assistantIds := make([]string, 0, len(req.AssistantList))
	// 	for _, a := range req.AssistantList {
	// 		assistantIds = append(assistantIds, a.AssistantID)
	// 	}
	// 	assistantResp, err := assistant.GetAssistantByIds(ctx.Request.Context(), &assistant_service.GetAssistantByIdsReq{
	// 		AssistantIdList: assistantIds,
	// 		Identity: &assistant_service.Identity{
	// 			UserId: userId,
	// 			OrgId:  orgId,
	// 		},
	// 	})
	// 	if err != nil {
	// 		return fmt.Errorf("assistant check failed")
	// 	}
	// 	if len(assistantResp.AssistantInfos) != len(req.AssistantList) {
	// 		return fmt.Errorf("assistant not found")
	// 	}
	// }

	// if len(req.ToolList) > 0 {
	// 	agentCategories, err := wga.GetAgentToolCategories(config.WgaCfg().AgentID)
	// 	if err != nil {
	// 		return fmt.Errorf("get agent tool categories failed: %v", err)
	// 	}

	// 	validToolTitles := make(map[string]bool)
	// 	for _, tc := range agentCategories {
	// 		for _, t := range tc.Tools {
	// 			validToolTitles[t.Title] = true
	// 		}
	// 	}

	// 	var opts []wga_option.Option

	// 	for _, t := range req.ToolList {
	// 		switch t.ToolType {
	// 		case constant.ToolTypeBuiltIn:
	// 			toolResp, err := mcp.GetSquareTool(ctx.Request.Context(), &mcp_service.GetSquareToolReq{
	// 				ToolSquareId: t.ToolID,
	// 				Identity: &mcp_service.Identity{
	// 					UserId: userId,
	// 					OrgId:  orgId,
	// 				},
	// 			})
	// 			if err != nil {
	// 				return fmt.Errorf("tool not found: %s", t.ToolID)
	// 			}

	// 			if !validToolTitles[toolResp.Info.Name] {
	// 				return fmt.Errorf("tool %s is not in wga tool categories", toolResp.Info.Name)
	// 			}

	// 			toolDetail := toToolSquareDetail(ctx, toolResp)

	// 			authType := toolDetail.ApiAuth.AuthType
	// 			if authType == "" {
	// 				authType = util.AuthTypeNone
	// 			}
	// 			apiAuth := &util.ApiAuthWebRequest{
	// 				AuthType:           authType,
	// 				ApiKeyHeaderPrefix: toolDetail.ApiAuth.ApiKeyHeaderPrefix,
	// 				ApiKeyHeader:       toolDetail.ApiAuth.ApiKeyHeader,
	// 				ApiKeyQueryParam:   toolDetail.ApiAuth.ApiKeyQueryParam,
	// 				ApiKeyValue:        toolDetail.ApiAuth.ApiKeyValue,
	// 			}

	// 			toolConfig := wga_option.ToolConfig{
	// 				Title:   toolDetail.ToolSquareInfo.Name,
	// 				APIAuth: apiAuth,
	// 			}
	// 			opts = append(opts, wga_option.WithToolConfig(toolConfig))
	// 		}
	// 	}

	// 	checkResult, err := wga.CheckOptions(ctx.Request.Context(), config.WgaCfg().AgentID, opts...)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	for _, tc := range checkResult.ToolCategories {
	// 		if tc.Condition == "required" && !tc.Meet {
	// 			return fmt.Errorf("required tool category not met: %s", tc.Category)
	// 		}
	// 	}
	// }

	return nil
}

func buildFileTree(dirPath, parentPath string) ([]response.GeneralAgentFileInfo, error) {
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
			children, err := buildFileTree(filepath.Join(dirPath, entry.Name()), filePath)
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

func calculateTotalSize(files []response.GeneralAgentFileInfo) int64 {
	var total int64
	for _, f := range files {
		if f.Type == "directory" {
			total += calculateTotalSize(f.Children)
		} else {
			total += f.Size
		}
	}
	return total
}
