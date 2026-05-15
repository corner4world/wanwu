package service

import (
	"fmt"
	"path/filepath"
	"strings"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	"github.com/UnicomAI/wanwu/api/proto/common"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
)

func ConvertGeneralAgentSkillConversation(ctx *gin.Context, userId, orgId string, req request.ConvertGeneralAgentSkillConversationReq) (*response.ConvertGeneralAgentSkillConversationResp, error) {
	if err := checkModelConfig(ctx, req.ModelConfig); err != nil {
		return nil, err
	}
	modelConfigString, err := req.ModelConfig.ConfigString()
	if err != nil {
		return nil, grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, err.Error())
	}

	sourceType := normalizeGeneralAgentSkillConvertType(req.Type)
	title := generalAgentSkillConvertTitle(sourceType)
	previewID := util.GenUUID()

	threadResp, err := assistant.WgaConversationCreate(ctx.Request.Context(), &assistant_service.WgaConversationCreateReq{
		Prompt: title,
		ModelConfig: &common.AppModelConfig{
			ModelId:   req.ModelConfig.ModelId,
			Provider:  req.ModelConfig.Provider,
			Model:     req.ModelConfig.Model,
			ModelType: req.ModelConfig.ModelType,
			Config:    modelConfigString,
		},
		Identity: &assistant_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
	})
	if err != nil {
		return nil, err
	}

	customSkillResp, err := mcp.CustomSkillCreate(ctx.Request.Context(), &mcp_service.CustomSkillCreateReq{
		Name:            title,
		Author:          req.Author,
		WgaThreadId:     threadResp.ThreadId,
		PreviewThreadId: previewID,
		SourceType:      customSkillSourceTypeConvert,
		Identity:        &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		rollbackImportedSkillConversation(ctx, userId, orgId, threadResp.ThreadId, "")
		return nil, err
	}
	customSkillID := customSkillResp.SkillId

	outputDir, err := prepareGeneralAgentSkillConvertOutputDir(customSkillID)
	if err != nil {
		rollbackImportedSkillConversation(ctx, userId, orgId, threadResp.ThreadId, customSkillID)
		return nil, err
	}
	if err := generateGeneralAgentSkillFromSource(ctx, sourceType, req.ID, outputDir); err != nil {
		rollbackImportedSkillConversation(ctx, userId, orgId, threadResp.ThreadId, customSkillID)
		return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("convert skill err: %v", err))
	}

	fm, err := readImportedSkillFrontMatter(outputDir)
	if err != nil {
		rollbackImportedSkillConversation(ctx, userId, orgId, threadResp.ThreadId, customSkillID)
		return nil, grpc_util.ErrorStatus(errs.Code_BFFSkillParse, err.Error())
	}
	if _, err = mcp.UpdateCustomSkillBasicMeta(ctx.Request.Context(), &mcp_service.UpdateCustomSkillBasicMetaReq{
		SkillId: customSkillID,
		Name:    fm.Name,
		Desc:    fm.Description,
	}); err != nil {
		rollbackImportedSkillConversation(ctx, userId, orgId, threadResp.ThreadId, customSkillID)
		return nil, err
	}

	return &response.ConvertGeneralAgentSkillConversationResp{
		CustomSkillID: customSkillID,
		ThreadID:      threadResp.ThreadId,
		PreviewID:     previewID,
	}, nil
}

func prepareGeneralAgentSkillConvertOutputDir(customSkillID string) (string, error) {
	store, err := NewGeneralAgentSkillWorkspaceStore(customSkillID)
	if err != nil {
		return "", err
	}
	outputDir := filepath.Join(GetWgaWorkspaceThreadDir(store), generalAgentSkillImportDirName)
	if err := recreateDir(outputDir); err != nil {
		return "", grpc_util.ErrorStatus(errs.Code_BFFGeneral, fmt.Sprintf("prepare skill dir err: %v", err))
	}
	return outputDir, nil
}

func generateGeneralAgentSkillFromSource(ctx *gin.Context, sourceType, id, outputDir string) error {
	switch sourceType {
	case "mcp":
		return GenerateSkillFromMCP(ctx, id, outputDir)
	case "tool":
		return GenerateSkillFromCustomTool(ctx, id, outputDir)
	case "agent":
		return GenerateSkillFromAgent(ctx, id, outputDir)
	case "workflow":
		return GenerateSkillFromWorkflow(ctx, id, outputDir)
	case "rag":
		return GenerateSkillFromRAG(ctx, id, outputDir)
	default:
		return fmt.Errorf("unsupported type: %s", sourceType)
	}
}

func normalizeGeneralAgentSkillConvertType(sourceType string) string {
	return strings.TrimSpace(strings.ToLower(sourceType))
}

func generalAgentSkillConvertTitle(sourceType string) string {
	switch sourceType {
	case "mcp":
		return "Convert MCP Skill"
	case "tool":
		return "Convert Tool Skill"
	case "agent":
		return "Convert Agent Skill"
	case "workflow":
		return "Convert Workflow Skill"
	case "rag":
		return "Convert RAG Skill"
	default:
		return "Convert Skill"
	}
}
