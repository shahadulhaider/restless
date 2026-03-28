package tui

import tea "charm.land/bubbletea/v2"

type SearchModel struct {
	width, height int
}

func NewSearchModel() SearchModel {
	return SearchModel{}
}

func (m SearchModel) Init() tea.Cmd {
	return nil
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		wsm := msg.(tea.WindowSizeMsg)
		m.width = wsm.Width
		m.height = wsm.Height
	}
	return m, nil
}

func (m SearchModel) View() string {
	return "/ Search..."
}
