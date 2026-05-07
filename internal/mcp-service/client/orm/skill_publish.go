package orm

import (
	"context"
	"encoding/json"
	"errors"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/mcp-service/client/model"
	"github.com/UnicomAI/wanwu/internal/mcp-service/client/orm/sqlopt"
	"github.com/UnicomAI/wanwu/pkg/util"
	"gorm.io/gorm"
)

const textKeyCustomSkillPublishNotFound = "mcp_custom_skill_publish_not_found"

type CustomSkillPublishSnapshot struct {
	Variables  []*model.CustomSkillVariable
	Markdown   string
	ObjectPath string
}

func (c *Client) PublishCustomSkill(ctx context.Context, publish *model.CustomSkillPublish, snapshot *CustomSkillPublishSnapshot) *errs.Status {
	if publish.SkillID == "" {
		return toErrStatus("mcp_custom_skill_publish_skill_id_required")
	}
	if publish.Version == "" {
		return toErrStatus("mcp_custom_skill_publish_version_required")
	}

	customSkill, errStatus := c.GetCustomSkill(ctx, publish.SkillID)
	if errStatus != nil {
		return errStatus
	}

	var variableInfos []byte
	var err error
	if snapshot != nil && len(snapshot.Variables) > 0 {
		variableInfos, err = json.Marshal(customSkillVariablesForPublishBlob(snapshot.Variables, publish.SkillID))
		if err != nil {
			return toErrStatus("mcp_custom_skill_publish_variable_marshal", err.Error())
		}
	} else {
		variables, est := c.GetCustomSkillVars(ctx, customSkill.UserID, customSkill.OrgID, publish.SkillID)
		if est != nil {
			return est
		}
		variableInfos, err = json.Marshal(customSkillVariablesForPublishBlob(variables, publish.SkillID))
		if err != nil {
			return toErrStatus("mcp_custom_skill_publish_variable_marshal", err.Error())
		}
	}

	markdown := customSkill.Markdown
	objectPath := customSkill.ObjectPath
	if snapshot != nil {
		if snapshot.Markdown != "" {
			markdown = snapshot.Markdown
		}
		if snapshot.ObjectPath != "" {
			objectPath = snapshot.ObjectPath
		}
	}

	publish.Markdown = markdown
	publish.ObjectPath = objectPath
	publish.VariableInfos = string(variableInfos)
	if err := c.db.WithContext(ctx).Create(publish).Error; err != nil {
		return toErrStatus("mcp_custom_skill_publish", err.Error())
	}
	return nil
}

// customSkillVariablesForPublishBlob 构造写入 CustomSkillPublish.variable_infos 的变量列表。
// 只保留业务字段，不显式携带配置表主键/时间戳（零值），与 assistant 复制子表时「先置 ID=0 再 Create」
// （见 internal/assistant-service/client/orm/assistant.go 复制 workflow/mcp/tool 等）同一语义：快照是版本产物，
// 回写草稿时 OverwriteCustomSkillDraft 会 Unmarshal 后再 CreateInBatches，不应依赖旧行主键。
// Rag 发布是把整份草稿 json.Marshal 进 RagInfo；此处变量单独成表，故用显式字段拷贝而非带 ORM id 的行快照。
func customSkillVariablesForPublishBlob(src []*model.CustomSkillVariable, skillID string) []*model.CustomSkillVariable {
	out := make([]*model.CustomSkillVariable, 0, len(src))
	for _, v := range src {
		if v == nil {
			continue
		}
		out = append(out, cloneCustomSkillVariableForPublish(v, skillID))
	}
	return out
}

func cloneCustomSkillVariableForPublish(v *model.CustomSkillVariable, skillID string) *model.CustomSkillVariable {
	return &model.CustomSkillVariable{
		SkillID:       skillID,
		Name:          v.Name,
		Desc:          v.Desc,
		VariableKey:   v.VariableKey,
		VariableValue: v.VariableValue,
	}
}

func (c *Client) UpdatePublishCustomSkill(ctx context.Context, skillId, desc string) *errs.Status {
	publish, errStatus := c.getLatestCustomSkillPublish(ctx, skillId)
	if errStatus != nil {
		return errStatus
	}
	if err := c.db.WithContext(ctx).Model(&model.CustomSkillPublish{}).
		Where("id = ?", publish.ID).
		Update("description", desc).Error; err != nil {
		return toErrStatus("mcp_custom_skill_publish_update", err.Error())
	}
	return nil
}

// GetPublishCustomSkillHistoryList 返回的 total：无分页，表示该 skill 发布历史全量条数，与 len(list) 一致。
func (c *Client) GetPublishCustomSkillHistoryList(ctx context.Context, skillId string) ([]*model.CustomSkillPublish, int64, *errs.Status) {
	if skillId == "" {
		return nil, 0, toErrStatus("mcp_skill_config_invalid_arg")
	}
	var list []*model.CustomSkillPublish
	if err := sqlopt.SQLOptions(
		sqlopt.WithSkillID(skillId),
	).Apply(c.db).WithContext(ctx).Order("created_at DESC").
		Find(&list).Error; err != nil {
		return nil, 0, toErrStatus("mcp_custom_skill_publish_history", err.Error())
	}
	return list, int64(len(list)), nil
}

func (c *Client) OverwriteCustomSkillDraft(ctx context.Context, skillId, version string) *errs.Status {
	if skillId == "" || version == "" {
		return toErrStatus("mcp_custom_skill_overwrite_draft_invalid_arg", skillId, version)
	}
	skillPK := util.MustU32(skillId)

	var publish model.CustomSkillPublish
	if err := sqlopt.SQLOptions(
		sqlopt.WithSkillID(skillId),
		sqlopt.WithVersion(version),
	).Apply(c.db).WithContext(ctx).First(&publish).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return toErrStatus("mcp_custom_skill_publish_not_found_for_version", skillId, version)
		}
		return toErrStatus("mcp_custom_skill_publish_get", skillId, err.Error())
	}

	return c.transaction(ctx, func(tx *gorm.DB) *errs.Status {
		if err := tx.Model(&model.CustomSkill{}).
			Where("id = ?", skillPK).
			Updates(map[string]interface{}{
				"desc":        publish.Description,
				"markdown":    publish.Markdown,
				"object_path": publish.ObjectPath,
			}).Error; err != nil {
			return toErrStatus("mcp_custom_skill_overwrite_draft", err.Error())
		}

		if err := sqlopt.SQLOptions(
			sqlopt.WithSkillID(skillId),
		).Apply(tx).Delete(&model.CustomSkillVariable{}).Error; err != nil {
			return toErrStatus("mcp_custom_skill_overwrite_draft_variable_delete", err.Error())
		}

		var variables []*model.CustomSkillVariable
		if publish.VariableInfos != "" {
			if err := json.Unmarshal([]byte(publish.VariableInfos), &variables); err != nil {
				return toErrStatus("mcp_custom_skill_overwrite_draft_variable_unmarshal", err.Error())
			}
		}
		for _, variable := range variables {
			variable.ID = 0
			variable.SkillID = skillId
		}
		if len(variables) > 0 {
			if err := tx.CreateInBatches(variables, len(variables)).Error; err != nil {
				return toErrStatus("mcp_custom_skill_overwrite_draft_variable_create", err.Error())
			}
		}
		return nil
	})
}

func (c *Client) GetPublishCustomSkillDesc(ctx context.Context, skillId string) (*model.CustomSkillPublish, *errs.Status) {
	return c.getLatestCustomSkillPublish(ctx, skillId)
}

func (c *Client) GetPublishCustomSkillDescBatch(ctx context.Context, skillIdList []string) ([]*model.CustomSkillPublish, *errs.Status) {
	if len(skillIdList) == 0 {
		return []*model.CustomSkillPublish{}, nil
	}
	for _, skillId := range skillIdList {
		if skillId == "" {
			return nil, toErrStatus("mcp_skill_config_invalid_arg")
		}
	}
	seen := make(map[string]struct{}, len(skillIdList))
	uniq := make([]string, 0, len(skillIdList))
	for _, id := range skillIdList {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	var rows []*model.CustomSkillPublish
	if err := c.db.WithContext(ctx).
		Where("skill_id IN ?", uniq).
		Order("skill_id ASC, created_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, toErrStatus("mcp_custom_skill_publish_get", err.Error())
	}
	latestBySkill := make(map[string]*model.CustomSkillPublish, len(uniq))
	for _, p := range rows {
		if p == nil {
			continue
		}
		if _, ok := latestBySkill[p.SkillID]; ok {
			continue
		}
		latestBySkill[p.SkillID] = p
	}
	list := make([]*model.CustomSkillPublish, 0, len(skillIdList))
	for _, skillId := range skillIdList {
		if p, ok := latestBySkill[skillId]; ok {
			list = append(list, p)
		}
	}
	return list, nil
}

func (c *Client) getLatestCustomSkillPublish(ctx context.Context, skillId string) (*model.CustomSkillPublish, *errs.Status) {
	if skillId == "" {
		return nil, toErrStatus("mcp_skill_config_invalid_arg")
	}
	var publish model.CustomSkillPublish
	if err := sqlopt.SQLOptions(
		sqlopt.WithSkillID(skillId),
	).Apply(c.db).WithContext(ctx).Order("created_at DESC").
		First(&publish).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, toErrStatus(textKeyCustomSkillPublishNotFound, skillId)
		}
		return nil, toErrStatus("mcp_custom_skill_publish_get", skillId, err.Error())
	}
	return &publish, nil
}
