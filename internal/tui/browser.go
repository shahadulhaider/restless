package tui

import tea "charm.land/bubbletea/v2"

type BrowserModel struct {
	width, height int
}

func NewBrowserModel() BrowserModel {
	return BrowserModel{}
}

func (m BrowserModel) Init() tea.Cmd {
	return nil
}

func (m BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		wsm := msg.(tea.WindowSizeMsg)
		m.width = wsm.Width
		m.height = wsm.Height
	}
	return m, nil
}

func (m BrowserModel) View() string {
	return "Collection Browser\n\n(no requests loaded)"
}
