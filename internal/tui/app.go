package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/history"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/shahadulhaider/restless/internal/writer"
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

type collectionReloadMsg struct{}
type envsReloadMsg struct{}
type statusMsg struct{ text string }
type clearStatusMsg struct{}
type editorOpenedInExternalEditor struct{ filePath string }

// setStatus sets the status text and returns a command to clear it after 3 seconds.
func setStatus(text string) (string, tea.Cmd) {
	return text, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

type App struct {
	rootDir       string
	width, height int
	splitPct      int // browser width as percentage (10-70, default 30)
	focus         Pane
	browser       BrowserModel
	detail        DetailModel
	search        SearchModel
	envSwitch     EnvModel
	editor        EditorModel
	confirm       ConfirmModel
	prompt        PromptModel
	showSearch    bool
	showEnvSwitch bool
	showEditor    bool
	showConfirm   bool
	showPrompt    bool
	showHelp      bool
	help          HelpModel
	editingReq    *model.Request // nil = create mode, non-nil = edit mode
	currentEnv    string
	envFile       *model.EnvironmentFile
	envVars       map[string]string
	chainCtx      *parser.ChainContext
	cookies       *engine.CookieManager
	statusText    string // ephemeral status message
}

func New(rootDir string) App {
	chainCtx := parser.NewChainContext()
	cookies := engine.NewCookieManager()
	return App{
		rootDir:   rootDir,
		splitPct:  30,
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
		browserWidth := m.width * m.splitPct / 100
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
		m.envSwitch.SetHasFile(parser.EnvFilePath(m.rootDir) != "")
		return m, nil

	case collectionReloadMsg:
		rootDir := m.rootDir
		return m, func() tea.Msg {
			c, _ := LoadCollection(rootDir)
			return collectionLoaded{collection: c}
		}

	case statusMsg:
		m.statusText = msg.text
		return m, nil

	case clearStatusMsg:
		m.statusText = ""
		return m, nil

	case RequestSelected:
		m.detail, _ = m.detail.Update(msg)
		m.showSearch = false
		return m, nil

	case SearchSelected:
		m.showSearch = false
		m.detail, _ = m.detail.Update(RequestSelected{Request: msg.Request})
		return m, nil

	case EditorSaved:
		m.showEditor = false
		req := msg.Request
		var err error
		if m.editingReq != nil {
			// Edit mode — update the existing request
			err = writer.UpdateRequest(m.editingReq.SourceFile, *m.editingReq, req)
		} else {
			// Create mode — insert into current file or default
			targetFile := m.currentEditFile()
			err = writer.InsertRequest(targetFile, req)
		}
		m.editingReq = nil
		if err != nil {
			m.statusText = "Error: " + err.Error()
		}
		return m, func() tea.Msg { return collectionReloadMsg{} }

	case EditorCancelled:
		m.showEditor = false
		m.editingReq = nil
		return m, nil

	case ConfirmResult:
		m.showConfirm = false
		if !msg.Confirmed {
			return m, nil
		}
		switch result := msg.Context.(type) {
		case confirmDeleteRequest:
			if err := writer.DeleteRequest(result.req.SourceFile, result.req); err != nil {
				m.statusText = "Error: " + err.Error()
			} else {
				return m, func() tea.Msg { return collectionReloadMsg{} }
			}
		case confirmDeleteEntry:
			if err := writer.DeleteEntry(m.rootDir, result.relPath); err != nil {
				m.statusText = "Error: " + err.Error()
			} else {
				return m, func() tea.Msg { return collectionReloadMsg{} }
			}
		}
		return m, nil

	case PromptResult:
		m.showPrompt = false
		if !msg.OK {
			return m, nil
		}
		switch ctx := msg.Context.(type) {
		case promptCreateFile:
			if err := writer.CreateHTTPFile(m.rootDir, ctx.dir+"/"+msg.Value+".http"); err != nil {
				m.statusText = "Error: " + err.Error()
			} else {
				return m, func() tea.Msg { return collectionReloadMsg{} }
			}
		case promptCreateDir:
			name := msg.Value
			if ctx.parent != "" {
				name = ctx.parent + "/" + name
			}
			if err := writer.CreateDirectory(m.rootDir, name); err != nil {
				m.statusText = "Error: " + err.Error()
			} else {
				return m, func() tea.Msg { return collectionReloadMsg{} }
			}
		case promptRename:
			if err := writer.RenameEntry(m.rootDir, ctx.relPath, filepath.Dir(ctx.relPath)+"/"+msg.Value); err != nil {
				m.statusText = "Error: " + err.Error()
			} else {
				return m, func() tea.Msg { return collectionReloadMsg{} }
			}
		case promptMove:
			if err := writer.MoveEntry(m.rootDir, ctx.relPath, msg.Value); err != nil {
				m.statusText = "Error: " + err.Error()
			} else {
				return m, func() tea.Msg { return collectionReloadMsg{} }
			}
		}
		return m, nil

	case EnvCreateRequested:
		m.showEnvSwitch = false
		envPath := filepath.Join(m.rootDir, "restless.env.json")
		if _, err := os.Stat(envPath); err == nil {
			m.statusText, _ = setStatus("restless.env.json already exists — use 'e' to edit")
			return m, nil
		}
		// Write boilerplate
		boilerplate := `{
  "$shared": {
    "baseUrl": "https://api.example.com"
  },
  "dev": {
    "baseUrl": "http://localhost:8000",
    "token": "dev-token"
  },
  "staging": {
    "baseUrl": "https://staging.example.com",
    "token": "staging-token"
  },
  "prod": {
    "token": "prod-token"
  }
}
`
		if err := os.WriteFile(envPath, []byte(boilerplate), 0644); err != nil {
			m.statusText, _ = setStatus("Error creating env file: " + err.Error())
			return m, nil
		}
		// Open in $EDITOR
		editorBin := os.Getenv("EDITOR")
		if editorBin == "" {
			editorBin = os.Getenv("VISUAL")
		}
		if editorBin != "" {
			c := buildEditorCmd(editorBin, envPath)
			return m, tea.ExecProcess(c, func(err error) tea.Msg {
				return envsReloadMsg{}
			})
		}
		m.statusText, _ = setStatus("Created restless.env.json — set $EDITOR to edit")
		return m, func() tea.Msg { return envsReloadMsg{} }

	case EnvEditRequested:
		m.showEnvSwitch = false
		envPath := parser.EnvFilePath(m.rootDir)
		if envPath == "" {
			m.statusText, _ = setStatus("No env file found — press 'c' to create one")
			return m, nil
		}
		editorBin := os.Getenv("EDITOR")
		if editorBin == "" {
			editorBin = os.Getenv("VISUAL")
		}
		if editorBin == "" {
			m.statusText, _ = setStatus("$EDITOR not set — can't open env file")
			return m, nil
		}
		c := buildEditorCmd(editorBin, envPath)
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return envsReloadMsg{}
		})

	case envsReloadMsg:
		rootDir := m.rootDir
		return m, func() tea.Msg {
			ef, _ := parser.LoadEnvironments(rootDir)
			return envsLoaded{envFile: ef}
		}

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

	case yankResult:
		var cmd tea.Cmd
		if msg.err != nil {
			m.statusText, cmd = setStatus("Copy failed: " + msg.err.Error())
		} else {
			m.statusText, cmd = setStatus("Copied " + msg.label + " to clipboard")
		}
		return m, cmd

	case responseReceived:
		if msg.resp != nil && msg.resp.Request != nil {
			_ = history.Save(m.rootDir, msg.resp.Request, msg.resp, m.currentEnv)
			if msg.resp.Request.Name != "" {
				m.chainCtx.StoreResponse(msg.resp.Request.Name, msg.resp)
			}
			m.browser.RecordStatus(msg.resp.Request, msg.resp.StatusCode)
		}
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" || msg.String() == "f1" {
				m.showHelp = false
				return m, nil
			}
			var cmd tea.Cmd
			m.help, cmd = m.help.Update(msg)
			return m, cmd
		}
		if m.showEditor {
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}
		if m.showConfirm {
			var cmd tea.Cmd
			m.confirm, cmd = m.confirm.Update(msg)
			return m, cmd
		}
		if m.showPrompt {
			var cmd tea.Cmd
			m.prompt, cmd = m.prompt.Update(msg)
			return m, cmd
		}
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
		case "?":
			m.help = NewHelpModel(helpFull, "")
			m.help.width = m.width
			m.help.height = m.height
			m.showHelp = true
			return m, nil
		case "f1":
			ctx := m.helpContext()
			m.help = NewHelpModel(helpContext, ctx)
			m.help.width = m.width
			m.help.height = m.height
			m.showHelp = true
			return m, nil
		case "ctrl+left", "[":
			if m.splitPct > 15 {
				m.splitPct -= 5
			}
			return m, nil
		case "ctrl+right", "]":
			if m.splitPct < 65 {
				m.splitPct += 5
			}
			return m, nil
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
		case "ctrl+e":
			// Environment switch (was: e)
			m.showEnvSwitch = true
			return m, nil
		case "n":
			// Create new request
			m.editor = NewEditorModel()
			m.editor.SetAvailableVars(m.collectAvailableVars())
			m.editingReq = nil
			m.showEditor = true
			return m, nil
		case "e":
			// Edit with $EDITOR — fallback to internal editor if $EDITOR is not set
			if sel := m.browser.selected; sel != nil && sel.SourceFile != "" {
				editorBin := os.Getenv("EDITOR")
				if editorBin == "" {
					editorBin = os.Getenv("VISUAL")
				}
				if editorBin != "" {
					// Use tea.ExecProcess to properly hand off the terminal
					c := buildEditorCmd(editorBin, sel.SourceFile)
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return collectionReloadMsg{}
					})
				}
				// No $EDITOR set — fall back to internal editor
				m.editor = NewEditorModelFromRequest(*sel)
				m.editor.SetAvailableVars(m.collectAvailableVars())
				m.editingReq = sel
				m.showEditor = true
			}
			return m, nil
		case "E":
			// Always open internal editor
			if sel := m.browser.selected; sel != nil {
				m.editor = NewEditorModelFromRequest(*sel)
				m.editor.SetAvailableVars(m.collectAvailableVars())
				m.editingReq = sel
				m.showEditor = true
			}
			return m, nil
		case "D":
			// Delete selected request (with confirmation)
			if sel := m.browser.selected; sel != nil {
				m.confirm = NewConfirmModel("Delete this request?", confirmDeleteRequest{req: *sel})
				m.showConfirm = true
			}
			return m, nil
		case "Y":
			// Duplicate selected request
			if sel := m.browser.selected; sel != nil {
				targetFile := sel.SourceFile
				if err := writer.DuplicateRequest(*sel, targetFile); err != nil {
					m.statusText = "Error: " + err.Error()
				}
				return m, func() tea.Msg { return collectionReloadMsg{} }
			}
			return m, nil


		case "N":
			// Create new .http file in current directory
			dir := m.currentDir()
			relDir, _ := filepath.Rel(m.rootDir, dir)
			m.prompt = NewPromptModel("New file name (without .http)", promptCreateFile{dir: relDir})
			m.showPrompt = true
			return m, nil
		case "F":
			// Create new folder
			dir := m.currentDir()
			parent, _ := filepath.Rel(m.rootDir, dir)
			if parent == "." {
				parent = ""
			}
			m.prompt = NewPromptModel("New folder name", promptCreateDir{parent: parent})
			m.showPrompt = true
			return m, nil
		case "R":
			// Rename selected item
			if item := m.browser.CurrentItem(); item != nil {
				rel, _ := filepath.Rel(m.rootDir, item.Path)
				m.prompt = NewPromptModel("Rename to", promptRename{relPath: rel})
				m.showPrompt = true
			}
			return m, nil
		case "M":
			// Move selected item
			if item := m.browser.CurrentItem(); item != nil {
				rel, _ := filepath.Rel(m.rootDir, item.Path)
				m.prompt = NewPromptModel("Move to (relative path)", promptMove{relPath: rel})
				m.showPrompt = true
			}
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

	browserWidth := m.width * m.splitPct / 100
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
	statusLine := m.statusText
	if statusLine == "" {
		switch {
		case m.showEditor:
			statusLine = " Ctrl+S: save │ Esc: cancel │ Tab: next field │ Shift+Tab: prev field"
		case m.showConfirm:
			statusLine = " ←/→: select │ Enter: confirm │ y/n: shortcut"
		case m.showPrompt:
			statusLine = " Enter: confirm │ Esc: cancel"
		case m.focus == PaneBrowser:
			statusLine = fmt.Sprintf(" env:%s │ e:edit │ n:new │ D:del │ Y:dup │ N:file │ F:folder │ ?:help │ q:quit", envLabel)
		case m.focus == PaneDetail:
			if m.detail.response != nil {
				if m.detail.mode == modeResponse {
					statusLine = " [r]req [s]resp │ 1/2/3:fold │ space:toggle │ y:yank(b/h/a/c) │ p:pretty │ f:find │ Enter:send"
				} else {
					statusLine = " [r]req [s]resp │ 1/2/3:fold │ space:toggle │ y:yank(b/h/a/c) │ Enter:send │ e:edit"
				}
			} else {
				statusLine = " Enter:send │ e:edit │ h:history │ q:quit"
			}
		default:
			statusLine = fmt.Sprintf(" env:%s │ tab:switch │ /:search │ ctrl+e:env │ q:quit", envLabel)
		}
	}
	statusBar := statusBarStyle.Width(m.width).Render(statusLine)

	content := lipgloss.JoinVertical(lipgloss.Left, mainContent, statusBar)

	if m.showEditor {
		editorView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderActive).
			Padding(1, 2).
			Width(m.width * 8 / 10).
			Render(m.editor.View())
		content = lipgloss.JoinVertical(lipgloss.Left, editorView, content)
	}
	if m.showConfirm {
		content = lipgloss.JoinVertical(lipgloss.Left, m.confirm.View(), content)
	}
	if m.showPrompt {
		content = lipgloss.JoinVertical(lipgloss.Left, m.prompt.View(), content)
	}
	if m.showSearch {
		content = lipgloss.JoinVertical(lipgloss.Left, m.search.View(), content)
	}
	if m.showEnvSwitch {
		content = lipgloss.JoinVertical(lipgloss.Left, m.envSwitch.View(), content)
	}
	if m.showHelp {
		helpView := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderActive).
			Padding(1, 2).
			Width(m.width * 8 / 10).
			Height(m.height - 4).
			Render(m.help.View())
		content = helpView
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

// confirmDeleteRequest is the Context payload for delete confirmations.
type confirmDeleteRequest struct{ req model.Request }
type confirmDeleteEntry struct{ relPath string }

// Prompt context types
type promptCreateFile struct{ dir string }
type promptCreateDir struct{ parent string }
type promptRename struct{ relPath string }
type promptMove struct{ relPath string }

// currentEditFile returns the file where a new request should be inserted.
// Prefers the currently selected file in the browser; falls back to a default.
func (m App) currentEditFile() string {
	if sel := m.browser.selected; sel != nil && sel.SourceFile != "" {
		return sel.SourceFile
	}
	return filepath.Join(m.rootDir, "requests.http")
}

// currentDir returns the directory of the currently selected browser item.
func (m App) currentDir() string {
	if item := m.browser.CurrentItem(); item != nil {
		if item.Type == ItemTypeDir {
			return filepath.Join(m.rootDir, item.Path)
		}
		return filepath.Dir(item.Path)
	}
	return m.rootDir
}

// helpContext returns the context string for F1 context-sensitive help.
func (m App) helpContext() string {
	if m.showEditor {
		return "editor"
	}
	if m.showSearch {
		return "search"
	}
	if m.focus == PaneBrowser {
		return "browser"
	}
	if m.focus == PaneDetail {
		if m.detail.response != nil && m.detail.mode == modeResponse {
			return "detail-response"
		}
		return "detail-request"
	}
	return ""
}

// collectAvailableVars gathers variable names from env + file vars for auto-complete.
func (m App) collectAvailableVars() []string {
	seen := make(map[string]bool)
	var vars []string
	for k := range m.envVars {
		if !seen[k] {
			vars = append(vars, k)
			seen[k] = true
		}
	}
	// Add common dynamic vars
	for _, dv := range []string{"$uuid", "$timestamp", "$isoTimestamp", "$date", "$randomInt", "$randomEmail"} {
		if !seen[dv] {
			vars = append(vars, dv)
			seen[dv] = true
		}
	}
	return vars
}

// buildEditorCmd creates an *exec.Cmd for the given editor binary and file path.
// Supports EDITOR values with args like "code --wait".
func buildEditorCmd(editorBin, filePath string) *exec.Cmd {
	parts := strings.Fields(editorBin)
	args := append(parts[1:], filePath)
	return exec.Command(parts[0], args...)
}
