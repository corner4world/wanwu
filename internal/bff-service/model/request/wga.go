package request

// UpdateGeneralAgentConfigReq 更新通用智能体配置请求
type UpdateGeneralAgentConfigReq struct {
	Mcp       []GeneralAgentConfigItem     `json:"mcp"`
	Workflow  []GeneralAgentConfigItem     `json:"workflow"`
	Skill     []GeneralAgentConfigItem     `json:"skill"`
	Assistant []GeneralAgentConfigItem     `json:"assistant"`
	Knowledge []GeneralAgentConfigItem     `json:"knowledge"`
	Tool      []GeneralAgentConfigToolItem `json:"tool"`
}

// GeneralAgentConfigItem 配置项（带type）
type GeneralAgentConfigItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// GeneralAgentConfigToolItem tool配置项
type GeneralAgentConfigToolItem struct {
	ID   string `json:"toolId"`
	Type string `json:"toolType"`
}

func (c *UpdateGeneralAgentConfigReq) Check() error { return nil }

type GetGeneralAgentConversationConfigReq struct {
	ThreadID string `json:"threadId" form:"threadId" validate:"required"` // 对话ID
}

func (c *GetGeneralAgentConversationConfigReq) Check() error { return nil }

type CreateGeneralAgentConversationReq struct {
	Title       string          `json:"title" validate:"required"`       // 标题
	ModelConfig *AppModelConfig `json:"modelConfig" validate:"required"` // 模型
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
}

func (c *GetGeneralAgentConversationDetailReq) Check() error { return nil }

type GeneralAgentConfigCheckRequest struct {
	AgentID  string `json:"agentId"`                                      // 子智能体ID
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
	AssistantID string `json:"assistantId" validate:"required"` // 智能体ID
}

func (c *UpdateGeneralAgentConversationConfigReq) Check() error { return nil }

type GeneralAgentConversationChatReq struct {
	AgentID  string                            `json:"agentId"`                      // 智能体ID
	ThreadID string                            `json:"threadId" validate:"required"` // 对话ID
	Messages []GeneralAgentConversationMessage `json:"messages" validate:"required"` // 消息
}

func (c *GeneralAgentConversationChatReq) Check() error { return nil }

type GeneralAgentConversationMessage struct {
	ID      string      `json:"id"`                          // 消息id
	Role    string      `json:"role" validate:"required"`    // 角色 user
	Content interface{} `json:"content" validate:"required"` // 内容 string 或者 [{"type":"text","text":"这张图片是什么？"},{"type":"binary","mimeType":"image/png","url":"https://..."}]
}

func (m *GeneralAgentConversationMessage) GetURLs() map[string]string {
	urls := make(map[string]string)
	switch v := m.Content.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "binary" {
					urlStr, _ := m["url"].(string)
					fileName, _ := m["fileName"].(string)
					if urlStr != "" && fileName != "" {
						urls[fileName] = urlStr
					}
				}
			}
		}
	}
	return urls
}

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
