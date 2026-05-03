package mp_qwen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/UnicomAI/wanwu/pkg/log"
	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
	"github.com/UnicomAI/wanwu/pkg/util"
)

type MultiModalRerank struct {
	ApiKey              string   `json:"apiKey"`
	EndpointUrl         string   `json:"endpointUrl"`
	ContextSize         *int     `json:"contextSize"`
	MaxTextLength       *int64   `json:"maxTextLength"`
	MaxImageSize        *int64   `json:"maxImageSize,omitempty"`
	MaxVideoClipSize    *int64   `json:"maxVideoClipSize,omitempty"`
	SupportFileTypes    []string `json:"supportFileTypes"`
	SupportImageInQuery bool     `json:"supportImageInQuery"`
}

func (cfg *MultiModalRerank) Tags() []mp_common.Tag {
	tags := []mp_common.Tag{
		{
			Text: mp_common.TagMultiModalRerank,
		},
	}
	tags = append(tags, mp_common.GetTagsByContentSize(cfg.ContextSize)...)
	return tags
}

func (cfg *MultiModalRerank) NewReq(req *mp_common.MultiModalRerankReq) (mp_common.IMultiModalRerankReq, error) {
	m := map[string]interface{}{
		"model": req.Model,
	}

	queryMap, err := processRerankQuery(req.Query)
	if err != nil {
		return nil, err
	}

	docsMap, err := processRerankDocuments(req.Documents)
	if err != nil {
		return nil, err
	}

	m["input"] = map[string]interface{}{
		"query":     queryMap,
		"documents": docsMap,
	}

	if req.TopN != nil || req.ReturnDocuments != nil || req.Fps != nil {
		parameters := make(map[string]interface{})
		if req.TopN != nil {
			parameters["top_n"] = *req.TopN
		}
		if req.ReturnDocuments != nil {
			parameters["return_documents"] = *req.ReturnDocuments
		}
		if req.Fps != nil {
			parameters["fps"] = *req.Fps
		}
		m["parameters"] = parameters
	}

	return mp_common.NewMultiModalRerankReq(m), nil
}

func processRerankDocuments(documents []mp_common.MultiDocument) ([]map[string]interface{}, error) {
	content := make([]map[string]interface{}, 0, len(documents))
	for idx, doc := range documents {
		item := make(map[string]interface{})
		if doc.Text != nil && *doc.Text != "" {
			item["text"] = *doc.Text
		} else if doc.Image != nil && *doc.Image != "" {
			item["image"] = *doc.Image
		} else if doc.Video != nil && *doc.Video != "" {
			item["video"] = *doc.Video
		} else {
			return nil, fmt.Errorf("documents第%d个元素无效: image/text/video必选其一", idx+1)
		}
		content = append(content, item)
	}
	return content, nil
}

func processRerankQuery(query interface{}) (map[string]interface{}, error) {
	switch q := query.(type) {
	case string:
		if q == "" {
			return nil, fmt.Errorf("query字符串不能为空")
		}
		return map[string]interface{}{"text": q}, nil

	case map[string]interface{}:
		image, _ := q["image"].(string)
		text, _ := q["text"].(string)
		if image == "" && text == "" {
			return nil, fmt.Errorf("query对象无效: image和text必选其一，不能都为空")
		}
		result := make(map[string]interface{})
		if text != "" {
			result["text"] = text
		}
		if image != "" {
			result["image"] = image
		}
		return result, nil

	default:
		return nil, fmt.Errorf("query类型不支持: %T，仅支持字符串或{image:string,text:string}对象", q)
	}
}

func (cfg *MultiModalRerank) MultiModalRerank(ctx context.Context, req mp_common.IMultiModalRerankReq, headers ...mp_common.Header) (mp_common.IMultiModalRerankResp, error) {
	b, err := mp_common.MultiModalRerank(ctx, "qwen", cfg.ApiKey, cfg.rerankUrl(), req.Data(), headers...)
	if err != nil {
		return nil, err
	}
	return &multiRerankResp{raw: string(b)}, nil
}

func (cfg *MultiModalRerank) rerankUrl() string {
	ret, _ := url.JoinPath(cfg.EndpointUrl, "/services/rerank/text-rerank/text-rerank")
	return ret
}

type multiRerankResp struct {
	raw string
}

type multiRerankSuccessResp struct {
	Output    multiRerankRespOutput `json:"output"`
	Usage     mp_common.Usage       `json:"usage"`
	RequestId string                `json:"request_id"`
}

type multiRerankRespOutput struct {
	Results []multiRerankResultItem `json:"results"`
}

type multiRerankResultItem struct {
	Index          int                    `json:"index"`
	Document       map[string]interface{} `json:"document"`
	RelevanceScore float64                `json:"relevance_score"`
}

type multiRerankErrorResp struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"request_id"`
}

func (resp *multiRerankResp) String() string {
	return resp.raw
}

func (resp *multiRerankResp) Data() (interface{}, bool) {
	ret := make(map[string]interface{})
	if err := json.Unmarshal([]byte(resp.raw), &ret); err != nil {
		log.Errorf("qwen multi_rerank resp (%v) convert to data err: %v", resp.raw, err)
		return nil, false
	}
	return ret, true
}

func (resp *multiRerankResp) ConvertResp() (*mp_common.MultiModalRerankResp, bool) {
	var errResp multiRerankErrorResp
	if err := json.Unmarshal([]byte(resp.raw), &errResp); err == nil {
		if errResp.Code != "" {
			log.Errorf("qwen multi_rerank error: code=%s, message=%s", errResp.Code, errResp.Message)
			return nil, false
		}
	}

	var successResp multiRerankSuccessResp
	if err := json.Unmarshal([]byte(resp.raw), &successResp); err != nil {
		log.Errorf("qwen multi_rerank resp (%v) convert to data err: %v", resp.raw, err)
		return nil, false
	}

	if err := util.Validate(&successResp); err != nil {
		log.Errorf("qwen multi_rerank resp validate err: %v", err)
		return nil, false
	}

	res := &mp_common.MultiModalRerankResp{
		Usage:     successResp.Usage,
		RequestId: &successResp.RequestId,
	}

	res.Results = make([]mp_common.Result, 0, len(successResp.Output.Results))
	for _, item := range successResp.Output.Results {
		res.Results = append(res.Results, mp_common.Result{
			Index:          item.Index,
			RelevanceScore: item.RelevanceScore,
			Document:       item.Document,
		})
	}

	return res, true
}
