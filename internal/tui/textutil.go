package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// stripANSI removes ANSI escape sequences from s. It is a thin wrapper over
// ansi.Strip, which correctly handles CSI/OSC/SGR sequences and preserves
// printable runes (including multi-byte UTF-8).
func stripANSI(s string) string {
	return ansi.Strip(s)
}

// wrapLines hard-wraps each input line to maxWidth display cells, returning the
// resulting lines. Wrapping is grapheme- and width-aware (wide CJK/emoji are
// counted correctly and never split mid-rune) and preserves ANSI styling across
// the wrap. Lines that already fit are returned unchanged.
func wrapLines(lines []string, maxWidth int) []string {
	if maxWidth <= 0 {
		return lines
	}
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		// preserveSpace=true keeps leading indentation on continuation lines,
		// which matters for pretty-printed JSON/XML.
		w := ansi.Hardwrap(line, maxWidth, true)
		wrapped = append(wrapped, strings.Split(w, "\n")...)
	}
	return wrapped
}

// sanitizeInline strips characters unsuitable for single-line input fields:
// newlines, carriage returns, tabs, and other control runes, plus invalid
// UTF-8. Used when routing pasted text into single-line editors (URL, headers,
// search, prompt) so a stray newline can't corrupt the buffer.
func sanitizeInline(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == utf8.RuneError || unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
