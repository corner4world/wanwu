package response

import "github.com/UnicomAI/wanwu/internal/bff-service/model/request"

// JoinerSkillDetail 资源库-我添加的skill详情
type JoinerSkillDetail struct {
	SkillId       string         `json:"skillId"`
	Name          string         `json:"name"`
	Avatar        request.Avatar `json:"avatar"`
	Author        string         `json:"author"`
	Desc          string         `json:"desc"`
	SkillMarkdown string         `json:"skillMarkdown"`
	DownloadUrl   string         `json:"downloadUrl"`
}
