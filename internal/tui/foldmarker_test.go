package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithFoldMarker(t *testing.T) {
	assert.True(t, strings.HasPrefix(stripANSI(withFoldMarker("  x", false)), "▾ x"))
	assert.True(t, strings.HasPrefix(stripANSI(withFoldMarker("  x", true)), "▸ x"))
}

func TestRenderCursorLineKeepsText(t *testing.T) {
	m := DetailModel{selectCol: 2}
	out := m.renderCursorLine([]rune("hello"), 0, 3, true)
	assert.Equal(t, "hello", stripANSI(out))
	assert.Contains(t, out, "\x1b[", "cursor cell should be styled")
}

func TestRenderCursorLineAtEnd(t *testing.T) {
	m := DetailModel{selectCol: 5}
	out := m.renderCursorLine([]rune("hello"), 0, 5, true)
	assert.Equal(t, "hello ", stripANSI(out))
}

func TestFoldRenderShowsMarkers(t *testing.T) {
	m := foldTestModel()
	out := stripANSI(m.View())
	assert.Contains(t, out, "▾", "expanded foldable nodes show a ▾ marker")

	m.respCollapsed[2] = true
	out = stripANSI(m.View())
	assert.Contains(t, out, "▸", "collapsed nodes show a ▸ marker")
}
