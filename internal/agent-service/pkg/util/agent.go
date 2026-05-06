package util

import (
	"encoding/json"

	"github.com/cloudwego/eino/schema"
)

const (
	AgentSearchKnowledgeName = "智能体知识库检索"
	AgentSkillPrefix         = "skill-"
	AgentStartLabel          = "transfer_to_agent"
	MainAgentExitLabel       = "exit"
)

func BuildAssistantMessage(content string, extra map[string]any) string {
	message := &schema.Message{
		Role:    schema.Assistant,
		Content: content,
		Extra:   extra,
	}
	marshal, _ := json.Marshal(message)
	return string(marshal)
}

func BuildToolParamsMessage(toolCall []schema.ToolCall) *schema.Message {
	return &schema.Message{ //模拟智能体切换消息
		Role:      schema.Assistant,
		ToolCalls: toolCall,
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: "tool_calls",
		},
	}
}

func BuildToolFinishMessage(content string) string {
	message := &schema.Message{
		Role:    schema.Assistant,
		Content: content,
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: "stop",
		},
	}
	marshal, _ := json.Marshal(message)
	return string(marshal)
}
