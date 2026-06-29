package tui

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/shahadulhaider/restless/internal/model"
)

func TestViewFitsTerminalHeight(t *testing.T) {
	const w, h = 100, 24
	model0, _ := New("").Update(tea.WindowSizeMsg{Width: w, Height: h})
	app := model0.(App)

	var items []string
	for i := 0; i < 80; i++ {
		items = append(items, `{"id":`+strconv.Itoa(i)+`,"name":"item"}`)
	}
	body := []byte(`{"items":[` + strings.Join(items, ",") + `]}`)
	app.detail.request = &model.Request{Method: "GET", URL: "http://x"}
	app.detail.response = &model.Response{StatusCode: 200, Status: "OK", ContentType: "application/json", Body: body}
	app.detail.mode = modeResponse
	app.focus = PaneDetail
	app.detail.respOffset = 30

	v := app.View()
	lines := strings.Split(strings.TrimRight(v.Content, "\n"), "\n")
	t.Logf("terminal height=%d, rendered View lines=%d", h, len(lines))
	assert.LessOrEqual(t, len(lines), h,
		"composed View exceeds terminal height — the status bar is pushed off-screen")
}

func TestViewFitsTerminalHeightSmallBody(t *testing.T) {
	const w, h = 100, 24
	model0, _ := New("").Update(tea.WindowSizeMsg{Width: w, Height: h})
	app := model0.(App)
	app.detail.request = &model.Request{Method: "GET", URL: "http://x"}
	app.detail.response = &model.Response{StatusCode: 200, Status: "OK", ContentType: "application/json", Body: []byte(`{"ok":true}`)}
	app.detail.mode = modeResponse
	app.focus = PaneDetail

	v := app.View()
	lines := strings.Split(strings.TrimRight(v.Content, "\n"), "\n")
	t.Logf("terminal height=%d, rendered View lines=%d (small body)", h, len(lines))
	assert.LessOrEqual(t, len(lines), h)
}
