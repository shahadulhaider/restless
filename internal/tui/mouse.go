package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

const mouseWheelStep = 3

type paneRect struct {
	x, y, w, h int
}

func (r paneRect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

type paneLayout struct {
	dividerX int
	browser  paneRect
	detail   paneRect
}

// computeLayout derives screen geometry from the same dimensions View() uses:
// each pane is a rounded-border box, so its content is inset by 1 cell on every
// side and the two borders account for the "-4" in the detail content width.
func (m App) computeLayout() paneLayout {
	browserWidth := m.width * m.splitPct / 100
	contentHeight := m.height - 2
	if contentHeight < 0 {
		contentHeight = 0
	}
	detailWidth := m.width - browserWidth - 4
	if detailWidth < 0 {
		detailWidth = 0
	}
	return paneLayout{
		dividerX: browserWidth + 1,
		browser:  paneRect{x: 1, y: 1, w: browserWidth, h: contentHeight},
		detail:   paneRect{x: browserWidth + 3, y: 1, w: detailWidth, h: contentHeight},
	}
}

func (m App) handleMouseWheel(msg tea.MouseWheelMsg) (App, tea.Cmd) {
	if m.anyOverlayActive() {
		return m, nil
	}
	var delta int
	switch msg.Button {
	case tea.MouseWheelUp:
		delta = -mouseWheelStep
	case tea.MouseWheelDown:
		delta = mouseWheelStep
	default:
		return m, nil
	}
	if msg.X <= m.computeLayout().dividerX {
		m.browser.ScrollBy(delta)
	} else {
		m.detail.ScrollBy(delta)
	}
	return m, nil
}

func (m App) handleMouseClick(msg tea.MouseClickMsg) (App, tea.Cmd) {
	if m.anyOverlayActive() || msg.Button != tea.MouseLeft {
		return m, nil
	}
	lay := m.computeLayout()
	if msg.X == lay.dividerX || msg.X == lay.dividerX+1 {
		m.draggingSplit = true
		return m, nil
	}
	if next, cmd, ok := m.handleZoneClick(msg); ok {
		return next, cmd
	}
	if msg.X <= lay.dividerX {
		m.focus = PaneBrowser
	} else {
		m.focus = PaneDetail
	}
	return m, nil
}

// handleZoneClick dispatches a left click to a bubblezone-marked element
// (browser row, [r]/[s] toggle, accordion header). The bool reports a hit.
func (m App) handleZoneClick(msg tea.MouseClickMsg) (App, tea.Cmd, bool) {
	for i := range m.browser.items {
		if z := zone.Get(fmt.Sprintf("br:%d", i)); z != nil && z.InBounds(msg) {
			m.focus = PaneBrowser
			m.browser.cursor = i
			var cmd tea.Cmd
			m.browser, cmd = m.browser.Activate()
			return m, cmd, true
		}
	}
	if z := zone.Get("dt:req"); z != nil && z.InBounds(msg) {
		m.focus = PaneDetail
		m.detail.mode = modeRequest
		return m, nil, true
	}
	if z := zone.Get("dt:resp"); z != nil && z.InBounds(msg) {
		m.focus = PaneDetail
		if m.detail.response != nil {
			m.detail.mode = modeResponse
		}
		return m, nil, true
	}
	for _, key := range []string{"1", "2", "3", "4"} {
		if z := zone.Get("dt:sec:" + key); z != nil && z.InBounds(msg) {
			m.focus = PaneDetail
			m.detail.toggleSection(int(key[0] - '1'))
			return m, nil, true
		}
	}
	for j := range lastFold.jsonFolds {
		if z := zone.Get(fmt.Sprintf("fold:%d", j)); z != nil && z.InBounds(msg) {
			m.focus = PaneDetail
			if cm := m.detail.collapsedFor(); cm[j] {
				delete(cm, j)
			} else {
				cm[j] = true
			}
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m App) handleMouseMotion(msg tea.MouseMotionMsg) (App, tea.Cmd) {
	if !m.draggingSplit || msg.Button != tea.MouseLeft || m.width <= 0 {
		return m, nil
	}
	pct := msg.X * 100 / m.width
	if pct < 15 {
		pct = 15
	}
	if pct > 65 {
		pct = 65
	}
	if pct == m.splitPct {
		return m, nil
	}
	m.splitPct = pct
	return m, m.resizePanes()
}

func (m App) handleMouseRelease(_ tea.MouseReleaseMsg) (App, tea.Cmd) {
	m.draggingSplit = false
	return m, nil
}

// resizePanes propagates the current split to the child models, mirroring the
// WindowSizeMsg handler so body wrapping stays correct after a live resize.
func (m *App) resizePanes() tea.Cmd {
	browserWidth := m.width * m.splitPct / 100
	detailWidth := m.width - browserWidth
	var bc, dc tea.Cmd
	m.browser, bc = m.browser.Update(tea.WindowSizeMsg{Width: browserWidth, Height: m.height - 1})
	m.detail, dc = m.detail.Update(tea.WindowSizeMsg{Width: detailWidth, Height: m.height - 1})
	return tea.Batch(bc, dc)
}
