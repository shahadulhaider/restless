package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type helpMode int

const (
	helpFull    helpMode = iota // ? — full keybinding reference
	helpContext                 // F1 — context-sensitive help
)

type HelpModel struct {
	mode   helpMode
	ctx    string // context identifier for F1
	offset int
	width  int
	height int
}

func NewHelpModel(mode helpMode, ctx string) HelpModel {
	return HelpModel{mode: mode, ctx: ctx}
}

func (m HelpModel) Init() tea.Cmd { return nil }

func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			m.offset++
		case "k", "up":
			if m.offset > 0 {
				m.offset--
			}
		case "ctrl+d":
			m.offset += 10
		case "ctrl+u":
			m.offset -= 10
			if m.offset < 0 {
				m.offset = 0
			}
		case "g":
			m.offset = 0
		case "G":
			m.offset = 999999
		}
	}
	return m, nil
}

func (m HelpModel) View() string {
	var content string
	if m.mode == helpFull {
		content = fullKeybindingReference()
	} else {
		content = contextHelp(m.ctx)
	}

	lines := strings.Split(content, "\n")

	// Clamp offset
	viewH := m.height - 4
	if viewH < 1 {
		viewH = 10
	}
	maxOff := len(lines) - viewH
	if maxOff < 0 {
		maxOff = 0
	}
	if m.offset > maxOff {
		m.offset = maxOff
	}

	end := m.offset + viewH
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	for _, l := range lines[m.offset:end] {
		sb.WriteString(l + "\n")
	}
	sb.WriteString("\n" + dimStyle.Render("? or Esc: close  │  j/k: scroll  │  g/G: top/bottom"))
	return sb.String()
}

// --- Full keybinding reference (?) ---

func fullKeybindingReference() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4"))
	section := lipgloss.NewStyle().Bold(true).Foreground(colorBorderActive)
	key := lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))

	var sb strings.Builder
	sb.WriteString(title.Render("restless — Keyboard Reference") + "\n\n")

	sb.WriteString(section.Render("Global") + "\n")
	writeKeys(&sb, key, [][2]string{
		{"Tab", "Switch between browser and detail panes"},
		{"/", "Fuzzy search requests"},
		{"Ctrl+E", "Switch environment"},
		{"n", "Create new request (internal editor)"},
		{"e", "Edit with $EDITOR (falls back to internal)"},
		{"E", "Edit with internal form editor"},
		{"D", "Delete request (with confirmation)"},
		{"Y", "Duplicate request"},
		{"?", "This help screen"},
		{"F1", "Context-sensitive help"},
		{"q / Ctrl+C", "Quit"},
	})

	sb.WriteString("\n" + section.Render("Browser Pane") + "\n")
	writeKeys(&sb, key, [][2]string{
		{"j/k / ↑/↓", "Navigate requests"},
		{"Enter", "Select / expand folder or file"},
		{"N", "Create new .http file"},
		{"F", "Create new folder"},
		{"R", "Rename file or folder"},
		{"M", "Move file or folder"},
	})

	sb.WriteString("\n" + section.Render("Detail Pane — Navigation") + "\n")
	writeKeys(&sb, key, [][2]string{
		{"r / s", "Switch to Request / Response view"},
		{"Enter / Ctrl+R", "Send request"},
		{"h", "Toggle response history"},
		{"d", "Diff two history entries"},
	})

	sb.WriteString("\n" + section.Render("Detail Pane — Sections") + "\n")
	writeKeys(&sb, key, [][2]string{
		{"1 / 2 / 3 / 4", "Toggle section fold"},
		{"Space", "Toggle fold on section under cursor"},
		{"zo", "Expand section under cursor"},
		{"zc", "Collapse section under cursor"},
		{"zR", "Expand all sections"},
		{"zM", "Collapse all sections"},
	})

	sb.WriteString("\n" + section.Render("Detail Pane — Scrolling") + "\n")
	writeKeys(&sb, key, [][2]string{
		{"j/k / ↑/↓", "Scroll line by line"},
		{"Ctrl+D / Ctrl+U", "Scroll half page down / up"},
		{"g / G", "Jump to top / bottom"},
	})

	sb.WriteString("\n" + section.Render("Detail Pane — Body Viewer") + "\n")
	writeKeys(&sb, key, [][2]string{
		{"p", "Toggle pretty-print / raw"},
		{"w", "Toggle word wrap"},
		{"l", "Toggle line numbers"},
		{"f", "Search in body"},
		{"n / N", "Next / previous search match"},
	})

	sb.WriteString("\n" + section.Render("Yank (Copy to Clipboard)") + "\n")
	writeKeys(&sb, key, [][2]string{
		{"yb", "Copy body"},
		{"yh", "Copy headers"},
		{"ya", "Copy all (full request or response)"},
		{"yc", "Copy as curl command"},
		{"yg + key", "Generate code (see below)"},
	})

	sb.WriteString("\n" + section.Render("Code Generation (yg + key)") + "\n")
	writeKeys(&sb, key, [][2]string{
		{"ygp", "Python (requests)"},
		{"ygj", "JavaScript (fetch)"},
		{"ygg", "Go (net/http)"},
		{"ygv", "Java (HttpClient)"},
		{"ygr", "Ruby (net/http)"},
		{"ygh", "HTTPie"},
		{"ygc", "curl"},
		{"ygw", "PowerShell"},
	})

	sb.WriteString("\n" + section.Render("Internal Editor") + "\n")
	writeKeys(&sb, key, [][2]string{
		{"Tab / Shift+Tab", "Navigate fields"},
		{"←/→", "Move cursor in text field"},
		{"Ctrl+A / Home", "Jump to start of field"},
		{"Ctrl+E / End", "Jump to end of field"},
		{"Ctrl+W", "Delete word backward"},
		{"Ctrl+U", "Clear to start of line"},
		{"Ctrl+K", "Clear to end of line"},
		{"Ctrl+D", "Delete header row"},
		{"Ctrl+S", "Save"},
		{"Esc", "Cancel"},
	})

	return sb.String()
}

// --- Context-sensitive help (F1) ---

func contextHelp(ctx string) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CDD6F4"))
	section := lipgloss.NewStyle().Bold(true).Foreground(colorBorderActive)
	key := lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))
	tip := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Italic(true)

	var sb strings.Builder

	switch ctx {
	case "browser":
		sb.WriteString(title.Render("Browser Pane — Help") + "\n\n")
		sb.WriteString(tip.Render("Navigate your collection of .http files and requests.") + "\n\n")

		sb.WriteString(section.Render("Navigation") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"j/k", "Move up/down through files and requests"},
			{"Enter", "Expand file to see requests, or select a request"},
			{"/", "Fuzzy search across all requests by name, method, or URL"},
		})
		sb.WriteString("\n" + section.Render("Request Operations") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"n", "Create a new request (opens form editor)"},
			{"e", "Edit selected request in $EDITOR (nvim, vim, etc.)"},
			{"E", "Edit selected request in internal form editor"},
			{"D", "Delete selected request (with confirmation)"},
			{"Y", "Duplicate selected request in the same file"},
		})
		sb.WriteString("\n" + section.Render("Collection Management") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"N", "Create a new .http file (prompts for name)"},
			{"F", "Create a new folder"},
			{"R", "Rename a file or folder"},
			{"M", "Move a file or folder"},
		})
		sb.WriteString("\n" + tip.Render("Tip: Press Tab to switch to the detail pane to send requests.") + "\n")

	case "detail-request":
		sb.WriteString(title.Render("Request View — Help") + "\n\n")
		sb.WriteString(tip.Render("View and send the selected request. Sections show body, headers, and metadata.") + "\n\n")

		sb.WriteString(section.Render("Sending") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"Enter / Ctrl+R", "Send the request — response view opens automatically"},
		})
		sb.WriteString("\n" + section.Render("Editing") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"e", "Open in $EDITOR for full editing power"},
			{"E", "Open internal form editor"},
		})
		sb.WriteString("\n" + section.Render("Variables") + "\n")
		sb.WriteString("  Use " + key.Render("{{varName}}") + " in URL, headers, or body.\n")
		sb.WriteString("  Define inline: " + key.Render("@baseUrl = http://localhost:8000") + "\n")
		sb.WriteString("  Or in " + key.Render("http-client.env.json") + " and switch with " + key.Render("Ctrl+E") + "\n")
		sb.WriteString("  Dynamic: " + key.Render("{{$uuid}}") + " " + key.Render("{{$timestamp}}") + " " + key.Render("{{$randomInt}}") + "\n")
		sb.WriteString("\n" + section.Render("Assertions") + "\n")
		sb.WriteString("  Add " + key.Render("# @assert status == 200") + " to validate responses.\n")
		sb.WriteString("  Run headless: " + key.Render("restless run api.http --env dev") + "\n")
		sb.WriteString("\n" + section.Render("Yank") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"yb/yh/ya", "Copy body / headers / full request"},
			{"yc", "Copy as curl command"},
			{"yg + key", "Generate code (Python, JS, Go, etc.)"},
		})

	case "detail-response":
		sb.WriteString(title.Render("Response View — Help") + "\n\n")
		sb.WriteString(tip.Render("Inspect the response. Body is expanded by default. Fold sections with 1/2/3/4.") + "\n\n")

		sb.WriteString(section.Render("Sections") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"1", "[Body] — response body with JSON/XML formatting"},
			{"2", "[Headers] — response headers"},
			{"3", "[Timing] — DNS, TLS, TTFB waterfall"},
			{"4", "[Assertions] — pass/fail results (if assertions defined)"},
			{"Space", "Toggle fold on section under cursor"},
			{"zo/zc", "Open/close section (vim fold)"},
			{"zR/zM", "Expand all / collapse all"},
		})
		sb.WriteString("\n" + section.Render("Body Viewer") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"p", "Toggle pretty-print vs raw"},
			{"w", "Toggle word wrap"},
			{"l", "Toggle line numbers"},
			{"f", "Search in response body"},
			{"n/N", "Next/previous search match"},
		})
		sb.WriteString("\n" + section.Render("Copy") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"yb", "Copy response body"},
			{"yh", "Copy response headers"},
			{"ya", "Copy full response (status + headers + body)"},
			{"yg + key", "Generate code from the request"},
		})
		sb.WriteString("\n" + section.Render("History") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"h", "Browse past responses"},
			{"d", "Diff two history entries"},
		})

	case "editor":
		sb.WriteString(title.Render("Request Editor — Help") + "\n\n")
		sb.WriteString(tip.Render("Edit request fields. Tab between fields, Ctrl+S to save.") + "\n\n")

		sb.WriteString(section.Render("Navigation") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"Tab / Shift+Tab", "Move to next / previous field"},
			{"←/→ on Method", "Cycle through HTTP methods"},
			{"Enter on Header Key", "Move to header value"},
			{"Enter on Header Value", "Add new header row"},
		})
		sb.WriteString("\n" + section.Render("Text Editing (readline/emacs)") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"←/→", "Move cursor left/right"},
			{"Ctrl+A / Home", "Jump to start"},
			{"Ctrl+E / End", "Jump to end"},
			{"Ctrl+F / Ctrl+B", "Forward/backward one char"},
			{"Alt+F / Alt+B", "Forward/backward one word"},
			{"Ctrl+W", "Delete word backward"},
			{"Ctrl+U", "Clear from start to cursor"},
			{"Ctrl+K", "Clear from cursor to end"},
			{"Ctrl+D", "Delete character / delete header row"},
		})
		sb.WriteString("\n" + section.Render("Save / Cancel") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"Ctrl+S", "Save and close editor"},
			{"Esc", "Cancel without saving"},
		})

	case "search":
		sb.WriteString(title.Render("Search — Help") + "\n\n")
		sb.WriteString(tip.Render("Fuzzy search across all requests in your collection.") + "\n\n")
		writeKeys(&sb, key, [][2]string{
			{"Type", "Filter requests by name, method, or URL"},
			{"j/k / ↑/↓", "Navigate results"},
			{"Enter", "Select request and jump to it"},
			{"Backspace", "Delete search character"},
			{"Esc", "Close search"},
		})

	default:
		sb.WriteString(title.Render("Help") + "\n\n")
		sb.WriteString("Press " + key.Render("?") + " for full keybinding reference.\n")
		sb.WriteString("Press " + key.Render("F1") + " for context-sensitive help.\n")
	}

	return sb.String()
}

func writeKeys(sb *strings.Builder, keyStyle lipgloss.Style, keys [][2]string) {
	maxKeyLen := 0
	for _, k := range keys {
		if len(k[0]) > maxKeyLen {
			maxKeyLen = len(k[0])
		}
	}
	for _, k := range keys {
		padding := strings.Repeat(" ", maxKeyLen-len(k[0])+2)
		sb.WriteString("  " + keyStyle.Render(k[0]) + padding + k[1] + "\n")
	}
}
