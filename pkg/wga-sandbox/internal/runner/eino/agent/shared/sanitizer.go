package shared

import "strings"

// SanitizeForLog removes CRLF characters from untrusted input
// to prevent log forging attacks.
func SanitizeForLog(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\\r\\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}
