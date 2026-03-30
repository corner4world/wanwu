package shared

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// 创建关闭思考模式的 ChatModel。
func NewNoReasonChatModel(ctx context.Context, cfg AppConfig) (model.ToolCallingChatModel, error) {
	m, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.ModelID,
		ExtraFields: map[string]any{
			"enable_thinking": false,
			"thinking": map[string]any{
				"type": "disabled",
			},
			"chat_template_kwargs": map[string]any{
				"enable_thinking": false,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}
	return m, nil
}
