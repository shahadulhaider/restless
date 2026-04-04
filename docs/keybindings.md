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
| `Space` | Toggle fold section under cursor |
| `1` / `2` / `3` / `4` | Toggle sections |
| `zo` | Expand section under cursor |
| `zc` | Collapse section under cursor |
| `zR` | Expand all sections |
| `zM` | Collapse all sections |
| `j` / `k` | Scroll line by line |
| `Ctrl+D` / `Ctrl+U` | Scroll half page |
| `g` / `G` | Jump to top / bottom |
| `p` | Toggle pretty-print / raw |
| `w` | Toggle word wrap |
| `l` | Toggle line numbers |
| `f` | Search in body |
| `n` / `N` | Next / previous search match |
| `h` | Toggle response history |
| `d` | Diff two history entries |

## Yank (Copy)

| Key | Action |
|-----|--------|
| `yb` | Copy body |
| `yh` | Copy headers |
| `ya` | Copy all (full request or response) |
| `yc` | Copy as curl |
| `yg` + key | Generate code (see below) |

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
| `Ctrl+S` | Save |
| `Esc` | Cancel |
