package option

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// NeedsDeepSeekCompat checks if the model requires DeepSeek API compatibility handling.
//
// DeepSeek models have two specific requirements:
//  1. "Invalid assistant message: content or tool_calls must be set"
//     — assistant messages must have non-empty content or tool_calls
//  2. "The reasoning_content in the thinking mode must be passed back to the API"
//     — reasoning_content must be echoed back in subsequent requests
//
// Not all platforms hosting DeepSeek models have these issues; some platforms
// (e.g., Alibaba Cloud DashScope) handle reasoning_content internally.
// This function uses a combination of provider and model name heuristics.
func NeedsDeepSeekCompat(m ModelConfig) bool {
	// DeepSeek v4 series models (detected by model name)
	model := strings.ToLower(m.Model)
	return strings.Contains(model, "deepseek-v4") || strings.Contains(model, "deepseek_v4")
}

// DeepSeekCompatMiddleware returns an AgentMiddleware that fixes DeepSeek API compatibility issues.
//
// DeepSeek API requires every assistant message to have non-empty "content" or "tool_calls".
// The "reasoning_content" field does NOT satisfy this requirement.
// This middleware sets a placeholder Content (" ") for assistant messages that lack both,
// preventing the "Invalid assistant message: content or tool_calls must be set" error.
//
// The reasoning_content echo-back is handled at the vendor level (openai chat_model.go genRequest),
// which propagates schema.Message.ReasoningContent to the API request.
func DeepSeekCompatMiddleware() adk.AgentMiddleware {
	return adk.AgentMiddleware{
		BeforeChatModel: func(ctx context.Context, state *adk.ChatModelAgentState) error {
			for _, msg := range state.Messages {
				if msg.Role == schema.Assistant && msg.Content == "" && len(msg.ToolCalls) == 0 {
					msg.Content = " "
				}
			}
			return nil
		},
	}
}
