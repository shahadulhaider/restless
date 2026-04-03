package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ConfirmResult is emitted when the user makes a choice in the confirmation dialog.
type ConfirmResult struct {
	Confirmed bool
	// Context carries arbitrary caller-supplied data so the handler knows what was confirmed.
	Context interface{}
}

// ConfirmModel is a simple yes/no confirmation overlay.
type ConfirmModel struct {
	message string
	context interface{}
	cursor  int // 0 = Yes, 1 = No
	width   int
	height  int
}

// NewConfirmModel creates a ConfirmModel with the given message.
// context is any value the caller needs back in ConfirmResult to identify the action.
func NewConfirmModel(message string, context interface{}) ConfirmModel {
	return ConfirmModel{
		message: message,
		context: context,
		cursor:  1, // default: No (safer)
	}
}

func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "left", "h":
			m.cursor = 0 // Yes
		case "right", "l":
			m.cursor = 1 // No
		case "tab":
			m.cursor = 1 - m.cursor
		case "y", "Y":
			return m, func() tea.Msg {
				return ConfirmResult{Confirmed: true, Context: m.context}
			}
		case "n", "N", "esc":
			return m, func() tea.Msg {
				return ConfirmResult{Confirmed: false, Context: m.context}
			}
		case "enter":
			confirmed := m.cursor == 0
			return m, func() tea.Msg {
				return ConfirmResult{Confirmed: confirmed, Context: m.context}
			}
		}
	}
	return m, nil
}

func (m ConfirmModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CDD6F4")).
		Bold(true)

	selectedStyle := lipgloss.NewStyle().
		Background(colorBorderActive).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)

	normalStyle := lipgloss.NewStyle().
		Foreground(colorStatusText).
		Padding(0, 2)

	yes := normalStyle.Render("Yes")
	no := normalStyle.Render("No")
	if m.cursor == 0 {
		yes = selectedStyle.Render("Yes")
	} else {
		no = selectedStyle.Render("No")
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(m.message) + "\n\n")
	sb.WriteString(yes + "  " + no + "\n")
	sb.WriteString("\n" + dimStyle.Render("←/→ select  Enter confirm  y/n shortcut"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Padding(1, 2).
		Render(sb.String())
}
