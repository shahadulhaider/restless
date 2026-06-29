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
| `e` | Edit with `$EDITOR` (falls back to internal editor) |
| `E` | Edit with internal form editor |
| `D` | Delete request (with confirmation) |
| `Y` | Duplicate request |
| `?` | Full keybinding reference |
| `F1` | Context-sensitive help |
| `q` / `Ctrl+C` | Quit |

## Mouse

Mouse support is always on. Hold `Option`/`Shift` for your terminal's native text selection.

| Action | Result |
|--------|--------|
| Wheel | Scroll the pane under the cursor |
| Click | Focus a pane; select a request; expand a file/folder |
| Click `[r]`/`[s]` | Switch Request / Response view |
| Click a section header | Fold / unfold that accordion section |
| Click a JSON node (`▾`/`▸`) | Fold / unfold that object or array |
| Drag the pane divider | Resize the browser / detail split |

## Browser Pane

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate |
| `Enter` | Select / expand |
| `N` | New `.http` file |
| `F` | New folder |
| `R` | Rename |
| `M` | Move |

## Detail Pane

| Key | Action |
|-----|--------|
| `r` / `s` | Switch to Request / Response view |
| `Enter` / `Ctrl+R` | Send request |
| `Space` | Toggle fold on section under cursor |
| `1` / `2` / `3` | Toggle Body / Headers / Timing section |
| `za` | Fold / unfold the JSON object or array under the cursor (marked `▾`/`▸`) |
| `zo` / `zc` | Expand / collapse section under cursor |
| `zR` | Expand all sections and JSON nodes |
| `zM` | Collapse all sections |
| `j` / `k` | Scroll line by line |
| `Ctrl+D` / `Ctrl+U` | Scroll half page |
| `g` / `G` | Jump to top / bottom |
| `p` | Toggle pretty-print / raw |
| `w` | Toggle word wrap |
| `l` | Toggle line numbers |
| `f` | Search in body |
| `n` / `N` | Next / previous search match |
| `gp` | Jump to a JSON path (type the path, then Enter) |

## Visual Selection

Press `v` in the detail pane for character-level selection. A block cursor marks the active position.

| Key | Action |
|-----|--------|
| `v` | Enter visual selection mode |
| `h` / `l` / `j` / `k` | Move the cursor |
| `w` / `b` | Move by word |
| `0` / `$` | Jump to start / end of line |
| `y` | Copy the exact selection |
| `Esc` | Cancel |

## Yank (Copy)

| Key | Action |
|-----|--------|
| `yb` | Copy body |
| `yh` | Copy headers |
| `ya` | Copy all (full request or response) |
| `yc` | Copy as curl |
| `yl` | Copy the current line |
| `yp` | Copy the JSON path at the cursor |
| `yv` | Copy the JSON value at the cursor |
| `yi` | Copy the individual header or line |
| `yg` + key | Generate code (see below) |

> Copies use OSC52 in addition to the local clipboard tool, so they work over SSH and tmux.

## Code Generation (`yg` + key)

| Key | Language |
|-----|----------|
| `ygp` | Python (requests) |
| `ygj` | JavaScript (fetch) |
| `ygg` | Go (net/http) |
| `ygv` | Java (HttpClient) |
| `ygr` | Ruby (net/http) |
| `ygh` | HTTPie |
| `ygc` | curl |
| `ygw` | PowerShell |

## Internal Editor

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Navigate fields |
| `←` / `→` | Move cursor |
| `Ctrl+A` / `Home` | Start of field |
| `Ctrl+E` / `End` | End of field |
| `Ctrl+F` / `Ctrl+B` | Forward / backward char |
| `Alt+F` / `Alt+B` | Forward / backward word |
| `Ctrl+W` | Delete word backward |
| `Ctrl+U` | Clear to start |
| `Ctrl+K` | Clear to end |
| `Ctrl+D` | Delete char / header row |
| `Ctrl+V` / `Cmd+V` | Paste (also works in search and prompts) |
| `Ctrl+S` | Save |
| `Esc` | Cancel |
