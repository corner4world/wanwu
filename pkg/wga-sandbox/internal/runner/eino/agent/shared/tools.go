package shared

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type bashArgs struct {
	Command string `json:"command"`
}

// 创建 bash 执行工具（含安全拦截、退出码、截断处理）。
func NewBashTool(ctx context.Context, backend *ShellOnlyBackend) (tool.BaseTool, error) {
	bashTool, err := utils.InferTool("bash",
		"执行 shell 命令并返回输出结果",
		func(ctx context.Context, input bashArgs) (string, error) {
			result, err := backend.Execute(ctx, &filesystem.ExecuteRequest{
				Command: input.Command,
			})
			if err != nil {
				return "", err
			}
			output := result.Output

			// 检测安全拦截，返回拦截信息让 agent 处理
			if strings.HasPrefix(output, "安全拦截：") {
				output += "\n\n[系统提示] 检测到安全违规操作，请严格遵守安全规范"
				return output, nil
			}

			if result.ExitCode != nil && *result.ExitCode != 0 {
				output += fmt.Sprintf("\n[命令执行失败，退出码: %d]", *result.ExitCode)
			}
			if result.Truncated {
				output += "\n[输出因大小限制被截断]"
			}
			return output, nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create bash tool: %w", err)
	}
	return bashTool, nil
}

// 创建技能中间件（含 LocalBackend 初始化）。
func NewSkillMiddleware(ctx context.Context, workspace string) (adk.AgentMiddleware, error) {
	backend, err := skill.NewLocalBackend(&skill.LocalBackendConfig{
		BaseDir: workspace + "/skills",
	})
	if err != nil {
		return adk.AgentMiddleware{}, fmt.Errorf("failed to create skill backend: %w", err)
	}

	skillMiddleware, err := skill.New(ctx, &skill.Config{
		Backend:    backend,
		UseChinese: true,
	})
	if err != nil {
		return adk.AgentMiddleware{}, fmt.Errorf("failed to create skill middleware: %w", err)
	}
	return skillMiddleware, nil
}
