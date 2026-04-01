package wga_sandbox_converter

import (
	"encoding/json"

	"github.com/cloudwego/eino/schema"
)

type einoChatModelConverter struct{}

func newEinoChatModelConverter() *einoChatModelConverter {
	return &einoChatModelConverter{}
}

// agentEventMsg 仅提取 adk.AgentEvent 中需要的字段（PascalCase，无 json tag）
type agentEventMsg struct {
	Output *struct {
		MessageOutput *struct {
			Message *schema.Message `json:"Message"`
		} `json:"MessageOutput"`
	} `json:"Output"`
}

func (c *einoChatModelConverter) Convert(line string) ([]*schema.Message, error) {
	var event agentEventMsg
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return nil, err
	}
	if event.Output == nil || event.Output.MessageOutput == nil || event.Output.MessageOutput.Message == nil {
		return nil, nil
	}
	msg := event.Output.MessageOutput.Message
	if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 && msg.Content != "" {
		msg.Content = ""
	}
	return []*schema.Message{msg}, nil
}
