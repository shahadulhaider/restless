<div align="center">

# restless

**Your API workbench lives in the terminal.**

A full-featured HTTP client that runs entirely in your terminal. Uses `.http` files — the same plain-text format supported by JetBrains IDEs and VS Code REST Client. No Electron. No cloud sync. No account required.

[![CI](https://github.com/shahadulhaider/restless/actions/workflows/ci.yml/badge.svg)](https://github.com/shahadulhaider/restless/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/shahadulhaider/restless)](https://goreportcard.com/report/github.com/shahadulhaider/restless)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/shahadulhaider/restless)](https://github.com/shahadulhaider/restless/releases)

<img src="https://github.com/shahadulhaider/restless/releases/download/demo-assets/hero.gif" alt="Browsing a collection, sending a request, and folding the JSON response in restless" width="100%">

</div>

---

## Install

```bash
# Homebrew
brew tap shahadulhaider/tap && brew install restless

# Go
go install github.com/shahadulhaider/restless/cmd/restless@latest

# Binary — download from Releases
```

## Quick Start

```bash
# Create a request file
cat > api.http << 'EOF'
@baseUrl = https://httpbin.org

# @name health
GET {{baseUrl}}/get
Accept: application/json

###

# @name echo
POST {{baseUrl}}/post
Content-Type: application/json

{"message": "hello from restless", "time": "{{$isoTimestamp}}"}

# @assert status == 200
# @assert body.$.json.message == "hello from restless"
EOF

# Launch the TUI
restless .

# Or run headless (CI/CD)
restless run api.http
```

## Features

- **Interactive TUI** — browse collections, send requests, inspect responses in a tabbed detail pane
- **Mouse support** — wheel-scroll, click to select/focus, click a tab to switch, click `▾`/`▸` to fold JSON, drag the divider to resize
- **`.http` files** — plain text, Git-friendly, JetBrains-compatible
- **Request/Response toggle** — `r`/`s` to switch views, both with syntax highlighting, fold/scroll/search/yank
- **JSON folding** — collapse/expand individual objects and arrays with `za`
- **Environments** — `restless.env.json` with `$shared` + per-env variables, switch with `Ctrl+E`
- **Inline variables** — `@baseUrl = http://localhost:8000` right in your `.http` file
- **Dynamic variables** — `{{$uuid}}`, `{{$timestamp}}`, `{{$randomInt}}`, `{{$date}}`, and more
- **Request chaining** — `{{login.response.body.token}}` passes data between requests
- **Response assertions** — `# @assert status == 200` for CI/CD testing
- **Collection runner** — `--data users.csv` to run requests with parameterized data from CSV/JSON
- **Code generation** — `yg` + key to copy as Python, JavaScript, Go, Java, Ruby, HTTPie, curl, PowerShell
- **Import from anywhere** — Postman, Insomnia, Bruno, curl commands, OpenAPI/Swagger
- **$EDITOR integration** — press `e` to edit in nvim/vim/code
- **Vim-style commands** — `za`/`zR`/`zM` JSON folds, `v`/`V` visual selection, `yb`/`yh`/`ya`/`yc`/`yf` yank
- **Cookie jar** — cookies persist per environment automatically
- **Pre-request & post-response scripting** — JavaScript (ES5.1) via `# @pre-request { ... }` and `# @post-response { ... }` with crypto builtins (hmac, sha256, base64)
- **Proxy & SSL** — `# @insecure`, `# @proxy`, `--insecure`, `--proxy` flags
- **Readline/emacs editing** — `Ctrl+A/E/W/U/K`, arrow keys, word navigation, and paste (`Ctrl`/`Cmd`+`V`) in editor, search, and prompts

## Demo

<details>
<summary><b>JSON folding and tabs</b> — collapse nodes with <code>za</code>, jump tabs with <code>1</code>–<code>4</code></summary>

<img src="https://github.com/shahadulhaider/restless/releases/download/demo-assets/folding.gif" alt="Folding JSON nodes and switching between the Body, Headers and Timing tabs" width="100%">

</details>

<details>
<summary><b>Visual selection and yank</b> — <code>v</code>/<code>V</code> to select, <code>yb</code>/<code>yc</code> to copy</summary>

<img src="https://github.com/shahadulhaider/restless/releases/download/demo-assets/yank.gif" alt="Selecting response lines in visual mode and copying the body and a curl command" width="100%">

</details>

<details>
<summary><b>Code generation</b> — <code>yg</code> then a language key</summary>

<img src="https://github.com/shahadulhaider/restless/releases/download/demo-assets/codegen.gif" alt="The yg which-key popup listing languages, then generating JavaScript" width="100%">

</details>

<details>
<summary><b>Environment switching</b> — <code>Ctrl+E</code> to swap variables</summary>

<img src="https://github.com/shahadulhaider/restless/releases/download/demo-assets/envs.gif" alt="Switching environments and re-sending a request with the new variables resolved" width="100%">

</details>

> Recorded with [VHS](https://github.com/charmbracelet/vhs). The tapes and demo collection live in [`docs/demo/`](docs/demo/) — run `vhs docs/demo/hero.tape` to regenerate.

## Keyboard Shortcuts

Press `?` in the TUI for the full reference. Press `F1` for context-sensitive help.

| Key | Action |
|-----|--------|
| `j/k` | Navigate |
| `Enter` | Send request / select |
| `e` | Edit in `$EDITOR` |
| `r/s` | Request / Response view |
| `1/2/3` | Select tab |
| `4` | Assertions tab (response, when present) |
| `Space` | Next tab |
| `za` | Fold/unfold JSON node |
| `v/V` | Visual selection (character / line) |
| Mouse | Scroll, click, drag to resize |
| `yb/yh/ya/yc` | Copy body/headers/all/curl |
| `yf` | Copy JSON fold block |
| `yg` + key | Generate code |
| `p` | Pretty/raw toggle |
| `f` | Search in body |
| `?` | Help |

[Full keybinding reference →](docs/keybindings.md)

## CLI

```bash
restless [directory]                    # Launch TUI
restless run <file> [--env name]        # Run headless (CI/CD)
restless run <file> --data data.csv    # Parameterized run with data file
restless import postman <file>          # Import Postman collection
restless import insomnia <file>         # Import Insomnia export
restless import bruno <dir>             # Import Bruno collection
restless import curl "<command>"        # Import curl command
restless import openapi <spec>          # Import OpenAPI/Swagger
```

## Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](https://github.com/shahadulhaider/restless/wiki/Getting-Started) | First collection, environments, CI/CD |
| [.http File Format](https://github.com/shahadulhaider/restless/wiki/HTTP-File-Format) | Full syntax reference, variables, assertions |
| [All Keybindings](docs/keybindings.md) | Complete keyboard reference |
| [Environments](https://github.com/shahadulhaider/restless/wiki/Environments) | Inline vars, env files, dynamic vars |
| [Assertions](https://github.com/shahadulhaider/restless/wiki/Assertions) | Response assertions for CI/CD |
| [Scripting](https://github.com/shahadulhaider/restless/wiki/Scripting) | Pre-request & post-response JavaScript |
| [Collection Runner](https://github.com/shahadulhaider/restless/wiki/Collection-Runner) | Parameterized runs with CSV/JSON data |
| [Importing Collections](https://github.com/shahadulhaider/restless/wiki/Importing-Collections) | Postman, Insomnia, Bruno, curl, OpenAPI |
| [Code Generation](https://github.com/shahadulhaider/restless/wiki/Code-Generation) | Python, JS, Go, Java, Ruby, HTTPie, curl, PowerShell |
| [FAQ](https://github.com/shahadulhaider/restless/wiki/FAQ) | Common questions and troubleshooting |

## Contributing

Contributions welcome. Please open an issue first for non-trivial changes.

```bash
git clone https://github.com/shahadulhaider/restless.git
cd restless
go build ./cmd/restless
go test ./...
```

## License

[MIT](LICENSE) — Shahadul Haider
