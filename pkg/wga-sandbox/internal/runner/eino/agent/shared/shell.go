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

// ShellOnlyBackend 仅提供 shell 命令执行能力。
// 不实现 filesystem.Backend 的其他文件操作方法，因为 bash 工具只调用 Execute。
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

// --- 安全规则 ---
//
// 设计原则：
//   1. wga-sandbox 本身已是隔离容器，本层不再承担"防止一切危险字符串"的职责，
//      只防"破坏沙箱自身 + 泄露宿主敏感信息"。
//   2. 区分读/写：cat /proc/cpuinfo、cat /etc/hosts 这类只读操作不拦，只在
//      写入或删除时拦截敏感路径。
//   3. 高危且无合法替代的动作（mkfs、shutdown、写块设备、反向 shell 通道、
//      rm 系统目录）无条件拦截。

var dangerousPatterns = []*regexp.Regexp{
	// 无可逆/无合法用途的系统破坏命令
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\b(shutdown|reboot|halt|poweroff|init\s+[06])\b`),

	// 直接向块设备写入
	regexp.MustCompile(`(?i)\bdd\b[^|;&]*\bof=/dev/(sd|nvme|hd|mmcblk|loop|mapper)`),
	regexp.MustCompile(`(?i)>\s*/dev/(sd|nvme|hd|mmcblk)`),

	// bash 反向 shell 通道
	regexp.MustCompile(`/dev/(tcp|udp)/`),

	// 删除根目录或系统目录（工作目录内 rm -rf 放行）。
	// 注意：故意不列 /home —— 工作区前缀就是 /home/root/workspace/...，列了会误杀。
	// /root\b 是兜底；走到 sensitiveWritePatterns 的 /root/ + writeAction 也会再拦一次。
	regexp.MustCompile(`(?i)\brm\s+-[a-zA-Z]*[rRf][a-zA-Z]*\s+(/\s|/$|/\*|\$HOME\b|~/?\*?\s|~/?\*?$|/etc\b|/var\b|/usr\b|/bin\b|/sbin\b|/lib\b|/lib64\b|/boot\b|/root\b)`),

	// 明显的代码注入
	regexp.MustCompile(`(?i)\b__import__\s*\(\s*['"]os['"]\s*\)`),
}

// sensitiveReadPatterns：无论读写，命中即拦。
// 真正存放凭证/秘密的位置才进来。
var sensitiveReadPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/etc/(shadow|sudoers)\b`),
	regexp.MustCompile(`\.ssh/(id_rsa|id_ed25519|id_ecdsa|id_dsa)\b`),
	regexp.MustCompile(`\.aws/credentials\b`),
	regexp.MustCompile(`/var/lib/(mysql|postgresql)\b`),
	regexp.MustCompile(`/proc/\d+/(mem|maps|environ)\b`),
	regexp.MustCompile(`/proc/sys/kernel/`),
}

// sensitiveWritePatterns：仅当紧邻写动作（>, >>, tee, rm, mv, cp, chmod, chown）时拦截。
var sensitiveWritePatterns = []*regexp.Regexp{
	regexp.MustCompile(`/etc/`),
	regexp.MustCompile(`/proc/`),
	regexp.MustCompile(`/sys/`),
	regexp.MustCompile(`/var/lib/`),
	regexp.MustCompile(`/boot/`),
	regexp.MustCompile(`/root/`),
}

// writeActionPattern：命中点之前若以这些动作结尾，则视为"对该路径写"。
// 用 $ 锚点配合"取前缀子串"的方式判定。
var writeActionPattern = regexp.MustCompile(`(?i)(>\s*|>>\s*|\|\s*tee\s+(-[a-zA-Z]+\s+)*|\btee\s+(-[a-zA-Z]+\s+)*|\brm\s+(-[a-zA-Z]+\s+)*|\bmv\s+\S+\s+|\bcp\s+(-[a-zA-Z]+\s+)*\S+\s+|\bchmod\s+\S+\s+|\bchown\s+\S+\s+)$`)

var (
	pathTraversalPattern = regexp.MustCompile(`\.\./`)
	symlinkPattern       = regexp.MustCompile(`(?i)\bln\s+-[a-z]*s`)
)

// hasWriteActionBefore 判断 command[:idx] 末尾是否以"写动作"结束。
// lookback 限制回看窗口避免误判跨命令的远距离 token。
func hasWriteActionBefore(command string, idx int) bool {
	const lookback = 64
	start := max(idx-lookback, 0)
	prefix := command[start:idx]
	return writeActionPattern.MatchString(prefix)
}

func validateCommand(command string) error {
	// 1) 高危动作：无论上下文都拦
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(command) {
			matched := pattern.FindString(command)
			log.Printf("[安全拦截] 检测到危险操作: %s", matched)
			return fmt.Errorf("安全拦截：检测到高危命令片段 [%s]，已拒绝执行", matched)
		}
	}

	// 2) 读敏感路径：命中即拦
	for _, pattern := range sensitiveReadPatterns {
		if matched := pattern.FindString(command); matched != "" {
			log.Printf("[安全拦截] 检测到对敏感路径读取尝试: %s", matched)
			return fmt.Errorf("安全拦截：检测到对敏感路径 [%s] 的访问尝试，已拒绝执行", matched)
		}
	}

	// 3) 写敏感路径：仅当命中位置前紧邻写动作时拦截
	for _, pattern := range sensitiveWritePatterns {
		for _, loc := range pattern.FindAllStringIndex(command, -1) {
			if hasWriteActionBefore(command, loc[0]) {
				matched := command[loc[0]:loc[1]]
				log.Printf("[安全拦截] 检测到对敏感路径写入尝试: %s", matched)
				return fmt.Errorf("安全拦截：检测到对敏感路径 [%s] 的写入尝试，已拒绝执行", matched)
			}
		}
	}

	// 4) 路径穿越
	if strings.Contains(command, "..") && pathTraversalPattern.MatchString(command) {
		return fmt.Errorf("安全拦截：检测到路径穿越尝试（../），已拒绝执行")
	}

	// 5) 指向敏感路径的符号链接
	if symlinkPattern.MatchString(command) {
		for _, pattern := range sensitiveReadPatterns {
			if matched := pattern.FindString(command); matched != "" {
				return fmt.Errorf("安全拦截：禁止创建指向敏感路径 [%s] 的符号链接", matched)
			}
		}
		for _, pattern := range sensitiveWritePatterns {
			if matched := pattern.FindString(command); matched != "" {
				return fmt.Errorf("安全拦截：禁止创建指向敏感路径 [%s] 的符号链接", matched)
			}
		}
	}

	return nil
}

// Execute 执行 shell 命令，附带安全校验、超时、输出截断与退出码处理。
func (b *ShellOnlyBackend) Execute(ctx context.Context, req *filesystem.ExecuteRequest) (*filesystem.ExecuteResponse, error) {
	if err := validateCommand(req.Command); err != nil {
		exitCode := 1
		return &filesystem.ExecuteResponse{
			Output:   err.Error(),
			ExitCode: &exitCode,
		}, nil
	}

	execCtx, cancel := context.WithTimeout(ctx, b.commandTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(execCtx, "cmd", "/C", req.Command)
	} else {
		cmd = exec.CommandContext(execCtx, "sh", "-c", req.Command)
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
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return nil, err
		}
		exitCode = exitErr.ExitCode()
	}

	return &filesystem.ExecuteResponse{
		Output:    output,
		ExitCode:  &exitCode,
		Truncated: truncated,
	}, nil
}
