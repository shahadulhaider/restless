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

// EnvCreateRequested is emitted when user wants to create a new env file.
type EnvCreateRequested struct{}

// EnvEditRequested is emitted when user wants to edit the env file in $EDITOR.
type EnvEditRequested struct{}

type EnvModel struct {
	envFile  *model.EnvironmentFile
	envNames []string
	current  string
	cursor   int
	width    int
	height   int
	hasFile  bool // whether an env file exists on disk
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
	m.hasFile = ef != nil && len(ef.Environments) > 0
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

func (m *EnvModel) SetHasFile(has bool) {
	m.hasFile = has
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
		case "c":
			return m, func() tea.Msg { return EnvCreateRequested{} }
		case "e":
			if m.hasFile {
				return m, func() tea.Msg { return EnvEditRequested{} }
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
	if len(m.envNames) <= 1 {
		sb.WriteString("\n" + dimStyle.Render("No environments configured."))
	}

	// Keybinding hints
	sb.WriteString("\n")
	var hints []string
	hints = append(hints, "Enter: select")
	if m.hasFile {
		hints = append(hints, "e: edit env file")
	}
	hints = append(hints, "c: create new env file")
	hints = append(hints, "Esc: close")
	sb.WriteString(dimStyle.Render(strings.Join(hints, "  │  ")))

	return sb.String()
}
