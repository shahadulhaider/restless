package tui

import tea "charm.land/bubbletea/v2"

type EnvModel struct {
	width, height int
}

func NewEnvModel() EnvModel {
	return EnvModel{}
}

func (m EnvModel) Init() tea.Cmd {
	return nil
}

func (m EnvModel) Update(msg tea.Msg) (EnvModel, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		wsm := msg.(tea.WindowSizeMsg)
		m.width = wsm.Width
		m.height = wsm.Height
	}
	return m, nil
}

func (m EnvModel) View() string {
	return "Environment Switcher"
}
