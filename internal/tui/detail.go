package tui

import tea "charm.land/bubbletea/v2"

type DetailModel struct {
	width, height int
}

func NewDetailModel() DetailModel {
	return DetailModel{}
}

func (m DetailModel) Init() tea.Cmd {
	return nil
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		wsm := msg.(tea.WindowSizeMsg)
		m.width = wsm.Width
		m.height = wsm.Height
	}
	return m, nil
}

func (m DetailModel) View() string {
	return "Request / Response\n\n(select a request to view)"
}
