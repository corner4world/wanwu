// Package opencode 提供 opencode 智能体的运行器实现（基于 HTTP API）。
//
// 基于 opencode >= v1.4.11 的 SSE 事件格式实现。
// 通过 SSE 连接接收 opencode 事件流，转换为统一的 JSON 格式输出。
package opencode

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/UnicomAI/wanwu/pkg/log"
	openapi3_util "github.com/UnicomAI/wanwu/pkg/openapi3-util"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/UnicomAI/wanwu/pkg/wga-sandbox/internal/runner"
	"github.com/UnicomAI/wanwu/pkg/wga-sandbox/internal/sandbox"
	wga_sandbox_option "github.com/UnicomAI/wanwu/pkg/wga-sandbox/wga-sandbox-option"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
)

// ============================================================================
// 常量
// ============================================================================

const (
	// opencode.json 配置文件模板
	configTemplate = `{
  "$schema": "https://opencode.ai/config.json",
  "permission": {
    "*": "allow"{{if .EnableHumanInTheLoop}},
    "question": "ask"{{end}}
  },
  "provider": {
    "{{.Provider}}": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "{{.ProviderName}}",
      "options": {
        "baseURL": "{{.BaseURL}}",
        "apiKey": "{{.APIKey}}"
      },
      "models": {
        "{{.Model}}": {
          "name": "{{.ModelName}}"
        }
      }
    }
  }{{if .MCPs}},
  "mcp": {
{{range $i, $mcp := .MCPs}}    "{{$mcp.Name}}": {
      "type": "remote",
      "url": "{{$mcp.URL}}",
      "enabled": true
    }{{if ne $i (sub (len $.MCPs) 1)}},{{end}}
{{end}}  }{{end}}
}`

	// 系统提示词模板
	systemTemplate = `# 任务要求

{{if .Instruction}}---

## 系统提示词

{{.Instruction}}

{{end}}{{if .OverallTask}}---

## 整体任务

{{.OverallTask}}

{{end}}{{if .Messages}}---

## 历史信息

{{range .Messages}}### role: {{.Role}}

{{.ReasoningContent}}{{if .UserInputMultiContent}}{{$parts := listParts .UserInputMultiContent}}{{range $i, $part := $parts}}{{if $i}}

{{end}}{{$part}}{{end}}{{else}}{{.Content}}{{end}}

{{end}}{{end}}`
)

// ============================================================================
// 类型
// ============================================================================

// 确保 Runner 实现 runner.Runner 接口
var _ runner.Runner = (*Runner)(nil)

// Runner 实现 opencode 智能体运行器（基于 HTTP API）。
// 通过 SSE 连接接收事件流，转换为 JSON 格式输出。
type Runner struct {
	sb              sandbox.Sandbox
	opt             wga_sandbox_option.RunOption
	sessionID       string
	userMsgIDs      map[string]bool // 用户消息 ID 集合，用于过滤
	logPrefix       string
	partTypeTracker *partTypeTracker // partID -> part 类型映射，用于 delta 事件判断 text/reasoning
}

// partTypeTracker 跟踪 partID 到 part 类型的映射。
// message.part.delta 事件只携带 partID 而不携带 part 类型，
// 需要通过 message.part.updated 事件中记录的 part 类型来推断 delta 的类型。
type partTypeTracker struct {
	types map[string]string // partID -> type ("text" | "reasoning" | "tool" | ...)
}

func newPartTypeTracker() *partTypeTracker {
	return &partTypeTracker{types: make(map[string]string)}
}

// set 记录 partID 对应的类型。
func (t *partTypeTracker) set(partID, partType string) {
	if partID != "" && partType != "" {
		t.types[partID] = partType
	}
}

// get 获取 partID 对应的类型。
func (t *partTypeTracker) get(partID string) string {
	return t.types[partID]
}

// stepStart 在新的 step 开始时调用，清理旧的 part 类型映射。
func (t *partTypeTracker) stepStart(stepID string) {
	// 新 step 开始时清理上一轮的映射，避免内存泄漏
	t.types = make(map[string]string)
}

// stepFinish 在 step 结束时调用。
func (t *partTypeTracker) stepFinish() {
	// step 结束时不清理，因为可能还有后续 delta 事件
}

// ============================================================================
// 公开方法
// ============================================================================

// NewRunner 创建 opencode 运行器实例。
func NewRunner(sb sandbox.Sandbox, opt wga_sandbox_option.RunOption) runner.Runner {
	logPrefix := fmt.Sprintf("[wga-sandbox][%s]", opt.RunSession.RunID)
	return &Runner{
		sb:              sb,
		opt:             opt,
		userMsgIDs:      make(map[string]bool),
		logPrefix:       logPrefix,
		partTypeTracker: newPartTypeTracker(),
	}
}

// BeforeRun 执行前准备工作：
// 1. 创建 opencode 配置文件
// 2. 复制 skills 和 tools
// 3. 复制输入文件
// 4. 创建 opencode session
// 注意：沙箱环境已在 Manager.Create 时通过 Prepare 初始化，此处不再调用
func (r *Runner) BeforeRun(ctx context.Context) error {
	if err := r.setupConfig(ctx); err != nil {
		return err
	}

	if err := r.setupSkills(ctx); err != nil {
		return err
	}

	if err := r.setupTools(ctx); err != nil {
		return err
	}

	if r.opt.InputDir != "" {
		if err := r.sb.CopyToSandbox(ctx, r.opt.InputDir); err != nil {
			return fmt.Errorf("failed to copy input to workspace: %w", err)
		}
	}

	if err := r.createSession(ctx); err != nil {
		return fmt.Errorf("failed to create opencode session: %w", err)
	}

	return nil
}

// Run 执行智能体任务，返回 JSON 格式事件流。
// 通过 SSE 连接接收 opencode 事件，过滤并转换为 JSON 格式输出。
func (r *Runner) Run(ctx context.Context) (<-chan string, error) {
	sseCh, err := r.connectSSE(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect SSE: %w", err)
	}

	if err := r.sendPromptAsync(ctx); err != nil {
		return nil, fmt.Errorf("failed to send prompt: %w", err)
	}

	outputCh := make(chan string, 1024)

	go func() {
		defer util.PrintPanicStack()
		defer close(outputCh)
		r.processSSEEvents(ctx, sseCh, outputCh)
	}()

	return outputCh, nil
}

// AfterRun 执行后处理：
// 1. 删除 opencode session
// 2. 复制输出文件到本地（如果指定了 OutputDir）
// 沙箱清理由外部统一管理，不在此处处理
func (r *Runner) AfterRun(ctx context.Context) error {
	r.deleteSession(ctx)

	if r.opt.OutputDir != "" {
		return r.copyOutput(ctx)
	}
	return nil
}

// ============================================================================
// 生命周期 - 准备
// ============================================================================

// setupConfig 创建 opencode 配置文件。
func (r *Runner) setupConfig(ctx context.Context) error {
	if _, err := r.sb.ExecuteSync(ctx, "mkdir", "-p", ".opencode"); err != nil {
		return fmt.Errorf("failed to create .opencode directory: %w", err)
	}

	content, err := renderConfig(r.opt.ModelConfig, r.opt.MCPs, r.opt.EnableHumanInTheLoop)
	if err != nil {
		return fmt.Errorf("failed to render config: %w", err)
	}
	if err := r.sb.WriteFile(ctx, ".opencode/opencode.json", []byte(content)); err != nil {
		return fmt.Errorf("failed to create opencode.json: %w", err)
	}

	return nil
}

// setupSkills 复制 skills 到工作目录，并注入变量到 SKILL.md。
func (r *Runner) setupSkills(ctx context.Context) error {
	if len(r.opt.Skills) == 0 {
		return nil
	}

	if _, err := r.sb.ExecuteSync(ctx, "mkdir", "-p", ".opencode/skills"); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	for _, skill := range r.opt.Skills {
		dirName := path.Base(skill.Dir)
		if err := r.sb.CopyToSandbox(ctx, skill.Dir, ".opencode/skills"); err != nil {
			return fmt.Errorf("failed to copy skill %s to workspace: %w", dirName, err)
		}

		// 追加变量信息到 SKILL.md
		if len(skill.Variables) > 0 {
			skillDir := ".opencode/skills/" + dirName
			skillPath := fmt.Sprintf("%s/SKILL.md", skillDir)
			varsContent := formatVariablesContent(skill.Variables)
			encoded := base64.StdEncoding.EncodeToString([]byte(varsContent))
			cmd := fmt.Sprintf("echo '%s' | base64 -d >> \"%s\"", encoded, skillPath)
			if _, err := r.sb.ExecuteSync(ctx, cmd); err != nil {
				return fmt.Errorf("failed to update SKILL.md for skill %s: %w", dirName, err)
			}
		}
	}

	return nil
}

// setupTools 转换 tools 为 skills 并复制到工作目录。
func (r *Runner) setupTools(ctx context.Context) error {
	if len(r.opt.Tools) == 0 {
		return nil
	}

	if _, err := r.sb.ExecuteSync(ctx, "mkdir", "-p", ".opencode/tools"); err != nil {
		return fmt.Errorf("failed to create tools directory: %w", err)
	}

	if _, err := r.sb.ExecuteSync(ctx, "mkdir", "-p", ".opencode/skills"); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	for _, tool := range r.opt.Tools {
		if err := r.setupTool(ctx, tool); err != nil {
			return err
		}
	}

	return nil
}

// setupTool 处理单个 tool。
func (r *Runner) setupTool(ctx context.Context, tool wga_sandbox_option.Tool) error {
	// 写入 OpenAPI schema 文件
	schemaData, err := json.Marshal(tool.OpenAPI3Schema)
	if err != nil {
		return fmt.Errorf("failed to marshal tool schema %s: %w", tool.Name, err)
	}

	dstFileName := fmt.Sprintf("%s.%s.json", toSkillName(tool.Name), uuid.New().String()[:8])
	dstPath := ".opencode/tools/" + dstFileName
	if err := r.sb.WriteFile(ctx, dstPath, schemaData); err != nil {
		return fmt.Errorf("failed to write tool schema %s: %w", tool.Name, err)
	}

	// 使用 openapi-to-skills 转换为 skill
	skillName := toSkillName(tool.Name)
	if _, err := r.sb.ExecuteSync(ctx, "openapi-to-skills", dstPath, "-o", ".opencode/skills", "-n", skillName, "-f"); err != nil {
		return fmt.Errorf("failed to convert tool %s to skill: %w", tool.Name, err)
	}

	// 追加 API 认证信息到 SKILL.md
	if tool.APIAuth != nil && tool.APIAuth.Type != "none" && tool.APIAuth.Value != "" {
		skillDir := ".opencode/skills/" + skillName
		skillPath := fmt.Sprintf("%s/SKILL.md", skillDir)
		authContent := formatAuthContent(tool.APIAuth)
		encoded := base64.StdEncoding.EncodeToString([]byte(authContent))
		cmd := fmt.Sprintf("echo '%s' | base64 -d >> \"%s\"", encoded, skillPath)
		if _, err := r.sb.ExecuteSync(ctx, cmd); err != nil {
			return fmt.Errorf("failed to update SKILL.md for tool %s: %w", tool.Name, err)
		}
	}

	return nil
}

// copyOutput 复制输出文件到本地，并移除隐藏文件。
func (r *Runner) copyOutput(ctx context.Context) error {
	if err := r.sb.CopyFromSandbox(ctx, r.opt.OutputDir); err != nil {
		return fmt.Errorf("failed to copy output from workspace: %w", err)
	}

	entries, err := os.ReadDir(r.opt.OutputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory: %w", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			removePath := r.opt.OutputDir + "/" + entry.Name()
			if err := os.RemoveAll(removePath); err != nil {
				return fmt.Errorf("failed to remove hidden file %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// ============================================================================
// Session 管理
// ============================================================================

// createSession 通过 API 创建 opencode session。
func (r *Runner) createSession(ctx context.Context) error {
	var result struct {
		ID string `json:"id"`
	}
	resp, err := resty.New().R().
		SetContext(ctx).
		SetQueryParam("directory", r.sb.WorkDir()).
		SetBody(map[string]any{}).
		SetResult(&result).
		Post(r.opt.Sandbox.OpencodeEndpoint() + "/session")
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	if resp.StatusCode() >= 300 {
		return fmt.Errorf("create session failed: [%d] %s", resp.StatusCode(), resp.String())
	}
	r.sessionID = result.ID
	return nil
}

// deleteSession 通过 API 删除 opencode session。
func (r *Runner) deleteSession(ctx context.Context) {
	if r.sessionID == "" {
		return
	}
	resp, err := resty.New().R().
		SetContext(ctx).
		SetQueryParam("directory", r.sb.WorkDir()).
		Delete(fmt.Sprintf("%s/session/%s", r.opt.Sandbox.OpencodeEndpoint(), r.sessionID))
	if err != nil {
		log.Warnf("%s failed to delete session %s: %v", r.logPrefix, r.sessionID, err)
		return
	}
	if resp.StatusCode() >= 300 {
		log.Warnf("%s delete session %s failed: [%d] %s", r.logPrefix, r.sessionID, resp.StatusCode(), resp.String())
	}
}

// ============================================================================
// SSE 连接
// ============================================================================

// connectSSE 连接到 opencode 全局事件流。
func (r *Runner) connectSSE(ctx context.Context) (<-chan string, error) {
	sseCh := make(chan string, 1024)
	errCh := make(chan error, 1)
	connected := make(chan struct{})

	go func() {
		defer util.PrintPanicStack()
		defer close(sseCh)
		defer close(errCh)

		resp, err := resty.New().
			SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
			SetTimeout(0).
			R().
			SetContext(ctx).
			SetHeader("Accept", "text/event-stream").
			SetDoNotParseResponse(true).
			Get(r.opt.Sandbox.OpencodeEndpoint() + "/global/event")
		if err != nil {
			errCh <- fmt.Errorf("SSE connect failed: %w", err)
			return
		}
		defer func() {
			if resp != nil && resp.RawResponse != nil {
				_ = resp.RawResponse.Body.Close()
			}
		}()

		// context 已取消，直接返回
		select {
		case <-ctx.Done():
			return
		default:
		}

		if resp.StatusCode() >= 300 {
			b, _ := io.ReadAll(resp.RawResponse.Body)
			errCh <- fmt.Errorf("SSE connect failed: [%d] %s", resp.StatusCode(), string(b))
			return
		}

		// 连接成功，通知主 goroutine
		close(connected)
		r.readSSEStream(resp.RawResponse.Body, sseCh, ctx)
	}()

	select {
	case err := <-errCh:
		return nil, err
	case <-connected:
		return sseCh, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// readSSEStream 读取 SSE 流，提取 data 字段并发送到通道。
func (r *Runner) readSSEStream(body io.ReadCloser, sseCh chan<- string, ctx context.Context) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB buffer
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			select {
			case sseCh <- data:
			case <-ctx.Done():
				return
			}
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF && err != context.Canceled {
		log.Warnf("%s SSE stream error: %v", r.logPrefix, err)
	}
}

// processSSEEvents 处理 SSE 事件流，过滤并转换为 JSON 输出。
func (r *Runner) processSSEEvents(ctx context.Context, sseCh <-chan string, outputCh chan<- string) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-sseCh:
			if !ok {
				return
			}
			line, isIdle := r.handleEvent(data)
			if line != "" {
				select {
				case outputCh <- line:
				case <-ctx.Done():
					return
				}
			}
			if isIdle {
				return
			}
		}
	}
}

// ============================================================================
// 提示词
// ============================================================================

// sendPromptAsync 异步发送提示词到 opencode session。
func (r *Runner) sendPromptAsync(ctx context.Context) error {
	system, prompt, err := r.buildSystemAndPrompt()
	if err != nil {
		return fmt.Errorf("failed to build system and prompt: %w", err)
	}

	reqBody := map[string]any{
		"parts": []map[string]any{
			{"type": "text", "text": prompt},
		},
	}
	if system != "" {
		reqBody["system"] = system
	}

	resp, err := resty.New().R().
		SetContext(ctx).
		SetQueryParam("directory", r.sb.WorkDir()).
		SetBody(reqBody).
		Post(fmt.Sprintf("%s/session/%s/prompt_async", r.opt.Sandbox.OpencodeEndpoint(), r.sessionID))
	if err != nil {
		return fmt.Errorf("failed to send prompt: %w", err)
	}
	if resp.StatusCode() >= 300 && resp.StatusCode() != 204 {
		return fmt.Errorf("send prompt failed: [%d] %s", resp.StatusCode(), resp.String())
	}
	return nil
}

// buildSystemAndPrompt 构建系统提示词和用户提示词。
// historyMessages: 倒序找到第一条 role=user 之前的所有消息
// prompt: 倒序找到第一条 role=user 消息，将这条及之后的所有消息拼接成 prompt
func (r *Runner) buildSystemAndPrompt() (system string, prompt string, err error) {
	// 从后往前找到第一条 role=user 的消息
	firstUserIndex := -1
	for i := len(r.opt.Messages) - 1; i >= 0; i-- {
		if r.opt.Messages[i].Role == schema.User {
			firstUserIndex = i
			break
		}
	}

	var historyMessages []adk.Message
	var promptMessages []adk.Message

	if firstUserIndex >= 0 {
		// historyMessages: 第一条 user 之前的消息
		if firstUserIndex > 0 {
			historyMessages = r.opt.Messages[:firstUserIndex]
		}
		// promptMessages: 第一条 user 及之后的消息
		promptMessages = r.opt.Messages[firstUserIndex:]
	} else {
		// 没有 user 消息，全部作为 prompt
		promptMessages = r.opt.Messages
	}

	system, err = renderSystem(r.opt.Instruction, r.opt.OverallTask, historyMessages)
	if err != nil {
		return "", "", err
	}

	// 拼接 promptMessages
	for i, msg := range promptMessages {
		if i > 0 {
			prompt += "\n\n"
		}
		if len(msg.UserInputMultiContent) > 0 {
			b, _ := json.Marshal(msg.UserInputMultiContent)
			prompt += string(b)
		} else {
			prompt += msg.Content
		}
	}

	return system, prompt, nil
}

// ============================================================================
// 事件转换
// ============================================================================

// handleEvent 处理 SSE 事件，返回 (输出内容, 是否会话结束)。
func (r *Runner) handleEvent(data string) (string, bool) {
	var event sseEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return "", false
	}

	// 过滤其他沙箱的事件
	if event.Directory != r.sb.WorkDir() {
		return "", false
	}

	// 区分 BusEvent 和 SyncEvent
	if event.Payload.Type == "sync" && event.Payload.SyncEvent != nil {
		return r.handleSyncEvent(&event)
	}

	return r.handleBusEvent(&event)
}

// handleBusEvent 处理 BusEvent 类型的事件（对应 opencode/src/bus/bus-event.ts）。
// BusEvent 用于实时增量推送，包括：
// - message.part.delta: 流式文本/推理增量
// - session.idle: 会话空闲
// - session.error: 会话错误
func (r *Runner) handleBusEvent(event *sseEvent) (string, bool) {
	switch event.Payload.Type {
	case "message.part.delta":
		// 流式增量事件（对应 MessageV2.Event.PartDelta）
		if event.Payload.Properties.SessionID != r.sessionID {
			return "", false
		}
		return r.convertDeltaEvent(event), false
	case "session.idle":
		// 会话空闲事件（对应 SessionStatus.Event.Idle）
		if event.Payload.Properties.SessionID != r.sessionID {
			return "", false
		}
		return "", true
	case "session.error":
		// 会话错误事件（对应 Session.Event.Error）
		if event.Payload.Properties.SessionID != r.sessionID {
			return "", false
		}
		return r.convertErrorEvent(event), false
	case string(OpencodeEventTypeQuestionAsked):
		// 问题提出事件（Human-in-the-Loop）
		return r.convertQuestionEvent(event, OpencodeEventTypeQuestionAsked, "pending"), false
	case string(OpencodeEventTypeQuestionReplied):
		// 问题已回答事件（Human-in-the-Loop）
		return r.convertQuestionEvent(event, OpencodeEventTypeQuestionReplied, "answered"), false
	case string(OpencodeEventTypeQuestionRejected):
		// 问题被拒绝事件（Human-in-the-Loop）
		return r.convertQuestionEvent(event, OpencodeEventTypeQuestionRejected, "rejected"), false
	default:
		return "", false
	}
}

// handleSyncEvent 处理 SyncEvent 类型的事件（对应 opencode/src/sync/index.ts）。
// SyncEvent 用于状态变更通知，包括：
// - message.updated.1: 消息创建/更新
// - message.part.updated.1: Part 状态更新
func (r *Runner) handleSyncEvent(event *sseEvent) (string, bool) {
	syncEvent := event.Payload.SyncEvent
	if syncEvent == nil {
		return "", false
	}

	switch syncEvent.Type {
	case "message.updated.1":
		// 消息更新事件（对应 MessageV2.Event.Updated）
		r.trackUserMessageIDFromSyncEvent(syncEvent)
		return "", false
	case "message.part.updated.1":
		// Part 更新事件（对应 MessageV2.Event.PartUpdated）
		if syncEvent.Data.SessionID != r.sessionID {
			return "", false
		}
		return r.convertMessagePartEventFromSync(syncEvent), false
	default:
		return "", false
	}
}

// trackUserMessageIDFromSyncEvent 从 message.updated 事件中记录用户消息 ID。
// 用于过滤用户消息的 delta 输出（用户消息不需要输出）。
func (r *Runner) trackUserMessageIDFromSyncEvent(syncEvent *sseSyncEvent) {
	info := syncEvent.Data.Info
	if info != nil && info.Role == "user" && info.ID != "" {
		r.userMsgIDs[info.ID] = true
	}
}

// convertDeltaEvent 转换 message.part.delta 事件（对应 MessageV2.Event.PartDelta）。
// delta 事件用于流式传输文本和推理内容的增量，只携带 partID 而不携带 part 类型，
// 需要通过 partTypeTracker 查找类型（从 message.part.updated 事件中记录）。
func (r *Runner) convertDeltaEvent(event *sseEvent) string {
	props := event.Payload.Properties
	delta := props.Delta
	if delta == "" {
		return ""
	}

	// 通过 messageID 判断是否为用户消息的 delta（用户消息不需要输出）
	if r.userMsgIDs[props.MessageID] {
		return ""
	}

	// 根据 part 类型判断是 text 还是 reasoning
	partType := r.partTypeTracker.get(props.PartID)
	switch partType {
	case "reasoning":
		if !r.opt.EnableThinking {
			return ""
		}
		return r.buildReasoningEvent(delta)
	default:
		// 未知 part 类型或 text 类型，作为 text 处理
		return r.buildTextEvent(delta)
	}
}

// convertMessagePartEventFromSync 转换 message.part.updated 事件（对应 MessageV2.Event.PartUpdated）。
// PartUpdated 事件用于通知 Part 状态变更，包括：
// - step-start/step-finish: 步骤标记
// - text/reasoning: 跟踪 part 类型供 delta 事件使用
// - tool: 工具调用状态变更
func (r *Runner) convertMessagePartEventFromSync(syncEvent *sseSyncEvent) string {
	part := syncEvent.Data.Part
	if part == nil {
		return ""
	}

	if r.userMsgIDs[part.MessageID] {
		return ""
	}

	switch part.Type {
	case "step-start":
		// 跟踪当前 step 的 part 列表
		r.partTypeTracker.stepStart(part.ID)
		return r.convertStepStartEvent(part)
	case "step-finish":
		r.partTypeTracker.stepFinish()
		return r.convertStepFinishEvent(part)
	case "text":
		// 跟踪 part 类型，供 delta 事件使用
		r.partTypeTracker.set(part.ID, "text")
		// SyncEvent 中 text part.updated 不携带增量文本
		return ""
	case "reasoning":
		// 跟踪 part 类型，供 delta 事件使用
		r.partTypeTracker.set(part.ID, "reasoning")
		return ""
	case "tool":
		return r.convertToolEvent(part)
	default:
		return ""
	}
}

// convertStepStartEvent 转换步骤开始事件。
func (r *Runner) convertStepStartEvent(part *sseEventPart) string {
	event := OpencodeEvent{
		Type:      OpencodeEventTypeStepStart,
		Timestamp: time.Now().UnixMilli(),
		SessionID: r.sessionID,
	}
	stepP := stepStartPart{Type: "step_start"}
	event.Part, _ = json.Marshal(stepP)

	data, _ := json.Marshal(event)
	return string(data)
}

// convertStepFinishEvent 转换步骤结束事件。
func (r *Runner) convertStepFinishEvent(part *sseEventPart) string {
	event := OpencodeEvent{
		Type:      OpencodeEventTypeStepFinish,
		Timestamp: time.Now().UnixMilli(),
		SessionID: r.sessionID,
	}
	finishP := stepFinishPart{
		Type:   "step_finish",
		Reason: part.Reason,
		Tokens: stepFinishPartTokens{
			Input:     part.Tokens.Input,
			Output:    part.Tokens.Output,
			Reasoning: part.Tokens.Reasoning,
			Cache: struct {
				Read  float64 `json:"read,omitempty"`
				Write float64 `json:"write,omitempty"`
			}{
				Read:  part.Tokens.Cache.Read,
				Write: part.Tokens.Cache.Write,
			},
		},
		Cost: part.Cost,
	}
	event.Part, _ = json.Marshal(finishP)

	data, _ := json.Marshal(event)
	return string(data)
}

// buildTextEvent 构建文本增量事件。
func (r *Runner) buildTextEvent(delta string) string {
	event := OpencodeEvent{
		Type:      OpencodeEventTypeText,
		Timestamp: time.Now().UnixMilli(),
		SessionID: r.sessionID,
	}
	textP := textPart{Type: "text", Text: delta}
	event.Part, _ = json.Marshal(textP)

	data, _ := json.Marshal(event)
	return string(data)
}

// buildReasoningEvent 构建推理增量事件。
func (r *Runner) buildReasoningEvent(delta string) string {
	event := OpencodeEvent{
		Type:      OpencodeEventTypeReasoning,
		Timestamp: time.Now().UnixMilli(),
		SessionID: r.sessionID,
	}
	reasoningP := reasoningPart{Type: "reasoning", Text: delta}
	event.Part, _ = json.Marshal(reasoningP)

	data, _ := json.Marshal(event)
	return string(data)
}

// convertToolEvent 转换工具调用事件。
// 只发送 completed 或 error 状态的事件。
func (r *Runner) convertToolEvent(part *sseEventPart) string {
	if part.State.Status != "completed" && part.State.Status != "error" {
		return ""
	}

	callID := part.CallID
	if callID == "" {
		callID = part.ID
	}

	event := OpencodeEvent{
		Type:      OpencodeEventTypeToolUse,
		Timestamp: time.Now().UnixMilli(),
		SessionID: r.sessionID,
	}

	toolP := toolPart{
		Type:   "tool_use",
		CallID: callID,
		Tool:   part.Tool,
		State: toolState{
			Status: part.State.Status,
			Input:  part.State.Input,
			Output: part.State.Output,
			Error:  part.State.Error,
		},
	}
	event.Part, _ = json.Marshal(toolP)

	data, _ := json.Marshal(event)
	return string(data)
}

// convertErrorEvent 转换错误事件。
func (r *Runner) convertErrorEvent(event *sseEvent) string {
	errInfo := event.Payload.Properties.Error
	evt := OpencodeEvent{
		Type:      OpencodeEventTypeError,
		Timestamp: time.Now().UnixMilli(),
		SessionID: r.sessionID,
	}
	errorP := errorPart{}
	errorP.Error.Name = errInfo.Name
	errorP.Error.Data.Message = errInfo.Data.Message
	evt.Part, _ = json.Marshal(errorP)

	data, _ := json.Marshal(evt)
	return string(data)
}

// convertQuestionEvent 转换问题事件（Human-in-the-Loop）。
func (r *Runner) convertQuestionEvent(event *sseEvent, eventType OpencodeEventType, status string) string {
	evt := OpencodeEvent{
		Type:      eventType,
		Timestamp: time.Now().UnixMilli(),
		SessionID: r.sessionID,
	}

	props := event.Payload.Properties
	questionID := props.ID
	if questionID == "" {
		questionID = props.RequestID
	}

	questionP := questionPart{
		Type:       "question",
		QuestionID: questionID,
		SessionID:  props.SessionID,
		Status:     status,
		Questions:  make([]questionItem, 0, len(props.Questions)),
	}

	for _, q := range props.Questions {
		custom := r.opt.EnableHumanInTheLoopCustom
		if q.Custom != nil {
			custom = *q.Custom
		}
		item := questionItem{
			Question: q.Question,
			Header:   q.Header,
			Multiple: q.Multiple,
			Custom:   custom,
			Options:  make([]questionOption, 0, len(q.Options)),
		}
		for _, opt := range q.Options {
			item.Options = append(item.Options, questionOption(opt))
		}
		questionP.Questions = append(questionP.Questions, item)
	}

	if len(props.Answers) > 0 {
		questionP.Answers = props.Answers
	}

	evt.Part, _ = json.Marshal(questionP)
	data, _ := json.Marshal(evt)

	return string(data)
}

// ============================================================================
// 模板渲染
// ============================================================================

// renderConfig 渲染 opencode 配置文件。
func renderConfig(config wga_sandbox_option.ModelConfig, mcps []wga_sandbox_option.MCP, enableHITL bool) (string, error) {
	tmpl, err := template.New("config").Funcs(template.FuncMap{
		"sub": func(a, b int) int { return a - b },
		"len": func(v interface{}) int {
			switch s := v.(type) {
			case []wga_sandbox_option.MCP:
				return len(s)
			}
			return 0
		},
	}).Parse(configTemplate)
	if err != nil {
		return "", fmt.Errorf("parse config template failed: %w", err)
	}
	data := struct {
		wga_sandbox_option.ModelConfig
		MCPs                 []wga_sandbox_option.MCP
		EnableHumanInTheLoop bool
	}{
		ModelConfig:          config,
		MCPs:                 mcps,
		EnableHumanInTheLoop: enableHITL,
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute config template failed: %w", err)
	}
	return buf.String(), nil
}

// renderSystem 渲染系统提示词模板。
func renderSystem(instruction, overallTask string, messages []adk.Message) (string, error) {
	tmpl, err := template.New("system").Funcs(template.FuncMap{
		"formatPart": formatMessageInputPart,
		"listParts":  listMessageInputParts,
	}).Parse(systemTemplate)
	if err != nil {
		return "", fmt.Errorf("parse system template failed: %w", err)
	}

	data := struct {
		Instruction string
		OverallTask string
		Messages    []adk.Message
	}{
		Instruction: instruction,
		OverallTask: overallTask,
		Messages:    messages,
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute system template failed: %w", err)
	}
	return buf.String(), nil
}

// listMessageInputParts 将消息部分列表格式化为字符串列表，过滤空结果。
func listMessageInputParts(parts []schema.MessageInputPart) []string {
	var result []string
	for _, part := range parts {
		if s := formatMessageInputPart(part); s != "" {
			result = append(result, s)
		}
	}
	return result
}

// formatMessageInputPart 格式化消息部分为文本表示。
// text: 直接输出文本内容
// image_url/audio_url/video_url/file_url: 输出 URL
func formatMessageInputPart(part schema.MessageInputPart) string {
	switch part.Type {
	case schema.ChatMessagePartTypeText:
		return part.Text
	case schema.ChatMessagePartTypeImageURL:
		if part.Image != nil && part.Image.URL != nil {
			return fmt.Sprintf("[image](%s)", *part.Image.URL)
		}
	case schema.ChatMessagePartTypeAudioURL:
		if part.Audio != nil && part.Audio.URL != nil {
			return fmt.Sprintf("[audio](%s)", *part.Audio.URL)
		}
	case schema.ChatMessagePartTypeVideoURL:
		if part.Video != nil && part.Video.URL != nil {
			return fmt.Sprintf("[video](%s)", *part.Video.URL)
		}
	case schema.ChatMessagePartTypeFileURL:
		if part.File != nil && part.File.URL != nil {
			name := "file"
			if part.File.Name != "" {
				name = part.File.Name
			}
			return fmt.Sprintf("[%s](%s)", name, *part.File.URL)
		}
	}
	return ""
}

// ============================================================================
// 工具函数
// ============================================================================

// toSkillName 将工具名称转换为 skill 名称，替换空格为连字符，移除括号。
func toSkillName(name string) string {
	result := strings.ReplaceAll(name, " ", "-")
	result = strings.ReplaceAll(result, "(", "")
	result = strings.ReplaceAll(result, ")", "")
	return result
}

// formatAuthContent 格式化认证信息为 Markdown 格式。
func formatAuthContent(auth *openapi3_util.Auth) string {
	if auth.Type == "none" || auth.Value == "" {
		return ""
	}
	var authDesc string
	switch auth.In {
	case "header":
		authDesc = fmt.Sprintf("Header: `%s: %s`", auth.Name, auth.Value)
	case "query":
		authDesc = fmt.Sprintf("Query Parameter: `%s=%s`", auth.Name, auth.Value)
	default:
		authDesc = fmt.Sprintf("Auth Value: `%s`", auth.Value)
	}
	return fmt.Sprintf("\n## API Key\n\n%s\n", authDesc)
}

// formatVariablesContent 格式化变量信息为 Markdown 格式。
func formatVariablesContent(variables []wga_sandbox_option.SkillVariable) string {
	var buf strings.Builder
	buf.WriteString("\n## Variables\n\n")
	buf.WriteString("The following variables are configured for this skill:\n\n")
	for _, v := range variables {
		// 转义反引号
		escapedKey := strings.ReplaceAll(v.VariableKey, "`", "\\`")
		escapedValue := strings.ReplaceAll(v.VariableValue, "`", "\\`")

		if v.Description != "" {
			escapedDesc := strings.ReplaceAll(v.Description, "`", "\\`")
			fmt.Fprintf(&buf, "- **%s** (%s): `%s` = `%s`\n",
				v.Name, escapedDesc, escapedKey, escapedValue)
		} else {
			fmt.Fprintf(&buf, "- **%s**: `%s` = `%s`\n",
				v.Name, escapedKey, escapedValue)
		}
	}
	buf.WriteString("\n")
	return buf.String()
}
