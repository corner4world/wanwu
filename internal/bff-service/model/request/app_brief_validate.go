package request

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	appNameMaxLen = 30
	appDescMaxLen = 200
)

var appNameRegex = regexp.MustCompile(`^[A-Za-z0-9.\x{4e00}-\x{9fa5}_-]+$`)

func validateAppBrief(b AppBriefConfig, subject string) error {
	name := strings.TrimSpace(b.Name)
	if name == "" {
		return fmt.Errorf("请填写%s名称", subject)
	}
	if utf8.RuneCountInString(name) > appNameMaxLen {
		return fmt.Errorf("%s名称须在%d字符以内", subject, appNameMaxLen)
	}
	if !appNameRegex.MatchString(name) {
		return fmt.Errorf("%s名称仅支持中文、英文、数字、下划线、中划线、英文(.)", subject)
	}
	if utf8.RuneCountInString(b.Desc) > appDescMaxLen {
		return fmt.Errorf("%s描述限制%d字符以内", subject, appDescMaxLen)
	}
	return nil
}
