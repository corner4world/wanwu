package response

// ModelExperienceDialog 模型体验对话
type ModelExperienceDialog struct {
	ID           string `json:"id"` // modelExperienceId
	ModelId      string `json:"modelId"`
	SessionId    string `json:"sessionId"`
	Title        string `json:"title"`
	ModelSetting string `json:"modelSetting"`
	CreatedAt    int64  `json:"createdAt"`
}

// ModelExperienceDialogRecord 模型体验对话记录
type ModelExperienceDialogRecord struct {
	ModelExperienceId string                 `json:"modelExperienceId"` // 模型体验 ID
	ModelId           string                 `json:"modelId"`           // 模型 ID
	SessionId         string                 `json:"sessionId"`         // Session ID
	OriginalContent   string                 `json:"originalContent"`   // 原始内容
	ReasoningContent  string                 `json:"reasoningContent"`  // 思考过程
	Role              string                 `json:"role"`              // 角色
	RequestFiles      []AssistantRequestFile `json:"requestFiles"`      // 文件信息
}
