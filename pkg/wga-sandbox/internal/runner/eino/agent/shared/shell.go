package shared

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk/filesystem"
)

const (
	maxOutputSize  = 1024 * 1024 // 1MB
	commandTimeout = 5 * time.Minute
	// ScriptFileMaxSize 读取脚本文件做 body-scan 的大小上限。
	// 越权脚本通常几 KB - 几十 KB；超过 1 MB 一般是数据文件或大型工具，
	// 静态扫描收益低、成本高，直接放行让 shell 自己去跑。
	ScriptFileMaxSize = 1 << 20 // 1 MB
)

// SkillEnvFileName sandbox workspace 下注入 skill 变量的文件名。
// 写侧（eino runner.injectEnvVariables）与读侧（ShellOnlyBackend.Execute）共用此常量，
// 避免两处字面值漂移。JSON 格式不与 dotenv 兼容，故文件名带 .json 后缀。
const SkillEnvFileName = ".skill_env.json"

// ShellOnlyBackend 仅提供 shell 命令执行能力。
// 不实现 filesystem.Backend 的其他文件操作方法，因为 bash 工具只调用 Execute。
type ShellOnlyBackend struct {
	maxOutputSize  int
	commandTimeout time.Duration
	workDir        string
	// skillEnvPath 指向 workspace 下的 SkillEnvFileName 文件；Execute 每次会重新读，
	// 文件不存在时保持 cmd.Env=nil（透明继承父进程 env），从而对无 skill var 的请求零侵入。
	skillEnvPath string
	// halt 累计单次 sandbox 会话内的连续 [BLOCKED:...] 次数；连续到阈值触发熔断。
	// 可空：nil 表示不启用熔断（兼容 tests + oneshot 沙箱路径）。
	halt *HaltState
}

// NewShellOnlyBackend 构造 ShellOnlyBackend。halt 可空——为 nil 时不启用连续 BLOCKED 熔断。
func NewShellOnlyBackend(workDir string, halt *HaltState) *ShellOnlyBackend {
	return &ShellOnlyBackend{
		maxOutputSize:  maxOutputSize,
		commandTimeout: commandTimeout,
		workDir:        workDir,
		skillEnvPath:   filepath.Join(workDir, SkillEnvFileName),
		halt:           halt,
	}
}

// haltedResponse 返回熔断后 Execute 的短路响应。命令不进入 cmd.Run，直接向 LLM 返回终止提示。
func haltedResponse() *filesystem.ExecuteResponse {
	exitCode := 1
	return &filesystem.ExecuteResponse{
		Output:   "[BLOCKED:HALT] session terminated by sandbox security guard due to repeated policy violations; no further commands will execute.",
		ExitCode: &exitCode,
	}
}

// readSkillEnv 读取并反序列化 .skill_env.json。文件不存在返回 nil（合法路径）；
// 存在但解析失败打日志后返回 nil（不阻断命令执行，保持"无 var = 透明继承"的兜底行为）。
// 错误信息只带文件路径，不含内容片段。
func (b *ShellOnlyBackend) readSkillEnv() map[string]string {
	if b.skillEnvPath == "" {
		return nil
	}
	data, err := os.ReadFile(b.skillEnvPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[Shell] read %s failed: %v", b.skillEnvPath, err)
		}
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	var envMap map[string]string
	if err := json.Unmarshal(data, &envMap); err != nil {
		log.Printf("[Shell] parse %s failed: %v", b.skillEnvPath, err)
		return nil
	}
	return envMap
}

// Execute 执行 shell 命令，附带安全校验、超时、输出截断与退出码处理。
//
// 三层防线（按顺序）：
//  1. halt 短路：已熔断则直接返回 [BLOCKED:HALT]，命令永不进入 cmd.Run。
//  2. validateCommand：通用系统安全（rm -rf / 路径穿越 / 写敏感文件等）。
//     与 skill vars 状态无关，**永远生效**——predates 本次 skill-var 工作。
//  3. precheck 家族（precheckCommand + precheckScriptFile）：skill 变量保护层。
//     仅在 skill 配置了至少一个 var（envMap 非空）时启用——**无 vars 时跳过**，
//     backward compat 到 v4 之前的行为。
//
// 熔断计数：precheck 拦截或 validate 拦截触发的 output 会以 [BLOCKED:xxx] 或
// `安全拦截：` 起头；连续 BLOCKED 到阈值触发 haltFn（绑定到 sessionCtx 的
// CancelCauseFunc），eino runner iterator 关闭，上层通过 BuildFinalAgentEvent
// 下发 error[agent] 兜底 SSE。**halt 计数同样只在有 skill vars 时启用**——
// 无 vars 时 precheck 不跑，halt 也无意义；validate 拦截在无 vars 场景不计入
// halt（保持 backward compat 语义纯粹）。
func (b *ShellOnlyBackend) Execute(ctx context.Context, req *filesystem.ExecuteRequest) (*filesystem.ExecuteResponse, error) {
	// 已熔断：短路返回，命令不进入执行链。放在最前面确保后续步骤都不跑。
	if b.halt != nil && b.halt.Halted() {
		return haltedResponse(), nil
	}

	// 通用安全校验（永远生效，与 skill vars 状态无关）
	if err := validateCommand(req.Command); err != nil {
		exitCode := 1
		return &filesystem.ExecuteResponse{
			Output:   err.Error(),
			ExitCode: &exitCode,
		}, nil
	}

	// 读取 skill 已注入的环境变量。文件格式：JSON map[string]string（由 runner BeforeRun
	// 的 injectEnvVariables 写入）。两条不变量：
	//   1. 文件不存在时 cmd.Env 保持 nil（Go exec 自动继承父进程 env），透明无侵入；
	//   2. 文件存在时 cmd.Env = append(os.Environ(), kv...)，PATH/PYTHONPATH/HOME 不丢失，
	//      同名 key 以 skill var 为准（os/exec 取后出现的值）。
	// 安全：读 / 解析失败的错误信息只带文件路径不带值；同样不打印 envMap 内容。
	envMap := b.readSkillEnv()
	hasVars := len(envMap) > 0

	// Skill 变量保护层：仅在有 skill vars 时启用。
	// 无 vars 时保护对象不存在，precheck 跳过——backward compat 到 v4 之前的行为。
	if hasVars {
		// cmd.Run 前对命令字符串做静态拦截，阻止 LLM 通过 echo/cat/printenv 等方式回流敏感 value。
		if err := precheckCommand(req.Command, envMap); err != nil {
			exitCode := 1
			resp := &filesystem.ExecuteResponse{
				Output:   err.Error(),
				ExitCode: &exitCode,
			}
			b.recordHalt(resp.Output, hasVars)
			return resp, nil
		}
		// 文件形式脚本 body-scan：拦下 `python3 x.py` / `bash script.sh` 等通过文件运行脚本的越权手法。
		if err := precheckScriptFile(req.Command, b.workDir, envMap); err != nil {
			exitCode := 1
			resp := &filesystem.ExecuteResponse{
				Output:   err.Error(),
				ExitCode: &exitCode,
			}
			b.recordHalt(resp.Output, hasVars)
			return resp, nil
		}
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

	if len(envMap) > 0 {
		env := os.Environ()
		for k, v := range envMap {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
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

	resp := &filesystem.ExecuteResponse{
		Output:    output,
		ExitCode:  &exitCode,
		Truncated: truncated,
	}
	// exec 正常收尾也走 recordHalt：output 若是 sh 报错回显"[BLOCKED:..."/"安全拦截："
	// 之类的字符串（合法场景下几乎不可能，兜底），也一并计入；否则 NoteSuccess 重置计数。
	b.recordHalt(resp.Output, hasVars)
	return resp, nil
}

// recordHalt 根据 output 是否为拦截产物累计或重置 halt 计数。
// hasVars=false 时短路——无 skill vars 场景下 halt 不启用（backward compat）。
func (b *ShellOnlyBackend) recordHalt(output string, hasVars bool) {
	if b.halt == nil || !hasVars {
		return
	}
	if isBlockedOutput(output) {
		b.halt.NoteBlocked()
	} else {
		b.halt.NoteSuccess()
	}
}

// isBlockedOutput 判断 Execute 的 Output 是否为安全拦截产物。
// 覆盖两个前缀：precheck 家族的 [BLOCKED:xxx] 与 validateCommand 家族的 `安全拦截：...`。
func isBlockedOutput(output string) bool {
	return strings.HasPrefix(output, "[BLOCKED:") || strings.HasPrefix(output, "安全拦截：")
}

// scriptFileExtSet 与 precheck.go 里 scriptFileExtRe 保持同源。
// script-file 扫 tokens 位置参数时用它做后缀快速匹配。
var scriptFileExtSet = map[string]struct{}{
	".py": {}, ".pyw": {}, ".sh": {}, ".bash": {},
	".js": {}, ".mjs": {}, ".cjs": {},
	".pl": {}, ".rb": {}, ".php": {}, ".awk": {},
}

// devStdinPaths 解释器 "从 stdin/fd 读脚本" 的伪路径集合。命中即拒（stdin-source 标签）。
var devStdinPaths = map[string]struct{}{
	"/dev/stdin": {}, "/proc/self/fd/0": {},
}

// precheckScriptFile 拦截 "命令行传入 workspace 内脚本文件时脚本 body 含越权代码"。
//
// 扫所有 tokens 位置参数：任何位置参数（不论 head 是 python / bash / cat / grep / less / cp / mv 等）
// 只要指向 workspace 内的脚本后缀文件，都读进来做 body-scan。
//
// 触发规则：
//  1. 位置参数是 /dev/stdin / /dev/fd/* / /proc/self/fd/* → 拒（stdin-source 标签）
//  2. 位置参数后缀在 scriptFileExtSet 内 + 路径在 workDir 边界内 + 文件存在 + size ≤ 上限
//     → 读进来跑 precheckScriptReadSensitive + precheckScriptBody；任一命中即拒
//
// 路径信任分级：
//   - workspace/skills/** 下的脚本视为可信预置（构建期审计），免除三条件 body-scan；
//     但 precheckScriptReadSensitive（读 .env 两条件）仍生效，作为硬防线。
//   - workspace 其它路径（tmp / output）不豁免，两层扫描都走。
//
// 兜底策略：
//   - 文件不存在 / 超大 / 读失败 / 路径越出 workDir → 跳过此 arg（不阻断命令）
//   - workDir 为空 → 放行（测试或极端 corner case）
//   - envMap 空时 precheckScriptBody 不触发，但 precheckScriptReadSensitive 仍会拦（护 .env 本身）
//
// 历史注：早期版本这里有 "位置参数以 $ 开头一律拒"（indirect-target）分支，因误伤
// `python3 script.py --token "$KEY"` 这类合法 skill 调用模式已删除。头位置的变量
// 展开（$X / $(...)）仍由 precheck.go 顶层 indirectCmdRe 拦。
func precheckScriptFile(cmdStr, workDir string, envMap map[string]string) error {
	// 用引号感知 tokenizer 替代 strings.Fields：把 `awk 'BEGIN{print $0}'` 的整块 body
	// 视为一个 token，避免 awk 的字段引用 $0/$1/... 被独立看作 "$-prefix 位置参数"。
	// （历史遗留原因：早期这里的 $-prefix 判定会拒 script-file 层的位置参数，现已删除；
	// 但引号感知分词保留，因其对其它 $KEY 相关判定也更准确。）
	tokens := splitFieldsRespectQuotes(cmdStr)
	if len(tokens) < 2 {
		return nil
	}
	head := strings.TrimPrefix(tokens[0], "/usr/bin/")
	head = strings.TrimPrefix(head, "/bin/")
	head = strings.ToLower(head)

	if workDir == "" {
		return nil
	}
	absWorkDir, wderr := filepath.Abs(workDir)
	if wderr != nil {
		return nil
	}

	// "flag + 值" 组合：这些 flag 后紧跟的一个 arg 不是位置参数
	skipNextArg := map[string]bool{
		"-c": true, "-e": true, "--eval": true, "-p": true, "--print": true, "-r": true,
		"-m": true, "-E": true,
	}

	i := 1
	for i < len(tokens) {
		tok := tokens[i]
		if tok == "" || tok == "-" || tok == "--" {
			i++
			continue
		}
		if strings.HasPrefix(tok, "--") {
			i++
			continue
		}
		if strings.HasPrefix(tok, "-") {
			if skipNextArg[tok] {
				i += 2
			} else {
				i++
			}
			continue
		}
		// 位置参数：判定
		// 注：早期版本这里有 "$-prefix 位置参数一律拒" 分支（indirect-target 标签），
		// 但会误伤合法用法——例如 skill 官方调用模式 `python3 script.py --token "$KEY"`
		// 里的 flag value。由于本函数上层的 skipNextArg 无法穷举所有 skill / 用户自定义
		// flag（--token / --url / --arguments 等），"位置参数 = 脚本路径" 这一判断在
		// 混着 flag value 的现实命令上不成立。已删除该分支：
		//   - head 位置的 $X / $(...) 变量展开仍由 precheck.go 顶层 indirectCmdRe 拦；
		//   - 位置参数是变量展开时，shell 展开后要么落到 workspace 内脚本文件继续走
		//     script-file 路径检查，要么展开失败让 bash 报错——不打开新的泄露通道。
		// 剥外层配对引号
		arg := tok
		if len(arg) >= 2 {
			if (arg[0] == '"' || arg[0] == '\'') && arg[len(arg)-1] == arg[0] {
				arg = arg[1 : len(arg)-1]
			}
		}
		// /dev/stdin / /proc/self/fd/0 / /dev/fd/N 伪路径（stdin 喂脚本）→ 拒
		if _, ok := devStdinPaths[arg]; ok {
			return fmt.Errorf("%s", blockedMsg("stdin-source", head))
		}
		if strings.HasPrefix(arg, "/dev/fd/") || strings.HasPrefix(arg, "/proc/self/fd/") {
			return fmt.Errorf("%s", blockedMsg("stdin-source", head))
		}
		// 脚本后缀判定：不在集合内跳过此 arg
		ext := strings.ToLower(filepath.Ext(arg))
		if _, ok := scriptFileExtSet[ext]; !ok {
			i++
			continue
		}
		// 路径解析：绝对路径直接用；相对路径相对 workDir。
		var absPath string
		if filepath.IsAbs(arg) {
			absPath = filepath.Clean(arg)
		} else {
			absPath = filepath.Join(absWorkDir, arg)
		}
		// 边界：workDir 内
		relToWork, rerr := filepath.Rel(absWorkDir, absPath)
		if rerr != nil || relToWork == ".." || strings.HasPrefix(relToWork, ".."+string(filepath.Separator)) {
			i++
			continue
		}
		stat, serr := os.Stat(absPath)
		if serr != nil {
			if !os.IsNotExist(serr) {
				log.Printf("[Shell] script-file stat failed: %v (path=%s)", serr, absPath)
			}
			i++
			continue
		}
		if stat.IsDir() || stat.Size() > ScriptFileMaxSize {
			i++
			continue
		}
		data, rerr2 := os.ReadFile(absPath)
		if rerr2 != nil {
			log.Printf("[Shell] script-file read failed: %v (path=%s)", rerr2, absPath)
			i++
			continue
		}
		body := string(data)
		// 硬防线：任何路径下的脚本读 .env / .skill_env 都拒（含 skill 目录）
		if perr := precheckScriptReadSensitive(body); perr != nil {
			return fmt.Errorf("%s", blockedMsg("script-file", head))
		}
		// 三条件 body-scan：workspace/skills/** 下的可信预置脚本免除，其它路径（tmp / output）走。
		if len(envMap) > 0 && !isTrustedSkillScript(absPath, absWorkDir) {
			if perr := precheckScriptBody(body, envMap); perr != nil {
				return fmt.Errorf("%s", blockedMsg("script-file", head))
			}
		}
		i++
	}
	return nil
}

// isTrustedSkillScript 判断 absPath 是否位于 workspace/skills/ 下。
// 这些脚本是 sandbox 启动前静态挑选、可审计的预置文件，不是 LLM 或攻击者写入的
// 运行时产物，免除三条件 body-scan；但仍受 precheckScriptReadSensitive 两条件
// 保护（读 .env / .skill_env 不豁免）。
//
// 前提：workspace/skills/ 目录内容由构建/打包流程审计；运行时 LLM 不应有写入
// 该目录的权限（该权限边界由 sandbox 文件系统层保障，不在本函数职责内）。
func isTrustedSkillScript(absPath, absWorkDir string) bool {
	rel, err := filepath.Rel(absWorkDir, absPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, "skills/")
}
