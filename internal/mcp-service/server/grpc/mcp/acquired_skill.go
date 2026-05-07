package mcp

import (
	"context"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/mcp-service/client/model"
	"github.com/UnicomAI/wanwu/pkg/util"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Service) AcquiredSkillCreate(ctx context.Context, req *mcp_service.AcquiredSkillCreateReq) (*mcp_service.AcquiredSkillCreateResp, error) {
	if req.GetIdentity() == nil {
		return nil, errStatus(errs.Code_MCPAcquiredSkillErr, toErrStatus("mcp_acquired_skill_create", "identity is empty"))
	}
	acquiredSkillId, err := s.cli.CreateAcquiredSkill(ctx, &model.AcquiredSkill{
		SquareSkillID:      req.SquareSkillId,
		Name:               req.Name,
		Avatar:             req.Avatar,
		Author:             req.Author,
		AuthorID:           req.AuthorId,
		Desc:               req.Desc,
		ObjectPath:         req.ObjectPath,
		Markdown:           req.Markdown,
		Version:            req.Version,
		VersionDescription: req.VersionDescription,
		UserID:             req.GetIdentity().GetUserId(),
		OrgID:              req.GetIdentity().GetOrgId(),
	})
	if err != nil {
		return nil, errStatus(errs.Code_MCPAcquiredSkillErr, err)
	}

	return &mcp_service.AcquiredSkillCreateResp{AcquiredSkillId: acquiredSkillId}, nil
}

func (s *Service) AcquiredSkillDelete(ctx context.Context, req *mcp_service.AcquiredSkillDeleteReq) (*emptypb.Empty, error) {
	err := s.cli.DeleteAcquiredSkill(ctx, req.AcquiredSkillId)
	if err != nil {
		return nil, errStatus(errs.Code_MCPAcquiredSkillErr, err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) AcquiredSkillGet(ctx context.Context, req *mcp_service.AcquiredSkillGetReq) (*mcp_service.AcquiredSkill, error) {
	acquiredSkill, err := s.cli.GetAcquiredSkill(ctx, req.AcquiredSkillId)
	if err != nil {
		return nil, errStatus(errs.Code_MCPAcquiredSkillErr, err)
	}

	variables, err := s.cli.GetAcquiredSkillVars(ctx, acquiredSkill.UserID, acquiredSkill.OrgID, req.AcquiredSkillId)
	if err != nil {
		return nil, errStatus(errs.Code_MCPAcquiredSkillErr, err)
	}

	return toAcquiredSkillInfo(acquiredSkill, toAcquiredSkillVariables(variables)), nil
}

func (s *Service) AcquiredSkillGetList(ctx context.Context, req *mcp_service.AcquiredSkillGetListReq) (*mcp_service.AcquiredSkillGetListResp, error) {
	if req.GetIdentity() == nil {
		return nil, errStatus(errs.Code_MCPAcquiredSkillErr, toErrStatus("mcp_acquired_skill_list", "identity is empty"))
	}
	userId, orgId := req.GetIdentity().GetUserId(), req.GetIdentity().GetOrgId()
	acquiredSkills, total, err := s.cli.GetAcquiredSkillList(ctx, userId, orgId, req.Name)
	if err != nil {
		return nil, errStatus(errs.Code_MCPAcquiredSkillErr, err)
	}

	skillIDs := make([]string, 0, len(acquiredSkills))
	for _, as := range acquiredSkills {
		skillIDs = append(skillIDs, util.Int2Str(as.ID))
	}
	varsBySkill, err := s.cli.GetAcquiredSkillVarsBySkillIDs(ctx, userId, orgId, skillIDs)
	if err != nil {
		return nil, errStatus(errs.Code_MCPAcquiredSkillErr, err)
	}

	acquiredSkillList := make([]*mcp_service.AcquiredSkill, 0, len(acquiredSkills))
	for _, acquiredSkill := range acquiredSkills {
		sid := util.Int2Str(acquiredSkill.ID)
		acquiredSkillList = append(acquiredSkillList, toAcquiredSkillInfo(acquiredSkill, toAcquiredSkillVariables(varsBySkill[sid])))
	}

	return &mcp_service.AcquiredSkillGetListResp{
		List:  acquiredSkillList,
		Total: total,
	}, nil
}

func toAcquiredSkillInfo(acquiredSkill *model.AcquiredSkill, variables []*mcp_service.Variable) *mcp_service.AcquiredSkill {
	if acquiredSkill == nil {
		return nil
	}
	return &mcp_service.AcquiredSkill{
		AcquiredSkillId:    util.Int2Str(acquiredSkill.ID),
		SquareSkillId:      acquiredSkill.SquareSkillID,
		Name:               acquiredSkill.Name,
		Avatar:             acquiredSkill.Avatar,
		Author:             acquiredSkill.Author,
		AuthorId:           acquiredSkill.AuthorID,
		Desc:               acquiredSkill.Desc,
		ObjectPath:         acquiredSkill.ObjectPath,
		Markdown:           acquiredSkill.Markdown,
		Version:            acquiredSkill.Version,
		VersionDescription: acquiredSkill.VersionDescription,
		Variables:          variables,
		CreatedAt:          acquiredSkill.CreatedAt,
		UpdatedAt:          acquiredSkill.UpdatedAt,
	}
}

func toAcquiredSkillVariables(variables []*model.AcquiredSkillVariable) []*mcp_service.Variable {
	ret := make([]*mcp_service.Variable, 0, len(variables))
	for _, variable := range variables {
		ret = append(ret, &mcp_service.Variable{
			Id:            util.Int2Str(variable.ID),
			Name:          variable.Name,
			Desc:          variable.Desc,
			VariableKey:   variable.VariableKey,
			VariableValue: variable.VariableValue,
		})
	}
	return ret
}
