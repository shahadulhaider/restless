# Restless Plan — Issues & Gotchas

## Bubbletea v2 Breaking Changes (CRITICAL)
- Import paths changed: use `charm.land/*` NOT `github.com/charmbracelet/*`
- `View()` returns `tea.View`, not `string` — use `tea.NewView(s)`
- `KeyMsg` → `KeyPressMsg`
- Alt screen via view fields, not `tea.WithAltScreen()`

## .http Spec Edge Cases
- Body ends at `###` or EOF — empty lines WITHIN body are body content (not end-of-body)
- First request doesn't need a leading `###`
- `< ./path` is relative to the .http file location, NOT CWD
- Path traversal must be blocked for file body loading

## Environment Merge Priority (highest wins)
1. Private env-specific vars
2. Public env-specific vars
3. Private $shared vars
4. Public $shared vars
