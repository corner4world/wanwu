package wga_option

import (
	"github.com/UnicomAI/wanwu/pkg/wga/internal/config"
	"github.com/UnicomAI/wanwu/pkg/wga/internal/option"
	"github.com/cloudwego/eino/adk"
)

type Option = option.Option
type ModelConfig = option.ModelConfig
type ToolConfig = option.ToolConfig
type ExtraTool = option.ExtraTool
type RunSession = option.RunSession

type CheckResult = option.CheckResult
type CheckModel = option.CheckModel

type ToolCategoryInfo = config.ToolCategory

// WithModelConfig 设置模型配置。
func WithModelConfig(model ModelConfig) Option {
	return option.WithModelConfig(model)
}

// WithToolConfig 添加工具配置，工具标题不能重复。
func WithToolConfig(tool ToolConfig) Option {
	return option.WithToolConfig(tool)
}

// WithExtraTool 添加额外工具（非配置文件中的工具）。
// 工具标题不能与配置文件中的工具重复，也不能与已添加的额外工具重复。
func WithExtraTool(tool ExtraTool) Option {
	return option.WithExtraTool(tool)
}

// WithInputDir 设置输入目录。
// 输入目录的内容会在执行前复制到沙箱工作目录。
// 支持 "/." 后缀：如 "/path/to/dir/." 表示复制目录内容而非目录本身。
func WithInputDir(inputDir string) Option {
	return option.WithInputDir(inputDir)
}

// WithOutputDir 设置输出目录。
// 沙箱工作目录的内容会在执行后复制到该目录。
// 注意：隐藏文件（以 "." 开头）不会被复制。
func WithOutputDir(outputDir string) Option {
	return option.WithOutputDir(outputDir)
}

// WithRunSession 设置执行会话标识（ThreadID 和 RunID）。
func WithRunSession(session RunSession) Option {
	return option.WithRunSession(session)
}

// WithMessages 设置历史消息。
func WithMessages(messages []adk.Message) Option {
	return option.WithMessages(messages)
}
