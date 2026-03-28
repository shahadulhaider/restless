package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Pane int

const (
	PaneBrowser Pane = iota
	PaneDetail
)

type App struct {
	width, height int
	focus         Pane
	browser       BrowserModel
	detail        DetailModel
	search        SearchModel
	envSwitch     EnvModel
	showSearch    bool
	showEnvSwitch bool
	currentEnv    string
}

func New() App {
	return App{
		currentEnv: "default",
		browser:    NewBrowserModel(),
		detail:     NewDetailModel(),
		search:     NewSearchModel(),
		envSwitch:  NewEnvModel(),
	}
}

func (m App) Init() tea.Cmd {
	return nil
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		browserWidth := m.width * 3 / 10
		detailWidth := m.width - browserWidth
		browserMsg := tea.WindowSizeMsg{Width: browserWidth, Height: m.height - 1}
		detailMsg := tea.WindowSizeMsg{Width: detailWidth, Height: m.height - 1}
		var bc, dc tea.Cmd
		m.browser, bc = m.browser.Update(browserMsg)
		m.detail, dc = m.detail.Update(detailMsg)
		return m, tea.Batch(bc, dc)

	case RequestSelected:
		m.detail, _ = m.detail.Update(msg)
		m.showSearch = false
		return m, nil

	case SearchSelected:
		m.showSearch = false
		fakeMsg := RequestSelected{Request: msg.Request}
		m.detail, _ = m.detail.Update(fakeMsg)
		return m, nil

	case EnvChanged:
		m.currentEnv = msg.Name
		m.showEnvSwitch = false
		return m, nil

	case responseReceived:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.showSearch {
			switch msg.String() {
			case "esc":
				m.showSearch = false
				return m, nil
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}
		if m.showEnvSwitch {
			switch msg.String() {
			case "esc":
				m.showEnvSwitch = false
				return m, nil
			}
			var cmd tea.Cmd
			m.envSwitch, cmd = m.envSwitch.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.focus == PaneBrowser {
				m.focus = PaneDetail
			} else {
				m.focus = PaneBrowser
			}
			return m, nil
		case "/":
			m.showSearch = true
			return m, nil
		case "e":
			m.showEnvSwitch = true
			return m, nil
		}

		var cmd tea.Cmd
		switch m.focus {
		case PaneBrowser:
			m.browser, cmd = m.browser.Update(msg)
		case PaneDetail:
			m.detail, cmd = m.detail.Update(msg)
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m App) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	browserWidth := m.width * 3 / 10
	detailWidth := m.width - browserWidth - 4

	browserStyle := paneStyle
	detailStyle := paneStyle
	if m.focus == PaneBrowser {
		browserStyle = paneStyleActive
	} else {
		detailStyle = paneStyleActive
	}

	contentHeight := m.height - 2
	if contentHeight < 0 {
		contentHeight = 0
	}

	browserView := browserStyle.
		Width(browserWidth).
		Height(contentHeight).
		Render(m.browser.View())

	detailView := detailStyle.
		Width(detailWidth).
		Height(contentHeight).
		Render(m.detail.View())

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, browserView, detailView)

	statusText := fmt.Sprintf(" env: %s │ tab: switch pane │ /: search │ e: env │ q: quit", m.currentEnv)
	statusBar := statusBarStyle.Width(m.width).Render(statusText)

	content := lipgloss.JoinVertical(lipgloss.Left, mainContent, statusBar)

	if m.showSearch {
		content = lipgloss.JoinVertical(lipgloss.Left, m.search.View(), content)
	}
	if m.showEnvSwitch {
		content = lipgloss.JoinVertical(lipgloss.Left, m.envSwitch.View(), content)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func RunApp() error {
	p := tea.NewProgram(New())
	_, err := p.Run()
	return err
}
