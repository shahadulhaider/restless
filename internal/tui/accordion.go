package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// sectionRange maps a section to its line range in the rendered output.
type sectionRange struct {
	sec   section
	start int
	end   int
}

// accordionSection describes one collapsible section in the accordion view.
type accordionSection struct {
	key      string // "1", "2", "3"
	label    string // "Body", "Headers", etc.
	summary  string // one-line summary shown next to label when collapsed
	preview  string // 2-line preview content when collapsed (may be empty)
	content  string // full expanded content
	expanded bool
}

// accordionResult holds the rendered output and section line ranges.
type accordionResult struct {
	content string
	ranges  []sectionRange
}

// renderAccordionSections renders a list of sections as a unified scrollable accordion.
func renderAccordionSections(sections []accordionSection, width int) accordionResult {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4"))
	numStyle := lipgloss.NewStyle().Foreground(colorBorderActive)
	divider := dimStyle.Render(strings.Repeat("─", max(width-6, 10)))

	var sb strings.Builder
	var ranges []sectionRange
	lineIdx := 0

	countLines := func(s string) int {
		if s == "" {
			return 0
		}
		return strings.Count(s, "\n") + 1
	}

	for _, sec := range sections {
		start := lineIdx

		if sec.expanded {
			// ▼ [N] Label ────────
			sb.WriteString(headerStyle.Render("▼ ") + numStyle.Render("["+sec.key+"]") + headerStyle.Render(" "+sec.label) + " " + divider + "\n")
			lineIdx++
			if sec.content != "" {
				sb.WriteString(sec.content)
				lineIdx += countLines(sec.content)
			}
			sb.WriteString("\n")
			lineIdx++
		} else {
			// ▶ [N] Label ── summary
			summaryPart := ""
			if sec.summary != "" {
				summaryPart = " " + dimStyle.Render("── "+sec.summary)
			}
			sb.WriteString(headerStyle.Render("▶ ") + numStyle.Render("["+sec.key+"]") + headerStyle.Render(" "+sec.label) + summaryPart + "\n")
			lineIdx++
			if sec.preview != "" {
				sb.WriteString(sec.preview + "\n")
				lineIdx += countLines(sec.preview)
			}
			sb.WriteString("\n")
			lineIdx++
		}

		// Map section key to section enum
		var secEnum section
		switch sec.key {
		case "1":
			secEnum = sectionBody
		case "2":
			secEnum = sectionHeaders
		case "3":
			secEnum = sectionTiming
		}
		ranges = append(ranges, sectionRange{secEnum, start, lineIdx - 1})
	}

	return accordionResult{
		content: sb.String(),
		ranges:  ranges,
	}
}

// truncate shortens s to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// previewLines returns the first n lines of content, indented and dimmed.
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

// formatRequestHeaders returns request headers as indented text.
func formatRequestHeaders(headers []struct{ Key, Value string }) string {
	var sb strings.Builder
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB"))
	for _, h := range headers {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", keyStyle.Render(h.Key), h.Value))
	}
	return sb.String()
}
