package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/minio"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
)

func FileUrlConvertBase64(ctx *gin.Context, req *request.FileUrlConvertBase64Req) (string, error) {
	resp, err := http.Get(req.FileUrl)
	if err != nil {
		return "", grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_file_http_get", err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_file_http_get", fmt.Sprintf("StatusCode: %d", resp.StatusCode))
	}
	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_file_read", err.Error())
	}

	base64Str, base64StrWithPrefix, err := util.FileData2Base64(fileData, req.CustomPrefix)
	if err != nil {
		return "", grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_file_convert_base64", err.Error())
	}
	if req.AddPrefix {
		return base64StrWithPrefix, nil
	} else {
		return base64Str, nil
	}
}

func UploadFileByBase64(ctx *gin.Context, req *request.UploadFileByBase64Req) (*response.UploadFileByBase64Resp, error) {
	var finalFileName string
	if req.FileName == "" {
		finalFileName = util.GenUUID()
	} else {
		finalFileName = req.FileName
	}

	base64Data := req.File
	var inferredExt string

	// 尝试从标准 Data URL 提取 MIME 类型
	if strings.HasPrefix(base64Data, "data:") && strings.Contains(base64Data, ";base64,") {
		parts := strings.SplitN(base64Data, ",", 2)
		if len(parts) == 2 {
			header := parts[0]
			base64Data = parts[1]

			mimeType := strings.TrimPrefix(header, "data:")
			if idx := strings.Index(mimeType, ";"); idx != -1 {
				mimeType = mimeType[:idx]
			}

			if mimeType != "" {
				if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
					inferredExt = exts[0]
				}
			}
		}
	}

	var finalExt string
	if req.FileExt != "" {
		ext := strings.TrimPrefix(req.FileExt, ".")
		finalExt = "." + ext
	} else if inferredExt != "" {
		finalExt = inferredExt
	}

	if finalExt != "" {
		finalFileName += finalExt
	}

	// 解码 base64 数据
	fileData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_file_base64_decode", err.Error())
	}

	// 使用 minio 上传文件
	reader := bytes.NewReader(fileData)
	_, _, err = minio.UploadFile(ctx, minio.BucketFileUpload, minio.DirFileExpire, finalFileName, reader, int64(len(fileData)))
	if err != nil {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_file_upload_minio", err.Error())
	}

	objectPath := path.Join(minio.BucketFileUpload, minio.DirFileExpire, finalFileName)
	filePath, _ := url.JoinPath(config.Cfg().Minio.DownloadURL, objectPath)

	return &response.UploadFileByBase64Resp{
		Url: filePath,
		Uri: objectPath,
	}, nil
}
