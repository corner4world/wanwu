package option

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

func (options *Options) checkModel() error {
	if options.Model.Model == "" {
		return fmt.Errorf("model required")
	}
	if options.Model.BaseURL == "" {
		return fmt.Errorf("model base url empty")
	}
	return nil
}

// ToChatModel 创建聊天模型实例。
func (options *Options) ToChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	if err := options.checkModel(); err != nil {
		return nil, err
	}
	cfg := &openai.ChatModelConfig{
		Model:   options.Model.Model,
		APIKey:  options.Model.APIKey,
		BaseURL: options.Model.BaseURL,
	}
	if options.Model.Params != nil {
		if options.Model.Params.TemperatureEnable {
			temp := float32(options.Model.Params.Temperature)
			cfg.Temperature = &temp
		}
		if options.Model.Params.TopPEnable {
			topP := float32(options.Model.Params.TopP)
			cfg.TopP = &topP
		}
		if options.Model.Params.FrequencyPenaltyEnable {
			fp := float32(options.Model.Params.FrequencyPenalty)
			cfg.FrequencyPenalty = &fp
		}
		if options.Model.Params.PresencePenaltyEnable {
			pp := float32(options.Model.Params.PresencePenalty)
			cfg.PresencePenalty = &pp
		}
	}
	return openai.NewChatModel(ctx, cfg)
}
