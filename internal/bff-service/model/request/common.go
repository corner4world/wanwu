package request

import (
	"regexp"
	"strings"
)

// filterBlankStrings 过滤掉切片里的空字符串和纯空白字符串
func filterBlankStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, s := range items {
		if strings.TrimSpace(s) != "" {
			result = append(result, s)
		}
	}
	return result
}

// 预编译恶意文件名检测正则表达式（不区分大小写）
var (
	// 匹配双管道后跟常见解释器名称（如 python, perl, sh 等）
	rePipeWithInterpreter = regexp.MustCompile(`(?i)\|\|.*(?:python|perl|ruby|sh|bash|cmd|powershell)`)
	// 匹配 python -c 直接执行代码
	rePythonC = regexp.MustCompile(`(?i)python\d?\s+-c\s+`)
	// 匹配 exec( 函数调用
	reExecCall = regexp.MustCompile(`(?i)exec\s*\(`)
	// 匹配命令替换：$(...) 或反引号 `
	reCmdSubst = regexp.MustCompile(`\$\x60|` + "`")
	// 匹配分号、与号、管道（单独使用或组合）后跟 shell 命令特征（可选，增强检测）
	reShellMeta = regexp.MustCompile(`(?i)[;&|]\s*(?:python|perl|ruby|sh|bash|wget|curl|nc|powershell)`)
)

type CommonCheck struct {
}

func (c *CommonCheck) Check() error {
	return nil
}

type PageSearch struct {
	PageSize int `json:"pageSize" form:"pageSize" validate:"required"`
	PageNo   int `json:"pageNo" form:"pageNo"`
}

type LoginEmailCheck struct {
	Email string `json:"email" validate:"required"` // 邮箱
	Code  string `json:"code" validate:"required"`  // 邮箱验证码
}

func (l *LoginEmailCheck) Check() error {
	return nil
}

type ChangeUserPasswordByEmail struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required"`
	Email       string `json:"email" validate:"required"` // 邮箱
	Code        string `json:"code" validate:"required"`  // 邮箱验证码
}

func (c *ChangeUserPasswordByEmail) Check() error {
	return nil
}

// IsMaliciousFilename 检查文件名是否包含潜在的恶意命令执行模式
// 返回 true 表示文件名危险，应拒绝上传；false 表示相对安全
func IsMaliciousFilename(filename string) bool {
	// 可选：仅检查文件名的基础部分，避免路径干扰
	// base := path.Base(filename)
	// 此处直接使用完整字符串，攻击者可能直接传入文件名
	if rePipeWithInterpreter.MatchString(filename) {
		return true
	}
	if rePythonC.MatchString(filename) {
		return true
	}
	if reExecCall.MatchString(filename) {
		return true
	}
	if reCmdSubst.MatchString(filename) {
		return true
	}
	if reShellMeta.MatchString(filename) {
		return true
	}
	return false
}
