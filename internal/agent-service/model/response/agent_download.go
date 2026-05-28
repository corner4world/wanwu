package response

import (
	"path/filepath"
	"strings"

	"github.com/UnicomAI/wanwu/internal/agent-service/pkg/util"
	"github.com/UnicomAI/wanwu/pkg/log"
)

var MaxDownloadFileCount = 3

type AgentFile struct {
	Name     string     `json:"name"`
	Size     int        `json:"size"`
	FileUrl  string     `json:"fileUrl"`
	FileType string     `json:"fileType"`
	Metadata *AgentMeta `json:"metadata"`
}

type AgentMeta struct {
	Desc     string `json:"desc"`
	CreateAt string `json:"createAt"`
	Name     string `json:"name"`
}

type DownloadFileInfo struct {
	FileName string `json:"fileName"`
	FilePath string `json:"filePath"`
	FileSize int64  `json:"fileSize"`
	CreateAt string `json:"createAt"`
}

// ParseDownloadFileInfoList 从 interface{} 转换为 []*DownloadFileInfo
// 支持以下输入类型:
// 1. []*DownloadFileInfo - 直接返回
// 2. []interface{} - 每个 element 是 map[string]interface{} 或 DownloadFileInfo
// 3. []map[string]interface{} - JSON 反序列化后的类型
func ParseDownloadFileInfoList(data interface{}) []*DownloadFileInfo {
	if data == nil {
		return nil
	}

	// 尝试直接类型断言
	if fileList, ok := data.([]*DownloadFileInfo); ok {
		return fileList
	}

	// 尝试 []interface{} 类型 (JSON 反序列化常见情况)
	if slice, ok := data.([]interface{}); ok {
		var result []*DownloadFileInfo
		for _, item := range slice {
			info := parseDownloadFileInfo(item)
			if info != nil {
				result = append(result, info)
			}
		}
		return result
	}

	// 尝试 []map[string]interface{} 类型
	if slice, ok := data.([]map[string]interface{}); ok {
		var result []*DownloadFileInfo
		for _, item := range slice {
			info := parseDownloadFileInfoFromMap(item)
			if info != nil {
				result = append(result, info)
			}
		}
		return result
	}

	log.Warnf("ParseDownloadFileInfoList: unsupported type %T", data)
	return nil
}

// parseDownloadFileInfo 从单个 interface{} 解析 DownloadFileInfo
func parseDownloadFileInfo(data interface{}) *DownloadFileInfo {
	if data == nil {
		return nil
	}

	// 尝试直接类型断言
	if info, ok := data.(*DownloadFileInfo); ok {
		return info
	}

	// 尝试 map[string]interface{} 类型 (JSON 反序列化)
	if m, ok := data.(map[string]interface{}); ok {
		return parseDownloadFileInfoFromMap(m)
	}

	log.Warnf("parseDownloadFileInfo: unsupported item type %T", data)
	return nil
}

// parseDownloadFileInfoFromMap 从 map[string]interface{} 解析 DownloadFileInfo
func parseDownloadFileInfoFromMap(m map[string]interface{}) *DownloadFileInfo {
	if m == nil {
		return nil
	}

	info := &DownloadFileInfo{}

	// FileName
	if v, ok := m["fileName"]; ok {
		if s, ok := v.(string); ok {
			info.FileName = s
		}
	}

	// FilePath
	if v, ok := m["filePath"]; ok {
		if s, ok := v.(string); ok {
			info.FilePath = s
		}
	}

	// FileSize
	if v, ok := m["fileSize"]; ok {
		switch val := v.(type) {
		case int64:
			info.FileSize = val
		case int:
			info.FileSize = int64(val)
		case float64:
			info.FileSize = int64(val)
		case float32:
			info.FileSize = int64(val)
		}
	}

	// CreateAt
	if v, ok := m["createAt"]; ok {
		if s, ok := v.(string); ok {
			info.CreateAt = s
		}
	}

	return info
}

type DownloadContext struct {
	DownloadMap  map[string][]*DownloadFileInfo
	DownloadList [][]*DownloadFileInfo
}

func NewDownloadContext() *DownloadContext {
	return &DownloadContext{
		DownloadMap:  make(map[string][]*DownloadFileInfo),
		DownloadList: make([][]*DownloadFileInfo, 0),
	}
}

func (d *DownloadContext) AddDownloadFile(toolId string, fileList []*DownloadFileInfo) {
	if fileList == nil {
		fileList = make([]*DownloadFileInfo, 0)
	}
	d.DownloadMap[toolId] = fileList
	d.DownloadList = append(d.DownloadList, fileList)
}

func (d *DownloadContext) ResponseFiles(finish int) []*AgentFile {
	if finish != 1 || len(d.DownloadList) == 0 {
		return make([]*AgentFile, 0)
	}
	var responseFiles []*AgentFile
	start, end := buildFileRange(len(d.DownloadList))
	for i := start; i >= end; i-- {
		fileList := d.DownloadList[i]
		for _, file := range fileList {
			responseFiles = append(responseFiles, buildAgentFile(file))
		}
	}

	return responseFiles
}

// buildFileRange 构建遍历范围，倒叙遍历
func buildFileRange(listLen int) (start int, end int) {
	start = listLen - 1
	end = listLen - MaxDownloadFileCount
	if end < 0 {
		end = 0
	}
	return start, end
}

// buildAgentFile 构造AgentFile
func buildAgentFile(file *DownloadFileInfo) *AgentFile {
	fileExt := filepath.Ext(file.FileName)
	fileType := strings.TrimPrefix(fileExt, ".")
	fileNameWithoutExt := strings.TrimSuffix(file.FileName, fileExt)
	var fileName = file.FileName
	if len(fileName) == 0 {
		fileName = util.ExtractFileNameFromURL(file.FilePath)
	}

	return &AgentFile{
		Name:     fileName,
		Size:     int(file.FileSize),
		FileUrl:  file.FilePath,
		FileType: fileType,
		Metadata: &AgentMeta{
			CreateAt: file.CreateAt,
			Name:     fileNameWithoutExt,
		},
	}
}
