package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/history"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

type Pane int

const (
	PaneBrowser Pane = iota
	PaneDetail
)

type collectionLoaded struct {
	collection *model.Collection
}

type envsLoaded struct {
	envFile *model.EnvironmentFile
}

type envVarsMsg struct {
	vars    map[string]string
	envName string
}

type App struct {
	rootDir       string
	width, height int
	focus         Pane
	browser       BrowserModel
	detail        DetailModel
	search        SearchModel
	envSwitch     EnvModel
	showSearch    bool
	showEnvSwitch bool
	currentEnv    string
	envFile       *model.EnvironmentFile
	envVars       map[string]string
	chainCtx      *parser.ChainContext
	cookies       *engine.CookieManager
}

func New(rootDir string) App {
	chainCtx := parser.NewChainContext()
	cookies := engine.NewCookieManager()
	return App{
		rootDir:   rootDir,
		browser:   NewBrowserModel(),
		detail:    NewDetailModel(rootDir, chainCtx, cookies),
		search:    NewSearchModel(),
		envSwitch: NewEnvModel(),
		chainCtx:  chainCtx,
		cookies:   cookies,
		envVars:   make(map[string]string),
	}
}

func (m App) Init() tea.Cmd {
	rootDir := m.rootDir
	return tea.Batch(
		func() tea.Msg {
			c, _ := LoadCollection(rootDir)
			return collectionLoaded{collection: c}
		},
		func() tea.Msg {
			ef, _ := parser.LoadEnvironments(rootDir)
			return envsLoaded{envFile: ef}
		},
	)
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		browserWidth := m.width * 3 / 10
		detailWidth := m.width - browserWidth
		var bc, dc tea.Cmd
		m.browser, bc = m.browser.Update(tea.WindowSizeMsg{Width: browserWidth, Height: m.height - 1})
		m.detail, dc = m.detail.Update(tea.WindowSizeMsg{Width: detailWidth, Height: m.height - 1})
		return m, tea.Batch(bc, dc)

	case collectionLoaded:
		if msg.collection != nil {
			m.browser.SetCollection(msg.collection)
			var items []SearchResult
			for _, f := range msg.collection.Files {
				for i := range f.Requests {
					items = append(items, SearchResult{Request: &f.Requests[i], File: f.Path})
				}
			}
			m.search.SetItems(items)
		}
		return m, nil

	case envsLoaded:
		m.envFile = msg.envFile
		m.envSwitch.SetEnvFile(msg.envFile, m.currentEnv)
		return m, nil

	case RequestSelected:
		m.detail, _ = m.detail.Update(msg)
		m.showSearch = false
		return m, nil

	case SearchSelected:
		m.showSearch = false
		m.detail, _ = m.detail.Update(RequestSelected{Request: msg.Request})
		return m, nil

	case EnvChanged:
		m.currentEnv = msg.Name
		m.showEnvSwitch = false
		if m.envFile != nil && msg.Name != "" {
			vars, _ := parser.ResolveEnvironment(m.envFile, msg.Name)
			m.envVars = vars
		} else {
			m.envVars = make(map[string]string)
		}
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(envVarsMsg{vars: m.envVars, envName: msg.Name})
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case responseReceived:
		if msg.resp != nil && msg.resp.Request != nil {
			_ = history.Save(m.rootDir, msg.resp.Request, msg.resp, m.currentEnv)
			if msg.resp.Request.Name != "" {
				m.chainCtx.StoreResponse(msg.resp.Request.Name, msg.resp)
			}
		}
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.showSearch {
			if msg.String() == "esc" {
				m.showSearch = false
				return m, nil
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		}
		if m.showEnvSwitch {
			if msg.String() == "esc" {
				m.showEnvSwitch = false
				return m, nil
			}
			var cmd tea.Cmd
			m.envSwitch, cmd = m.envSwitch.Update(msg)
			return m, cmd
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

	browserView := browserStyle.Width(browserWidth).Height(contentHeight).Render(m.browser.View())
	detailView := detailStyle.Width(detailWidth).Height(contentHeight).Render(m.detail.View())
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, browserView, detailView)

	envLabel := m.currentEnv
	if envLabel == "" {
		envLabel = "none"
	}
	statusText := fmt.Sprintf(" env: %s │ tab: switch │ /: search │ e: env │ q: quit", envLabel)
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

func RunApp(rootDir string) error {
	p := tea.NewProgram(New(rootDir))
	_, err := p.Run()
	return err
}
