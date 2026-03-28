package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/shahadulhaider/restless/internal/model"
)

type EnvChanged struct {
	Name string
}

type EnvModel struct {
	envFile  *model.EnvironmentFile
	envNames []string
	current  string
	cursor   int
	width    int
	height   int
}

func NewEnvModel() EnvModel {
	return EnvModel{}
}

func (m EnvModel) Init() tea.Cmd {
	return nil
}

func (m *EnvModel) SetEnvFile(ef *model.EnvironmentFile, current string) {
	m.envFile = ef
	m.current = current
	m.envNames = []string{"(no environment)"}
	if ef != nil {
		for name := range ef.Environments {
			m.envNames = append(m.envNames, name)
		}
	}
	for i, n := range m.envNames {
		if n == current {
			m.cursor = i
			break
		}
	}
}

func (m EnvModel) Update(msg tea.Msg) (EnvModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.envNames)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.cursor < len(m.envNames) {
				name := m.envNames[m.cursor]
				if name == "(no environment)" {
					name = ""
				}
				m.current = name
				return m, func() tea.Msg { return EnvChanged{Name: name} }
			}
		}
	}
	return m, nil
}

func (m EnvModel) View() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CDD6F4")).
		Bold(true).
		Render("Select Environment") + "\n\n")

	for i, name := range m.envNames {
		indicator := "○"
		if name == m.current || (name == "(no environment)" && m.current == "") {
			indicator = "●"
		}
		line := indicator + " " + name
		if i == m.cursor {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#3D3D5C")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Render(line)
		} else if indicator == "●" {
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50")).Render(line)
		}
		sb.WriteString(line + "\n")
	}
	if len(m.envNames) == 0 {
		sb.WriteString(dimStyle.Render("(no environments found)"))
	}
	return sb.String()
}
