package response

import (
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
)

type GetGeneralAgentConfigResp struct {
	ToolList      []request.ToolSelected      `json:"toolList"`      // 工具列表
	AssistantList []request.AssistantSelected `json:"assistantList"` // 智能体列表
	MCPList       []request.MCPSelected       `json:"mcpList"`       // MCP列表
	WorkflowList  []request.WorkflowSelected  `json:"workflowList"`  // 工作流列表
	SkillList     []request.SkillSelected     `json:"skillList"`     // 技能列表
}

type WgaAgentInfo struct {
	AgentID     string         `json:"agentId"`     // 子智能体ID
	AgentName   string         `json:"agentName"`   // 子智能体名称
	Avatar      request.Avatar `json:"avatar"`      // logo
	Placeholder string         `json:"placeholder"` // 占位提示文本
}

type GetGeneralAgentSubListResp struct {
	WgaAgentList []WgaAgentInfo `json:"wgaAgentList"` // 子智能体列表
}

type GetGeneralAgentConversationConfigResp struct {
	ThreadID    string                 `json:"threadId"`    // 对话ID
	ModelConfig request.AppModelConfig `json:"modelConfig"` // 模型
}

type CreateGeneralAgentConversationResp struct {
	ThreadID string `json:"threadId"` // 对话ID
}

type GeneralAgentConversationInfo struct {
	ThreadID  string `json:"threadId"`  // 对话ID
	Title     string `json:"title"`     // 对话标题
	CreatedAt string `json:"createdAt"` // 创建时间
}

type GetGeneralAgentToolSelectResp struct {
	Category  string     `json:"category"`  // 类型
	Condition string     `json:"condition"` // 条件 none | optional | required
	ToolList  []ToolInfo `json:"toolList"`  // 工具列表
}

type GeneralAgentToolInfoResp struct {
	Actions []*protocol.Tool `json:"actions"` // action列表
	ToolInfo
}

type GeneralAgentConfigCheckResponse struct {
	Meet      bool                         `json:"meet"`      // 是否符合要求
	ModelMeet bool                         `json:"modelMeet"` // 是否符合模型要求 FIXME 去掉模型检查
	ToolsMeet []GeneralAgentToolCategories `json:"toolsMeet"` // 工具是否符合要求
}

type GeneralAgentToolCategories struct {
	Category  string                  `json:"category"`  // 工具类别类型
	Condition string                  `json:"condition"` // 工具类别条件
	Meet      bool                    `json:"meet"`      // 是否满足条件
	Tools     []GeneralAgentCheckTool `json:"tools"`     // 工具检查结果
}
type GeneralAgentCheckTool struct {
	ToolID string `json:"toolId"` // 工具ID
	Meet   bool   `json:"meet"`   // 是否符合要求
}

type GeneralAgentFileInfo struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // "file" or "directory"
	Size     int64                  `json:"size,omitempty"`
	MimeType string                 `json:"mimeType,omitempty"`
	Children []GeneralAgentFileInfo `json:"children,omitempty"`
}

type GeneralAgentWorkspaceResp struct {
	GeneralAgentConversationWorkspaceInfo
	Path  string                 `json:"path"`
	Files []GeneralAgentFileInfo `json:"files"`
}

type GeneralAgentConversationDetailInfo struct {
	ThreadID  string        `json:"threadId"`
	RunID     string        `json:"runId"`
	CreatedAt int64         `json:"createdAt"`
	Events    []interface{} `json:"events"`
}

type GeneralAgentConversationWorkspaceInfo struct {
	ThreadID  string `json:"threadId"`
	RunID     string `json:"runId"`
	FileCount int32  `json:"fileCount"`
	TotalSize int64  `json:"totalSize"`
	IsDisplay bool   `json:"isDisplay"`
}

type GeneralAgentUploadLimitResp struct {
	UploadLimitList []*GeneralAgentUploadLimit `json:"uploadLimitList"`
}

type GeneralAgentUploadLimit struct {
	FileType string   `json:"fileType"` // 文件类型，如：image、video、audio、document
	MaxSize  int      `json:"maxSize"`  // 文件大小限制，单位MB
	ExtList  []string `json:"extList"`  // 支持的文件后缀列表
}

type GeneralAgentCopilotRuntimeInfoAgent struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ClassName   string `json:"className"`
}

type GeneralAgentCopilotRuntimeInfoResp struct {
	Version                       string                                         `json:"version"`
	Agents                        map[string]GeneralAgentCopilotRuntimeInfoAgent `json:"agents"`
	Mode                          string                                         `json:"mode"`
	AudioFileTranscriptionEnabled bool                                           `json:"audioFileTranscriptionEnabled"`
	A2UIEnabled                   bool                                           `json:"a2uiEnabled,omitempty"`
}

type GeneralAgentResourceSelectItem struct {
	ID     string         `json:"id"`     // ID
	Name   string         `json:"name"`   // 名称
	Desc   string         `json:"desc"`   // 描述
	Avatar request.Avatar `json:"avatar"` // 头像
	Type   string         `json:"type"`   // 类型
	Author string         `json:"author"` // 作者
}

type GeneralAgentResourceSelectList struct {
	ListType string                            `json:"listType"` // 列表类型: mcp, workflow, skill, assistant
	List     []*GeneralAgentResourceSelectItem `json:"list"`     // 列表项
}
