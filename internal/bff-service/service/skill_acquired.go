package service

import (
	"net/url"

	mcp_service "github.com/UnicomAI/wanwu/api/proto/mcp-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/gin-gonic/gin"
)

// GetAcquiredSkillList 资源库-我添加的skill列表
func GetAcquiredSkillList(ctx *gin.Context, userId, orgId, name string) (*response.ListResult, error) {
	resp, err := mcp.AcquiredSkillGetList(ctx.Request.Context(), &mcp_service.AcquiredSkillGetListReq{
		Identity: &mcp_service.Identity{UserId: userId, OrgId: orgId},
		Name:     name,
	})
	if err != nil {
		return nil, err
	}

	list := make([]*response.AcquiredSkillDetail, 0, len(resp.List))
	for _, skill := range resp.List {
		list = append(list, toAcquiredSkillDetail(ctx, skill, false))
	}

	return &response.ListResult{
		List:  list,
		Total: resp.Total,
	}, nil
}

// DeleteAcquiredSkill 资源库-删除已添加的skill
func DeleteAcquiredSkill(ctx *gin.Context, userId, orgId, acquiredSkillId string) error {
	if _, err := getOwnedAcquiredSkill(ctx, userId, orgId, acquiredSkillId); err != nil {
		return err
	}
	_, err := mcp.AcquiredSkillDelete(ctx.Request.Context(), &mcp_service.AcquiredSkillDeleteReq{
		AcquiredSkillId: acquiredSkillId,
	})
	return err
}

// GetAcquiredSkill 资源库-获取已添加skill详情
func GetAcquiredSkill(ctx *gin.Context, userId, orgId, acquiredSkillId string) (*response.AcquiredSkillDetail, error) {
	resp, err := getOwnedAcquiredSkill(ctx, userId, orgId, acquiredSkillId)
	if err != nil {
		return nil, err
	}

	return toAcquiredSkillDetail(ctx, resp, true), nil
}

// --- internal ---

func toAcquiredSkillDetail(ctx *gin.Context, skill *mcp_service.AcquiredSkill, includeVariables bool) *response.AcquiredSkillDetail {
	if skill == nil {
		return nil
	}
	filePath, _ := url.JoinPath(config.Cfg().Minio.DownloadURL, skill.ObjectPath)
	ret := &response.AcquiredSkillDetail{
		SkillId:       skill.AcquiredSkillId,
		SquareSkillID: skill.SquareSkillId,
		Name:          skill.Name,
		Avatar:        cacheSkillAvatar(ctx, skill.Avatar),
		Author:        skill.Author,
		Desc:          skill.Desc,
		SkillMarkdown: config.FixFrontMatterFormat(skill.Markdown),
		DownloadUrl:   filePath,
	}
	if includeVariables {
		ret.Variables = toSkillVariables(skill.Variables)
	}
	return ret
}
