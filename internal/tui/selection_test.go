package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/parser"
)

func selKey(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: s, Code: []rune(s)[0]}
}

func TestRuneSlice(t *testing.T) {
	assert.Equal(t, "ell", runeSlice("hello", 1, 4))
	assert.Equal(t, "hello", runeSlice("hello", 0, -1))
	assert.Equal(t, "", runeSlice("hello", 3, 2))
	assert.Equal(t, "lo", runeSlice("hello", 3, 99))
	assert.Equal(t, "日本", runeSlice("日本語", 0, 2))
}

func TestSelectionBoundsNormalize(t *testing.T) {
	m := DetailModel{selectAnchor: 2, selectAnchorCol: 5, selectCursor: 1, selectCol: 3}
	loL, loC, hiL, hiC := m.selectionBounds()
	assert.Equal(t, 1, loL)
	assert.Equal(t, 3, loC)
	assert.Equal(t, 2, hiL)
	assert.Equal(t, 5, hiC)
}

func TestSelectedTextRangeSingleLine(t *testing.T) {
	m := DetailModel{
		selectLines:  []string{"hello world"},
		selectAnchor: 0, selectAnchorCol: 0,
		selectCursor: 0, selectCol: 4,
	}
	assert.Equal(t, "hello", m.selectedTextRange())
}

func TestSelectedTextRangeMultiLine(t *testing.T) {
	m := DetailModel{
		selectLines:  []string{"abcde", "fghij", "klmno"},
		selectAnchor: 0, selectAnchorCol: 2,
		selectCursor: 2, selectCol: 1,
	}
	assert.Equal(t, "cde\nfghij\nkl", m.selectedTextRange())
}

func TestLineSelCols(t *testing.T) {
	m := DetailModel{selectAnchor: 0, selectAnchorCol: 1, selectCursor: 0, selectCol: 3}
	a, b, ok := m.lineSelCols(0, 10)
	assert.True(t, ok)
	assert.Equal(t, 1, a)
	assert.Equal(t, 4, b)
	_, _, ok = m.lineSelCols(1, 10)
	assert.False(t, ok)
}

func TestNextPrevWordCol(t *testing.T) {
	r := []rune("hello world foo")
	assert.Equal(t, 6, nextWordCol(r, 0))
	assert.Equal(t, 12, nextWordCol(r, 6))
	assert.Equal(t, 6, prevWordCol(r, 11))
	assert.Equal(t, 0, prevWordCol(r, 5))
}

func TestUpdateSelectingMotions(t *testing.T) {
	m := NewDetailModel("", parser.NewChainContext(), engine.NewCookieManager())
	m.height = 20
	m.selecting = true
	m.selectLines = []string{"hello world", "second line"}

	for i := 0; i < 5; i++ {
		m, _ = m.updateSelecting(selKey("l"))
	}
	assert.Equal(t, 5, m.selectCol)

	m, _ = m.updateSelecting(selKey("w"))
	assert.Equal(t, 6, m.selectCol)

	m, _ = m.updateSelecting(selKey("$"))
	assert.Equal(t, 11, m.selectCol)

	m, _ = m.updateSelecting(selKey("j"))
	assert.Equal(t, 1, m.selectCursor)

	m, _ = m.updateSelecting(selKey("0"))
	assert.Equal(t, 0, m.selectCol)
}
