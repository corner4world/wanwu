package response

import "github.com/UnicomAI/wanwu/internal/bff-service/model/request"

// SquareSkillDetail 探索广场-skill列表项
type SquareSkillDetail struct {
	SkillId  string         `json:"skillId"`
	Name     string         `json:"name"`
	Avatar   request.Avatar `json:"avatar"`
	Author   string         `json:"author"`
	Desc     string         `json:"desc"`
	IsShared bool           `json:"isShared"`
}

// SquareSkillDetailInfo 探索广场-skill详情（含isShared、markdown、downloadUrl）
type SquareSkillDetailInfo struct {
	SkillId       string         `json:"skillId"`
	Name          string         `json:"name"`
	Avatar        request.Avatar `json:"avatar"`
	Author        string         `json:"author"`
	Desc          string         `json:"desc"`
	SkillMarkdown string         `json:"skillMarkdown"`
	DownloadUrl   string         `json:"downloadUrl"`
	IsShared      bool           `json:"isShared"`
}
