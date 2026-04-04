<div align="center">

# restless

**Your API workbench lives in the terminal.**

A full-featured HTTP client that runs entirely in your terminal. Uses `.http` files — the same plain-text format supported by JetBrains IDEs and VS Code REST Client. No Electron. No cloud sync. No account required.

[![CI](https://github.com/shahadulhaider/restless/actions/workflows/ci.yml/badge.svg)](https://github.com/shahadulhaider/restless/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/shahadulhaider/restless)](https://goreportcard.com/report/github.com/shahadulhaider/restless)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/shahadulhaider/restless)](https://github.com/shahadulhaider/restless/releases)

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

- **Interactive TUI** — browse collections, send requests, inspect responses with collapsible accordion view
- **`.http` files** — plain text, Git-friendly, JetBrains-compatible
- **Request/Response toggle** — `r`/`s` to switch views, both with fold/scroll/search/yank
- **Environments** — `http-client.env.json` with `$shared` + per-env variables, switch with `Ctrl+E`
- **Inline variables** — `@baseUrl = http://localhost:8000` right in your `.http` file
- **Dynamic variables** — `{{$uuid}}`, `{{$timestamp}}`, `{{$randomInt}}`, `{{$date}}`, and more
- **Request chaining** — `{{login.response.body.token}}` passes data between requests
- **Response assertions** — `# @assert status == 200` for CI/CD testing
- **Code generation** — `yg` + key to copy as Python, JavaScript, Go, Java, Ruby, HTTPie, curl, PowerShell
- **Import from anywhere** — Postman, Insomnia, Bruno, curl commands, OpenAPI/Swagger
- **$EDITOR integration** — press `e` to edit in nvim/vim/code
- **Vim-style commands** — `zo`/`zc`/`zR`/`zM` folds, `yb`/`yh`/`ya`/`yc` yank
- **Cookie jar** — cookies persist per environment automatically
- **Response history** — browse past responses with `h`, diff with `d`
- **Proxy & SSL** — `# @insecure`, `# @proxy`, `--insecure`, `--proxy` flags
- **Readline/emacs editing** — `Ctrl+A/E/W/U/K`, arrow keys, word navigation in editor

## Keyboard Shortcuts

Press `?` in the TUI for the full reference. Press `F1` for context-sensitive help.

| Key | Action |
|-----|--------|
| `j/k` | Navigate |
| `Enter` | Send request / select |
| `e` | Edit in `$EDITOR` |
| `r/s` | Request / Response view |
| `1/2/3/4` | Toggle sections |
| `Space` | Fold/unfold section |
| `yb/yh/ya/yc` | Copy body/headers/all/curl |
| `yg` + key | Generate code |
| `p` | Pretty/raw toggle |
| `f` | Search in body |
| `?` | Help |

[Full keybinding reference →](docs/keybindings.md)

## CLI

```bash
restless [directory]                    # Launch TUI
restless run <file> [--env name]        # Run headless (CI/CD)
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
| [All Keybindings](https://github.com/shahadulhaider/restless/wiki/Keybindings) | Complete keyboard reference |
| [Environments](https://github.com/shahadulhaider/restless/wiki/Environments) | Inline vars, env files, dynamic vars |
| [Assertions](https://github.com/shahadulhaider/restless/wiki/Assertions) | Response assertions for CI/CD |
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
