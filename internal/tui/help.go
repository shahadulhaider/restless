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

// fullKeybindingReference renders keyReference (see keyref.go) for the in-app
// `?` screen. docs/keybindings.md is rendered from the same data.
func fullKeybindingReference() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(colorText)
	section := lipgloss.NewStyle().Bold(true).Foreground(colorBorderActive)
	key := lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))

	var sb strings.Builder
	sb.WriteString(title.Render(keyReferenceTitle) + "\n\n")

	for i, sec := range keyReference {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(section.Render(sec.Title) + "\n")
		keys := make([][2]string, len(sec.Bindings))
		for j, b := range sec.Bindings {
			keys[j] = [2]string{b.Keys, b.Desc}
		}
		writeKeys(&sb, key, keys)
	}

	return sb.String()
}

// --- Context-sensitive help (F1) ---

func contextHelp(ctx string) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(colorText)
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
		sb.WriteString(tip.Render("View and send the selected request. Tabs hold the body, headers, and metadata.") + "\n\n")

		sb.WriteString(section.Render("Tabs") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"1 / 2 / 3", "Body / Headers / Metadata"},
			{"Space", "Cycle to the next tab"},
		})
		sb.WriteString("\n" + section.Render("Sending") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"Enter / Ctrl+R", "Send the request — response view opens automatically"},
		})
		sb.WriteString("\n" + section.Render("Editing") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"e", "Open in $EDITOR for full editing power"},
			{"E", "Open internal form editor"},
			{"i", "Inline-edit here (focuses headers on the Headers tab)"},
		})
		sb.WriteString("\n" + section.Render("Variables") + "\n")
		sb.WriteString("  Use " + key.Render("{{varName}}") + " in URL, headers, or body.\n")
		sb.WriteString("  Define inline: " + key.Render("@baseUrl = http://localhost:8000") + "\n")
		sb.WriteString("  Or in " + key.Render("http-client.env.json") + " and switch with " + key.Render("Ctrl+E") + "\n")
		sb.WriteString("  Dynamic: " + key.Render("{{$uuid}}") + " " + key.Render("{{$timestamp}}") + " " + key.Render("{{$randomInt}}") + "\n")
		sb.WriteString("\n" + section.Render("Assertions") + "\n")
		sb.WriteString("  Add " + key.Render("# @assert status == 200") + " to validate responses.\n")
		sb.WriteString("  Run headless: " + key.Render("restless run api.http --env dev") + "\n")
		sb.WriteString("\n" + section.Render("Yank & Selection") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"yb/yh/ya", "Copy body / headers / full request"},
			{"yc", "Copy as curl command"},
			{"yl/yp/yv/yi", "Copy line / JSON path / value / item"},
			{"yf", "Copy the JSON fold block under the cursor"},
			{"yg + key", "Generate code (Python, JS, Go, etc.)"},
			{"v / V", "Character / line selection (h/l/j/k/w/b, y to copy)"},
			{"gp", "Jump to JSON path"},
		})
		sb.WriteString("\n" + tip.Render("Press a prefix (g, z, y) or idle 1.5s for a which-key popup.") + "\n")

	case "detail-response":
		sb.WriteString(title.Render("Response View — Help") + "\n\n")
		sb.WriteString(tip.Render("Inspect the response. It opens on the Body tab; switch tabs with 1/2/3/4 or Space.") + "\n\n")

		sb.WriteString(section.Render("Tabs") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"1", "Body — response body with JSON/XML formatting"},
			{"2", "Headers — response headers"},
			{"3", "Timing — DNS, TLS, TTFB waterfall"},
			{"4", "Assertions — only when the request has assertions"},
			{"Space", "Cycle to the next tab"},
		})
		sb.WriteString("\n" + tip.Render("A failing assertion opens the response on the Assertions tab.") + "\n")
		sb.WriteString("\n" + section.Render("JSON Folding") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"za", "Fold/unfold the JSON object/array under the cursor"},
			{"zR / zM", "Open / close all JSON folds"},
		})
		sb.WriteString("\n" + section.Render("History") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"h", "Open the history overlay (j/k move, Enter loads, d diffs, Esc closes)"},
		})
		sb.WriteString("\n" + section.Render("Body Viewer") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"p", "Toggle pretty-print vs raw"},
			{"w", "Toggle word wrap"},
			{"l", "Toggle line numbers"},
			{"f", "Search in response body"},
			{"n/N", "Next/previous search match"},
		})
		sb.WriteString("\n" + section.Render("Copy & Selection") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"yb", "Copy response body"},
			{"yh", "Copy response headers"},
			{"ya", "Copy full response (status + headers + body)"},
			{"yl", "Copy current line"},
			{"yp", "Copy JSON path at cursor"},
			{"yv", "Copy JSON value at cursor path"},
			{"yi", "Copy individual header or line"},
			{"yf", "Copy the JSON fold block under the cursor"},
			{"yg + key", "Generate code from the request"},
			{"v / V", "Character / line selection (h/l/j/k/w/b, y to copy)"},
			{"gp", "Jump to JSON path"},
		})
		sb.WriteString("\n" + tip.Render("Press a prefix (g, z, y) or idle 1.5s for a which-key popup.") + "\n")

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
		sb.WriteString("\n" + section.Render("Paste / Save / Cancel") + "\n")
		writeKeys(&sb, key, [][2]string{
			{"Ctrl+V / Cmd+V", "Paste into the focused field (multi-line into body)"},
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
