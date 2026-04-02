package mp_qwen

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/UnicomAI/wanwu/pkg/log"
	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
	"github.com/UnicomAI/wanwu/pkg/util"
)

type MultiModalEmbedding struct {
	ApiKey           string   `json:"apiKey"`
	EndpointUrl      string   `json:"endpointUrl"`
	ContextSize      *int     `json:"contextSize"`
	MaxTextLength    *int64   `json:"maxTextLength"`
	MaxImageSize     *int64   `json:"maxImageSize"`
	MaxVideoClipSize *int64   `json:"maxVideoClipSize"`
	SupportFileTypes []string `json:"supportFileTypes"`
}

func (cfg *MultiModalEmbedding) Tags() []mp_common.Tag {
	tags := []mp_common.Tag{
		{
			Text: mp_common.TagMultiModalEmbedding,
		},
	}
	tags = append(tags, mp_common.GetTagsByContentSize(cfg.ContextSize)...)
	return tags
}

func (cfg *MultiModalEmbedding) NewReq(req *mp_common.MultiModalEmbeddingReq) (mp_common.IMultiModalEmbeddingReq, error) {
	m := map[string]interface{}{
		"model": req.Model,
		"input": map[string]interface{}{
			"contents": req.Input,
		},
	}
	if req.EncodingFormat != "" {
		m["encoding_format"] = req.EncodingFormat
	}
	if req.Dimensions != nil {
		m["dimensions"] = *req.Dimensions
	}
	if req.Parameters != nil {
		m["parameters"] = req.Parameters
	}
	return mp_common.NewMultiModalEmbeddingReq(m), nil
}

func (cfg *MultiModalEmbedding) MultiModalEmbeddings(ctx context.Context, req mp_common.IMultiModalEmbeddingReq, headers ...mp_common.Header) (mp_common.IMultiModalEmbeddingResp, error) {
	b, err := mp_common.MultiModalEmbeddings(ctx, "qwen", cfg.ApiKey, cfg.embeddingsUrl(), req.Data(), headers...)
	if err != nil {
		return nil, err
	}
	return &multiModalEmbeddingResp{raw: string(b)}, nil
}

func (cfg *MultiModalEmbedding) embeddingsUrl() string {
	ret, _ := url.JoinPath(cfg.EndpointUrl, "")
	return ret
}

type multiModalEmbeddingResp struct {
	raw       string
	Output    respOutput      `json:"output" validate:"required"`
	Usage     mp_common.Usage `json:"usage" validate:"required"`
	RequestId string          `json:"request_id"`
}

type respOutput struct {
	Embeddings []respEmbedding `json:"embeddings" validate:"required,dive"`
}

type respEmbedding struct {
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
	Type      string    `json:"type"`
}

func (resp *multiModalEmbeddingResp) String() string {
	return resp.raw
}

func (resp *multiModalEmbeddingResp) Data() (map[string]interface{}, bool) {
	ret := make(map[string]interface{})
	if err := json.Unmarshal([]byte(resp.raw), &ret); err != nil {
		log.Errorf("qwen multimodal embedding resp (%v) convert to data err: %v", resp.raw, err)
		return nil, false
	}
	return ret, true
}

func (resp *multiModalEmbeddingResp) ConvertResp() (*mp_common.MultiModalEmbeddingResp, bool) {
	if err := json.Unmarshal([]byte(resp.raw), resp); err != nil {
		log.Errorf("qwen multimodal embedding resp (%v) convert to data err: %v", resp.raw, err)
		return nil, false
	}

	if err := util.Validate(resp); err != nil {
		log.Errorf("qwen multimodal embedding resp validate err: %v", err)
		return nil, false
	}

	data := make([]mp_common.EmbeddingData, len(resp.Output.Embeddings))
	for i, e := range resp.Output.Embeddings {
		data[i] = mp_common.EmbeddingData{
			Embedding: e.Embedding,
			Index:     e.Index,
			Type:      &e.Type,
		}
	}

	res := &mp_common.MultiModalEmbeddingResp{
		Model: "",
		Data:  data,
		Usage: resp.Usage,
	}
	return res, true
}
