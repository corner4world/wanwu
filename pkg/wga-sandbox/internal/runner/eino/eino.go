// Package eino 提供 eino-agent 智能体的运行器实现（基于 HTTP API）
package eino

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/UnicomAI/wanwu/pkg/wga-sandbox/internal/runner"
	"github.com/UnicomAI/wanwu/pkg/wga-sandbox/internal/sandbox"
	wga_sandbox_option "github.com/UnicomAI/wanwu/pkg/wga-sandbox/wga-sandbox-option"
	"github.com/go-resty/resty/v2"
)

// 确保 Runner 实现 runner.Runner 接口
var _ runner.Runner = (*Runner)(nil)

// 实现 eino-agent 智能体运行器
type Runner struct {
	sb        sandbox.Sandbox
	req       wga_sandbox_option.RunOption
	agentType string
	logPrefix string
}

// 创建 eino-agent 运行器实例
func NewRunner(sb sandbox.Sandbox, req wga_sandbox_option.RunOption, agentType string) runner.Runner {
	logPrefix := fmt.Sprintf("[wga-sandbox][%s]", req.RunSession.RunID)
	return &Runner{
		sb:        sb,
		req:       req,
		agentType: agentType,
		logPrefix: logPrefix,
	}
}

// BeforeRun 执行前准备工作：
// 1. 创建 .env 配置文件
// 2. 复制 skills 到 skills 目录
// 3. 复制输入文件到 input 目录
func (r *Runner) BeforeRun(ctx context.Context) error {
	log.Infof("%s BeforeRun - req.Skills count: %d", r.logPrefix, len(r.req.Skills))
	for i, skill := range r.req.Skills {
		log.Infof("%s BeforeRun - skill[%d]: Dir=%s", r.logPrefix, i, skill.Dir)
	}
	log.Infof("%s BeforeRun - req.InputDir: %s", r.logPrefix, r.req.InputDir)
	log.Infof("%s BeforeRun - req.OutputDir: %s", r.logPrefix, r.req.OutputDir)
	log.Infof("%s BeforeRun - sandbox.WorkDir: %s", r.logPrefix, r.sb.WorkDir())

	if err := r.setupEnv(ctx); err != nil {
		return err
	}

	// 创建 skills 目录
	if _, err := r.sb.ExecuteSync(ctx, "mkdir", "-p", "skills"); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// 复制 skills （eino-agent HTTP 服务从 workspace/skills/ 加载技能）
	if len(r.req.Skills) > 0 {
		for _, skill := range r.req.Skills {
			log.Infof("%s Copying skill from %s to skills/", r.logPrefix, skill.Dir)
			if err := r.sb.CopyToSandbox(ctx, skill.Dir, "skills"); err != nil {
				return fmt.Errorf("failed to copy skill to workspace: %w", err)
			}
			log.Infof("%s Successfully copied skill dir=%s", r.logPrefix, skill.Dir)
		}
	}

	// 复制输入文件到 input 目录
	if r.req.InputDir != "" {
		if err := r.sb.CopyToSandbox(ctx, r.req.InputDir); err != nil {
			return fmt.Errorf("failed to copy input to workspace: %w", err)
		}
	}

	if _, err := r.sb.ExecuteSync(ctx, "mkdir", "-p", "output"); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if _, err := r.sb.ExecuteSync(ctx, "mkdir", "-p", "tmp"); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	return nil
}

// 执行 eino-agent 任务，通过 HTTP API 调用，返回 SSE 事件流
func (r *Runner) Run(ctx context.Context) (<-chan string, error) {
	sseCh, err := r.connectSSE(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect SSE: %w", err)
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
// 复制输出文件到本地（如果指定了 OutputDir）
func (r *Runner) AfterRun(ctx context.Context) error {
	log.Infof("%s AfterRun start, OutputDir: %s", r.logPrefix, r.req.OutputDir)

	if r.req.OutputDir != "" {
		err := r.copyOutput(ctx)
		if err != nil {
			log.Errorf("%s AfterRun failed: %v", r.logPrefix, err)
			return err
		}
		log.Infof("%s AfterRun completed", r.logPrefix)
		return nil
	}

	log.Infof("%s AfterRun skipped (no OutputDir)", r.logPrefix)
	return nil
}

// setupEnv 创建 .env 文件，供 eino-agent 读取模型配置
func (r *Runner) setupEnv(ctx context.Context) error {
	var lines []string
	if r.req.ModelConfig.APIKey != "" {
		lines = append(lines, fmt.Sprintf("OPENAI_API_KEY=%s", r.req.ModelConfig.APIKey))
	}
	if r.req.ModelConfig.BaseURL != "" {
		lines = append(lines, fmt.Sprintf("OPENAI_BASE_URL=%s", r.req.ModelConfig.BaseURL))
	}
	if r.req.ModelConfig.Model != "" {
		lines = append(lines, fmt.Sprintf("OPENAI_MODEL_ID=%s", r.req.ModelConfig.Model))
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := r.sb.WriteFile(ctx, ".env", []byte(content)); err != nil {
		return fmt.Errorf("failed to create .env: %w", err)
	}
	log.Infof("%s .env file created in sandbox workspace", r.logPrefix)
	return nil
}

// 复制输出文件到本地，并移除隐藏文件
func (r *Runner) copyOutput(ctx context.Context) error {
	log.Infof("%s copyOutput start", r.logPrefix)

	// 从沙箱复制输出文件
	if err := r.sb.CopyFromSandbox(ctx, r.req.OutputDir); err != nil {
		log.Errorf("%s copyOutput CopyFromSandbox failed: %v", r.logPrefix, err)
		return fmt.Errorf("failed to copy output from workspace: %w", err)
	}

	// 读取 outputDir 内容
	entries, err := os.ReadDir(r.req.OutputDir)
	if err != nil {
		log.Errorf("%s copyOutput ReadDir failed: %v", r.logPrefix, err)
		return fmt.Errorf("failed to read output directory: %w", err)
	}

	// 处理每个条目
	for _, entry := range entries {
		entryPath := filepath.Join(r.req.OutputDir, entry.Name())

		// 删除隐藏文件
		if strings.HasPrefix(entry.Name(), ".") {
			log.Infof("%s copyOutput removing hidden file: %s", r.logPrefix, entry.Name())
			if err := os.RemoveAll(entryPath); err != nil {
				log.Errorf("%s copyOutput remove hidden file failed: %s, err: %v", r.logPrefix, entry.Name(), err)
				return fmt.Errorf("failed to remove hidden file %s: %w", entry.Name(), err)
			}
			continue
		}

		// 删除 skills 目录
		if entry.Name() == "skills" && entry.IsDir() {
			log.Infof("%s copyOutput removing skills dir", r.logPrefix)
			if err := os.RemoveAll(entryPath); err != nil {
				log.Errorf("%s copyOutput remove skills dir failed: %v", r.logPrefix, err)
				return fmt.Errorf("failed to remove skills directory: %w", err)
			}
			continue
		}

		// 删除 input 目录
		if entry.Name() == "input" && entry.IsDir() {
			log.Infof("%s copyOutput removing input dir", r.logPrefix)
			if err := os.RemoveAll(entryPath); err != nil {
				log.Errorf("%s copyOutput remove input dir failed: %v", r.logPrefix, err)
				return fmt.Errorf("failed to remove input directory: %w", err)
			}
			continue
		}

		// 删除 tmp 目录
		if entry.Name() == "tmp" && entry.IsDir() {
			log.Infof("%s copyOutput removing tmp dir", r.logPrefix)
			if err := os.RemoveAll(entryPath); err != nil {
				log.Errorf("%s copyOutput remove tmp dir failed: %v", r.logPrefix, err)
				return fmt.Errorf("failed to remove tmp directory: %w", err)
			}
			continue
		}

		// 将 output 子目录内容提升到 outputDir 根目录
		if entry.Name() == "output" && entry.IsDir() {
			log.Infof("%s copyOutput flattening output subdir", r.logPrefix)
			if err := flattenDir(entryPath, r.req.OutputDir); err != nil {
				log.Errorf("%s copyOutput flatten failed: %v", r.logPrefix, err)
				return fmt.Errorf("failed to flatten output directory: %w", err)
			}
			continue
		}
	}

	log.Infof("%s copyOutput completed", r.logPrefix)
	return nil
}

// flattenDir 将 src 目录中的所有内容移动到 dst 目录，然后删除空的 src 目录
func flattenDir(src, dst string) error {
	log.Infof("[flattenDir] start, src: %s, dst: %s", src, dst)

	subEntries, err := os.ReadDir(src)
	if err != nil {
		log.Errorf("[flattenDir] ReadDir failed: %v", err)
		return fmt.Errorf("failed to read dir %s: %w", src, err)
	}

	log.Infof("[flattenDir] moving %d items", len(subEntries))

	for _, sub := range subEntries {
		srcPath := filepath.Join(src, sub.Name())
		dstPath := filepath.Join(dst, sub.Name())

		if err := os.Rename(srcPath, dstPath); err != nil {
			log.Errorf("[flattenDir] move failed: %s, err: %v", sub.Name(), err)
			return fmt.Errorf("failed to move %s: %w", sub.Name(), err)
		}
	}

	if err := os.Remove(src); err != nil {
		log.Errorf("[flattenDir] remove src dir failed: %v", err)
		return err
	}

	log.Infof("[flattenDir] completed")
	return nil
}

// 连接到 eino-agent HTTP 服务的 /chat 端点
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
			SetQueryParam("workspace", r.sb.WorkDir()).
			SetQueryParam("agent_type", r.agentType).
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "text/event-stream").
			SetBody(map[string]interface{}{"messages": r.req.Messages}).
			SetDoNotParseResponse(true).
			Post(r.req.Sandbox.EinoEndpoint() + "/chat")

		if err != nil {
			errCh <- fmt.Errorf("SSE connect failed: %w", err)
			return
		}
		defer func() {
			if resp != nil && resp.RawResponse != nil && resp.RawResponse.Body != nil {
				_ = resp.RawResponse.Body.Close()
			}
		}()

		if resp == nil || resp.RawResponse == nil {
			errCh <- fmt.Errorf("SSE connect failed: empty response")
			return
		}

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

func (r *Runner) readSSEStream(body io.ReadCloser, sseCh chan<- string, ctx context.Context) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
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

func (r *Runner) processSSEEvents(ctx context.Context, sseCh <-chan string, outputCh chan<- string) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-sseCh:
			if !ok {
				return
			}
			if line := r.convertEvent(data); line != "" {
				select {
				case outputCh <- line:
				case <-ctx.Done():
					return
				}
			}
			if r.isDoneEvent(data) {
				return
			}
		}
	}
}

func (r *Runner) convertEvent(data string) string {
	var event struct {
		Role string `json:"role"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return ""
	}

	if event.Role == "done" || event.Role == "error" {
		return ""
	}

	return data
}

func (r *Runner) isDoneEvent(data string) bool {
	var event struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return false
	}
	return event.Role == "done"
}
