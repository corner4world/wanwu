package service

import (
	"context"
	"encoding/json"
	"fmt"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	"github.com/UnicomAI/wanwu/api/proto/common"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	model_service "github.com/UnicomAI/wanwu/api/proto/model-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/UnicomAI/wanwu/pkg/wga"
	wga_persistent "github.com/UnicomAI/wanwu/pkg/wga-persistent"
	wga_sandbox "github.com/UnicomAI/wanwu/pkg/wga-sandbox"
	wga_option "github.com/UnicomAI/wanwu/pkg/wga/wga-option"
	"github.com/gin-gonic/gin"
)

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
	store, err := NewGeneralAgentWorkspaceStore(req.ThreadID)
	if err != nil {
		log.Errorf("[wga] thread %v create persistent store err: %v", req.ThreadID, err)
	} else {
		if err := CleanupWgaWorkspace(store); err != nil {
			log.Errorf("[wga] thread %v delete persistent dir err: %v", req.ThreadID, err)
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

// GeneralAgentConversationChat 通用智能体对话接口
func GeneralAgentConversationChat(ctx *gin.Context, userId, orgId string, req request.GeneralAgentConversationChatReq) error {
	// 获取 threadId 的 ModelConfig
	configResp, err := assistant.GetWgaConversationConfig(ctx.Request.Context(), &assistant_service.GetWgaConversationConfigReq{
		ThreadId: req.ThreadID,
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return err
	}
	var modelConfig *common.AppModelConfig
	if configResp.Config != nil {
		modelConfig = configResp.Config.ModelConfig
	}

	// 获取 threadId 的 workspace store
	var workspaceStore *wga_persistent.Store
	if config.WgaCfg().Persistent.Enabled {
		store, err := NewGeneralAgentWorkspaceStore(req.ThreadID)
		if err != nil {
			log.Errorf("[wga] thread %v failed to create persistent store: %v", req.ThreadID, err)
		} else {
			workspaceStore = store
		}
	}

	agentID := req.AgentID
	if agentID == "" {
		agentID = config.WgaCfg().AgentID
	}

	return WgaConversationChat(ctx, &WgaChatParams{
		UserID:         userId,
		OrgID:          orgId,
		AgentID:        agentID,
		ThreadID:       req.ThreadID,
		Messages:       req.Messages,
		ModelConfig:    modelConfig,
		WorkspaceStore: workspaceStore,
	})
}

func GeneralAgentReplyQuestion(ctx context.Context, runID string, questionID string, answers [][]string) error {
	sandboxCfg, err := getWgaSandboxConfig()
	if err != nil {
		return err
	}

	if config.Cfg().Ontology.Enable != 0 {
		go func() {
			defer util.PrintPanicStack()
			ontologySandboxCfg, err := getWgaSandboxOntologyConfig()
			if err != nil {
				log.Errorf("get ontology sandbox config failed: %v", err)
				return
			}
			if err := wga_sandbox.ReplyQuestion(context.Background(), ontologySandboxCfg, runID, questionID, answers); err != nil {
				log.Warnf("reply ontology question failed, runID: %s, questionID: %s, err: %v", runID, questionID, err)
			}
		}()
	}

	return wga_sandbox.ReplyQuestion(ctx, sandboxCfg, runID, questionID, answers)
}

func GeneralAgentRejectQuestion(ctx context.Context, runID string, questionID string) error {
	sandboxCfg, err := getWgaSandboxConfig()
	if err != nil {
		return err
	}

	if config.Cfg().Ontology.Enable != 0 {
		go func() {
			defer util.PrintPanicStack()
			ontologySandboxCfg, err := getWgaSandboxOntologyConfig()
			if err != nil {
				log.Errorf("get ontology sandbox config failed: %v", err)
				return
			}
			if err := wga_sandbox.RejectQuestion(context.Background(), ontologySandboxCfg, runID, questionID); err != nil {
				log.Warnf("reject ontology question failed, runID: %s, questionID: %s, err: %v", runID, questionID, err)
			}
		}()
	}

	return wga_sandbox.RejectQuestion(ctx, sandboxCfg, runID, questionID)
}
