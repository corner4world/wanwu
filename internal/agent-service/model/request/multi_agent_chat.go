package request

type MultiAgentChatParams struct {
	MultiAgentId uint32 `json:"multiAgentId"  validate:"required"` //多智能体ID
	AgentChatBaseReq
}

type MultiAgentChatReq struct {
	Input               string                 `json:"input"`
	UploadFile          []string               `json:"uploadFile"`
	Stream              bool                   `json:"stream"`
	AgentChatBaseParams *AgentChatBaseParams   `json:"agentChatBaseParams"` // 模型参数
	AgentList           []*AgentChatBaseParams `json:"agentList"`
}

func (c *MultiAgentChatParams) Check() error {
	return nil
}
