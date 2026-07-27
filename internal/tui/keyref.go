package tui

import "strings"

// This file is the single source of truth for the keyboard reference. Both the
// in-app `?` screen (fullKeybindingReference) and docs/keybindings.md
// (renderKeybindingsMarkdown) are rendered from keyReference, so the two can
// never drift apart. TestKeybindingsDocInSync enforces that the committed
// markdown matches the generator output; regenerate with:
//
//	go test ./internal/tui -run TestKeybindingsDocInSync -update
//
// Only document keys that have a real handler in internal/tui.

// keyBinding is one row of the reference: the key (or mouse gesture) and what
// it does.
type keyBinding struct {
	Keys string
	Desc string
}

// keySection groups related bindings under a heading. Note is optional prose
// that is emitted into the markdown document only — the in-app help renders
// headings and bindings alone so it stays readable in narrow panes.
type keySection struct {
	Title    string
	Note     string
	Bindings []keyBinding
}

const (
	keyReferenceTitle = "restless — Keyboard Reference"
	keyReferenceDocH1 = "Keyboard Reference"
	keyReferenceIntro = "Press `?` in the TUI for an interactive version of this reference.\n" +
		"Press `F1` for context-sensitive help."
)

var keyReference = []keySection{
	{
		Title: "Global",
		Bindings: []keyBinding{
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
		},
	},
	{
		Title: "Mouse",
		Note:  "Mouse support is always on. Hold `Option`/`Shift` for your terminal's native text selection.",
		Bindings: []keyBinding{
			{"Wheel", "Scroll the pane under the cursor"},
			{"Click", "Focus a pane; select a request; expand a file/folder"},
			{"Click [r]/[s]", "Switch Request / Response view"},
			{"Click a tab", "Switch to that tab"},
			{"Click ▾/▸ line", "Fold/unfold the JSON node on that line"},
			{"Drag divider", "Resize the browser / detail split"},
		},
	},
	{
		Title: "Browser Pane",
		Bindings: []keyBinding{
			{"j/k / ↑/↓", "Navigate requests"},
			{"Enter", "Select / expand folder or file"},
			{"N", "Create new .http file"},
			{"F", "Create new folder"},
			{"R", "Rename file or folder"},
			{"M", "Move file or folder"},
		},
	},
	{
		Title: "Detail Pane — Navigation",
		Bindings: []keyBinding{
			{"r / s", "Switch to Request / Response view"},
			{"Enter / Ctrl+R", "Send request"},
			{"i", "Inline-edit the request (focuses headers on the Headers tab)"},
		},
	},
	{
		Title: "Detail Pane — Tabs",
		Note: "The detail pane is tabbed. The request view has Body, Headers and Metadata; " +
			"the response view has Body, Headers, Timing and — only when the request has " +
			"assertions — Assertions. Switching tabs resets the cursor, selection, search and " +
			"scroll position. A response with a failing assertion opens on the Assertions tab.",
		Bindings: []keyBinding{
			{"1 / 2 / 3", "Select the first / second / third tab"},
			{"4", "Select the Assertions tab (response view, when assertions exist)"},
			{"Space", "Cycle to the next tab"},
		},
	},
	{
		Title: "Detail Pane — JSON Folding",
		Bindings: []keyBinding{
			{"za", "Fold/unfold the JSON object/array under the cursor"},
			{"zR", "Open all JSON folds"},
			{"zM", "Close all JSON folds"},
		},
	},
	{
		Title: "Detail Pane — Scrolling",
		Bindings: []keyBinding{
			{"j/k / ↑/↓", "Scroll line by line"},
			{"Ctrl+D / Ctrl+U", "Scroll half page down / up"},
			{"gg / G", "Jump to top / bottom"},
		},
	},
	{
		Title: "Detail Pane — Selection",
		Note: "Press `v` for character-level selection; a block cursor marks the active position. " +
			"`V` selects whole lines.",
		Bindings: []keyBinding{
			{"v", "Enter visual selection mode (character-level)"},
			{"V", "Enter visual line selection mode"},
			{"h/l/j/k", "Move cursor while selecting"},
			{"w/b", "Move by word while selecting"},
			{"0/$", "Jump to start/end of line while selecting"},
			{"y", "Copy the exact selection to clipboard"},
			{"Esc", "Cancel selection"},
			{"gp", "Jump to JSON path (type path, Enter to jump)"},
		},
	},
	{
		Title: "Detail Pane — Body Viewer",
		Bindings: []keyBinding{
			{"p", "Toggle pretty-print / raw"},
			{"w", "Toggle word wrap"},
			{"l", "Toggle line numbers"},
			{"f", "Search in body"},
			{"n / N", "Next / previous search match"},
		},
	},
	{
		Title: "Detail Pane — History",
		Bindings: []keyBinding{
			{"h", "Open the history overlay for the selected request"},
			{"j / k", "Move through history entries"},
			{"Enter", "Load the selected historical response"},
			{"d", "Start a diff, then d again on a second entry"},
			{"Esc", "Close the overlay"},
		},
	},
	{
		Title: "Yank (Copy to Clipboard)",
		Note:  "Copies use OSC52 in addition to the local clipboard tool, so they work over SSH and tmux.",
		Bindings: []keyBinding{
			{"yb", "Copy body"},
			{"yh", "Copy headers"},
			{"ya", "Copy all (full request or response)"},
			{"yc", "Copy as curl command"},
			{"yl", "Copy current line"},
			{"yp", "Copy JSON path at cursor"},
			{"yv", "Copy JSON value at cursor path"},
			{"yi", "Copy individual item (header or line)"},
			{"yf", "Copy the JSON fold block under the cursor"},
			{"yg + key", "Generate code (see below)"},
		},
	},
	{
		Title: "Code Generation (yg + key)",
		Bindings: []keyBinding{
			{"ygp", "Python (requests)"},
			{"ygj", "JavaScript (fetch)"},
			{"ygg", "Go (net/http)"},
			{"ygv", "Java (HttpClient)"},
			{"ygr", "Ruby (net/http)"},
			{"ygh", "HTTPie"},
			{"ygc", "curl"},
			{"ygw", "PowerShell"},
		},
	},
	{
		Title: "Which-key",
		Note: "Pressing a prefix key — or idling for 1.5 seconds in the detail pane — pops up a " +
			"hint listing the keys available next.",
		Bindings: []keyBinding{
			{"g", "Goto prefix (gg, gp)"},
			{"z", "Fold prefix (za, zR, zM)"},
			{"y", "Yank prefix"},
			{"yg", "Code generation prefix"},
		},
	},
	{
		Title: "Internal Editor",
		Bindings: []keyBinding{
			{"Tab / Shift+Tab", "Navigate fields"},
			{"←/→", "Move cursor in text field"},
			{"Ctrl+A / Home", "Jump to start of field"},
			{"Ctrl+E / End", "Jump to end of field"},
			{"Ctrl+F / Ctrl+B", "Forward / backward one character"},
			{"Alt+F / Alt+B", "Forward / backward one word"},
			{"Ctrl+W", "Delete word backward"},
			{"Ctrl+U", "Clear to start of line"},
			{"Ctrl+K", "Clear to end of line"},
			{"Ctrl+D", "Delete header row"},
			{"Ctrl+V / Cmd+V", "Paste (also works in search and prompts)"},
			{"Ctrl+S", "Save"},
			{"Esc", "Cancel"},
		},
	},
}

// renderKeybindingsMarkdown renders keyReference as the full contents of
// docs/keybindings.md. The committed file must be byte-identical to this
// output; see TestKeybindingsDocInSync for how to regenerate it.
func renderKeybindingsMarkdown() string {
	var sb strings.Builder
	sb.WriteString("# " + keyReferenceDocH1 + "\n\n")
	sb.WriteString(keyReferenceIntro + "\n")

	for _, sec := range keyReference {
		sb.WriteString("\n## " + sec.Title + "\n\n")
		if sec.Note != "" {
			sb.WriteString(sec.Note + "\n\n")
		}
		sb.WriteString("| Key | Action |\n")
		sb.WriteString("|-----|--------|\n")
		for _, b := range sec.Bindings {
			sb.WriteString("| " + markdownKeys(b.Keys) + " | " + escapeTableCell(b.Desc) + " |\n")
		}
	}

	return sb.String()
}

// markdownKeys renders a binding's key column as code spans, splitting on the
// " / " separator so "q / Ctrl+C" becomes "`q` / `Ctrl+C`".
func markdownKeys(keys string) string {
	parts := strings.Split(keys, " / ")
	for i, p := range parts {
		parts[i] = "`" + p + "`"
	}
	return strings.Join(parts, " / ")
}

// escapeTableCell escapes the one character that would break a markdown table.
func escapeTableCell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
