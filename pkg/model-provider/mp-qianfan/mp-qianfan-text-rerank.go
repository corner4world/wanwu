package mp_qianfan

import (
	"context"
	"net/url"

	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
)

type Rerank struct {
	ApiKey      string `json:"apiKey"`      // ApiKey
	EndpointUrl string `json:"endpointUrl"` // 推理url
	ContextSize *int   `json:"contextSize"` // 上下文长度
}

func (cfg *Rerank) Tags() []mp_common.Tag {
	tags := []mp_common.Tag{
		{
			Text: mp_common.TagTextRerank,
		},
	}
	tags = append(tags, mp_common.GetTagsByContentSize(cfg.ContextSize)...)
	return tags
}

func (cfg *Rerank) NewReq(req *mp_common.TextRerankReq) (mp_common.ITextRerankReq, error) {
	m, err := req.Data()
	if err != nil {
		return nil, err
	}
	return mp_common.NewRerankReq(m), nil
}

func (cfg *Rerank) Rerank(ctx context.Context, req mp_common.ITextRerankReq, headers ...mp_common.Header) (mp_common.ITextRerankResp, error) {
	b, err := mp_common.Rerank(ctx, "qianfan", cfg.ApiKey, cfg.rerankUrl(), req.Data(), headers...)
	if err != nil {
		return nil, err
	}
	return mp_common.NewRerankResp(string(b)), nil
}

func (cfg *Rerank) rerankUrl() string {
	ret, _ := url.JoinPath(cfg.EndpointUrl, "/rerank")
	return ret
}
