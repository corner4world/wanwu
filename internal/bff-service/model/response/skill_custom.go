package response

import "github.com/UnicomAI/wanwu/internal/bff-service/model/request"

type CustomSkillDetail struct {
	SkillId       string           `json:"skillId"`
	Name          string           `json:"name"`
	Avatar        request.Avatar   `json:"avatar"`
	Author        string           `json:"author"`
	Desc          string           `json:"desc"`
	IsPublished   bool             `json:"isPublished"`
	Version       string           `json:"version"`
	PublishType   string           `json:"publishType"`
	Variables     []*SkillVariable `json:"variables,omitempty"`
	ThreadID      string           `json:"threadId,omitempty"`
	PreviewID     string           `json:"previewId,omitempty"`
	SkillMarkdown string           `json:"skillMarkdown,omitempty"`
}

type CustomSkillListItem struct {
	SkillId     string         `json:"skillId"`
	Name        string         `json:"name"`
	Avatar      request.Avatar `json:"avatar"`
	Author      string         `json:"author"`
	Desc        string         `json:"desc"`
	IsPublished bool           `json:"isPublished"`
	Version     string         `json:"version"`
	PublishType string         `json:"publishType"`
	ThreadID    string         `json:"threadId,omitempty"`
	PreviewID   string         `json:"previewId,omitempty"`
}

type CustomSkillIDResp struct {
	SkillId string `json:"skillId"`
}

type CustomSkillCheckResp struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}
