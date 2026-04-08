package middleware

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/pkg/imaging"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	"github.com/UnicomAI/wanwu/pkg/constant"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/minio"
	"github.com/gin-gonic/gin"
)

var (
	avatarCacheMu sync.Mutex

	avatarCacheLocalDir         = "cache/avatar"
	mcpAvatarCacheLocalDir      = "cache/avatar/mcp"
	workflowAvatarCacheLocalDir = "cache/avatar/workflow"

	workflowMinioPresign     = "workflow/minio/presign/"
	workflowAvatarStaticIcon = "/api/static/icon/"

	avatarHTTPClient  = &http.Client{Timeout: 10 * time.Second}
	avatarMaxBodySize = int64(10 << 20) // 10MB
)

// CacheAvatar 缓存中间件，当请求 cache/avatar 路径下的文件时，检查文件是否存在，如果不存在则从 MinIO 或其他服务下载并缓存
func CacheAvatar() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		urlPath := ctx.Request.URL.Path
		cacheIndex := strings.Index(urlPath, service.AvatarCachePrefix)
		prefixLen := len(service.AvatarCachePrefix)
		if cacheIndex == -1 {
			ctx.Next()
			return
		}
		relativePath := urlPath[cacheIndex+prefixLen:]
		relativePath = strings.TrimPrefix(relativePath, "/")
		if relativePath == "" {
			ctx.Next()
			return
		}
		if ctx.Request.URL.RawQuery != "" {
			relativePath += "?" + ctx.Request.URL.RawQuery
		}
		if strings.HasPrefix(relativePath, "mcp/") {
			// MCP/工具 图片请求
			avatarPath := strings.TrimPrefix(relativePath, "mcp/")
			if handleMCPAvatar(ctx, avatarPath) {
				return
			}
		} else if strings.HasPrefix(relativePath, "workflow/") {
			if handleWorkflowAvatarCache(ctx, relativePath) {
				return
			}
		} else {
			// 自定义图片请求
			isResize := true
			realPath := relativePath
			if strings.HasPrefix(realPath, "custom/") {
				realPath = strings.TrimPrefix(realPath, "custom/")
				isResize = false
			}
			avatarPath := downloadAndCacheAvatar(ctx, realPath, isResize)
			if avatarPath == "" {
				ctx.Next()
				return
			}
			if _, err := os.Stat(avatarPath); err == nil {
				ctx.File(avatarPath)
				ctx.Abort()
				return
			}
		}
		ctx.Next()
	}
}

// handleWorkflowAvatarCache 处理工作流图片缓存请求
func handleWorkflowAvatarCache(ctx *gin.Context, relativePath string) bool {
	relativePath = strings.TrimPrefix(relativePath, "/")
	var completeURL string
	if strings.Contains(relativePath, workflowMinioPresign) {
		pathWithoutWorkflow := strings.TrimPrefix(relativePath, "workflow/")
		completeURL = config.Cfg().Server.WebBaseUrl + "/" + pathWithoutWorkflow
	} else {
		fileName := strings.TrimPrefix(relativePath, "workflow/")
		completeURL = config.Cfg().Server.WebBaseUrl + workflowAvatarStaticIcon + fileName
	}
	avatarPath := cacheWorkflowAvatar(ctx, completeURL, constant.AppTypeWorkflow)
	if avatarPath == "" {
		return false
	}
	localPath := strings.TrimPrefix(avatarPath, "v1/")
	localPath = strings.TrimPrefix(localPath, "/")
	if _, err := os.Stat(localPath); err == nil {
		ctx.File(localPath)
		ctx.Abort()
		return true
	}
	return false
}

// handleMCPAvatar 处理 MCP 图片请求
func handleMCPAvatar(ctx *gin.Context, mcpPath string) bool {
	avatarPath := cacheMCPAvatar(ctx, mcpPath)
	if avatarPath == "" {
		return false
	}
	if _, err := os.Stat(avatarPath); err == nil {
		ctx.File(avatarPath)
		ctx.Abort()
		return true
	}
	return false
}

// downloadAndCacheAvatar 将avatar在minio的objectPath下载并缓存到本地，返回本地缓存路径
func downloadAndCacheAvatar(ctx *gin.Context, avatarObjectPath string, isResize bool) string {
	if avatarObjectPath == "" {
		return ""
	}
	avatarObjectPath = strings.TrimPrefix(avatarObjectPath, "/")
	avatarCacheMu.Lock()
	defer avatarCacheMu.Unlock()

	parts := strings.SplitN(avatarObjectPath, "/", 2)
	if len(parts) <= 1 {
		log.Errorf("cache avatar %v err: invalid objectPath", avatarObjectPath)
		return ""
	}
	bucketName := parts[0]
	objectName := parts[1]
	filePath := filepath.Join(avatarCacheLocalDir, objectName)
	if !isResize {
		filePath = filepath.Join(avatarCacheLocalDir, "raw", objectName)
	}

	if !strings.HasPrefix(filePath, avatarCacheLocalDir) {
		log.Warnf("avatar path traversal attempt: %s", avatarObjectPath)
		return ""
	}

	_, err := os.Stat(filePath)
	if err == nil {
		return filePath
	}
	if !os.IsNotExist(err) {
		log.Errorf("cache avatar %v check %v exist err: %v", avatarObjectPath, filePath, err)
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Errorf("cache avatar %v mkdir %v err: %v", avatarObjectPath, filepath.Dir(filePath), err)
		return ""
	}
	b, err := minio.Custom().GetObject(ctx.Request.Context(), bucketName, objectName)
	if err != nil {
		log.Errorf("cache avatar %v minio download err: %v", avatarObjectPath, err)
		return ""
	}
	if isResize {
		compressedData, err := resizeImage(b)
		if err != nil {
			log.Warnf("cache avatar %v compress failed, using original: %v", avatarObjectPath, err)
			compressedData = b
		}
		if err := os.WriteFile(filePath, compressedData, 0644); err != nil {
			log.Errorf("cache avatar %v write file %v err: %v", avatarObjectPath, filePath, err)
			return ""
		}
		return filePath
	}
	if err := os.WriteFile(filePath, b, 0644); err != nil {
		log.Errorf("cache avatar %v write file %v err: %v", avatarObjectPath, filePath, err)
		return ""
	}
	return filePath
}

// cacheMCPAvatar 下载MCP服务的图片到本地缓存，返回本地缓存路径
func cacheMCPAvatar(ctx *gin.Context, avatarPath string) string {
	if avatarPath == "" {
		return ""
	}
	avatarCacheMu.Lock()
	defer avatarCacheMu.Unlock()

	filePath := filepath.Join(mcpAvatarCacheLocalDir, avatarPath)
	if !strings.HasPrefix(filePath, mcpAvatarCacheLocalDir) {
		log.Warnf("mcp avatar path traversal attempt: %s", avatarPath)
		return ""
	}

	_, err := os.Stat(filePath)
	if err == nil {
		return filePath
	}
	if !os.IsNotExist(err) {
		log.Errorf("cache mcp avatar %v check %v exist err: %v", avatarPath, filePath, err)
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Errorf("cache mcp avatar %v mkdir %v err: %v", avatarPath, filepath.Dir(filePath))
		return ""
	}
	resp, err := service.GetMCPAvatar(ctx, avatarPath)
	if err != nil {
		log.Errorf("cache mcp avatar %v download err: %v", avatarPath, err)
		return ""
	}
	if err := os.WriteFile(filePath, resp.Data, 0644); err != nil {
		log.Errorf("cache mcp avatar %v write file %v err: %v", avatarPath, filePath, err)
		return ""
	}
	return filePath
}

// cacheWorkflowAvatar 下载并缓存工作流/对话流图片，返回本地缓存路径
func cacheWorkflowAvatar(ctx *gin.Context, avatarURL, appType string) string {
	switch appType {
	case constant.AppTypeWorkflow:
		if avatarURL == "" {
			return config.Cfg().DefaultIcon.WorkflowIcon
		}
	case constant.AppTypeChatflow:
		if avatarURL == "" {
			return config.Cfg().DefaultIcon.ChatflowIcon
		}
	}

	avatarCacheMu.Lock()
	defer avatarCacheMu.Unlock()

	baseURL := avatarURL
	if idx := strings.Index(avatarURL, "?"); idx != -1 {
		baseURL = avatarURL[:idx]
	}
	lastSlash := strings.LastIndex(baseURL, "/")
	fileName := baseURL[lastSlash+1:]
	filePath := path.Join(workflowAvatarCacheLocalDir, fileName)
	if _, err := os.Stat(filePath); err == nil {
		return filePath
	}
	var newAvatarURL string
	if strings.Contains(avatarURL, config.Cfg().Workflow.MinioProxyPrefix) {
		parsedURL, err := url.Parse(avatarURL)
		if err != nil {
			log.Errorf("cache avatar parse URL %v failed: %v", avatarURL, err)
			return ""
		}
		path := parsedURL.Path
		path = strings.TrimPrefix(path, config.Cfg().Workflow.MinioProxyPrefix)
		newAvatarURL, err = url.JoinPath(config.Cfg().Workflow.MinioProxyEndpoint, path)
		if err != nil {
			log.Errorf("join path failed: %v", err)
			return ""
		}
		if parsedURL.RawQuery != "" {
			newAvatarURL += "?" + parsedURL.RawQuery
		}
	} else if strings.Contains(avatarURL, config.Cfg().Server.WebBaseUrl) {
		parsedURL, err := url.Parse(avatarURL)
		if err != nil {
			log.Errorf("cache avatar parse URL %v failed: %v", avatarURL, err)
			return ""
		}
		internalURL, err := url.Parse(config.Cfg().Workflow.Endpoint)
		if err != nil {
			log.Errorf("cache avatar invalid Workflow.Endpoint %s", config.Cfg().Workflow.Endpoint)
			return ""
		}
		parsedURL.Host = internalURL.Host
		parsedURL.Scheme = internalURL.Scheme
		newAvatarURL = parsedURL.String()
	} else {
		newAvatarURL = avatarURL
	}
	resp, err := avatarHTTPClient.Get(newAvatarURL)
	if err != nil {
		log.Errorf("cache avatar %v download %v err: %v", avatarURL, newAvatarURL, err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		log.Errorf("cache avatar %v download %v HTTP error: %v", avatarURL, newAvatarURL, resp.Status)
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, avatarMaxBodySize))
	if err != nil {
		log.Errorf("cache avatar %v download %v read response err: %v", avatarURL, newAvatarURL, err)
		return ""
	}
	if len(body) >= int(avatarMaxBodySize) {
		log.Warnf("cache avatar %v download %v body too large, truncated", avatarURL, newAvatarURL)
		return ""
	}
	compressedData, err := resizeImage(body)
	if err != nil {
		log.Warnf("cache avatar %v compress failed, using original: %v", avatarURL, err)
		compressedData = body
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Errorf("cache avatar %v mkdir %v err: %v", avatarURL, filepath.Dir(filePath), err)
		return ""
	}
	if err := os.WriteFile(filePath, compressedData, 0644); err != nil {
		log.Errorf("cache avatar %v write file %v err: %v", avatarURL, filePath, err)
		return ""
	}
	return filePath
}

// resizeImage 压缩图像
func resizeImage(imageData []byte) ([]byte, error) {
	// 先解码获取图像尺寸
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()
	// 计算等比例缩放后的尺寸
	targetWidth, targetHeight := calculateResizeParameters(originalWidth, originalHeight, 200)
	// 重新创建 reader（因为之前的读取位置已经改变）
	reader := bytes.NewReader(imageData)
	// 压缩图像到计算后的尺寸
	compressedData, err := imaging.Resize(reader, targetWidth, targetHeight)
	if err != nil {
		return nil, fmt.Errorf("image resize failed: %w", err)
	}
	return compressedData, nil
}

// 计算等比例缩放尺寸
func calculateResizeParameters(originalWidth, originalHeight, maxSize int) (int, int) {
	if originalWidth <= maxSize && originalHeight <= maxSize {
		// 如果原图已经小于目标尺寸，返回原尺寸
		return originalWidth, originalHeight
	}
	var newWidth, newHeight int
	if originalWidth > originalHeight {
		// 宽图：以宽度为基准
		newWidth = maxSize
		newHeight = int(float64(originalHeight) * float64(maxSize) / float64(originalWidth))
	} else {
		// 高图或正方形：以高度为基准
		newHeight = maxSize
		newWidth = int(float64(originalWidth) * float64(maxSize) / float64(originalHeight))
	}
	// 确保最小尺寸为1
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}
	return newWidth, newHeight
}
