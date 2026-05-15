package service

import (
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/gin-gonic/gin"
)

func CreateCustomSkillConfig(ctx *gin.Context, userId, orgId string, req request.SkillConfigReq) error {
	_, err := mcp.CreateCustomSkillVar(ctx.Request.Context(), &mcp_service.CreateCustomSkillVarReq{
		SkillId:  req.SkillId,
		Variable: toMcpSkillVariable(req.Variable),
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	return err
}

func UpdateCustomSkillConfig(ctx *gin.Context, userId, orgId string, req request.UpdateSkillConfigReq) error {
	_, err := mcp.UpdateCustomSkillVar(ctx.Request.Context(), &mcp_service.UpdateCustomSkillVarReq{
		Id:       req.ID,
		Variable: toMcpSkillVariable(req.Variable),
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	return err
}

func DeleteCustomSkillConfig(ctx *gin.Context, userId, orgId string, req request.DeleteSkillConfigReq) error {
	_, err := mcp.DeleteCustomSkillVar(ctx.Request.Context(), &mcp_service.DeleteCustomSkillVarReq{
		Id:       req.ID,
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	return err
}

func CreateAcquiredSkillConfig(ctx *gin.Context, userId, orgId string, req request.SkillConfigReq) error {
	if _, err := getOwnedAcquiredSkill(ctx, userId, orgId, req.SkillId); err != nil {
		return err
	}
	_, err := mcp.CreateAcquiredSkillVar(ctx.Request.Context(), &mcp_service.CreateAcquiredSkillVarReq{
		SkillId:  req.SkillId,
		Variable: toMcpSkillVariable(req.Variable),
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	return err
}

func UpdateAcquiredSkillConfig(ctx *gin.Context, userId, orgId string, req request.UpdateSkillConfigReq) error {
	_, err := mcp.UpdateAcquiredSkillVar(ctx.Request.Context(), &mcp_service.UpdateAcquiredSkillVarReq{
		Id:       req.ID,
		Variable: toMcpSkillVariable(req.Variable),
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	return err
}

func DeleteAcquiredSkillConfig(ctx *gin.Context, userId, orgId string, req request.DeleteSkillConfigReq) error {
	_, err := mcp.DeleteAcquiredSkillVar(ctx.Request.Context(), &mcp_service.DeleteAcquiredSkillVarReq{
		Id:       req.ID,
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	return err
}

func CreateBuiltinSkillConfig(ctx *gin.Context, userId, orgId string, req request.SkillConfigReq) error {
	if _, exist := config.Cfg().AgentSkill(req.SkillId); !exist {
		return grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "skill not found in builtin skills")
	}
	_, err := mcp.CreateBuiltinSkillVar(ctx.Request.Context(), &mcp_service.CreateBuiltinSkillVarReq{
		SkillId:  req.SkillId,
		Variable: toMcpSkillVariable(req.Variable),
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	return err
}

func UpdateBuiltinSkillConfig(ctx *gin.Context, userId, orgId string, req request.UpdateSkillConfigReq) error {
	_, err := mcp.UpdateBuiltinSkillVar(ctx.Request.Context(), &mcp_service.UpdateBuiltinSkillVarReq{
		Id:       req.ID,
		Variable: toMcpSkillVariable(req.Variable),
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	return err
}

func DeleteBuiltinSkillConfig(ctx *gin.Context, userId, orgId string, req request.DeleteSkillConfigReq) error {
	_, err := mcp.DeleteBuiltinSkillVar(ctx.Request.Context(), &mcp_service.DeleteBuiltinSkillVarReq{
		Id:       req.ID,
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	return err
}

func getOwnedAcquiredSkill(ctx *gin.Context, userId, orgId, skillId string) (*mcp_service.AcquiredSkill, error) {
	if skillId == "" {
		return nil, grpc_util.ErrorStatus(errs.Code_BFFInvalidArg, "skillId is required")
	}
	resp, err := mcp.AcquiredSkillGetList(ctx.Request.Context(), &mcp_service.AcquiredSkillGetListReq{
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
	})
	if err != nil {
		return nil, err
	}
	for _, skill := range resp.List {
		if skill.AcquiredSkillId == skillId {
			return skill, nil
		}
	}
	return nil, grpc_util.ErrorStatus(errs.Code_BFFGeneral, "skill_not_found", "acquired skill not found")
}

func toMcpSkillVariable(v request.SkillVariable) *mcp_service.Variable {
	return &mcp_service.Variable{
		Name:          v.Name,
		Desc:          v.Desc,
		VariableKey:   v.VariableKey,
		VariableValue: v.VariableValue,
	}
}

func toSkillVariables(variables []*mcp_service.Variable) []*response.SkillVariable {
	ret := make([]*response.SkillVariable, 0, len(variables))
	for _, variable := range variables {
		if variable == nil {
			continue
		}
		ret = append(ret, &response.SkillVariable{
			ID:            variable.Id,
			Name:          variable.Name,
			Desc:          variable.Desc,
			VariableKey:   variable.VariableKey,
			VariableValue: variable.VariableValue,
		})
	}
	return ret
}
