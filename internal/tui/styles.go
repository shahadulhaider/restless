package tui

import "charm.land/lipgloss/v2"

var (
	colorBorder       = lipgloss.Color("#444444")
	colorBorderActive = lipgloss.Color("#7C3AED")
	colorStatusBar    = lipgloss.Color("#1E1E2E")
	colorStatusText   = lipgloss.Color("#CDD6F4")
	colorDim          = lipgloss.Color("#6C7086")

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	paneStyleActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderActive)

	statusBarStyle = lipgloss.NewStyle().
			Background(colorStatusBar).
			Foreground(colorStatusText).
			Padding(0, 1)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorDim)
)
