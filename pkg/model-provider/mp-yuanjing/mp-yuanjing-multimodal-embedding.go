package mp_yuanjing

import (
	"context"
	"net/url"

	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
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
	m, err := req.Data()
	if err != nil {
		return nil, err
	}
	return mp_common.NewMultiModalEmbeddingReq(m), nil
}

func (cfg *MultiModalEmbedding) MultiModalEmbeddings(ctx context.Context, req mp_common.IMultiModalEmbeddingReq, headers ...mp_common.Header) (mp_common.IMultiModalEmbeddingResp, error) {
	b, err := mp_common.MultiModalEmbeddings(ctx, "yuanjing", cfg.ApiKey, cfg.embeddingsUrl(), req.Data(), headers...)
	if err != nil {
		return nil, err
	}
	return mp_common.NewMultiModalEmbeddingResp(string(b)), nil
}

func (cfg *MultiModalEmbedding) embeddingsUrl() string {
	ret, _ := url.JoinPath(cfg.EndpointUrl, "/embeddings")
	return ret
}
