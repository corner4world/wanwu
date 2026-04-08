package shared

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk/filesystem"
)

const (
	maxOutputSize  = 1024 * 1024 // 1MB
	commandTimeout = 5 * time.Minute
)

// ShellOnlyBackend 实现 filesystem.Backend 接口，
// 仅提供 shell 命令执行能力，文件操作方法返回 not implemented。
type ShellOnlyBackend struct {
	maxOutputSize  int
	commandTimeout time.Duration
	workDir        string
}

func NewShellOnlyBackend(workDir string) *ShellOnlyBackend {
	return &ShellOnlyBackend{
		maxOutputSize:  maxOutputSize,
		commandTimeout: commandTimeout,
		workDir:        workDir,
	}
}

// --- filesystem.Backend 空实现 ---
func (b *ShellOnlyBackend) LsInfo(_ context.Context, _ *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *ShellOnlyBackend) Read(_ context.Context, _ *filesystem.ReadRequest) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (b *ShellOnlyBackend) GrepRaw(_ context.Context, _ *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *ShellOnlyBackend) GlobInfo(_ context.Context, _ *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *ShellOnlyBackend) Write(_ context.Context, _ *filesystem.WriteRequest) error {
	return fmt.Errorf("not implemented")
}

func (b *ShellOnlyBackend) Edit(_ context.Context, _ *filesystem.EditRequest) error {
	return fmt.Errorf("not implemented")
}

// --- 安全规则 ---
var dangerousPatterns = []*regexp.Regexp{
	// 系统级破坏命令
	regexp.MustCompile(`(?i)\brm\s+(-[a-z]*f|-[a-z]*r|--force|--recursive)\b`),
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bdd\s+if=`),
	regexp.MustCompile(`(?i)\bchmod\s+777\b`),
	regexp.MustCompile(`(?i)\b(shutdown|reboot|init\s+[06])\b`),
	regexp.MustCompile(`(?i)>\s*/dev/sd`),
	regexp.MustCompile(`(?i)\bcurl\b.*\|\s*(ba)?sh`),
	regexp.MustCompile(`(?i)\bwget\b.*\|\s*(ba)?sh`),
	regexp.MustCompile(`(?i)\bnc\s+-[a-z]*l`),
	regexp.MustCompile(`(?i)/dev/(tcp|udp)/`),
	// 通用代码注入（仅拦截明显危险的）
	regexp.MustCompile(`(?i)\b__import__\s*\(\s*['"]os['"]\s*\)`),
}

var sensitivePathPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/etc/(passwd|shadow|sudoers|hosts)`),
	regexp.MustCompile(`/proc/`),
	regexp.MustCompile(`/sys/`),
	regexp.MustCompile(`/dev/`),
	regexp.MustCompile(`~/`),
	regexp.MustCompile(`/var/lib/(mysql|postgresql)`),
}

var allowedWorkspacePattern = regexp.MustCompile(`/home/root/workspace/[a-zA-Z0-9_-]+/workspace`)

func validateCommand(command string) error {
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(command) {
			matched := pattern.FindString(command)
			log.Printf("[安全拦截] 检测到危险操作: %s", matched)
			return fmt.Errorf("安全拦截：检测到高危命令片段 [%s]，已拒绝执行", matched)
		}
	}

	// 检查是否访问允许的工作目录
	if !allowedWorkspacePattern.MatchString(command) {
		// 不在允许的工作目录内，检查敏感路径
		for _, pattern := range sensitivePathPatterns {
			if matched := pattern.FindString(command); matched != "" {
				return fmt.Errorf("安全拦截：检测到对敏感路径 [%s] 的访问尝试，仅允许操作工作目录", matched)
			}
		}

		// 检查 /root/ 和 /home/ 路径
		if strings.Contains(command, "/root/") {
			return fmt.Errorf("安全拦截：检测到对敏感路径 [/root/] 的访问尝试，仅允许操作工作目录")
		}
		if matched := regexp.MustCompile(`/home/[^/]+`).FindString(command); matched != "" {
			return fmt.Errorf("安全拦截：检测到对敏感路径 [%s] 的访问尝试，仅允许操作工作目录", matched)
		}
	}

	if strings.Contains(command, "..") {
		pathTraversal := regexp.MustCompile(`\.\./`)
		if pathTraversal.MatchString(command) {
			return fmt.Errorf("安全拦截：检测到路径穿越尝试（../），已拒绝执行")
		}
	}

	symlinkPattern := regexp.MustCompile(`(?i)\bln\s+-[a-z]*s`)
	if symlinkPattern.MatchString(command) {
		for _, pattern := range sensitivePathPatterns {
			if matched := pattern.FindString(command); matched != "" {
				return fmt.Errorf("安全拦截：禁止创建指向敏感路径 [%s] 的符号链接", matched)
			}
		}
	}

	return nil
}

// var pythonCmdPattern = regexp.MustCompile(`(?i)(^|&&\s*|\|\|\s*|;\s*)(python3?|pip3?)\b`)
// var apkAddPyPattern = regexp.MustCompile(`(?i)\bapk\s+add\s+[^\n]*\bpy3?-`)

// // pip和python命令在Python虚拟环境中执行
// func wrapWithVenv(command, workDir string) string {
// 	venvDir := filepath.Join(workDir, ".venv")
// 	// 追加pip国内源配置（阿里云）
// 	pipConfigCmd := `pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
//                      pip config set global.trusted-host mirrors.aliyun.com`

// 	return fmt.Sprintf(
// 		`if [ ! -d "%s" ]; then
// 			[ ! -x "$(command -v python3)" ] && echo "ERROR: python3未安装" && exit 1
// 			if ! python3 -m venv "%s" >/dev/null 2>&1; then
// 				if command -v apt-get >/dev/null; then
// 					apt-get update -qq >/dev/null 2>&1 && \
// 					DEBIAN_FRONTEND=noninteractive apt-get install -y -qq python3-venv >/dev/null 2>&1;
// 				elif command -v apk >/dev/null; then
// 					apk add --no-cache python3-venv >/dev/null 2>&1;
// 				fi;
// 				python3 -m venv "%s" >/dev/null 2>&1 || { echo "ERROR: 创建虚拟环境失败"; exit 1; };
// 			fi;
// 		fi && . "%s/bin/activate" && %s && %s`,
// 		venvDir, venvDir, venvDir, venvDir, pipConfigCmd, command, // 先配置pip源，再执行原命令
// 	)
// }

// --- Execute ---

func (b *ShellOnlyBackend) Execute(ctx context.Context, req *filesystem.ExecuteRequest) (*filesystem.ExecuteResponse, error) {
	if err := validateCommand(req.Command); err != nil {
		exitCode := 1
		return &filesystem.ExecuteResponse{
			Output:   err.Error(),
			ExitCode: &exitCode,
		}, nil
	}

	command := req.Command

	// 不启用虚拟环境
	// if pythonCmdPattern.MatchString(command) {
	// 	command = wrapWithVenv(command, b.workDir)
	// 	log.Printf("[Execute] Python 命令已自动改写为虚拟环境执行: %s", command)
	// }

	// if apkAddPyPattern.MatchString(command) {
	// 	exitCode := 1
	// 	return &filesystem.ExecuteResponse{
	// 		Output:   "安全拦截：禁止通过 apk 安装 Python 依赖包（py3-*），请使用 pip install 安装。",
	// 		ExitCode: &exitCode,
	// 	}, nil
	// }

	execCtx, cancel := context.WithTimeout(ctx, b.commandTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(execCtx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(execCtx, "sh", "-c", command)
	}

	if b.workDir != "" {
		if absWorkDir, err := filepath.Abs(b.workDir); err == nil {
			cmd.Dir = absWorkDir
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if execCtx.Err() == context.DeadlineExceeded {
		exitCode := 124
		return &filesystem.ExecuteResponse{
			Output:   fmt.Sprintf("命令执行超时（限制 %v），已终止。请考虑拆分任务或优化命令。", b.commandTimeout),
			ExitCode: &exitCode,
		}, nil
	}

	output := stdout.String()
	if stderr.Len() > 0 {
		if len(output) > 0 {
			output += "\n"
		}
		output += stderr.String()
	}

	truncated := false
	if len(output) > b.maxOutputSize {
		output = output[:b.maxOutputSize]
		truncated = true
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr != nil {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, err
		}
	}

	return &filesystem.ExecuteResponse{
		Output:    output,
		ExitCode:  &exitCode,
		Truncated: truncated,
	}, nil
}
