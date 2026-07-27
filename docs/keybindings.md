# Keyboard Reference

Press `?` in the TUI for an interactive version of this reference.
Press `F1` for context-sensitive help.

## Global

| Key | Action |
|-----|--------|
| `Tab` | Switch between browser and detail panes |
| `/` | Fuzzy search requests |
| `Ctrl+E` | Switch environment |
| `n` | Create new request (internal editor) |
| `e` | Edit with $EDITOR (falls back to internal) |
| `E` | Edit with internal form editor |
| `D` | Delete request (with confirmation) |
| `Y` | Duplicate request |
| `?` | This help screen |
| `F1` | Context-sensitive help |
| `q` / `Ctrl+C` | Quit |

## Mouse

Mouse support is always on. Hold `Option`/`Shift` for your terminal's native text selection.

| Key | Action |
|-----|--------|
| `Wheel` | Scroll the pane under the cursor |
| `Click` | Focus a pane; select a request; expand a file/folder |
| `Click [r]/[s]` | Switch Request / Response view |
| `Click a tab` | Switch to that tab |
| `Click ▾/▸ line` | Fold/unfold the JSON node on that line |
| `Drag divider` | Resize the browser / detail split |

## Browser Pane

| Key | Action |
|-----|--------|
| `j/k` / `↑/↓` | Navigate requests |
| `Enter` | Select / expand folder or file |
| `N` | Create new .http file |
| `F` | Create new folder |
| `R` | Rename file or folder |
| `M` | Move file or folder |

## Detail Pane — Navigation

| Key | Action |
|-----|--------|
| `r` / `s` | Switch to Request / Response view |
| `Enter` / `Ctrl+R` | Send request |
| `i` | Inline-edit the request (focuses headers on the Headers tab) |

## Detail Pane — Tabs

The detail pane is tabbed. The request view has Body, Headers and Metadata; the response view has Body, Headers, Timing and — only when the request has assertions — Assertions. Switching tabs resets the cursor, selection, search and scroll position. A response with a failing assertion opens on the Assertions tab.

| Key | Action |
|-----|--------|
| `1` / `2` / `3` | Select the first / second / third tab |
| `4` | Select the Assertions tab (response view, when assertions exist) |
| `Space` | Cycle to the next tab |

## Detail Pane — JSON Folding

| Key | Action |
|-----|--------|
| `za` | Fold/unfold the JSON object/array under the cursor |
| `zR` | Open all JSON folds |
| `zM` | Close all JSON folds |

## Detail Pane — Scrolling

| Key | Action |
|-----|--------|
| `j/k` / `↑/↓` | Scroll line by line |
| `Ctrl+D` / `Ctrl+U` | Scroll half page down / up |
| `gg` / `G` | Jump to top / bottom |

## Detail Pane — Selection

Press `v` for character-level selection; a block cursor marks the active position. `V` selects whole lines.

| Key | Action |
|-----|--------|
| `v` | Enter visual selection mode (character-level) |
| `V` | Enter visual line selection mode |
| `h/l/j/k` | Move cursor while selecting |
| `w/b` | Move by word while selecting |
| `0/$` | Jump to start/end of line while selecting |
| `y` | Copy the exact selection to clipboard |
| `Esc` | Cancel selection |
| `gp` | Jump to JSON path (type path, Enter to jump) |

## Detail Pane — Body Viewer

| Key | Action |
|-----|--------|
| `p` | Toggle pretty-print / raw |
| `w` | Toggle word wrap |
| `l` | Toggle line numbers |
| `f` | Search in body |
| `n` / `N` | Next / previous search match |

## Detail Pane — History

| Key | Action |
|-----|--------|
| `h` | Open the history overlay for the selected request |
| `j` / `k` | Move through history entries |
| `Enter` | Load the selected historical response |
| `d` | Start a diff, then d again on a second entry |
| `Esc` | Close the overlay |

## Yank (Copy to Clipboard)

Copies use OSC52 in addition to the local clipboard tool, so they work over SSH and tmux.

| Key | Action |
|-----|--------|
| `yb` | Copy body |
| `yh` | Copy headers |
| `ya` | Copy all (full request or response) |
| `yc` | Copy as curl command |
| `yl` | Copy current line |
| `yp` | Copy JSON path at cursor |
| `yv` | Copy JSON value at cursor path |
| `yi` | Copy individual item (header or line) |
| `yf` | Copy the JSON fold block under the cursor |
| `yg + key` | Generate code (see below) |

## Code Generation (yg + key)

| Key | Action |
|-----|--------|
| `ygp` | Python (requests) |
| `ygj` | JavaScript (fetch) |
| `ygg` | Go (net/http) |
| `ygv` | Java (HttpClient) |
| `ygr` | Ruby (net/http) |
| `ygh` | HTTPie |
| `ygc` | curl |
| `ygw` | PowerShell |

## Which-key

Pressing a prefix key — or idling for 1.5 seconds in the detail pane — pops up a hint listing the keys available next.

| Key | Action |
|-----|--------|
| `g` | Goto prefix (gg, gp) |
| `z` | Fold prefix (za, zR, zM) |
| `y` | Yank prefix |
| `yg` | Code generation prefix |

## Internal Editor

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Navigate fields |
| `←/→` | Move cursor in text field |
| `Ctrl+A` / `Home` | Jump to start of field |
| `Ctrl+E` / `End` | Jump to end of field |
| `Ctrl+F` / `Ctrl+B` | Forward / backward one character |
| `Alt+F` / `Alt+B` | Forward / backward one word |
| `Ctrl+W` | Delete word backward |
| `Ctrl+U` | Clear to start of line |
| `Ctrl+K` | Clear to end of line |
| `Ctrl+D` | Delete header row |
| `Ctrl+V` / `Cmd+V` | Paste (also works in search and prompts) |
| `Ctrl+S` | Save |
| `Esc` | Cancel |
