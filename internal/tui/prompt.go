package tui

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// PromptResult is emitted when the user confirms or cancels a text prompt.
type PromptResult struct {
	Value   string
	Context interface{}
	OK      bool
}

// PromptModel is a single-line text input overlay used for naming files/folders.
type PromptModel struct {
	label   string
	value   string
	context interface{}
}

func NewPromptModel(label string, context interface{}) PromptModel {
	return PromptModel{label: label, context: context}
}

func (m PromptModel) Init() tea.Cmd { return nil }

func (m PromptModel) Update(msg tea.Msg) (PromptModel, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "enter":
			val := strings.TrimSpace(m.value)
			return m, func() tea.Msg {
				return PromptResult{Value: val, Context: m.context, OK: val != ""}
			}
		case "esc":
			return m, func() tea.Msg {
				return PromptResult{Context: m.context, OK: false}
			}
		case "backspace":
			if len(m.value) > 0 {
				_, size := utf8.DecodeLastRuneInString(m.value)
				m.value = m.value[:len(m.value)-size]
			}
		default:
			if k := kp.String(); len([]rune(k)) == 1 {
				m.value += k
			}
		}
	}
	return m, nil
}

func (m PromptModel) View() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4")).Bold(true)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorderActive).
		Padding(0, 1).
		Render(labelStyle.Render(m.label) + ": " + m.value + "█\n" +
			dimStyle.Render("Enter: confirm  Esc: cancel"))
}
