// Package wga 提供 AI 智能体的统一管理和执行接口。
//
// 支持多种智能体类型：react、sandbox、sequential、loop、parallel、deep、supervisor。
package wga

import (
	"context"
	"errors"
	"fmt"

	wga_sandbox "github.com/UnicomAI/wanwu/pkg/wga-sandbox"
	"github.com/UnicomAI/wanwu/pkg/wga/internal/config"
	"github.com/UnicomAI/wanwu/pkg/wga/internal/factory"
	"github.com/UnicomAI/wanwu/pkg/wga/internal/option"
	wga_option "github.com/UnicomAI/wanwu/pkg/wga/wga-option"
	"github.com/cloudwego/eino/adk"
)

var (
	ErrWgaNotInit     = errors.New("wga not init")
	ErrWgaAlreadyInit = errors.New("wga already init")
)

var _agents []*config.Agent

// Init 初始化智能体配置。
func Init(ctx context.Context, configPath string) error {
	if _agents != nil {
		return ErrWgaAlreadyInit
	}
	agents, err := config.LoadAgents(ctx, configPath)
	if err != nil {
		return err
	}
	_agents = agents
	return nil
}

// GetAgentToolCategories 获取智能体及其子智能体的工具类别配置。
func GetAgentToolCategories(id string) ([]*wga_option.ToolCategoryInfo, error) {
	agent, err := getAgent(id)
	if err != nil {
		return nil, err
	}
	return agent.CollectToolCategories(), nil
}

// CheckToolOptions 检查智能体工具配置是否满足运行要求，返回检查结果。
func CheckToolOptions(_ context.Context, id string, opts ...option.Option) (*wga_option.CheckResult, error) {
	agentCfg, err := getAgent(id)
	if err != nil {
		return nil, err
	}
	var options option.Options
	if err := options.Apply(opts...); err != nil {
		return nil, err
	}
	// 检查工具配置（包括配置文件工具条件和额外工具冲突检查）
	toolCategories, err := options.CheckTools(agentCfg)
	if err != nil {
		return nil, err
	}
	return &wga_option.CheckResult{
		ToolCategories: toolCategories,
	}, nil
}

// Run 执行智能体任务，返回会话标识和事件迭代器。
func Run(ctx context.Context, id string, opts ...option.Option) (wga_option.RunSession, *adk.AsyncIterator[*adk.AgentEvent], error) {
	agentCfg, err := getAgent(id)
	if err != nil {
		return wga_option.RunSession{}, nil, err
	}
	var options option.Options
	if err := options.Apply(opts...); err != nil {
		return wga_option.RunSession{}, nil, err
	}
	if err := options.CheckModelConfig(); err != nil {
		return wga_option.RunSession{}, nil, err
	}
	if err := options.CheckMessages(); err != nil {
		return wga_option.RunSession{}, nil, err
	}
	// 暂不在 Run 阶段进行工具检查，由业务层决定是否调用 CheckOptions 进行工具检查；wga 内部不加载配置无效的工具
	// toolCategories, err := options.CheckToolOptions(agentCfg)
	// if err != nil {
	// 	return wga_option.RunSession{}, nil, err
	// }
	// for _, tc := range toolCategories {
	// 	if !tc.Meet {
	// 		return wga_option.RunSession{}, nil, fmt.Errorf("tool category (%s) condition (%s) not meet", tc.Category, tc.Condition)
	// 	}
	// }
	agent, err := factory.NewAgent(ctx, agentCfg, options)
	if err != nil {
		return wga_option.RunSession{}, nil, err
	}
	return options.RunSession, agent.Run(ctx, &adk.AgentInput{Messages: options.Messages, EnableStreaming: true}), nil
}

// Cleanup 清理指定 runID 的沙箱工作目录。
func Cleanup(ctx context.Context, runID string) error {
	return wga_sandbox.Cleanup(ctx, runID)
}

// --- internal ---

func getAgent(id string) (*config.Agent, error) {
	for _, agent := range _agents {
		if agent.ID == id {
			return agent, nil
		}
	}
	return nil, fmt.Errorf("agent (%s) not found", id)
}
