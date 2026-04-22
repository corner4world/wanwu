package chatmodel

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/UnicomAI/wanwu/pkg/wga-sandbox/internal/runner/eino/agent/shared"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type App struct {
	runner    *adk.Runner
	workspace string
}

func NewApp(ctx context.Context, cfg shared.AppConfig) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.ApplyDefaults()

	chatModel, err := shared.NewNoReasonChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}

	skillMiddleware, err := shared.NewSkillMiddleware(ctx, cfg.Workspace)
	if err != nil {
		return nil, err
	}

	fileBackend := shared.NewShellOnlyBackend(cfg.Workspace)
	bashTool, err := shared.NewBashTool(ctx, fileBackend)
	if err != nil {
		return nil, err
	}

	skillMiddleware.AdditionalTools = append(skillMiddleware.AdditionalTools, bashTool)

	tmplData := struct {
		Workspace   string
		CurrentTime string
	}{
		Workspace:   cfg.Workspace,
		CurrentTime: time.Now().Format("2006/01/02 Mon"),
	}

	tmpl, err := template.New("instruction").Parse(instructionTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse instruction template: %w", err)
	}
	var instrBuf bytes.Buffer
	if err := tmpl.Execute(&instrBuf, tmplData); err != nil {
		return nil, fmt.Errorf("failed to render instruction template: %w", err)
	}

	// 自定义 GenModelInput，使用传入的完整 messages（包含历史）
	genModelInput := func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
		msgs := make([]adk.Message, 0, len(input.Messages)+1)

		if instruction != "" {
			msgs = append(msgs, schema.SystemMessage(instruction))
		}

		msgs = append(msgs, input.Messages...)

		return msgs, nil
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model:         chatModel,
		Name:          "ChatModelAgent",
		Description:   "一个带有skills的智能体助手",
		Instruction:   instrBuf.String(),
		GenModelInput: genModelInput,
		Middlewares:   []adk.AgentMiddleware{skillMiddleware},
		MaxIterations: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	return &App{runner: runner, workspace: cfg.Workspace}, nil
}

func (a *App) Query(ctx context.Context, messages []adk.Message) *adk.AsyncIterator[*adk.AgentEvent] {
	log.Printf("[App] Query with %d messages | Workspace: %s", len(messages), shared.SanitizeForLog(a.workspace))
	return a.runner.Run(ctx, messages)
}

func (a *App) Close() error {
	return nil
}
