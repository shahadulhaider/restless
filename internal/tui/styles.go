package tui

import "charm.land/lipgloss/v2"

var (
	colorBorder       = lipgloss.Color("#444444")
	colorBorderActive = lipgloss.Color("#7C3AED")
	colorStatusBar    = lipgloss.Color("#1E1E2E")
	colorStatusText   = lipgloss.Color("#CDD6F4")
	colorDim          = lipgloss.Color("#6C7086")

	colorText    = lipgloss.Color("#CDD6F4")
	colorSelBg   = lipgloss.Color("#3D3D5C")
	colorSelFg   = lipgloss.Color("#FFFFFF")
	colorLineNum = lipgloss.Color("#585858")
	colorKey     = lipgloss.Color("#89DCEB")
	colorSuccess = lipgloss.Color("#4CAF50")
	colorWarning = lipgloss.Color("#FF9800")
	colorError   = lipgloss.Color("#F44336")

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
