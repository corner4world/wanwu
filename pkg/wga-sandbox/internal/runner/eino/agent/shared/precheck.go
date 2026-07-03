package shared

import (
	"fmt"
	"regexp"
	"strings"
)

// --- Skill 环境变量保护：命令前置拦截 ---
//
// 目的：防止 LLM 在恶意 prompt 引导下，通过 bash 输出回流 skill 已注入的敏感 env 值。
// 策略：在 cmd.Run 之前对命令字符串做静态 precheck，命中规则即拒绝执行；不做输出 redact,
// 因为字节匹配式 redact 会误伤 value 撞合法输出的场景（污染整个 sandbox 输出）。
//
// 指导思想：承认 shell + 内联解释器攻击面不可穷举；precheck 目标是抬高攻击者构造成本，
// 不是证明安全。skill 内脚本自带的安全隐患也尽力发现拦截，责任归属不做区分。
// 架构层根治方案（网络白名单 / 不落盘 / token broker）由部署层实现，不在本层。
//
// 拦截规则按攻击向量分类：
//
//   [枚举型]
//     - 独立命令：env / printenv / set / declare / export / compgen / typeset 无差别拒
//     - 命令替换：$(env) / `printenv` / <(env) / <(printenv) 等嵌套语义等价，同样拒
//
//   [引用型]（打印命令/子命令引用 $KEY / ${KEY} / bash 参数展开变体）
//     - 打印类命令 (echo/printf/cat/tee/less/more/head/tail/sed/awk/tr/rev/cut/paste/sort/uniq/grep/
//       nl/fold/fmt/column/strings/base64/xxd/od/hexdump/hd) 命令行含 $KEY 引用 → 拒
//     - NAME=...$KEY... 中转变量赋值（赋值步骤即拒，杜绝 N 跳链）
//     - tokens[0] 里含 $KEY 引用（如 nonexistent_$KEY 报错回显泄露）
//     - trap 'echo $KEY' EXIT 之类延迟执行
//     - bash 参数展开变体覆盖：${K} / ${K:-x} / ${K:=x} / ${K:?x} / ${K:+x} / ${K#*} / ${K##*} /
//       ${K%*} / ${K%%*} / ${K:0:5} / ${K:5} / ${K/A/B} / ${K//A/B} / ${K^^} / ${K,,} /
//       ${K^} / ${K,} / ${K@Q} / ${K@P} / ${K@U} / ${K@A} / ${!K}
//
//   [文件读取]
//     - 命令体含 .env / .skill_env 系列路径（含 glob 通配符 * ? [）
//     - 命令体含 /proc/*/env* 路径（含 shell 变量展开）
//     - ln / cp / mv / rsync / install + .env / .skill_env 字面量（防别名化后再读）
//     - 打印命令 + `.*` / `.?<any>` 之类宽泛 glob（防 cat .* 展开到 .env）
//     - find / xargs 跳板到打印命令读敏感文件
//
//   [解释器 body]
//     - 内联脚本 body 同时命中 (env 访问) ∧ (stdout sink) ∧ (envMap key 字面量) 三条件
//     - heredoc body 同上（bash 家族用 $KEY 语法，其余用 os.environ / process.env 等）
//     - body 含文件读 API + .env / .skill_env 字面量（护 workspace/.env 里 OPENAI_API_KEY）
//
//   [dynamic exec / shell 语法]
//     - bash / sh / dash -c "<arg>" 递归 precheck，拦下"包一层 shell 绕过"
//     - eval 关键字无条件拒
//     - ${!VAR} 间接展开无条件拒
//     - head 是 $X / ${X} / $(...) 变量或命令替换展开 → 无条件拒
//     - bash -x / -v / -o xtrace debug flag（每条命令展开值输出到 stderr）
//     - 无参 bash / echo | bash / pipeline 尾接 shell（从 stdin 读命令的 dynamic exec）
//     - BASH_ENV / ENV / PROMPT_COMMAND / LD_PRELOAD / LD_LIBRARY_PATH / LD_AUDIT / SHELLOPTS /
//       BASH_XTRACEFD / PS4 等特殊 env 赋值（会触发 shell 启动 source 或运行时行为改变）
//     - 重定向 target / 脚本路径以 $ 开头（变量展开无法静态解析）
//     - base64 -d / --decode / xxd -r / --revert 解码命令（防编码后 pipe 到 shell）
//     - head normalize：剥掉 NAME=value 前缀赋值、command/exec/builtin 外壳、 (/{ 子 shell 前导、
//       反斜杠等常见绕 head 判定的语法糖后，再做后续所有 head 判定
//
//   [写盘]
//     - 重定向到 .py / .sh / .js / .rb / .pl / .php / .awk 等脚本后缀文件 + inline body（quoted
//       string 或 heredoc body）含越权代码或文件读 API + .env 字面量
//     - 重定向 target 是 shell rc 文件 (.bashrc / .zshrc / .profile 等) → 同样走 body 扫描
//     - 重定向 target 是 $VAR 变量展开 → 无论后缀强制走 body 扫描
//
// 另外文件形式脚本 body-scan (script-file) 规则在 ShellOnlyBackend.Execute 层做 IO check
// （precheck 通过后追加），扫所有 tokens 位置参数指向的 workspace 内脚本文件（不限 head，
// 因为无论 head 是 python / bash / cat / grep / less / cp / mv 都能达成"让越权代码被读"的效果）。
// 独立于 precheckCommand 分层：precheckCommand 是 pure-static（无 IO），script-file 需要读盘。

var (
	// secretEnumeratorCmds 无差别打印所有 env 的命令，envMap 是否为空都拒（给 LLM 一致 UX）。
	secretEnumeratorCmds = map[string]struct{}{
		"env": {}, "printenv": {}, "set": {}, "declare": {},
		"compgen": {}, "typeset": {}, "export": {},
	}
	// refPrinterCmds 必须叠加 "命令体引用注册 key" 条件才拒，避免 echo "hello" 这种合法用法被错杀。
	// 覆盖到 sed/awk/tr/rev/cut/paste/sort/uniq/grep/nl/fold/fmt/column 等所有会向 stdout 输出的
	// 文本处理命令。命令行无 $KEY 引用不受影响，FP 理论为 0；有 $KEY 引用即视为泄露。
	refPrinterCmds = map[string]struct{}{
		"echo": {}, "printf": {}, "cat": {}, "tee": {},
		"less": {}, "more": {}, "head": {}, "tail": {},
		"strings": {}, "base64": {}, "xxd": {}, "od": {},
		"hexdump": {}, "hd": {},
		"sed": {}, "awk": {}, "tr": {}, "rev": {}, "cut": {}, "paste": {},
		"sort": {}, "uniq": {}, "grep": {}, "nl": {}, "fold": {}, "fmt": {}, "column": {},
	}
	// sensitiveFileRe 匹配 .skill_env* / .env* 文件路径，需有 word/路径边界，避免误伤 readme.envsetup 这种。
	// 末尾边界含 glob 元字符 * ? [，覆盖 `cat .env*` / `cat .env?` / `cat .env[abc]` 这类通配符探查。
	// 前置边界保持 `^|[\s/]` 不含通配符：`readme.envsetup` 仍不会被误命中（`me` 不是路径分隔符）。
	sensitiveFileRe = regexp.MustCompile(`(^|[\s/])(\.env|\.skill_env)(\.[\w.]+)?($|[\s/*?\[])`)
	// procEnvironRe 放宽 pid 段与 environ 段均为"任意非空白/路径分隔符 + env 前缀"，
	// 覆盖 self / 数字 / $$ / $BASHPID / 通配符（如 se* / envi* / env*）。
	// 副作用：`/proc/<x>/env<any>` 之类不存在的路径也会命中，但那类命令不会成功执行，
	// precheck 提前拒无实质影响。
	procEnvironRe = regexp.MustCompile(`/proc/[^\s/]+/env[^\s/]*`)

	// scriptInterpreterCmds 解释器内联脚本拦截覆盖的命令集合及其"内联脚本"flag。
	// 匹配 head 后，extractInterpreterScript 在 cmdStr 中找任一 flag，取 flag 后的下一参数作为脚本 body。
	// awk/gawk/mawk 走单独的位置参数分支，不在本 map（脚本是首个非 flag 的位置参数）。
	scriptInterpreterCmds = map[string][]string{
		"python":  {"-c"},
		"python3": {"-c"},
		"pypy":    {"-c"},
		"pypy3":   {"-c"},
		"node":    {"-e", "--eval", "-p", "--print"},
		"nodejs":  {"-e", "--eval", "-p", "--print"},
		"perl":    {"-e", "-E"},
		"ruby":    {"-e"},
		"php":     {"-r"},
	}

	// scriptEnvAccessRe 命中即视为"脚本读取了 process env"。任一语言的常见 env 读法即可。
	scriptEnvAccessRe = regexp.MustCompile(
		`os\.environ|os\.getenv\s*\(|process\.env\b|\$ENV\{|\bENV\[|ENVIRON\[|\bgetenv\s*\(`)

	// scriptSinkRe 命中即视为"脚本会向 stdout/stderr 输出"。
	// 关键：\bprint\b 同时覆盖 Python 的 print(...) 与 Perl/awk 的裸 print（后者无括号）。
	// 不会误命中 printf / pprint —— 两者都不存在以 \b 切分的 "print" 边界。
	// \becho\b 主要覆盖 PHP / awk 内联用法；bash 的 echo 走的是打印型命令分支（head 在 refPrinterCmds），不会到这里。
	scriptSinkRe = regexp.MustCompile(
		`\bprint\b|console\.(log|error|warn|info|dir)\s*\(|` +
			`sys\.std(out|err)\.write\s*\(|process\.std(out|err)\.write\s*\(|` +
			`\bputs\b|\bsay\b|\bprintf\b|\becho\b`)

	// scriptFileExtRe 写入文件的目标若为可执行脚本后缀，需扫描 inline 内容做三条件检查。
	scriptFileExtRe = regexp.MustCompile(`\.(py|pyw|sh|bash|js|mjs|cjs|pl|rb|php|awk)\b`)

	// heredocStartRe 识别 `<head> ... <<-?['"]?TAG['"]?` 的 heredoc 起始 token，捕获 TAG 名。
	// body 与结束 TAG 由 extractHeredocBody 用 strings.Index 扫描提取（RE2 不支持反向引用）。
	heredocStartRe = regexp.MustCompile(`<<-?\s*['"]?([A-Za-z_][A-Za-z0-9_]*)['"]?`)

	// bashEnvAccessRe 仅在 bash/sh/dash heredoc body 内使用：$KEY 与 ${KEY} 即视为 env 访问。
	// Python/Node 脚本内的 $KEY 是字面量，不在此处用。
	bashEnvAccessRe = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*|\$\{[A-Za-z_][A-Za-z0-9_]*\}`)

	// evalKeywordRe head 形态或作为独立 token 出现的 eval 关键字。
	// 边界要求是空白/分隔符，避免误命中 eval_expr_func 这种函数名子串。
	evalKeywordRe = regexp.MustCompile(`(^|[\s;&|])eval(\s|$)`)

	// indirectExpandRe bash 的间接展开语法 ${!VAR}。LLM 自动生成 bash 几乎不用，可无条件拒。
	indirectExpandRe = regexp.MustCompile(`\$\{!\s*[A-Za-z_][A-Za-z0-9_]*\s*\}`)

	// cmdSubstEnumRe $( env|printenv|set|declare|compgen|typeset|export ... ) 或反引号形态。
	// 等价于直接执行枚举命令，复用 enumerator 标签。
	cmdSubstEnumRe = regexp.MustCompile(
		"\\$\\(\\s*(env|printenv|set|declare|compgen|typeset|export)\\b" +
			"|`\\s*(env|printenv|set|declare|compgen|typeset|export)\\b")

	// relayAssignRe `<NAME>=` 赋值语句首部。RHS 若引用 envMap 中任一 key 即拒。
	// 前导边界 (^|[\s;&|()])避免误命中 X==Y 比较 / 字符串中间的 VAR=value 子串。
	relayAssignRe = regexp.MustCompile(`(^|[\s;&|()])([A-Za-z_][A-Za-z0-9_]*)=`)

	// scriptFileReadRe 覆盖 python / node / perl / ruby / php / awk 的常见文件读 API。
	// 命中即视为"脚本读文件"。是 precheckScriptReadSensitive 两条件硬防线的第一条
	// （另一条件是 .env / .skill_env 字面量），单独命中不拒。
	// 匹配示例：open( / .read( / .readlines( / fs.readFile / require('fs') / File.read / IO.read /
	// file_get_contents( / fopen( / readfile( / getline < （awk）
	scriptFileReadRe = regexp.MustCompile(
		`\bopen\s*\(|\.read\s*\(|\.readlines\s*\(|` +
			`\bfs\.readFile|\bfs\.readFileSync|require\s*\(\s*['"]fs['"]|` +
			`\bFile\.(read|open)|\bIO\.(read|readlines)|` +
			`\bfile_get_contents\s*\(|\bfopen\s*\(|\breadfile\s*\(|` +
			`\bgetline\s*<`)

	// scriptSensitivePathLiteralRe 匹配脚本 body 中出现的 .env / .skill_env 字面量。
	// 三条件的第二条：脚本参数含敏感路径字面量（不做扁平路径展开，绕过手法列入残留）。
	scriptSensitivePathLiteralRe = regexp.MustCompile(`\.(env|skill_env)(\.[\w.]+)?`)

	// findExecReaderRe 匹配 `find ... -exec <printer>` / `find ... -exec <printer> ;` 形态。
	// 覆盖 find 跳板到打印命令的所有主流 pattern（sh 会拆解 -exec 后的命令行）。
	findExecReaderRe = regexp.MustCompile(
		`\bfind\b[^|;&]*?-exec\s+(cat|less|more|head|tail|xxd|hexdump|hd|od|strings|base64|tee|printf|echo)\b`)

	// xargsReaderRe 匹配 `<anything> | xargs <printer>` 形态。
	// xargs 可能带 -n / -I 等 flag，一起吞掉。
	xargsReaderRe = regexp.MustCompile(
		`\|\s*xargs\b(?:\s+-[a-zA-Z]+)*\s+(cat|less|more|head|tail|xxd|hexdump|hd|od|strings|base64)\b`)

	// findNamePatternEnvRe 检测 `find ... -name` 参数指向 env 文件族。
	// pattern 允许含 glob 元字符（* ? [ ]）和 . 后缀，覆盖 .env* / .skill_env* / .e / .s 等模糊探查。
	findNamePatternEnvRe = regexp.MustCompile(`-name\s+['"]?\.(env|skill_env|e|s)[\w*.?\[\]]*`)

	// echoPipeEnvRe 检测 pipeline 前段用 echo/printf 输出 env 路径字面量再通过 xargs 类跳板消费。
	// 例：echo .skill_env.json | xargs cat
	echoPipeEnvRe = regexp.MustCompile(`(echo|printf)[^|;&]*\.(env|skill_env)`)

	// indirectCmdRe head token 是 shell 变量/命令替换展开（$X / ${X} / $(...)）。
	// shell 会先展开再执行，precheck 静态分析看不到展开后的真实命令名——绕过 head 判定的唯一途径。
	// LLM 自动化任务里几乎无合法用途（合法用 printenv 就直接写 printenv），无条件拒。
	indirectCmdRe = regexp.MustCompile(`^\$[A-Za-z_{(]`)

	// shellDebugFlagRe bash -x / -v / -o xtrace 会把每条命令的展开值输出到 stderr（含 $KEY 展开值）。
	// LLM 自动化场景几乎无合法 debug 需求；bash 家族 + 这些 debug flag 无条件拒。
	shellDebugFlagRe = regexp.MustCompile(`\s(-[a-zA-Z]*[xv]|--xtrace|--verbose|-o\s+xtrace)\b`)

	// pipeShellRe pipeline 尾接 shell（... | bash / ... | sh / ... | dash）。
	// 无参 shell 从 stdin 读命令等价于 dynamic exec bash -c "<attacker code>"；无条件拒。
	pipeShellRe = regexp.MustCompile(`\|\s*(bash|sh|dash)\s*(\||;|&&|\|\||$)`)

	// specialEnvAssignRe 影响 shell 启动/运行时行为的特殊 env 名。
	// 攻击 pattern: BASH_ENV=/tmp/rc sh -c ':' 会在 bash 启动前自动 source /tmp/rc。
	// LHS 命中即拒（不管 RHS 引用什么，因为这些 env 无合法业务用途）。
	specialEnvAssignRe = regexp.MustCompile(
		`(^|[\s;&|()])(BASH_ENV|ENV|PROMPT_COMMAND|LD_PRELOAD|LD_LIBRARY_PATH|LD_AUDIT|SHELLOPTS|BASH_XTRACEFD|PS4)=`)

	// sensitiveFileHandlerRe 与 sensitiveFileRe 叠加判定（head 是文件操作 + 命令体含 .env / .skill_env 字面量）
	// 防止 ln -s .skill_env.json /tmp/x 之类的别名化绕过。
	sensitiveFileHandlerRe = regexp.MustCompile(`^(ln|cp|mv|rsync|install)$`)

	// broadGlobRe cat .* / less .* / head .? 之类宽泛 glob 展开到 . 起头文件。
	// shell 会展开为 .bashrc / .env / .skill_env.json 等；打印命令读所有匹配文件。
	// FP 低：合法用户几乎不 cat .* （一般用完整名或 ls .*）。
	// 匹配：空白 + .* / .? (后跟非路径分隔符或行尾) / .??* 等
	broadGlobRe = regexp.MustCompile(`\s\.(\*|\?($|[^/\s]))`)

	// base64DecodeRe / xxdRevertRe 解码命令：常与管道到 shell 组合成 dynamic exec
	// （base64 -d <<< 'Y2F0IC5lbnY=' | sh）。LLM 自动化场景无合法用途，无差别拒。
	base64DecodeRe = regexp.MustCompile(`(^|[\s;&|()])base64\s+(-D|-d|--decode)\b`)
	xxdRevertRe    = regexp.MustCompile(`(^|[\s;&|()])xxd\s+(-r|--revert)\b`)

	// shellRcFileRe 写入 shell rc 文件（.bashrc / .profile 等）会在下次 shell 启动时自动 source。
	// 补 script-write 覆盖面：脚本后缀正则只匹配 .sh / .py / ... 后缀，rc 文件后缀不同会漏掉。
	shellRcFileRe = regexp.MustCompile(`\.(bashrc|bash_profile|zshrc|zprofile|zshenv|profile|envrc)($|\s)`)

	// processSubstEnumRe 进程替换 <(env) / <(printenv) 等，语义等价命令替换嵌枚举命令。
	// 与 cmdSubstEnumRe 独立判定，标签同 enumerator。
	processSubstEnumRe = regexp.MustCompile(`<\(\s*(env|printenv|set|declare|compgen|typeset|export)\b`)
)

// precheckCommand 在 exec 前做静态拦截。envMap key 为已注入的 skill var keys；
// 命中拒绝规则返回非 nil error，error.Error() 即作为返回给 LLM 的 stdout。
// 不打印 envMap value、不打印命令字符串中的可疑片段。
//
// 规则顺序设计原则：先做高确定性、低 FP 的硬规则（head 间接展开 → 套娃递归 → 内联/heredoc 脚本体扫描 →
// eval / 间接展开 / 命令替换枚举 → 枚举型 head → 中转变量赋值 → find/xargs 跳板 → 敏感文件 →
// 打印型 head 配合 $KEY 引用 → 写脚本文件 inline 扫描）。写脚本文件检测放最后，因为它需要全局扫描所有
// quoted 串，成本最高。head 间接展开放最前，因为一旦 head 是 $X / $(...)，后续所有基于 head 的判定都
// 失去意义（shell 展开后真实命令名 precheck 看不到），直接拒最快。
// splitCompoundCommands 按引号感知的 shell 命令分隔符拆分 cmdStr。
// 分隔符：&& / || / ; / 换行。单双引号内的分隔符视为字面量不拆。
// 管道 | 不在此处拆分——pipeShellRe 等规则已覆盖 | bash / | sh 的 dynamic exec，
// printenv | grep 等已由 cmdSubstEnumRe / secretEnumeratorCmds 覆盖。
func splitCompoundCommands(cmdStr string) []string {
	var parts []string
	var buf strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(cmdStr); i++ {
		c := cmdStr[i]
		if inSingle {
			buf.WriteByte(c)
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			buf.WriteByte(c)
			if c == '"' && (i == 0 || cmdStr[i-1] != '\\') {
				inDouble = false
			}
			continue
		}
		if c == '\'' {
			inSingle = true
			buf.WriteByte(c)
			continue
		}
		if c == '"' {
			inDouble = true
			buf.WriteByte(c)
			continue
		}
		if c == '\\' && i+1 < len(cmdStr) {
			buf.WriteByte(c)
			buf.WriteByte(cmdStr[i+1])
			i++
			continue
		}
		// && / || / ; / 换行 作为命令分隔符
		if c == '&' && i+1 < len(cmdStr) && cmdStr[i+1] == '&' {
			flushCmdPart(&buf, &parts)
			i++ // 跳过第二个 &
			continue
		}
		if c == '|' && i+1 < len(cmdStr) && cmdStr[i+1] == '|' {
			flushCmdPart(&buf, &parts)
			i++ // 跳过第二个 |
			continue
		}
		if c == ';' {
			flushCmdPart(&buf, &parts)
			continue
		}
		buf.WriteByte(c)
	}
	flushCmdPart(&buf, &parts)
	return parts
}

func flushCmdPart(buf *strings.Builder, parts *[]string) {
	s := strings.TrimSpace(buf.String())
	if s != "" {
		*parts = append(*parts, s)
	}
	buf.Reset()
}

func precheckCommand(cmdStr string, envMap map[string]string) error {
	return precheckCommandWithDepth(cmdStr, envMap, 0)
}

// precheckCommandWithDepth 在 precheckCommand 基础上增加递归深度保护。
// depth 超过 4 时不再拆分复合命令，防止嵌套炸弹。
func precheckCommandWithDepth(cmdStr string, envMap map[string]string, depth int) error {
	if depth >= 4 {
		return nil
	}
	// 拆分复合命令：&& / || / ; / 换行 是 shell 命令分隔符，
	// 引号感知——单双引号内的分隔符不拆。
	// 拆分后逐条递归 precheck，任一命中即返回 error。
	subCmds := splitCompoundCommands(cmdStr)
	if len(subCmds) > 1 {
		for _, sub := range subCmds {
			if err := precheckCommandWithDepth(sub, envMap, depth+1); err != nil {
				return err
			}
		}
		return nil
	}

	tokens := strings.Fields(cmdStr) // 简化：按空格分词，复杂 shell quoting 不做完整解析（已知盲点）。
	if len(tokens) == 0 {
		return nil
	}

	// head normalize — 剥掉前缀赋值 (X=1) / command-exec-builtin 外壳 / (/{
	// / 反斜杠等常见绕 head 判定的语法糖，得到真实 head。
	head := normalizeHead(tokens)

	// head 是 shell 变量/命令替换展开（$X / ${X} / $(...)）—— shell 会先展开再执行，
	// precheck 静态分析看不到展开后的真实命令名，直接拒（绕过 head 判定的唯一途径）。
	if indirectCmdRe.MatchString(tokens[0]) {
		return fmt.Errorf("%s", blockedMsg("indirect-cmd", head))
	}

	// shell debug flags: bash -x / -v / -o xtrace 打印展开值
	if err := precheckShellFlags(cmdStr, head); err != nil {
		return err
	}
	// stdin-shell: 无参 bash / echo | bash 从 stdin 读命令（dynamic exec）
	if err := precheckStdinShell(cmdStr, tokens, head); err != nil {
		return err
	}
	// special-env-assign: BASH_ENV / LD_PRELOAD / PROMPT_COMMAND 等特殊 env 名赋值
	if err := precheckSpecialEnvAssign(cmdStr, head); err != nil {
		return err
	}

	// bash/sh/dash -c <arg> 套娃：把 <arg> 当新命令递归 precheck，拦下"包一层 shell 绕过"的尝试
	if head == "bash" || head == "sh" || head == "dash" {
		if inner, ok := extractDashCArg(cmdStr); ok {
			return precheckCommand(inner, envMap)
		}
	}

	// head-secret-ref: tokens[0] 含 envMap key 的 $KEY 引用（报错回显泄露）
	if err := precheckHeadSecretRef(tokens, envMap); err != nil {
		return err
	}

	// 解释器内联脚本（python/node/perl/ruby/php/awk 等）：三条件 AND（env 访问 ∧ 输出 sink ∧
	// envMap key 字面量）才拦，避免误伤合法的 "读 env 喂给 SDK" 模式（如 client=Cli(os.environ['K'])）。
	// 同时叠加 script-read-sensitive：body 内文件读 API + .env / .skill_env 字面量即拒（envMap 无关）。
	if body, ok := extractInterpreterScript(cmdStr, head, tokens); ok {
		if err := precheckScriptBody(body, envMap); err != nil {
			return err
		}
		if err := precheckScriptReadSensitive(body); err != nil {
			return err
		}
	}
	// heredoc 内联（仅在 head 是解释器或 bash/sh/dash 时触发，避免给 cat << EOF 这种普通文档误判）
	if _, isInterp := scriptInterpreterCmds[head]; isInterp || head == "bash" || head == "sh" || head == "dash" {
		if body, ok := extractHeredocBody(cmdStr); ok {
			if err := precheckHeredocBody(body, head, envMap); err != nil {
				return err
			}
			if err := precheckScriptReadSensitive(body); err != nil {
				return err
			}
		}
	}
	// eval 关键字 — 无条件拒（LLM 自动生成场景下 eval 几无合法用途）
	if evalKeywordRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("eval", head))
	}
	// 间接展开 ${!VAR} — 无条件拒（绕过 $KEY 字面匹配的唯一语法路径）
	if indirectExpandRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("indirect-expand", head))
	}
	// 命令替换嵌枚举命令 $(env)/$(printenv)/... — 等价于直接执行枚举命令，复用 enumerator 标签
	if cmdSubstEnumRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("enumerator", head))
	}
	// 进程替换 <(env) / <(printenv) 语义等价命令替换嵌枚举，同样拒
	if processSubstEnumRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("enumerator", head))
	}
	// 枚举型命令：env/printenv/set/... 无差别打印所有环境变量，无条件拒
	if _, ok := secretEnumeratorCmds[head]; ok {
		return fmt.Errorf("%s", blockedMsg("enumerator", head))
	}
	// 中转变量赋值：任何 NAME=...$KEY... 形态一律拒（赋值步骤即拒，杜绝 N 跳链）
	if err := precheckRelayAssign(cmdStr, envMap); err != nil {
		return err
	}
	// find/xargs 跳板：`find ... -exec cat` / `... | xargs cat` 到敏感文件。
	// 放在 sensitiveFileRe 之前，让 tool-jump 标签比通用 sensitive-file 更精准。
	if err := precheckToolJump(cmdStr); err != nil {
		return err
	}
	// ln/cp/mv/rsync/install + .env / .skill_env 字面量（防别名化后再读）
	if err := precheckSensitiveFileHandler(cmdStr, head); err != nil {
		return err
	}
	// 敏感文件 / /proc/<pid>/environ
	if sensitiveFileRe.MatchString(cmdStr) || procEnvironRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("sensitive-file", head))
	}
	// 写脚本文件 + inline 内容含越权代码三条件；同时覆盖 heredoc body / 变量展开 target / shell rc 文件 target。
	// 放在 ref-printer 之前，因为 `echo '...' > /tmp/x.py` 里 echo 引用 $KEY 会先命中 ref-printer；
	// 但真实语义是"写含泄露代码的脚本"，用 script-write 标签更精准。
	if err := precheckScriptWrite(cmdStr, envMap); err != nil {
		return err
	}
	// 打印型命令 + $KEY 引用，两条件必须同时满足，否则放行（避免 echo "hello" 这种被错杀）
	if _, ok := refPrinterCmds[head]; ok && referencesSecret(cmdStr, envMap) {
		return fmt.Errorf("%s", blockedMsg("ref-printer", head))
	}
	// 打印型 head + `.*` / `.?<any>` 之类宽泛 glob → 拒（防 cat .* 展开到 .env）
	if err := precheckBroadGlob(cmdStr, head); err != nil {
		return err
	}
	// base64 -d / xxd -r 解码命令：常用于 dynamic exec 后接 pipe 到 shell → 拒
	if err := precheckDecodePipe(cmdStr, head); err != nil {
		return err
	}
	// trap 'code' EXIT / SIGxxx 延迟执行：body 走 bash 三条件（防 trap 里 echo $KEY 泄露）
	if err := precheckTrapBody(cmdStr, head, envMap); err != nil {
		return err
	}
	return nil
}

// buildQuoteState 返回与 cmdStr 等长的 byte slice，标注每个字节的引号态：
//
//	0 = 引号外
//	1 = 单引号内（bash 单引号不处理转义）
//	2 = 双引号内
//
// 用途：让"按位置扫描字符串"的规则（NAME= 赋值、重定向 target、$-prefix token 判定）
// 能跳过引号内区间，避免把 echo "NAME=$KEY" / awk '$0' 里的字面内容误判为 shell 语法。
//
// 简化：不处理双引号内 \" 转义、$'...' ANSI-C quoting。最坏结果是引号态分类偏差，
// 不引入新泄露路径。
func buildQuoteState(cmdStr string) []byte {
	state := make([]byte, len(cmdStr))
	var mode byte // 0 out, 1 single, 2 double
	for i := 0; i < len(cmdStr); i++ {
		c := cmdStr[i]
		switch mode {
		case 0:
			switch c {
			case '\'':
				mode = 1
			case '"':
				mode = 2
			}
		case 1:
			if c == '\'' {
				mode = 0
			}
		case 2:
			if c == '"' {
				mode = 0
			}
		}
		state[i] = mode
	}
	return state
}

// maskQuotesToSpaces 返回一个与 cmdStr 等长的字符串，其中所有引号内的非引号、非换行字符
// 被替换为空格，引号字符本身和引号外内容保持不变。用于让"按位置匹配正则"的规则
// （如 extractRedirectTargets 找 `>` / `>>`）跳过引号内的语法字符。
//
// 位置保持：masked[i] 与 cmdStr[i] 一一对应，可直接把 masked 上的正则匹配位置映射回 cmdStr。
func maskQuotesToSpaces(cmdStr string) string {
	qs := buildQuoteState(cmdStr)
	out := []byte(cmdStr)
	for i, s := range qs {
		if s == 0 {
			continue
		}
		c := out[i]
		if c == '\'' || c == '"' || c == '\n' {
			continue
		}
		out[i] = ' '
	}
	return string(out)
}

// splitFieldsRespectQuotes 按空白分词，单/双引号包裹的整块视为一个 token
// （引号本身保留在 token 中，供下游做 $-prefix 判定）。
//
// 用途：替代 strings.Fields 供 precheckScriptFile 使用——避免 `awk 'BEGIN{print $0}'`
// 被拆散、awk 的 `$0`/`$1` 字段引用被误判为 shell 变量展开。
//
// 简化：不处理 \" / \' 转义、$'...' ANSI-C quoting。分词偏差最坏影响是"误合并/误拆分"
// 一小段 token，不引入新泄露路径。
func splitFieldsRespectQuotes(cmdStr string) []string {
	var out []string
	var cur []byte
	var mode byte // 0 out, 1 single, 2 double
	flush := func() {
		if len(cur) > 0 {
			out = append(out, string(cur))
			cur = cur[:0]
		}
	}
	for i := 0; i < len(cmdStr); i++ {
		c := cmdStr[i]
		switch mode {
		case 0:
			switch c {
			case '\'':
				mode = 1
				cur = append(cur, c)
			case '"':
				mode = 2
				cur = append(cur, c)
			case ' ', '\t', '\n', '\r':
				flush()
			default:
				cur = append(cur, c)
			}
		case 1:
			cur = append(cur, c)
			if c == '\'' {
				mode = 0
			}
		case 2:
			cur = append(cur, c)
			if c == '"' {
				mode = 0
			}
		}
	}
	flush()
	return out
}

// extractQuotedStrings 抽取 cmdStr 中所有单/双引号包裹的子串（不含引号本身）。
// 简化解析：不处理转义引号嵌套、$'...' ANSI-C quoting。
func extractQuotedStrings(cmdStr string) []string {
	var out []string
	var i int
	for i < len(cmdStr) {
		c := cmdStr[i]
		if c != '\'' && c != '"' {
			i++
			continue
		}
		// 找配对引号
		end := strings.IndexByte(cmdStr[i+1:], c)
		if end < 0 {
			break
		}
		out = append(out, cmdStr[i+1:i+1+end])
		i += 1 + end + 1
	}
	return out
}

// extractRedirectTargets 抽取 cmdStr 中所有重定向目标文件路径（> / >> / tee <flags...> <target> 后的 token）。
//
// 引号感知：先把引号内的非引号字符换成空格再跑正则，避免把 awk `'BEGIN{print $0 > "f"}'`
// 或 `echo "> path"` 里的 `>` 误识别成 shell 重定向。masked 与 cmdStr 位置一一对应，
// 匹配到的 target 直接取 masked 上的 substring；引号内的 target 因内容已被 mask，
// 天然不会作为有意义路径出现。
//
// 简化：tee 仅识别紧跟其后的首个非 flag token；不处理重定向到 /dev/fd/N 等特殊文件。
func extractRedirectTargets(cmdStr string) []string {
	masked := maskQuotesToSpaces(cmdStr)
	var out []string
	// > target / >> target
	redirRe := regexp.MustCompile(`>>?\s*([^\s;&|()<>]+)`)
	for _, m := range redirRe.FindAllStringSubmatch(masked, -1) {
		if len(m) > 1 && m[1] != "" {
			out = append(out, m[1])
		}
	}
	// tee [flags...] target
	teeRe := regexp.MustCompile(`\btee\b(?:\s+-[a-zA-Z]+)*\s+([^\s;&|()<>]+)`)
	for _, m := range teeRe.FindAllStringSubmatch(masked, -1) {
		if len(m) > 1 && m[1] != "" {
			out = append(out, m[1])
		}
	}
	return out
}

// extractHeredocBody 从 cmdStr 提取 heredoc body。
// 算法：
//  1. 用 heredocStartRe 找到 "<<TAG" 起始处，捕获 TAG 名
//  2. 从起始处往后找首个换行（heredoc body 起点）
//  3. 在 body 起点之后扫描 "\nTAG\n" / "\nTAG$" / "\nTAG;" 等结束定界（前面允许空白用于 <<- 缩进版）
//
// 不展开变量；C3 字面量仍可命中。简化：不处理多 heredoc 嵌套、heredoc 内的字符串恰好等于 TAG 的边缘情况。
func extractHeredocBody(cmdStr string) (string, bool) {
	loc := heredocStartRe.FindStringSubmatchIndex(cmdStr)
	if loc == nil {
		return "", false
	}
	tag := cmdStr[loc[2]:loc[3]]
	if tag == "" {
		return "", false
	}
	// 找起始 token 之后的首个换行
	bodyStart := strings.IndexByte(cmdStr[loc[1]:], '\n')
	if bodyStart < 0 {
		return "", false
	}
	bodyStart += loc[1] + 1
	// 在 body 起点之后扫描结束定界 "\n<空白>*TAG<空白>*(\n|$|;|&&|||)"
	rest := cmdStr[bodyStart:]
	for offset := 0; offset < len(rest); {
		idx := strings.Index(rest[offset:], "\n"+tag)
		if idx < 0 {
			// 也允许 body 单独占一行而 TAG 在文件末尾（无前导换行）
			break
		}
		absIdx := offset + idx
		// TAG 后面必须是空白/换行/分隔符/字符串结尾
		afterTag := absIdx + 1 + len(tag)
		if afterTag >= len(rest) {
			return rest[:absIdx], true
		}
		c := rest[afterTag]
		if c == '\n' || c == ' ' || c == '\t' || c == ';' || c == '&' || c == '|' || c == '\r' {
			return rest[:absIdx], true
		}
		offset = absIdx + 1
	}
	return "", false
}

// precheckHeredocBody 在 heredoc body 上跑三条件 AND（env 访问 ∧ 输出 sink ∧ envMap key 字面量）。
// bash/sh/dash 用 bashEnvAccessRe (识别 $KEY)；其它解释器沿用 scriptEnvAccessRe (os.environ / process.env / 等)。
// 命中即返回 [BLOCKED:heredoc]。
func precheckHeredocBody(body, head string, envMap map[string]string) error {
	if body == "" || len(envMap) == 0 {
		return nil
	}
	if head == "bash" || head == "sh" || head == "dash" {
		if !bashEnvAccessRe.MatchString(body) {
			return nil
		}
	} else {
		if !scriptEnvAccessRe.MatchString(body) {
			return nil
		}
	}
	if !scriptSinkRe.MatchString(body) {
		return nil
	}
	for k := range envMap {
		if k == "" {
			continue
		}
		matched, err := regexp.MatchString(`\b`+regexp.QuoteMeta(k)+`\b`, body)
		if err == nil && matched {
			return fmt.Errorf("%s", blockedMsg("heredoc", head))
		}
	}
	return nil
}

// precheckScriptWrite 拦截"把含越权代码的脚本写入文件"的命令。
// 触发：cmdStr 中存在重定向到脚本后缀文件（scriptFileExtRe）/ shell rc 文件（shellRcFileRe）
// / 变量展开 target `$VAR` 之一，AND quoted body 或 heredoc body 命中越权代码。
//
// 具体覆盖的三种 target：
//   - 脚本后缀文件（.py/.sh/.js/... 等）+ inline body 三条件命中
//   - shell rc 文件（.bashrc/.zshrc/.profile 等）+ inline body 三条件命中
//   - 变量展开 target（$VAR / ${VAR} / $(...)）+ inline body 三条件命中（无论后缀，静态无法解析路径）
//
// 其中 inline body 扫描同时覆盖 quoted string 和 heredoc body（`cat > x.py << EOF ... EOF`）。
//
// 命中返回 [BLOCKED:script-write]（第一优先）或原始规则标签。
func precheckScriptWrite(cmdStr string, envMap map[string]string) error {
	targets := extractRedirectTargets(cmdStr)
	if len(targets) == 0 {
		return nil
	}
	hasSuspiciousTarget := false
	for _, t := range targets {
		// 脚本后缀 target
		if scriptFileExtRe.MatchString(t) {
			hasSuspiciousTarget = true
			break
		}
		// shell rc 文件 target
		if shellRcFileRe.MatchString(t) {
			hasSuspiciousTarget = true
			break
		}
		// 变量展开 target
		if strings.HasPrefix(t, "$") {
			hasSuspiciousTarget = true
			break
		}
	}
	if !hasSuspiciousTarget {
		return nil
	}

	// 收集所有待扫描的 body：quoted string + heredoc body
	bodies := extractQuotedStrings(cmdStr)
	if hbody, ok := extractHeredocBody(cmdStr); ok {
		bodies = append(bodies, hbody)
	}
	for _, body := range bodies {
		if len(envMap) > 0 {
			// 解释器语义：os.environ / process.env + print/console.log + envMap key 字面量
			if err := precheckScriptBody(body, envMap); err != nil {
				return fmt.Errorf("%s", blockedMsg("script-write", "redirect"))
			}
			// bash 语义：$KEY / ${KEY} + echo/printf + envMap key 字面量
			// 覆盖 `echo 'echo $MY_TOKEN' > /tmp/x.sh` 这类"往脚本里写 shell 泄露命令"
			if err := precheckHeredocBody(body, "bash", envMap); err != nil {
				return fmt.Errorf("%s", blockedMsg("script-write", "redirect"))
			}
		}
		if err := precheckScriptReadSensitive(body); err != nil {
			return err
		}
	}
	return nil
}

// precheckRelayAssign 拦截"把 envMap key 赋值给另一个变量"的语句。
// 触发：cmdStr 中存在 NAME=<RHS>，且 RHS 中引用了 envMap 中任一 key（$KEY 或 ${KEY}）。
// 命中即返回 [BLOCKED:relay-assign]，不打印 RHS 内容。
//
// 设计：在赋值那一步就拒，杜绝中转变量的 N 跳链——A=$T 即拒，根本到不了 B=$A。
// FP 控制：
//   - 引号感知：若 NAME= 命中位置在单/双引号内（如 `echo "NAME=\$KEY"` 的字面串），跳过。
//     bash 引号内的 `NAME=...` 不构成赋值，`\$KEY` 也是转义字面 `$`（不展开、不泄露 value）。
//   - 所有中转变量在合法场景里都有 inline 等价写法（curl -H "X: $T" 替代 H=$T; curl -H "$H"），
//     拒赋值仅影响代码风格/可读性，功能完整性不受影响。
func precheckRelayAssign(cmdStr string, envMap map[string]string) error {
	if len(envMap) == 0 {
		return nil
	}
	qs := buildQuoteState(cmdStr)
	for _, m := range relayAssignRe.FindAllStringSubmatchIndex(cmdStr, -1) {
		// m 索引：0/1 整体匹配，2/3 前导边界 group，4/5 NAME group
		if len(m) < 6 || m[5] < 0 {
			continue
		}
		// NAME 起点在引号内 → 属于引号内字面串（echo/printf 输出等），跳过
		if m[4] < len(qs) && qs[m[4]] != 0 {
			continue
		}
		// "=" 在 NAME 之后；RHS 从 m[5]+1 (= 之后的位置) 开始
		rhs := extractAssignmentRHS(cmdStr, m[5]+1)
		if referencesSecret(rhs, envMap) {
			return fmt.Errorf("%s", blockedMsg("relay-assign", "assignment"))
		}
	}
	return nil
}

// extractAssignmentRHS 从 cmdStr[start:] 抽取赋值的 RHS 部分。
// 支持单/双引号包裹（引号内不切）；裸值到下个 shell 分隔符（空白 / ; / && / || / | / ) / 行尾）为止。
// 简化：不处理转义引号嵌套、$(...) 内部嵌套引号。
func extractAssignmentRHS(cmdStr string, start int) string {
	if start >= len(cmdStr) {
		return ""
	}
	// 引号包裹：取到配对引号为止
	c := cmdStr[start]
	if c == '\'' || c == '"' {
		end := strings.IndexByte(cmdStr[start+1:], c)
		if end < 0 {
			return cmdStr[start+1:] // 不配对，取剩余
		}
		return cmdStr[start+1 : start+1+end]
	}
	// 裸值：到下个 shell 分隔符为止
	i := start
	for i < len(cmdStr) {
		ch := cmdStr[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == ';' || ch == '|' || ch == '&' || ch == ')' {
			break
		}
		i++
	}
	return cmdStr[start:i]
}

// referencesSecret 判断 cmd 中是否引用了 envMap 中的任一 key。
// 覆盖 bash 参数展开全变体：
//   - $KEY / ${KEY}  基础形态
//   - ${KEY:-x} / ${KEY:=x} / ${KEY:?x} / ${KEY:+x}  默认值展开
//   - ${KEY#*} / ${KEY##*} / ${KEY%*} / ${KEY%%*}  前后缀移除
//   - ${KEY:0:5} / ${KEY:5}  子串
//   - ${KEY/A/B} / ${KEY//A/B}  替换
//   - ${KEY^^} / ${KEY,,} / ${KEY^} / ${KEY,}  大小写
//   - ${KEY@Q} / ${KEY@P} / ${KEY@U} / ${KEY@A}  @-operator (@P 展开成 value)
//   - ${!KEY} 间接展开也顺带命中（虽有独立 indirect-expand 规则拦，此处也算引用）
func referencesSecret(cmd string, envMap map[string]string) bool {
	for k := range envMap {
		if k == "" {
			continue
		}
		// $KEY\b：边界要求避免 $TOKEN 误命中 $TOKENS
		pattern := `\$` + regexp.QuoteMeta(k) + `\b`
		if matched, err := regexp.MatchString(pattern, cmd); err == nil && matched {
			return true
		}
		// ${KEY} 基础形态
		if strings.Contains(cmd, "${"+k+"}") {
			return true
		}
		// ${KEY<expansion-op>...} — 覆盖所有 bash 参数展开变体
		// KEY 后紧跟 bash 展开操作符或字符串结尾
		expandPattern := `\$\{[!]?` + regexp.QuoteMeta(k) + `[\s}:#%/,\^@]`
		if matched, err := regexp.MatchString(expandPattern, cmd); err == nil && matched {
			return true
		}
	}
	return false
}

// extractDashCArg 从 `bash -c <arg>` 中提取 <arg>，支持单/双引号或裸 token。
// 实现仅靠字符串切分，复杂 quoting（如转义引号嵌套）是已知盲点。
func extractDashCArg(cmd string) (string, bool) {
	_, after, found := strings.Cut(cmd, "-c")
	if !found {
		return "", false
	}
	rest := strings.TrimLeft(after, " \t")
	if rest == "" {
		return "", false
	}
	// 去掉首尾配对引号
	if (rest[0] == '\'' || rest[0] == '"') && len(rest) >= 2 && rest[len(rest)-1] == rest[0] {
		return rest[1 : len(rest)-1], true
	}
	return rest, true
}

// extractInterpreterScript 从命令字符串提取解释器内联脚本 body。
//   - 对 scriptInterpreterCmds 内的 head：在 cmdStr 中找首个匹配的 flag 紧跟空白后的下一参数（含配对引号）。
//   - 对 awk/gawk/mawk：脚本是首个位置参数（跳过 -F<sep> / -v <var=val> / -f <file> 这些 flag）。
//   - body 若被配对单/双引号包裹则剥掉一层；不命中或 body 为空返回 ("", false)。
//
// 实现：用 regexp 一次性匹配 `flag\s+('body'|"body"|bare)`；复杂 shell quoting（转义嵌套、shell 续行、
// 变量替换、--flag=value 等长选项写法）是已知盲点。
func extractInterpreterScript(cmdStr, head string, _ []string) (string, bool) {
	// awk 走位置参数语义
	if head == "awk" || head == "gawk" || head == "mawk" {
		return extractAwkProgram(cmdStr)
	}
	flags, ok := scriptInterpreterCmds[head]
	if !ok {
		return "", false
	}
	for _, f := range flags {
		// 匹配 "<空白><flag><空白>('body'|"body"|bare)"。flag 前必须有空白避免误命中 head 子串。
		// (?s) 让 . 跨行（罕见但 heredoc 替换后可能出现）。
		pattern := `\s` + regexp.QuoteMeta(f) + `\s+(?s)(?:'([^']*)'|"([^"]*)"|(\S+))`
		re := regexp.MustCompile(pattern)
		if m := re.FindStringSubmatch(" " + cmdStr); m != nil {
			for i := 1; i < len(m); i++ {
				if m[i] != "" {
					return m[i], true
				}
			}
		}
	}
	return "", false
}

// extractAwkProgram 跳过 awk 的 flag 后取首个位置参数作为 body。
// 已知盲点：仅识别独立 flag（-F,/-v X=1）形式，--field-separator= 等长格式不识别。
func extractAwkProgram(cmdStr string) (string, bool) {
	// 用正则一次性匹配：awk 头 + 零或多组（flag + 其值） + 首位置参数
	// flag 形态：-F<x>（粘连）/ -F <x> / -v <var=val> / -f <file> / 其它 -X
	// 位置参数：'body' / "body" / bare
	re := regexp.MustCompile(
		`^(?:awk|gawk|mawk)\b(?:\s+(?:-F\S*|-[vf]\s+\S+|-\S+))*\s+(?:'([^']*)'|"([^"]*)"|(\S+))`)
	m := re.FindStringSubmatch(cmdStr)
	if m == nil {
		return "", false
	}
	for i := 1; i < len(m); i++ {
		if m[i] != "" {
			return m[i], true
		}
	}
	return "", false
}

// precheckScriptBody 对解释器脚本 body 做三条件 AND 拦截，覆盖内联 (-c/-e)、
// heredoc、落盘 .py 三条 chain。三者全中返回 [BLOCKED:script-body]，任一缺失放行：
//   - 命中 scriptEnvAccessRe（脚本读取了 process env）
//   - 命中 scriptSinkRe（脚本会向 stdout/stderr 输出）
//   - body 中明文出现 envMap 任一 key 的 \b 边界匹配（确认引用了已注入的 skill key）
//
// 早期版本这里首步会跑 precheckScriptOpen —— 只要 body 出现 open( / .read( /
// fs.writeFile* 等 file I/O 内建就一律拒 —— 目的是防"open('.'+'env')" 拼装字面量
// 绕过 sensitive-file 检查。但该规则把 LLM 在 sandbox 写代码文件（heredoc /
// `cat > x.py <<EOF` / `python -c "with open(fp,'w')..."`）这一高频合法场景 100% 命中，
// 已删除；.env / .skill_env 字面量读取仍由 precheckScriptReadSensitive 兜底。
func precheckScriptBody(body string, envMap map[string]string) error {
	if body == "" || len(envMap) == 0 {
		return nil
	}
	if !scriptEnvAccessRe.MatchString(body) {
		return nil
	}
	if !scriptSinkRe.MatchString(body) {
		return nil
	}
	for k := range envMap {
		if k == "" {
			continue
		}
		matched, err := regexp.MatchString(`\b`+regexp.QuoteMeta(k)+`\b`, body)
		if err == nil && matched {
			return fmt.Errorf("%s", blockedMsg("script-body", "interpreter"))
		}
	}
	return nil
}

// precheckToolJump 检测 find / xargs 跳板到打印命令读取敏感文件的手法。
// 触发条件（任一命中）：
//   - `find ... -exec <printer>` + `-name <env pattern>`，或整条 cmdStr 命中 sensitiveFileRe
//   - `... | xargs <printer>` + pipeline 前段用 echo/printf 输出 env 路径字面量
//
// 命中即返回 [BLOCKED:tool-jump]。envMap 无关（护 .env / .skill_env 本身）。
func precheckToolJump(cmdStr string) error {
	hasFind := findExecReaderRe.MatchString(cmdStr)
	hasXargs := xargsReaderRe.MatchString(cmdStr)
	if !hasFind && !hasXargs {
		return nil
	}
	// 情况 1：find -exec printer + -name 指向 env 家族
	if hasFind && findNamePatternEnvRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("tool-jump", "find"))
	}
	// 情况 2：find -exec printer + 命令体命中 sensitiveFileRe（如 find -name .env* 已被上面命中，
	// 但如 find /workspace -path '*/.env' 走这里）
	if hasFind && sensitiveFileRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("tool-jump", "find"))
	}
	// 情况 3：pipeline 前段 echo/printf 敏感路径 → xargs cat
	if hasXargs && echoPipeEnvRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("tool-jump", "xargs"))
	}
	return nil
}

// precheckScriptReadSensitive 检测"解释器脚本读取 .env / .skill_env 文件"的路径。
// 触发条件（两条件 AND）：
//   - body 中出现文件读 API（open / .read( / fs.readFile / File.read / fopen / ...）
//   - body 中出现 .env / .skill_env 路径字面量
//
// 命中即返回 [BLOCKED:script-read-sensitive]。envMap 是否为空都拒：
// workspace/.env（含 OPENAI_API_KEY）的保护与 skill_env 独立，即使无 skill var 也不该被解释器直接读。
//
// 已知残留：字符串拼装（'.skill_e'+'nv'）/ chr / base64 编码路径静态分析穷不完，列入残留清单。
func precheckScriptReadSensitive(body string) error {
	if body == "" {
		return nil
	}
	if !scriptFileReadRe.MatchString(body) {
		return nil
	}
	if !scriptSensitivePathLiteralRe.MatchString(body) {
		return nil
	}
	return fmt.Errorf("%s", blockedMsg("script-read-sensitive", "interpreter"))
}

// blockedMsg 拼接给 LLM 的拒绝提示。固定文本不含 value，便于 LLM 模式匹配后调整策略。
func blockedMsg(rule, head string) string {
	return fmt.Sprintf(
		"[BLOCKED:%s] Command %q rejected: would print or dump skill environment variables. "+
			"Use $KEY references directly in network commands (curl, etc.) or in file writes to external systems, "+
			"but NEVER via echo / printf / cat / env / printenv / encode commands. See SKILL.md.",
		rule, head,
	)
}

// ─── 新增 helper（head 规范化 + shell/token 层规则） ───

// normalizeHead 对 tokens[0] 做规范化，剥掉常见"绕 head 判定"的语法糖，返回 (真实 head, 是否可继续 head 判定)。
// head 判定入口前置逻辑，闭合多个绕过变体：
//   - "X=fake env"（前缀赋值）→ 剥赋值前缀，返回 env
//   - "command env" / "exec env" / "builtin env"（bypass alias 内建）→ 剥外壳，返回 env
//   - "(env)" / "{ env; }"（子 shell / 命令组）→ strip 前导 (/{
//   - "\env"（backslash 绕 alias）→ strip 反斜杠
//
// 返回值：normalize 后的 head（小写、剥 /usr/bin/ 前缀）；若无法规范化返回原始 tokens[0] 处理结果。
// 递归深度限制 4 层，防止病态输入死循环。
func normalizeHead(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	idx := 0
	depth := 0
	for depth < 4 && idx < len(tokens) {
		tok := tokens[idx]
		// 剥前导 (/{
		tok = strings.TrimLeft(tok, "({")
		// 剥反斜杠
		tok = strings.TrimPrefix(tok, `\`)
		// 剥 /usr/bin/ /bin/
		tok = strings.TrimPrefix(tok, "/usr/bin/")
		tok = strings.TrimPrefix(tok, "/bin/")
		lower := strings.ToLower(tok)
		// NAME=value 前缀赋值：LHS 必须是合法 env name
		if eqIdx := strings.Index(tok, "="); eqIdx > 0 {
			lhs := tok[:eqIdx]
			if isValidEnvName(lhs) {
				idx++
				depth++
				continue
			}
		}
		// command / exec / builtin 外壳
		if lower == "command" || lower == "exec" || lower == "builtin" {
			idx++
			depth++
			continue
		}
		return lower
	}
	return ""
}

// isValidEnvName 判断 s 是否符合 POSIX env name（[A-Za-z_][A-Za-z0-9_]*）。
func isValidEnvName(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if i == 0 {
			if c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
				return false
			}
			continue
		}
		if c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// precheckShellFlags 检测 bash/sh/dash 的 debug flag（-x / -v / -o xtrace / --xtrace / --verbose）。
// 这些 flag 会 stderr 打印每条命令展开值，包含 $KEY 展开结果 → 泄露。
// LLM 自动化场景无合法用途；合法 debug 用 set -x 内联已被枚举命令规则拦下。
func precheckShellFlags(cmdStr, head string) error {
	if head != "bash" && head != "sh" && head != "dash" {
		return nil
	}
	if shellDebugFlagRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("shell-flags", head))
	}
	return nil
}

// precheckStdinShell 检测无参 bash / echo | bash 等 "从 stdin 读命令" 的 dynamic exec 手法。
// head=bash/sh/dash 无位置参数（无脚本文件）+ 无 -c → 拒；
// pipeline "... | bash" 尾接 shell → 拒。
func precheckStdinShell(cmdStr string, tokens []string, head string) error {
	// pipeline 版
	if pipeShellRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("stdin-shell", head))
	}
	// 单独 head 版
	if head != "bash" && head != "sh" && head != "dash" {
		return nil
	}
	// 检查 tokens[1:] 里是否有 "-c" 或位置参数（脚本文件）
	skipNextArg := map[string]bool{"-c": true, "-o": true, "--rcfile": true, "--init-file": true}
	hasPositional := false
	hasDashC := false
	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "-c" {
			hasDashC = true
			break
		}
		if strings.HasPrefix(tok, "-") {
			if skipNextArg[tok] {
				i++
			}
			continue
		}
		hasPositional = true
		break
	}
	if !hasDashC && !hasPositional {
		return fmt.Errorf("%s", blockedMsg("stdin-shell", head))
	}
	return nil
}

// precheckSpecialEnvAssign 检测赋值 LHS 是 BASH_ENV / ENV / PROMPT_COMMAND / LD_* 等特殊 env 名。
// 这些 env 的语义是"影响 shell 启动/运行时行为"，LLM 无合法用途；
// 攻击 pattern: BASH_ENV=/tmp/rc sh -c ':' 会 source /tmp/rc。
// 一律拒（不管 RHS 引用什么）。
func precheckSpecialEnvAssign(cmdStr, head string) error {
	if specialEnvAssignRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("special-env-assign", head))
	}
	return nil
}

// precheckHeadSecretRef 检测 tokens[0] 里出现 envMap 中任一 key 的 $KEY / ${KEY} 引用。
// 攻击 pattern: nonexistent_$MY_TOKEN 会展开后 bash 报错 "nonexistent_<VALUE>: command not found"
// 到 stderr，值泄露给 LLM。合法命令名不含 secret。
//
// 排除：tokens[0] 是 NAME=... 形态的赋值（如 MY=$MY_TOKEN），这类交给 relay-assign 规则处理；
// 此处只关心"看起来像命令名"的 head token 里含 secret 引用。
func precheckHeadSecretRef(tokens []string, envMap map[string]string) error {
	if len(tokens) == 0 || len(envMap) == 0 {
		return nil
	}
	tok := tokens[0]
	// 排除赋值形态（NAME=... 由 relay-assign 处理）
	if eqIdx := strings.Index(tok, "="); eqIdx > 0 && isValidEnvName(tok[:eqIdx]) {
		return nil
	}
	if referencesSecret(tok, envMap) {
		return fmt.Errorf("%s", blockedMsg("head-secret-ref", tok))
	}
	return nil
}

// precheckTrapBody 检测 trap 命令的 body 是否含 bash 三条件 (env 访问 + sink + envMap key)。
// trap 'echo $KEY' EXIT 在 EXIT 触发时执行 body，body 展开 $KEY 并输出。
// 复用 precheckHeredocBody(head="bash") 逻辑。
func precheckTrapBody(cmdStr, head string, envMap map[string]string) error {
	if head != "trap" {
		return nil
	}
	if len(envMap) == 0 {
		return nil
	}
	for _, body := range extractQuotedStrings(cmdStr) {
		if perr := precheckHeredocBody(body, "bash", envMap); perr != nil {
			return fmt.Errorf("%s", blockedMsg("trap-body", head))
		}
	}
	return nil
}

// precheckBroadGlob 检测 refPrinter head + `.*` / `.?<any>` 之类宽泛 glob。
// cat .* 展开为所有 . 起头文件（含 .env / .skill_env）。合法用途极少，拒。
// FP 边界：`ls .*` 常见但 ls 不在 refPrinter → 天然放行。
func precheckBroadGlob(cmdStr, head string) error {
	if _, ok := refPrinterCmds[head]; !ok {
		return nil
	}
	if broadGlobRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("broad-glob", head))
	}
	return nil
}

// precheckSensitiveFileHandler 检测 ln/cp/mv/rsync/install + .env / .skill_env 字面量。
// 攻击 pattern: ln -s .skill_env.json /tmp/x; cat /tmp/x（cat /tmp/x 不含 .env 绕敏感文件正则）。
// 由文件操作命令 + 敏感文件字面量 → 拒，防止别名化绕过。
func precheckSensitiveFileHandler(cmdStr, head string) error {
	if !sensitiveFileHandlerRe.MatchString(head) {
		return nil
	}
	if sensitiveFileRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("sensitive-file", head))
	}
	return nil
}

// precheckDecodePipe 检测 base64 -d / xxd -r 解码命令。
// base64 -d <<< 'Y2F0IC5lbnY=' | sh 之类编码后管道执行手法。
// LLM 自动化场景无合法用途；无差别拒。
func precheckDecodePipe(cmdStr, head string) error {
	if base64DecodeRe.MatchString(cmdStr) || xxdRevertRe.MatchString(cmdStr) {
		return fmt.Errorf("%s", blockedMsg("decode-pipe", head))
	}
	return nil
}
