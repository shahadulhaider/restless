package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestStripANSI(t *testing.T) {
	assert.Equal(t, "hi", stripANSI("\x1b[31mhi\x1b[0m"))
	assert.Equal(t, "plain", stripANSI("plain"))
	assert.Equal(t, "日本語", stripANSI("\x1b[32m日本語\x1b[0m"))
	// parity guard: colorized JSON must strip back to valid JSON for jsonpath.go
	colored := string([]byte("\x1b[34m{\x1b[0m\x1b[37m\"a\"\x1b[0m: \x1b[33m1\x1b[0m}"))
	assert.Equal(t, `{"a": 1}`, stripANSI(colored))
}

func TestWrapLinesShortUnchanged(t *testing.T) {
	in := []string{"short", "also short"}
	out := wrapLines(in, 40)
	assert.Equal(t, in, out)
}

func TestWrapLinesASCII(t *testing.T) {
	out := wrapLines([]string{"aaaabbbbcccc"}, 4)
	assert.Equal(t, []string{"aaaa", "bbbb", "cccc"}, out)
}

func TestWrapLinesMultibyteNoCorruption(t *testing.T) {
	// Each CJK glyph is width 2, so width-4 fits two per line.
	out := wrapLines([]string{"日本語ABC"}, 4)
	for _, l := range out {
		assert.True(t, utf8.ValidString(l), "wrapped line must be valid UTF-8: %q", l)
		assert.LessOrEqual(t, ansi.StringWidth(l), 4, "line exceeds max display width: %q", l)
	}
	// Round-trips to the original content when rejoined.
	assert.Equal(t, "日本語ABC", strings.Join(out, ""))
}

func TestWrapLinesPreservesANSIStyling(t *testing.T) {
	styled := "\x1b[31m" + strings.Repeat("x", 10) + "\x1b[0m"
	out := wrapLines([]string{styled}, 4)
	joined := strings.Join(out, "")
	assert.Contains(t, joined, "\x1b[", "ANSI escape codes must survive wrapping")
	// Stripped content is preserved across the wrap.
	assert.Equal(t, strings.Repeat("x", 10), stripANSI(joined))
}

func TestSanitizeInline(t *testing.T) {
	assert.Equal(t, "hello world", sanitizeInline("hello world"))
	assert.Equal(t, "abc", sanitizeInline("a\nb\rc"))
	assert.Equal(t, "ab", sanitizeInline("a\tb"))
	assert.Equal(t, "url", sanitizeInline("url\n"))
	assert.Equal(t, "日本語", sanitizeInline("日本語"))
}
