package mp_zhipu

import (
	"context"
	"fmt"
	"net/url"

	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
)

type LLM struct {
	ApiKey          string `json:"apiKey"`
	EndpointUrl     string `json:"endpointUrl"`
	FunctionCalling string `json:"functionCalling" validate:"oneof=noSupport toolCall"`
	VisionSupport   string `json:"visionSupport" validate:"oneof=noSupport support"`
	ThinkingSupport string `json:"thinkingSupport" validate:"oneof=noSupport support"`
	MaxTokens       *int   `json:"maxTokens"`
	ContextSize     *int   `json:"contextSize"`
	MaxImageSize    *int64 `json:"maxImageSize"`
}

func (cfg *LLM) Tags() []mp_common.Tag {
	tags := []mp_common.Tag{
		{
			Text: mp_common.TagChat,
		},
	}
	tags = append(tags, mp_common.GetTagsByVisionSupport(cfg.VisionSupport)...)
	tags = append(tags, mp_common.GetTagsByFunctionCall(cfg.FunctionCalling)...)
	tags = append(tags, mp_common.GetTagsByContentSize(cfg.ContextSize)...)
	return tags
}

func (cfg *LLM) NewReq(req *mp_common.LLMReq) (mp_common.ILLMReq, error) {
	if req.MaxTokens != nil && cfg.ContextSize != nil && *req.MaxTokens > *cfg.ContextSize {
		return nil, fmt.Errorf("max_tokens too large (max allowed: %d)", *cfg.ContextSize)
	}
	m, err := req.Data()
	if err != nil {
		return nil, err
	}
	if req.EnableThinking != nil {
		t := "disabled"
		if *req.EnableThinking {
			t = "enabled"
		}
		m["thinking"] = map[string]string{"type": t}
	}
	if req.Stream != nil && *req.Stream {
		if req.StreamOptions != nil && req.StreamOptions.IncludeUsage != nil {
			m["stream_options"] = map[string]bool{"include_usage": *req.StreamOptions.IncludeUsage}
		} else {
			m["stream_options"] = map[string]bool{"include_usage": true}
		}
	}
	return mp_common.NewLLMReq(m), nil
}

func (cfg *LLM) ChatCompletions(ctx context.Context, req mp_common.ILLMReq, headers ...mp_common.Header) (mp_common.ILLMResp, <-chan mp_common.ILLMResp, error) {
	return mp_common.ChatCompletions(ctx, "zhipu", cfg.ApiKey, cfg.chatCompletionsUrl(), req, mp_common.NewLLMResp, headers...)
}

func (cfg *LLM) chatCompletionsUrl() string {
	ret, _ := url.JoinPath(cfg.EndpointUrl, "/chat/completions")
	return ret
}
