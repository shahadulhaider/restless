# Restless Plan — Learnings

## Key Technical Decisions
- Go module: `github.com/shahadulhaider/restless`
- Bubbletea v2 (NOT v1): import `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`
- Bubbletea v2: `View()` returns `tea.View` (not string), use `tea.NewView(s)`, `tea.KeyPressMsg` (not `tea.KeyMsg`)
- JSON path: use `tidwall/gjson` for chain context body extraction
- Postman import: use `rbretecher/go-postman-collection`
- No CGo, no external runtime deps — pure Go single binary
