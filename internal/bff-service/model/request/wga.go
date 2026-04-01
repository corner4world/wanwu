package request

type UpdateGeneralAgentConfigReq struct {
	ToolList      []ToolSelected      `json:"toolList"`      // 工具ID
	AssistantList []AssistantSelected `json:"assistantList"` // 智能体ID
}

func (c *UpdateGeneralAgentConfigReq) Check() error { return nil }

type GetGeneralAgentConversationConfigReq struct {
	ThreadID string `json:"threadId" form:"threadId" validate:"required"` // 对话ID
}

func (c *GetGeneralAgentConversationConfigReq) Check() error { return nil }

type CreateGeneralAgentConversationReq struct {
	Title       string          `json:"title" validate:"required"` // 标题
	ModelConfig *AppModelConfig `json:"modelConfig"`               // 模型
}

func (c *CreateGeneralAgentConversationReq) Check() error { return nil }

type DeleteGeneralAgentConversationReq struct {
	ThreadID string `json:"threadId" validate:"required"` // 对话ID
}

func (c *DeleteGeneralAgentConversationReq) Check() error { return nil }

type GetGeneralAgentConversationListReq struct {
	PageNo   int `json:"pageNo" form:"pageNo" validate:"required"`     // 页码
	PageSize int `json:"pageSize" form:"pageSize" validate:"required"` // 每页数量
}

func (c *GetGeneralAgentConversationListReq) Check() error { return nil }

type GetGeneralAgentConversationDetailReq struct {
	ThreadID string `json:"threadId" form:"threadId" validate:"required"` // 对话ID
	PageNo   int    `json:"pageNo" form:"pageNo"`                         // 页码，默认1
	PageSize int    `json:"pageSize" form:"pageSize"`                     // 每页数量，默认100
}

func (c *GetGeneralAgentConversationDetailReq) Check() error { return nil }

type GeneralAgentConfigCheckRequest struct {
	ThreadID string `json:"threadId" form:"threadId" validate:"required"` // 对话ID
}

func (c *GeneralAgentConfigCheckRequest) Check() error { return nil }

type UpdateGeneralAgentConversationConfigReq struct {
	ThreadID    string          `json:"threadId" validate:"required"`    // 对话ID
	ModelConfig *AppModelConfig `json:"modelConfig" validate:"required"` // 模型
}

type ToolSelected struct {
	ToolID   string `json:"toolId" validate:"required"`   // 工具ID
	ToolType string `json:"toolType" validate:"required"` // 工具类型
}

type AssistantSelected struct {
	AssistantID   string `json:"assistantId" validate:"required"`   // 智能体ID
	AssistantType string `json:"assistantType" validate:"required"` // 智能体类型
}

func (c *UpdateGeneralAgentConversationConfigReq) Check() error { return nil }

type GeneralAgentConversationChatReq struct {
	ThreadID string                            `json:"threadId" validate:"required"` // 对话ID
	Messages []GeneralAgentConversationMessage `json:"messages" validate:"required"` // 消息
}

type GeneralAgentConversationMessage struct {
	ID      string      `json:"id"`                          // 消息id
	Role    string      `json:"role" validate:"required"`    // 角色 user
	Content interface{} `json:"content" validate:"required"` // 内容 string 或者 [{"type":"text","text":"这张图片是什么？"},{"type":"binary","mimeType":"image/png","url":"https://..."}]
}

func (c *GeneralAgentConversationChatReq) Check() error { return nil }

type GeneralAgentWorkspaceDownloadReq struct {
	ThreadID string `json:"threadId" form:"threadId" validate:"required"` // 对话ID
	RunID    string `json:"runId" form:"runId" validate:"required"`       // 运行ID
	Path     string `json:"path" form:"path"`                             // workspace中路径
}

func (c *GeneralAgentWorkspaceDownloadReq) Check() error { return nil }

type GeneralAgentWorkspacePreviewReq struct {
	ThreadID string `json:"threadId" form:"threadId" validate:"required"` // 对话ID
	RunID    string `json:"runId" form:"runId" validate:"required"`       // 运行ID
	Path     string `json:"path" form:"path" validate:"required"`         // 文件路径
}

func (c *GeneralAgentWorkspacePreviewReq) Check() error { return nil }

type GeneralAgentWorkspaceReq struct {
	ThreadID string `json:"threadId" form:"threadId" validate:"required"` // 对话ID
	RunID    string `json:"runId" form:"runId" validate:"required"`       // 运行ID
}

func (c *GeneralAgentWorkspaceReq) Check() error { return nil }

type GeneralAgentCopilotRuntimeReq struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
	Body   map[string]interface{} `json:"body,omitempty"`
}

func (c *GeneralAgentCopilotRuntimeReq) Check() error { return nil }

func (c *GeneralAgentCopilotRuntimeReq) GetThreadID() string {
	threadID, _ := c.Body["threadId"].(string)
	return threadID
}

func (c *GeneralAgentCopilotRuntimeReq) GetMessages() []GeneralAgentConversationMessage {
	if c.Body == nil {
		return nil
	}

	bodyMessages, ok := c.Body["messages"]
	if !ok || bodyMessages == nil {
		return nil
	}

	messagesSlice, ok := bodyMessages.([]interface{})
	if !ok {
		return nil
	}

	messages := make([]GeneralAgentConversationMessage, 0, len(messagesSlice))
	for _, m := range messagesSlice {
		msgMap, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		role, _ := msgMap["role"].(string)
		if role == "" {
			continue
		}

		id, _ := msgMap["id"].(string)
		content := msgMap["content"]
		messages = append(messages, GeneralAgentConversationMessage{
			ID:      id,
			Role:    role,
			Content: content,
		})
	}

	return messages
}
