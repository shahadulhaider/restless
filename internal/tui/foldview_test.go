package tui

import (
	"strings"
	"testing"

	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

func foldTestModel() DetailModel {
	zone.NewGlobal()
	m := NewDetailModel("", parser.NewChainContext(), engine.NewCookieManager())
	m.width = 80
	m.height = 40
	m.mode = modeResponse
	m.request = &model.Request{Method: "GET", URL: "http://x"}
	m.response = &model.Response{
		StatusCode:  200,
		Status:      "OK",
		ContentType: "application/json",
		Body:        []byte(`{"a":1,"big":{"x":1,"y":2,"z":3}}`),
	}
	return m
}

func TestFoldRenderCollapsesNode(t *testing.T) {
	m := foldTestModel()
	m.respCollapsed[2] = true // "big": { opens at JSON line 2
	out := stripANSI(m.View())
	assert.NotContains(t, out, `"x"`, "collapsed children must be hidden")
	assert.NotContains(t, out, `"z"`)
	assert.Contains(t, out, "lines)", "fold summary should be shown")
	assert.Contains(t, out, `"a"`, "siblings stay visible")
}

func TestFoldRenderExpandedShowsAll(t *testing.T) {
	m := foldTestModel()
	out := stripANSI(m.View())
	assert.Contains(t, out, `"x"`)
	assert.Contains(t, out, `"z"`)
}

func TestToggleFoldAtCursor(t *testing.T) {
	m := foldTestModel()
	_ = m.View() // populate lastFold (bodyStart, jsonFolds)
	m.cursorLine = lastFold.bodyStart + 2
	m.toggleFoldAtCursor()
	assert.True(t, m.respCollapsed[2])
	m.toggleFoldAtCursor()
	assert.False(t, m.respCollapsed[2])
}

func TestFoldDisabledWhileSearching(t *testing.T) {
	m := foldTestModel()
	m.respCollapsed[2] = true
	m.searching = true
	out := stripANSI(m.View())
	assert.Contains(t, out, `"x"`, "folding is disabled during search so all lines show")
	assert.True(t, strings.Contains(out, `"z"`))
}
