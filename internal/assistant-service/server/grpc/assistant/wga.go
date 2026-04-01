package assistant

import (
	"context"
	"encoding/json"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/assistant-service/client/model"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Service) GetWgaConversationConfig(ctx context.Context, req *assistant_service.GetWgaConversationConfigReq) (*assistant_service.GetWgaConversationConfigResp, error) {
	if req.ThreadId == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConversationGetErr, "thread_id is required")
	}
	if req.Identity == nil {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConversationGetErr, "identity is required")
	}

	config, status := s.cli.GetWgaConversationConfig(ctx, req.ThreadId, req.Identity.UserId, req.Identity.OrgId)
	if status != nil {
		log.Errorf("获取WGA对话配置失败，conversationId: %s, error: %v", req.ThreadId, status)
		return nil, errStatus(errs.Code_WgaConversationGetErr, status)
	}

	return &assistant_service.GetWgaConversationConfigResp{
		Config: toProtoWgaConversationConfig(config),
	}, nil
}

func (s *Service) UpdateWgaConversationConfig(ctx context.Context, req *assistant_service.UpdateWgaConversationConfigReq) (*emptypb.Empty, error) {
	if req.ThreadId == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConversationUpdateErr, "thread_id is required")
	}
	if req.Identity == nil {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConversationUpdateErr, "identity is required")
	}

	var modelConfigJSON string
	if req.ModelConfig != nil {
		modelConfigBytes, _ := json.Marshal(req.ModelConfig)
		modelConfigJSON = string(modelConfigBytes)
	} else {
		modelConfigJSON = "null"
	}

	config := &model.WgaConversationConfig{
		ThreadID:    req.ThreadId,
		UserID:      req.Identity.UserId,
		OrgID:       req.Identity.OrgId,
		ModelConfig: modelConfigJSON,
	}

	status := s.cli.UpdateWgaConversationConfig(ctx, config)
	if status != nil {
		log.Errorf("更新WGA对话配置失败，conversationId: %s, error: %v", req.ThreadId, status)
		return nil, errStatus(errs.Code_WgaConversationUpdateErr, status)
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) GetWgaConfig(ctx context.Context, req *assistant_service.GetWgaConfigReq) (*assistant_service.GetWgaConfigResp, error) {
	if req.Identity == nil {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigGetErr, "identity is required")
	}

	config, status := s.cli.GetWgaConfig(ctx, req.Identity.UserId, req.Identity.OrgId)
	if status != nil {
		log.Errorf("获取WGA配置失败，userId: %s, orgId: %s, error: %v", req.Identity.UserId, req.Identity.OrgId, status)
		return nil, errStatus(errs.Code_WgaConfigGetErr, status)
	}

	return &assistant_service.GetWgaConfigResp{
		Config: toProtoWgaConfig(config),
	}, nil
}

func (s *Service) UpdateWgaConfig(ctx context.Context, req *assistant_service.UpdateWgaConfigReq) (*emptypb.Empty, error) {
	if req.Identity == nil {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigUpdateErr, "identity is required")
	}

	toolListJSON, _ := json.Marshal(req.ToolList)
	assistantListJSON, _ := json.Marshal(req.AssistantList)

	config := &model.WgaConfig{
		UserID:        req.Identity.UserId,
		OrgID:         req.Identity.OrgId,
		AssistantList: string(assistantListJSON),
		ToolList:      string(toolListJSON),
	}

	status := s.cli.UpdateWgaConfig(ctx, config)
	if status != nil {
		log.Errorf("更新WGA配置失败，userId: %s, orgId: %s, error: %v", req.Identity.UserId, req.Identity.OrgId, status)
		return nil, errStatus(errs.Code_WgaConfigUpdateErr, status)
	}

	return &emptypb.Empty{}, nil
}

func toProtoWgaConversationConfig(m *model.WgaConversationConfig) *assistant_service.WgaConversationConfig {
	config := &assistant_service.WgaConversationConfig{
		ThreadId:  m.ThreadID,
		UserId:    m.UserID,
		OrgId:     m.OrgID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.ModelConfig != "" {
		var modelConfig assistant_service.WgaModelConfig
		if err := json.Unmarshal([]byte(m.ModelConfig), &modelConfig); err == nil {
			config.ModelConfig = &modelConfig
		}
	}

	return config
}

func toProtoWgaConfig(m *model.WgaConfig) *assistant_service.WgaConfig {
	config := &assistant_service.WgaConfig{
		UserId:    m.UserID,
		OrgId:     m.OrgID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.ToolList != "" {
		var tools []assistant_service.WgaConfigTool
		if err := json.Unmarshal([]byte(m.ToolList), &tools); err == nil {
			for _, t := range tools {
				config.ToolList = append(config.ToolList, &assistant_service.WgaConfigTool{
					ToolId:   t.ToolId,
					ToolType: t.ToolType,
				})
			}
		}
	}

	if m.AssistantList != "" {
		var assistants []assistant_service.WgaConfigAssistant
		if err := json.Unmarshal([]byte(m.AssistantList), &assistants); err == nil {
			for _, a := range assistants {
				config.AssistantList = append(config.AssistantList, &assistant_service.WgaConfigAssistant{
					AssistantId:   a.AssistantId,
					AssistantType: a.AssistantType,
				})
			}
		}
	}

	return config
}
