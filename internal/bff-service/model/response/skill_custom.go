package response

import "github.com/UnicomAI/wanwu/internal/bff-service/model/request"

type CustomSkillDetail struct {
	SkillId   string           `json:"skillId"`
	Name      string           `json:"name"`
	Avatar    request.Avatar   `json:"avatar"`
	Author    string           `json:"author"`
	Desc      string           `json:"desc"`
	Variables []*SkillVariable `json:"variables,omitempty"`
	ThreadID  string           `json:"threadId,omitempty"`
	PreviewID string           `json:"previewId,omitempty"`
}

type CustomSkillIDResp struct {
	SkillId string `json:"skillId"`
}

type CustomSkillCheckResp struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}
