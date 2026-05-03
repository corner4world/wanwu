package request

type AgentChatProxyReq struct {
	Input string `json:"input" validate:"required"`
	// UploadFile  []string `json:"uploadFile"`
}

func (r *AgentChatProxyReq) Check() error {
	return nil
}
