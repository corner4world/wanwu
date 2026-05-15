package mcp

import (
	"context"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/mcp-service/client/model"
	"github.com/UnicomAI/wanwu/pkg/util"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Service) CustomSkillCreate(ctx context.Context, req *mcp_service.CustomSkillCreateReq) (*mcp_service.CustomSkillCreateResp, error) {
	if req.GetIdentity() == nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, toErrStatus("mcp_custom_skill_create", "identity is empty"))
	}

	skillId, err := s.cli.CreateCustomSkill(ctx, &model.CustomSkill{
		Name:            req.Name,
		Avatar:          req.Avatar,
		Author:          req.Author,
		Desc:            req.Desc,
		SourceType:      req.SourceType,
		WgaThreadID:     req.WgaThreadId,
		PreviewThreadID: req.PreviewThreadId,
		UserID:          req.Identity.UserId,
		OrgID:           req.Identity.OrgId,
	})
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}

	return &mcp_service.CustomSkillCreateResp{SkillId: skillId}, nil
}

func (s *Service) CustomSkillDelete(ctx context.Context, req *mcp_service.CustomSkillDeleteReq) (*emptypb.Empty, error) {
	err := s.cli.DeleteCustomSkill(ctx, req.SkillId)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) CustomSkillGet(ctx context.Context, req *mcp_service.CustomSkillGetReq) (*mcp_service.CustomSkill, error) {
	customSkill, err := s.cli.GetCustomSkill(ctx, req.SkillId)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}

	variables, err := s.cli.GetCustomSkillVars(ctx, customSkill.UserID, customSkill.OrgID, req.SkillId)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}

	return toCustomSkillInfo(customSkill, toCustomSkillVariables(variables)), nil
}

func (s *Service) GetCustomSkillByPreviewID(ctx context.Context, req *mcp_service.GetCustomSkillByPreviewIDReq) (*mcp_service.GetCustomSkillByPreviewIDResp, error) {
	if req.GetIdentity() == nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, toErrStatus("mcp_custom_skill_get_by_preview_thread", "identity is empty"))
	}
	customSkill, st := s.cli.GetCustomSkillByPreviewThreadID(ctx, req.GetIdentity().GetUserId(), req.GetIdentity().GetOrgId(), req.GetPreviewThreadId())
	if st != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, st)
	}
	if customSkill == nil {
		return &mcp_service.GetCustomSkillByPreviewIDResp{}, nil
	}
	variables, err := s.cli.GetCustomSkillVars(ctx, customSkill.UserID, customSkill.OrgID, util.Int2Str(customSkill.ID))
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}
	return &mcp_service.GetCustomSkillByPreviewIDResp{Skill: toCustomSkillInfo(customSkill, toCustomSkillVariables(variables))}, nil
}

func (s *Service) GetCustomSkillByThreadID(ctx context.Context, req *mcp_service.GetCustomSkillByThreadIDReq) (*mcp_service.GetCustomSkillByThreadIDResp, error) {
	if req.GetIdentity() == nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, toErrStatus("mcp_custom_skill_get_by_wga_thread", "identity is empty"))
	}
	customSkill, st := s.cli.GetCustomSkillByWgaThreadID(ctx, req.GetIdentity().GetUserId(), req.GetIdentity().GetOrgId(), req.GetWgaThreadId())
	if st != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, st)
	}
	if customSkill == nil {
		return &mcp_service.GetCustomSkillByThreadIDResp{}, nil
	}
	variables, err := s.cli.GetCustomSkillVars(ctx, customSkill.UserID, customSkill.OrgID, util.Int2Str(customSkill.ID))
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}
	return &mcp_service.GetCustomSkillByThreadIDResp{Skill: toCustomSkillInfo(customSkill, toCustomSkillVariables(variables))}, nil
}

func (s *Service) GetCustomSkillListByThreadIDList(ctx context.Context, req *mcp_service.GetCustomSkillListByThreadIDListReq) (*mcp_service.CustomSkillGetListResp, error) {
	if req.GetIdentity() == nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, toErrStatus("mcp_custom_skill_get_by_wga_thread_list", "identity is empty"))
	}
	userId, orgId := req.GetIdentity().GetUserId(), req.GetIdentity().GetOrgId()
	customSkills, st := s.cli.GetCustomSkillListByWgaThreadIDList(ctx, userId, orgId, req.GetWgaThreadIdList())
	if st != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, st)
	}
	skillIDs := make([]string, 0, len(customSkills))
	for _, cs := range customSkills {
		skillIDs = append(skillIDs, util.Int2Str(cs.ID))
	}
	varsBySkill := map[string][]*model.CustomSkillVariable{}
	if len(skillIDs) > 0 {
		var errSt *errs.Status
		varsBySkill, errSt = s.cli.GetCustomSkillVarsBySkillIDs(ctx, userId, orgId, skillIDs)
		if errSt != nil {
			return nil, errStatus(errs.Code_MCPCustomSkillErr, errSt)
		}
	}
	out := make([]*mcp_service.CustomSkill, 0, len(customSkills))
	for _, cs := range customSkills {
		sid := util.Int2Str(cs.ID)
		out = append(out, toCustomSkillInfo(cs, toCustomSkillVariables(varsBySkill[sid])))
	}
	return &mcp_service.CustomSkillGetListResp{
		List:  out,
		Total: int64(len(out)),
	}, nil
}

func (s *Service) CustomSkillGetList(ctx context.Context, req *mcp_service.CustomSkillGetListReq) (*mcp_service.CustomSkillGetListResp, error) {
	if req.GetIdentity() == nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, toErrStatus("mcp_custom_skill_list", "identity is empty"))
	}
	userId, orgId := req.GetIdentity().GetUserId(), req.GetIdentity().GetOrgId()
	customSkills, total, err := s.cli.GetCustomSkillList(ctx, userId, orgId, req.Name)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}

	skillIDs := make([]string, 0, len(customSkills))
	for _, cs := range customSkills {
		skillIDs = append(skillIDs, util.Int2Str(cs.ID))
	}
	varsBySkill, err := s.cli.GetCustomSkillVarsBySkillIDs(ctx, userId, orgId, skillIDs)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}

	customSkillList := make([]*mcp_service.CustomSkill, 0, len(customSkills))
	for _, customSkill := range customSkills {
		sid := util.Int2Str(customSkill.ID)
		customSkillList = append(customSkillList, toCustomSkillInfo(customSkill, toCustomSkillVariables(varsBySkill[sid])))
	}

	return &mcp_service.CustomSkillGetListResp{
		List:  customSkillList,
		Total: total,
	}, nil
}

func (s *Service) CustomSkillGetBySaveIds(ctx context.Context, req *mcp_service.CustomSkillGetBySaveIdsReq) (*mcp_service.CustomSkillSaveIdsResp, error) {
	customSkills, err := s.cli.GetCustomSkillBySaveIds(ctx, req.SaveIds)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}

	saveIds := make([]string, 0, len(customSkills))
	for _, customSkill := range customSkills {
		saveIds = append(saveIds, customSkill.SaveId)
	}

	return &mcp_service.CustomSkillSaveIdsResp{
		SaveIds: saveIds,
	}, nil
}

func (s *Service) GetCustomSkillDetailByIdList(ctx context.Context, req *mcp_service.CustomSkillDetailByIdListReq) (*mcp_service.CustomSkillDetailByIdListResp, error) {
	customSkills, err := s.cli.GetCustomSkillBySkillIds(ctx, req.SkillIds)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}

	groups := make(map[[2]string][]string)
	for _, cs := range customSkills {
		k := [2]string{cs.UserID, cs.OrgID}
		groups[k] = append(groups[k], util.Int2Str(cs.ID))
	}
	varsBySkill := make(map[string][]*model.CustomSkillVariable, len(customSkills))
	for k, ids := range groups {
		m, err := s.cli.GetCustomSkillVarsBySkillIDs(ctx, k[0], k[1], ids)
		if err != nil {
			return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
		}
		for sid, vars := range m {
			varsBySkill[sid] = vars
		}
	}

	skillDetails := make([]*mcp_service.CustomSkill, 0, len(customSkills))
	for _, customSkill := range customSkills {
		sid := util.Int2Str(customSkill.ID)
		skillDetails = append(skillDetails, toCustomSkillInfo(customSkill, toCustomSkillVariables(varsBySkill[sid])))
	}

	return &mcp_service.CustomSkillDetailByIdListResp{
		SkillDetails: skillDetails,
	}, nil
}

func toCustomSkillInfo(customSkill *model.CustomSkill, variables []*mcp_service.Variable) *mcp_service.CustomSkill {
	if customSkill == nil {
		return nil
	}
	return &mcp_service.CustomSkill{
		SkillId:         util.Int2Str(customSkill.ID),
		Name:            customSkill.Name,
		Avatar:          customSkill.Avatar,
		Author:          customSkill.Author,
		Desc:            customSkill.Desc,
		ObjectPath:      customSkill.ObjectPath,
		WgaThreadId:     customSkill.WgaThreadID,
		PreviewThreadId: customSkill.PreviewThreadID,
		Variables:       variables,
		CreatedAt:       customSkill.CreatedAt,
		UpdatedAt:       customSkill.UpdatedAt,
	}
}

func toCustomSkillVariables(variables []*model.CustomSkillVariable) []*mcp_service.Variable {
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
