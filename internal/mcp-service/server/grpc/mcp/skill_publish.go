package mcp

import (
	"context"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/mcp-service/client/model"
	"github.com/UnicomAI/wanwu/internal/mcp-service/client/orm"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Service) PublishCustomSkill(ctx context.Context, req *mcp_service.PublishCustomSkillReq) (*emptypb.Empty, error) {
	if req.GetIdentity() == nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, toErrStatus("mcp_custom_skill_publish", "identity is empty"))
	}
	var snap *orm.CustomSkillPublishSnapshot
	if len(req.GetVariables()) > 0 || req.GetMarkdown() != "" || req.GetObjectPath() != "" {
		snap = &orm.CustomSkillPublishSnapshot{
			Markdown:   req.GetMarkdown(),
			ObjectPath: req.GetObjectPath(),
		}
		for _, v := range req.GetVariables() {
			if v == nil {
				continue
			}
			snap.Variables = append(snap.Variables, publishProtoToCustomSkillVariable(req.GetSkillId(), v))
		}
	}

	publish := &model.CustomSkillPublish{
		SkillID:     req.GetSkillId(),
		Version:     req.GetVersion(),
		Description: req.GetDesc(),
		UserId:      req.Identity.UserId,
		OrgId:       req.Identity.OrgId,
		GitCommit:   req.GetGitCommit(),
	}
	if err := s.cli.PublishCustomSkill(ctx, publish, snap); err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}
	return &emptypb.Empty{}, nil
}

func publishProtoToCustomSkillVariable(skillID string, v *mcp_service.Variable) *model.CustomSkillVariable {
	return &model.CustomSkillVariable{
		SkillID:       skillID,
		Name:          v.GetName(),
		Desc:          v.GetDesc(),
		VariableKey:   v.GetVariableKey(),
		VariableValue: v.GetVariableValue(),
	}
}

func (s *Service) UpdatePublishCustomSkill(ctx context.Context, req *mcp_service.UpdatePublishCustomSkillReq) (*emptypb.Empty, error) {
	if err := s.cli.UpdatePublishCustomSkill(ctx, req.SkillId, req.Desc); err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Service) GetPublishCustomSkillHistoryList(ctx context.Context, req *mcp_service.GetPublishCustomSkillHistoryListReq) (*mcp_service.PublishCustomSkillHistoryListResp, error) {
	history, total, err := s.cli.GetPublishCustomSkillHistoryList(ctx, req.SkillId)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}
	list := make([]*mcp_service.PublishCustomSkillHistory, 0, len(history))
	for _, item := range history {
		list = append(list, &mcp_service.PublishCustomSkillHistory{
			SkillId:   item.SkillID,
			Version:   item.Version,
			Desc:      item.Description,
			CreatedAt: item.CreatedAt,
		})
	}
	return &mcp_service.PublishCustomSkillHistoryListResp{HistoryList: list, Total: total}, nil
}

func (s *Service) OverwriteCustomSkillDraft(ctx context.Context, req *mcp_service.OverwriteCustomSkillDraftReq) (*emptypb.Empty, error) {
	if err := s.cli.OverwriteCustomSkillDraft(ctx, req.SkillId, req.Version); err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Service) GetPublishCustomSkillDesc(ctx context.Context, req *mcp_service.GetPublishCustomSkillDescReq) (*mcp_service.PublishCustomSkillDescResp, error) {
	publish, err := s.cli.GetPublishCustomSkillDesc(ctx, req.SkillId)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}
	return toPublishCustomSkillDesc(publish), nil
}

func (s *Service) GetPublishCustomSkillDescBatch(ctx context.Context, req *mcp_service.GetPublishCustomSkillDescBatchReq) (*mcp_service.PublishCustomSkillDescBatchResp, error) {
	publishList, err := s.cli.GetPublishCustomSkillDescBatch(ctx, req.SkillIdList)
	if err != nil {
		return nil, errStatus(errs.Code_MCPCustomSkillErr, err)
	}
	list := make([]*mcp_service.PublishCustomSkillDescResp, 0, len(publishList))
	for _, publish := range publishList {
		list = append(list, toPublishCustomSkillDesc(publish))
	}
	return &mcp_service.PublishCustomSkillDescBatchResp{List: list}, nil
}

func toPublishCustomSkillDesc(publish *model.CustomSkillPublish) *mcp_service.PublishCustomSkillDescResp {
	if publish == nil {
		return nil
	}
	return &mcp_service.PublishCustomSkillDescResp{
		SkillId:   publish.SkillID,
		Version:   publish.Version,
		Desc:      publish.Description,
		CreatedAt: publish.CreatedAt,
	}
}
