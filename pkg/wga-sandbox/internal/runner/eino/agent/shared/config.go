package shared

import (
	"context"

	"github.com/cloudwego/eino/adk"
)

// 模型相关默认值，用于 AppConfig.ApplyDefaults 兜底。
const (
	defaultBaseURL = "https://ark.cn-beijing.volces.com/api/v3"
	defaultModel   = "mimo-v2-flash"
)

// AppConfig 提供构建 App 所需的全部配置。
type AppConfig struct {
	Workspace string
	APIKey    string
	BaseURL   string
	ModelID   string
	// Halt 连续 [BLOCKED:...] 计数与熔断回调。可空——为 nil 时不启用熔断
	// （用于 tests + oneshot 沙箱路径的向后兼容）。
	Halt *HaltState
}

// ApplyDefaults 填充默认值。
func (c *AppConfig) ApplyDefaults() {
	if c.BaseURL == "" {
		c.BaseURL = defaultBaseURL
	}
	if c.ModelID == "" {
		c.ModelID = defaultModel
	}
}

// Validate 校验必填字段。
func (c *AppConfig) Validate() error {
	// if c.APIKey == "" {
	// 	return fmt.Errorf("OPENAI_API_KEY is required")
	// }
	return nil
}

// AgentApp 定义 agent 应用的统一接口。
type AgentApp interface {
	Query(ctx context.Context, messages []adk.Message) *adk.AsyncIterator[*adk.AgentEvent]
	Close() error
}
