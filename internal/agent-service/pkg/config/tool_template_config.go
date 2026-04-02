package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/UnicomAI/wanwu/internal/agent-service/model/request"
	"github.com/UnicomAI/wanwu/internal/agent-service/pkg"
)

const (
	DocParser = "doc_parser"
	SkillTool = "skill_tool"
)

// ToolTemplateMeta 工具模板元数据配置
type ToolTemplateMeta struct {
	ID       string `mapstructure:"id" json:"id"`               // 工具ID
	FilePath string `mapstructure:"file-path" json:"file-path"` // 配置文件路径
	ToolName string `mapstructure:"tool-name" json:"tool-name"` // 工具名称
	Avatar   string `mapstructure:"avatar" json:"avatar"`       // 工具头像
}

// ToolTemplateConfigData tool-template 配置数据
type ToolTemplateConfigData struct {
	Tools []*ToolTemplateMeta `mapstructure:"tools" json:"tools"` // 工具列表
}

type ToolTemplateConfig struct {
	ConfigPluginToolList   []*request.PluginToolInfo
	toolTemplateMap        map[string]*request.PluginToolInfo // 按ID索引的工具配置
	ToolTemplateConfigData *ToolTemplateConfigData            // 原始配置数据
}

var toolTemplateConfig = ToolTemplateConfig{
	toolTemplateMap: make(map[string]*request.PluginToolInfo),
}

func init() {
	pkg.AddContainer(toolTemplateConfig)
}

func GetToolTemplateConfig() *ToolTemplateConfig {
	return &toolTemplateConfig
}

func (c ToolTemplateConfig) LoadType() string {
	return "tool-template-config"
}

func (c ToolTemplateConfig) Load() error {
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("main config not loaded")
	}

	// 获取 tool-template 配置元数据
	templateMeta := cfg.ToolTemplateConfig
	if templateMeta == nil || len(templateMeta.Tools) == 0 {
		return fmt.Errorf("tool-template config not found or empty")
	}

	c.ToolTemplateConfigData = templateMeta

	// 加载所有工具配置
	for _, meta := range templateMeta.Tools {
		if meta.ID == "" || meta.FilePath == "" {
			fmt.Printf("skip tool template: id=%s,filePath=%s\n", meta.ID, meta.FilePath)
			continue
		}

		pluginTool, err := c.loadToolTemplateFile(meta.FilePath, meta.ToolName, meta.Avatar)
		if err != nil {
			fmt.Printf("load tool template file %s error: %v\n", meta.FilePath, err)
			continue
		}

		// 设置工具名称和头像
		if meta.ToolName != "" {
			pluginTool.ToolName = meta.ToolName
		}
		if meta.Avatar != "" {
			pluginTool.ToolAvatar = meta.Avatar
		}

		// 添加到列表和映射
		c.ConfigPluginToolList = append(c.ConfigPluginToolList, pluginTool)
		c.toolTemplateMap[meta.ID] = pluginTool

		fmt.Printf("load tool template success: id=%s,name=%s,path=%s\n", meta.ID, pluginTool.ToolName, meta.FilePath)
	}

	if len(c.ConfigPluginToolList) == 0 {
		return fmt.Errorf("no tool template loaded successfully")
	}

	return nil
}

// loadToolTemplateFile 加载单个工具模板配置文件
func (c ToolTemplateConfig) loadToolTemplateFile(filePath, toolName, avatar string) (*request.PluginToolInfo, error) {
	b, err := readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file %s err: %v", filePath, err)
	}

	// 替换 %s 为 endpoint
	toolConfig := fmt.Sprintf(string(b), GetConfig().ToolServer.Endpoint)

	// 解析 OpenAPI Schema
	var pluginTool = &request.PluginToolInfo{}
	err = json.Unmarshal([]byte(toolConfig), pluginTool)
	if err != nil {
		return nil, fmt.Errorf("unmarshal api schema %s err: %v", filePath, err)
	}

	pluginTool.ToolAvatar = avatar
	pluginTool.ToolName = toolName

	return pluginTool, nil
}

// GetToolByID 根据工具ID获取工具配置
func (c ToolTemplateConfig) GetToolByID(id string) (*request.PluginToolInfo, bool) {
	tool, ok := c.toolTemplateMap[id]
	return tool, ok
}

// GetAllTools 获取所有工具配置列表
func (c ToolTemplateConfig) GetAllTools() []*request.PluginToolInfo {
	return c.ConfigPluginToolList
}

// GetToolMetaList 获取工具元数据列表
func (c ToolTemplateConfig) GetToolMetaList() []*ToolTemplateMeta {
	if c.ToolTemplateConfigData == nil {
		return nil
	}
	return c.ToolTemplateConfigData.Tools
}

func (c ToolTemplateConfig) StopPriority() int {
	return pkg.DefaultPriority
}

func (c ToolTemplateConfig) Stop() error {
	return nil
}

// readFile 兼容性包装函数
var readFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}
