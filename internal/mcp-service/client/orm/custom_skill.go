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

func (c *Client) CreateCustomSkill(ctx context.Context, customSkill *model.CustomSkill) (string, *err_code.Status) {
	// 如果saveId不为空，检查是否已存在（根据source_type、save_id、user_id、org_id判断唯一性）
	if customSkill.SaveId != "" {
		var count int64
		if err := sqlopt.SQLOptions(
			sqlopt.WithUserID(customSkill.UserID),
			sqlopt.WithOrgID(customSkill.OrgID),
			sqlopt.WithCustomSkillSaveId(customSkill.SaveId),
			sqlopt.WithCustomSkillSourceType(customSkill.SourceType),
		).Apply(c.db).WithContext(ctx).Model(&model.CustomSkill{}).Count(&count).Error; err != nil {
			return "", toErrStatus("mcp_custom_skill_check_exists", err.Error())
		}
		if count > 0 {
			return "", toErrStatus("mcp_custom_skill_save_id_exists")
		}
	}

	if err := c.db.WithContext(ctx).Create(customSkill).Error; err != nil {
		return "", toErrStatus("mcp_custom_skill_create", err.Error())
	}

	return util.Int2Str(customSkill.ID), nil
}

func (c *Client) DeleteCustomSkill(ctx context.Context, skillId string) *err_code.Status {
	id := util.MustU32(skillId)
	return c.transaction(ctx, func(tx *gorm.DB) *err_code.Status {
		if err := sqlopt.WithSkillID(skillId).Apply(tx).Delete(&model.CustomSkillVariable{}).Error; err != nil {
			return toErrStatus("mcp_custom_skill_delete_variables", err.Error())
		}
		if err := sqlopt.WithSkillID(skillId).Apply(tx).Delete(&model.CustomSkillPublish{}).Error; err != nil {
			return toErrStatus("mcp_custom_skill_delete_publish", err.Error())
		}
		if err := sqlopt.WithID(id).Apply(tx).Delete(&model.CustomSkill{}).Error; err != nil {
			return toErrStatus("mcp_custom_skill_delete", err.Error())
		}
		return nil
	})
}

func (c *Client) GetCustomSkill(ctx context.Context, skillId string) (*model.CustomSkill, *err_code.Status) {
	var cs model.CustomSkill
	if err := sqlopt.SQLOptions(
		sqlopt.WithID(util.MustU32(skillId)),
	).Apply(c.db).WithContext(ctx).First(&cs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, toErrStatus("mcp_custom_skill_not_found", skillId)
		}
		return nil, toErrStatus("mcp_custom_skill_get", skillId, err.Error())
	}
	return &cs, nil
}

// GetCustomSkillByPreviewThreadID 仅匹配列 preview_thread_id。参数不完整或记录不存在时返回 (nil, nil)；仅查询失败返回 Status。
func (c *Client) GetCustomSkillByPreviewThreadID(ctx context.Context, userId, orgId, previewThreadID string) (*model.CustomSkill, *err_code.Status) {
	if previewThreadID == "" || userId == "" || orgId == "" {
		return nil, nil
	}
	var cs model.CustomSkill
	err := sqlopt.SQLOptions(
		sqlopt.WithUserID(userId),
		sqlopt.WithOrgID(orgId),
		sqlopt.WithCustomSkillPreviewThreadId(previewThreadID),
	).Apply(c.db).WithContext(ctx).Model(&model.CustomSkill{}).First(&cs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, toErrStatus("mcp_custom_skill_get_by_wga_thread", err.Error())
	}
	return &cs, nil
}

// GetCustomSkillByWgaThreadID 仅匹配列 wga_thread_id。参数不完整或记录不存在时返回 (nil, nil)；仅查询失败返回 Status。
func (c *Client) GetCustomSkillByWgaThreadID(ctx context.Context, userId, orgId, wgaThreadID string) (*model.CustomSkill, *err_code.Status) {
	if wgaThreadID == "" || userId == "" || orgId == "" {
		return nil, nil
	}
	var cs model.CustomSkill
	err := sqlopt.SQLOptions(
		sqlopt.WithUserID(userId),
		sqlopt.WithOrgID(orgId),
		sqlopt.WithCustomSkillWgaThreadId(wgaThreadID),
	).Apply(c.db).WithContext(ctx).Model(&model.CustomSkill{}).First(&cs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, toErrStatus("mcp_custom_skill_get_by_wga_thread", err.Error())
	}
	return &cs, nil
}

// GetCustomSkillListByWgaThreadIDList 在 identity 下按 wga_thread_id IN 批量查询。wgaThreadIdList 去空后为空时返回空切片；仅数据库错误返回 Status。
func (c *Client) GetCustomSkillListByWgaThreadIDList(ctx context.Context, userId, orgId string, wgaThreadIDList []string) ([]*model.CustomSkill, *err_code.Status) {
	nonEmpty := make([]string, 0, len(wgaThreadIDList))
	for _, id := range wgaThreadIDList {
		if id != "" {
			nonEmpty = append(nonEmpty, id)
		}
	}
	if len(nonEmpty) == 0 {
		return []*model.CustomSkill{}, nil
	}
	var list []*model.CustomSkill
	if err := sqlopt.SQLOptions(
		sqlopt.WithUserID(userId),
		sqlopt.WithOrgID(orgId),
		sqlopt.WithCustomSkillWgaThreadIdList(nonEmpty),
	).Apply(c.db).WithContext(ctx).Find(&list).Error; err != nil {
		return nil, toErrStatus("mcp_custom_skill_get_by_wga_thread_list", err.Error())
	}
	return list, nil
}

// GetCustomSkillList 返回的 total：无分页，表示当前筛选条件下全量条数，与 len(list) 一致。
func (c *Client) GetCustomSkillList(ctx context.Context, userId, orgId, name string) ([]*model.CustomSkill, int64, *err_code.Status) {
	var list []*model.CustomSkill
	if err := sqlopt.SQLOptions(
		sqlopt.WithUserID(userId),
		sqlopt.WithOrgID(orgId),
		sqlopt.LikeName(name),
	).Apply(c.db).WithContext(ctx).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, toErrStatus("mcp_custom_skill_list", err.Error())
	}

	return list, int64(len(list)), nil
}

func (c *Client) GetCustomSkillBySaveIds(ctx context.Context, saveIds []string) ([]*model.CustomSkill, *err_code.Status) {
	var list []*model.CustomSkill
	if err := sqlopt.SQLOptions(
		sqlopt.WithCustomSkillSaveIds(saveIds),
	).Apply(c.db).WithContext(ctx).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, toErrStatus("mcp_custom_skill_get_by_save_ids", err.Error())
	}

	return list, nil
}

func (c *Client) GetCustomSkillBySkillIds(ctx context.Context, skillIds []string) ([]*model.CustomSkill, *err_code.Status) {
	var list []*model.CustomSkill
	if err := sqlopt.SQLOptions(
		sqlopt.WithCustomSkillSkillId(skillIds),
	).Apply(c.db).WithContext(ctx).Find(&list).Error; err != nil {
		return nil, toErrStatus("mcp_custom_skill_get_by_skill_ids", err.Error())
	}

	return list, nil
}

func (c *Client) UpdateCustomSkillBasicMeta(ctx context.Context, skillId, name, desc string) *err_code.Status {
	updates := map[string]any{
		"name": name,
		"desc": desc,
	}
	if err := sqlopt.SQLOptions(
		sqlopt.WithID(util.MustU32(skillId)),
	).Apply(c.db).WithContext(ctx).Model(&model.CustomSkill{}).Updates(updates).Error; err != nil {
		return toErrStatus("mcp_custom_skill_update_basic_meta", skillId, err.Error())
	}
	return nil
}

func (c *Client) UpdateCustomSkillThreadMeta(ctx context.Context, skillId, wgaThreadId, previewThreadId string) *err_code.Status {
	updates := map[string]any{
		"wga_thread_id":     wgaThreadId,
		"preview_thread_id": previewThreadId,
	}
	if err := sqlopt.SQLOptions(
		sqlopt.WithID(util.MustU32(skillId)),
	).Apply(c.db).WithContext(ctx).Model(&model.CustomSkill{}).Updates(updates).Error; err != nil {
		return toErrStatus("mcp_custom_skill_update_thread_meta", skillId, err.Error())
	}
	return nil
}
