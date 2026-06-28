package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/parser"
)

func TestEditorPasteIntoURLFieldStripsNewline(t *testing.T) {
	m := NewEditorModel()
	m.focus = fieldURL
	m, _ = m.Update(tea.PasteMsg{Content: "https://api.example.com/x\n"})
	assert.Equal(t, "https://api.example.com/x", m.url.String())
}

func TestEditorPasteIntoTimeoutFiltersDigits(t *testing.T) {
	m := NewEditorModel()
	m.focus = fieldTimeout
	m, _ = m.Update(tea.PasteMsg{Content: "3a0s"})
	assert.Equal(t, "30", m.timeoutSecs.String())
}

func TestEditorPasteMultiLineBody(t *testing.T) {
	m := NewEditorModel()
	m.focus = fieldBody
	m, _ = m.Update(tea.PasteMsg{Content: "{\n  \"a\": 1\n}"})
	assert.Equal(t, "{\n  \"a\": 1\n}", m.Request().Body)
}

func TestEditorPasteBodyAtCursorSplitsLine(t *testing.T) {
	m := NewEditorModel()
	m.focus = fieldBody
	le := newLineEdit("AB")
	le.pos = 1
	m.body[0] = le
	m, _ = m.Update(tea.PasteMsg{Content: "X\nY"})
	assert.Equal(t, "AX\nYB", m.Request().Body)
}

func TestSearchPasteAppendsSanitized(t *testing.T) {
	m := NewSearchModel()
	m, _ = m.Update(tea.PasteMsg{Content: "users\n"})
	assert.Contains(t, stripANSI(m.View()), "users")
}

func TestPromptPasteAppendsSanitized(t *testing.T) {
	m := NewPromptModel("Name", nil)
	m, _ = m.Update(tea.PasteMsg{Content: "my-file\n"})
	assert.Contains(t, stripANSI(m.View()), "my-file")
}

func TestDetailPasteIntoSearch(t *testing.T) {
	m := NewDetailModel("", parser.NewChainContext(), engine.NewCookieManager())
	m.searching = true
	m, _ = m.Update(tea.PasteMsg{Content: "token\n"})
	assert.Equal(t, "token", m.searchQuery)
}

func TestDetailPasteIntoJumpPath(t *testing.T) {
	m := NewDetailModel("", parser.NewChainContext(), engine.NewCookieManager())
	m.jumpingPath = true
	m, _ = m.Update(tea.PasteMsg{Content: "data.id\n"})
	assert.Equal(t, "data.id", m.jumpQuery)
}
