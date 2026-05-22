package response

import "github.com/UnicomAI/wanwu/internal/bff-service/model/request"

type SkillDetailListResp struct {
	SkillList []*SkillDetail `json:"skillList"`
}

type CustomSkillDetailListResp struct {
	SkillList []*CustomSkillListDetail `json:"skillList"`
}

type CallbackAcquiredSkillDetailListResp struct {
	SkillList []*CallbackAcquiredSkillDetail `json:"skillList"`
}

type SkillDetail struct {
	SkillId       string           `json:"skillId"`             // 模板ID
	Name          string           `json:"name"`                // 模板名称
	Avatar        request.Avatar   `json:"avatar"`              // 模板头像
	Author        string           `json:"author"`              // 作者
	Desc          string           `json:"desc"`                // 模板描述
	SkillMarkdown string           `json:"skillMarkdown"`       // 模板markdown预览
	SkillPath     string           `json:"skillPath,omitempty"` // markdown地址，内部使用，不要对外
	Variables     []*SkillVariable `json:"variables,omitempty"`
}

type CustomSkillListDetail struct {
	SkillId    string           `json:"skillId"`
	Name       string           `json:"name"`
	Avatar     request.Avatar   `json:"avatar"`
	Author     string           `json:"author"`
	Desc       string           `json:"desc"`
	ObjectPath string           `json:"objectPath,omitempty"`
	Variables  []*SkillVariable `json:"variables,omitempty"`
}

type CallbackAcquiredSkillDetail struct {
	SkillId    string         `json:"skillId"`
	Name       string         `json:"name"`
	Avatar     request.Avatar `json:"avatar"`
	Author     string         `json:"author"`
	Desc       string         `json:"desc"`
	ObjectPath string         `json:"objectPath"`
}

type SkillInfo struct {
	SkillId   string         `json:"skillId"`   // skillId
	SkillName string         `json:"skillName"` // 名称
	SkillType string         `json:"skillType"` // 类型
	Desc      string         `json:"desc"`      // 描述
	Author    string         `json:"author"`    // 作者
	Avatar    request.Avatar `json:"avatar"`    // 图标
}

// CallbackSkillDetail 返回 skill 详情，字段与 agent-service 的 SkillToolInfo 对齐。
type CallbackSkillDetail struct {
	SkillId    string `json:"skillId"`
	SkillType  string `json:"skillType"`
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Avatar     string `json:"avatar"`
	ObjectPath string `json:"objectPath"`
}

type SkillVariable struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Desc          string `json:"desc"`
	VariableKey   string `json:"variableKey"`
	VariableValue string `json:"variableValue"`
}
