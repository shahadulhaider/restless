package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/shahadulhaider/restless/internal/exporter"
)

// copyCmd copies text to the clipboard via OSC52 (works over SSH/tmux) and a
// best-effort local clipboard tool. Success is reported for the status line
// since OSC52 is fire-and-forget; the local copy is a bonus on desktop.
func copyCmd(text, label string) tea.Cmd {
	return tea.Batch(
		tea.SetClipboard(text),
		func() tea.Msg {
			_ = exporter.CopyToClipboard(text)
			return yankResult{label: label}
		},
	)
}
