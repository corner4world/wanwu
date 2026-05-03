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
	acquiredSkillId, err := s.cli.CreateAcquiredSkill(ctx, &model.AcquiredSkill{
		SquareSkillID:      req.SquareSkillId,
		Name:               req.Name,
		Avatar:             req.Avatar,
		Author:             req.Author,
		AuthorID:           req.AuthorId,
		Desc:               req.Desc,
		ObjectPath:         req.ObjectPath,
		Markdown:           req.Markdown,
		AcquiredType:       req.AcquiredType,
		Version:            req.Version,
		VersionDescription: req.VersionDescription,
		UserID:             req.Identity.UserId,
		OrgID:              req.Identity.OrgId,
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

	return toAcquiredSkillInfo(acquiredSkill), nil
}

func (s *Service) AcquiredSkillGetList(ctx context.Context, req *mcp_service.AcquiredSkillGetListReq) (*mcp_service.AcquiredSkillGetListResp, error) {
	acquiredSkills, total, err := s.cli.GetAcquiredSkillList(ctx, req.Identity.UserId, req.Identity.OrgId, req.Name)
	if err != nil {
		return nil, errStatus(errs.Code_MCPAcquiredSkillErr, err)
	}

	acquiredSkillList := make([]*mcp_service.AcquiredSkill, 0, len(acquiredSkills))
	for _, acquiredSkill := range acquiredSkills {
		acquiredSkillList = append(acquiredSkillList, toAcquiredSkillInfo(acquiredSkill))
	}

	return &mcp_service.AcquiredSkillGetListResp{
		List:  acquiredSkillList,
		Total: total,
	}, nil
}

func toAcquiredSkillInfo(acquiredSkill *model.AcquiredSkill) *mcp_service.AcquiredSkill {
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
		AcquiredType:       acquiredSkill.AcquiredType,
		Version:            acquiredSkill.Version,
		VersionDescription: acquiredSkill.VersionDescription,
		CreatedAt:          acquiredSkill.CreatedAt,
		UpdatedAt:          acquiredSkill.UpdatedAt,
	}
}
