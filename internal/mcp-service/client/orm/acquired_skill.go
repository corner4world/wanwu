package orm

import (
	"context"
	"errors"

	err_code "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/mcp-service/client/model"
	"github.com/UnicomAI/wanwu/internal/mcp-service/client/orm/sqlopt"
	"github.com/UnicomAI/wanwu/pkg/util"
	"gorm.io/gorm"
)

func (c *Client) CreateAcquiredSkill(ctx context.Context, acquiredSkill *model.AcquiredSkill) (string, *err_code.Status) {
	if err := c.db.WithContext(ctx).Create(acquiredSkill).Error; err != nil {
		return "", toErrStatus("mcp_acquired_skill_create", err.Error())
	}
	return util.Int2Str(acquiredSkill.ID), nil
}

func (c *Client) DeleteAcquiredSkill(ctx context.Context, acquiredSkillId string) *err_code.Status {
	if err := sqlopt.SQLOptions(
		sqlopt.WithID(util.MustU32(acquiredSkillId)),
	).Apply(c.db).WithContext(ctx).Delete(&model.AcquiredSkill{}).Error; err != nil {
		return toErrStatus("mcp_acquired_skill_delete", err.Error())
	}
	return nil
}

func (c *Client) GetAcquiredSkill(ctx context.Context, acquiredSkillId string) (*model.AcquiredSkill, *err_code.Status) {
	var as model.AcquiredSkill
	if err := sqlopt.SQLOptions(
		sqlopt.WithID(util.MustU32(acquiredSkillId)),
	).Apply(c.db).WithContext(ctx).First(&as).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, toErrStatus("mcp_acquired_skill_not_found", acquiredSkillId)
		}
		return nil, toErrStatus("mcp_acquired_skill_get", acquiredSkillId, err.Error())
	}
	return &as, nil
}

func (c *Client) GetAcquiredSkillList(ctx context.Context, userId, orgId, name string) ([]*model.AcquiredSkill, int64, *err_code.Status) {
	var list []*model.AcquiredSkill
	if err := sqlopt.SQLOptions(
		sqlopt.LikeName(name),
		sqlopt.WithUserID(userId),
		sqlopt.WithOrgID(orgId),
	).Apply(c.db).WithContext(ctx).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, toErrStatus("mcp_acquired_skill_list", err.Error())
	}

	return list, int64(len(list)), nil
}
