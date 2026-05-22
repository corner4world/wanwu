package response

import "github.com/UnicomAI/wanwu/internal/bff-service/model/request"

// SquareSkillInfo 探索广场-skill列表项
type SquareSkillInfo struct {
	SkillId  string         `json:"skillId"`
	Name     string         `json:"name"`
	Avatar   request.Avatar `json:"avatar"`
	Author   string         `json:"author"`
	Desc     string         `json:"desc"`
	IsShared bool           `json:"isShared"`
}

// SquareBuiltinSkillInfo 探索广场-内置skill列表项
type SquareBuiltinSkillInfo struct {
	SkillId string         `json:"skillId"`
	Name    string         `json:"name"`
	Avatar  request.Avatar `json:"avatar"`
	Author  string         `json:"author"`
	Desc    string         `json:"desc"`
}

// SquareBuiltinSkillDetail 探索广场-内置skill详情
type SquareBuiltinSkillDetail struct {
	SquareBuiltinSkillInfo
	SkillMarkdown string `json:"skillMarkdown"`
}

// SquareSkillDetail 探索广场-skill详情
type SquareSkillDetail struct {
	SquareSkillInfo
	SkillMarkdown string `json:"skillMarkdown"`
}
