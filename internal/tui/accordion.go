package tui

import (
	"strings"
)

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func previewLines(content string, n, maxWidth int) string {
	if content == "" {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	var sb strings.Builder
	for _, l := range lines {
		plain := stripANSI(l)
		if len(plain) > maxWidth-4 {
			l = plain[:maxWidth-4] + "..."
		}
		sb.WriteString("  " + dimStyle.Render(l) + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
