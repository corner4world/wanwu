package util

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"strings"
)

func UserMessage(content string) []adk.Message {
	return []adk.Message{
		schema.UserMessage(content),
	}
}

func GetEnioReactChatHistory(ctx context.Context, destAgentName string) ([]adk.Message, error) {
	var messages []adk.Message
	var agentName string
	err := compose.ProcessState(ctx, func(ctx context.Context, st *adk.State) error {
		messages = make([]adk.Message, len(st.Messages)-1)
		copy(messages, st.Messages[:len(st.Messages)-1]) // remove the last assistant message, which is the tool call message
		agentName = st.AgentName
		return nil
	})

	a, t := adk.GenTransferMessages(ctx, destAgentName)
	messages = append(messages, a, t)
	history := make([]adk.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == schema.System {
			continue
		}

		if msg.Role == schema.Assistant || msg.Role == schema.Tool {
			msg = rewriteMessage(msg, agentName)
		}

		history = append(history, msg)
	}

	return history, err
}

func rewriteMessage(msg adk.Message, agentName string) adk.Message {
	var sb strings.Builder
	sb.WriteString("For context:")
	if msg.Role == schema.Assistant {
		if msg.Content != "" {
			text := fmt.Sprintf(" [%s] said: %s.", agentName, msg.Content)
			sb.WriteString(text)
		}
		if len(msg.ToolCalls) > 0 {
			for i := range msg.ToolCalls {
				f := msg.ToolCalls[i].Function
				text := fmt.Sprintf(" [%s] called tool: `%s` with arguments: %s.",
					agentName, f.Name, f.Arguments)
				sb.WriteString(text)
			}
		}
	} else if msg.Role == schema.Tool && msg.Content != "" {
		text := fmt.Sprintf(" [%s] `%s` tool returned result: %s.",
			agentName, msg.ToolName, msg.Content)
		sb.WriteString(text)
	}

	return schema.UserMessage(sb.String())
}
