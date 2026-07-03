package mp_common

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"

	"github.com/UnicomAI/wanwu/pkg/log"
	trace_util "github.com/UnicomAI/wanwu/pkg/trace-util"
)

// --- openapi request ---

type OcrReq struct {
	Url                 *string `json:"url,omitempty"`
	FileData            *string `json:"data,omitempty"`
	FileName            string  `json:"fileName" validate:"required"`
	Model               *string `json:"model,omitempty"`
	FileType            *int    `json:"fileType,omitempty"`
	ExtractImage        *int    `json:"extract_image,omitempty"`
	ExtractImageContent *int    `json:"extract_image_content,omitempty"`
}

func (req *OcrReq) Check() error {
	if req.FileName == "" {
		return fmt.Errorf("参数错误：fileName 必填")
	}
	hasUrl := req.Url != nil && *req.Url != ""
	hasData := req.FileData != nil && *req.FileData != ""
	if !hasUrl && !hasData {
		return fmt.Errorf("参数错误：url 和 data 必须传入一个有效参数")
	}
	if hasUrl && hasData {
		return fmt.Errorf("参数错误：url 和 data 只能传入一个有效参数")
	}
	return nil
}

func (req *OcrReq) Data() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// --- openapi response ---

type OcrResp struct {
	Code           int          `json:"code"`
	Message        string       `json:"message"`
	Meta           *OcrMeta     `json:"meta,omitempty"`
	Data           *OcrRespData `json:"data,omitempty"`
	Version        string       `json:"version,omitempty"`
	PrefixImageUrl string       `json:"prefixImageUrl,omitempty"`
}

type OcrMeta struct {
	RequestId string `json:"requestId,omitempty"`
	TraceId   string `json:"traceId,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type OcrRespData struct {
	FileInfo    *OcrFileInfo `json:"fileInfo,omitempty"`
	FullContent string       `json:"fullContent,omitempty"`
	OcrResults  []OcrResult  `json:"ocrResults,omitempty"`
}

type OcrFileInfo struct {
	FileName   string `json:"fileName,omitempty"`
	FileType   string `json:"fileType,omitempty"`
	TotalPages int64  `json:"totalPages,omitempty"`
}

type OcrResult struct {
	PageNumber int    `json:"pageNumber"`
	Type       string `json:"type"`
	Content    string `json:"content"`
}

// --- request ---

type IOcrReq interface {
	Data() *OcrReq
}

// ocrReq implementation of IOcrReq
type ocrReq struct {
	data *OcrReq
}

func NewOcrReq(data *OcrReq) IOcrReq {
	return &ocrReq{data: data}
}

func (req *ocrReq) Data() *OcrReq {
	return req.data
}

// --- response ---

type IOcrResp interface {
	String() string
	Data() (interface{}, bool)
	ConvertResp() (*OcrResp, bool)
}

// ocrResp implementation of IOcrResp
type ocrResp struct {
	raw string
}

func NewOcrResp(raw string) IOcrResp {
	return &ocrResp{raw: raw}
}

func (resp *ocrResp) String() string {
	return resp.raw
}

func (resp *ocrResp) Data() (interface{}, bool) {
	ret := make(map[string]interface{})
	if err := json.Unmarshal([]byte(resp.raw), &ret); err != nil {
		log.Errorf("ocr resp (%v) convert to data err: %v", resp.raw, err)
		return nil, false
	}
	return ret, true
}

func (resp *ocrResp) ConvertResp() (*OcrResp, bool) {
	var ret *OcrResp
	if err := json.Unmarshal([]byte(resp.raw), &ret); err != nil {
		log.Errorf("ocr resp (%v) convert to data err: %v", resp.raw, err)
		return nil, false
	}

	// code == 0 表示成功
	if ret.Code != 0 {
		log.Errorf("ocr resp code %v, message: %v", ret.Code, ret.Message)
		return nil, false
	}
	return ret, true
}

// --- ocr ---

// Ocr 向下游供应商发送 OCR 请求（JSON 格式）
func Ocr(ctx context.Context, provider, apiKey, url string, req *OcrReq, headers ...Header) ([]byte, error) {
	if apiKey != "" {
		headers = append(headers, Header{
			Key:   "Authorization",
			Value: "Bearer " + apiKey,
		})
	}

	request := trace_util.NewResty(ctx).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}). // 关闭证书校验
		SetTimeout(0).                                             // 关闭请求超时
		R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetBody(req).
		SetDoNotParseResponse(true)
	for _, header := range headers {
		request.SetHeader(header.Key, header.Value)
	}

	resp, err := request.Post(url)
	if err != nil {
		return nil, fmt.Errorf("request %v %v ocr err: %v", url, provider, err)
	}
	b, err := io.ReadAll(resp.RawResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("request %v %v ocr read response body err: %v", url, provider, err)
	}
	if resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("request %v %v ocr http status %v msg: %v", url, provider, resp.StatusCode(), string(b))
	}
	return b, nil
}
