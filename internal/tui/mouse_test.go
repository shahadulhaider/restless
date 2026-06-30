package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

func TestComputeLayout(t *testing.T) {
	m := App{width: 100, height: 30, splitPct: 30}
	lay := m.computeLayout()
	assert.Equal(t, 31, lay.dividerX)
	assert.Equal(t, paneRect{x: 1, y: 1, w: 30, h: 28}, lay.browser)
	assert.Equal(t, paneRect{x: 33, y: 1, w: 66, h: 28}, lay.detail)
}

func TestPaneRectContains(t *testing.T) {
	r := paneRect{x: 1, y: 1, w: 10, h: 5}
	assert.True(t, r.contains(1, 1))
	assert.True(t, r.contains(10, 5))
	assert.False(t, r.contains(11, 1))
	assert.False(t, r.contains(0, 0))
}

func TestBrowserScrollByClamps(t *testing.T) {
	m := NewBrowserModel()
	m.items = make([]BrowserItem, 5)
	m.height = 10
	m.ScrollBy(3)
	assert.Equal(t, 3, m.cursor)
	m.ScrollBy(100)
	assert.Equal(t, 4, m.cursor)
	m.ScrollBy(-100)
	assert.Equal(t, 0, m.cursor)
}

func TestDetailScrollByClampsAtTop(t *testing.T) {
	m := NewDetailModel("", parser.NewChainContext(), engine.NewCookieManager())
	m.ScrollBy(5)
	assert.Equal(t, 5, m.reqOffset)
	m.ScrollBy(-100)
	assert.Equal(t, 0, m.reqOffset)
}

func TestHandleMouseClickFocus(t *testing.T) {
	m := New("", "")
	m.width = 100
	m.height = 30
	m.focus = PaneBrowser

	got, _ := m.handleMouseClick(tea.MouseClickMsg{X: 80, Y: 5, Button: tea.MouseLeft})
	assert.Equal(t, PaneDetail, got.focus)

	got, _ = m.handleMouseClick(tea.MouseClickMsg{X: 5, Y: 5, Button: tea.MouseLeft})
	assert.Equal(t, PaneBrowser, got.focus)
}

func TestHandleMouseClickIgnoredWithOverlay(t *testing.T) {
	m := New("", "")
	m.width = 100
	m.height = 30
	m.focus = PaneBrowser
	m.showSearch = true
	got, _ := m.handleMouseClick(tea.MouseClickMsg{X: 80, Y: 5, Button: tea.MouseLeft})
	assert.Equal(t, PaneBrowser, got.focus)
}

func TestDividerDragResizes(t *testing.T) {
	m := New("", "")
	m.width = 100
	m.height = 30

	m, _ = m.handleMouseClick(tea.MouseClickMsg{X: 31, Y: 5, Button: tea.MouseLeft})
	assert.True(t, m.draggingSplit)

	m, _ = m.handleMouseMotion(tea.MouseMotionMsg{X: 50, Y: 5, Button: tea.MouseLeft})
	assert.Equal(t, 50, m.splitPct)

	m, _ = m.handleMouseRelease(tea.MouseReleaseMsg{X: 50, Y: 5, Button: tea.MouseLeft})
	assert.False(t, m.draggingSplit)
}

func TestDragClampsToBounds(t *testing.T) {
	m := New("", "")
	m.width = 100
	m.height = 30
	m.draggingSplit = true

	m, _ = m.handleMouseMotion(tea.MouseMotionMsg{X: 5, Y: 5, Button: tea.MouseLeft})
	assert.Equal(t, 15, m.splitPct)

	m, _ = m.handleMouseMotion(tea.MouseMotionMsg{X: 95, Y: 5, Button: tea.MouseLeft})
	assert.Equal(t, 65, m.splitPct)
}

func TestHandleMouseWheelRoutesToPane(t *testing.T) {
	m := New("", "")
	m.width = 100
	m.height = 30
	m.browser.items = make([]BrowserItem, 20)
	m.browser.height = 10

	m, _ = m.handleMouseWheel(tea.MouseWheelMsg{X: 5, Y: 5, Button: tea.MouseWheelDown})
	assert.Equal(t, 3, m.browser.cursor)
	assert.Equal(t, 0, m.detail.reqOffset)

	m, _ = m.handleMouseWheel(tea.MouseWheelMsg{X: 80, Y: 5, Button: tea.MouseWheelDown})
	assert.Equal(t, 3, m.detail.reqOffset)
	assert.Equal(t, 3, m.browser.cursor)
}

func TestAppHandlesMouseSequenceNoPanic(t *testing.T) {
	model0, _ := New("", "").Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app := model0.(App)

	for _, msg := range []tea.Msg{
		tea.MouseWheelMsg{X: 5, Y: 5, Button: tea.MouseWheelDown},
		tea.MouseClickMsg{X: 50, Y: 5, Button: tea.MouseLeft},
		tea.MouseMotionMsg{X: 55, Y: 5, Button: tea.MouseLeft},
		tea.MouseReleaseMsg{X: 55, Y: 5, Button: tea.MouseLeft},
	} {
		next, _ := app.Update(msg)
		app = next.(App)
	}
}

func TestAppViewScansMarkedRows(t *testing.T) {
	model0, _ := New("", "").Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app := model0.(App)

	app.browser.SetCollection(&model.Collection{
		RootDir: "/tmp",
		Files: []model.HTTPFile{{
			Path:     "/tmp/api.http",
			Requests: []model.Request{{Method: "GET", URL: "http://x", Name: "r1", SourceFile: "/tmp/api.http"}},
		}},
	})
	app.browser.expanded["/tmp/api.http"] = true
	app.browser.items = app.browser.buildItems()

	out := stripANSI(app.View().Content)
	assert.Contains(t, out, "r1")
	assert.NotContains(t, out, "br:0")
}
