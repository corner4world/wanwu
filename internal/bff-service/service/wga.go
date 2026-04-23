package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	mp "github.com/UnicomAI/wanwu/pkg/model-provider"
	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
	openapi3_util "github.com/UnicomAI/wanwu/pkg/openapi3-util"
	"github.com/UnicomAI/wanwu/pkg/util"
	wga_persistent "github.com/UnicomAI/wanwu/pkg/wga-persistent"
	wga_option "github.com/UnicomAI/wanwu/pkg/wga/wga-option"
	"github.com/gin-gonic/gin"
)

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
