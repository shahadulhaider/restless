<div align="center">

# restless

**Your API workbench lives in the terminal.**

Restless is a full-featured HTTP client that runs entirely in your terminal. It speaks `.http` files — the same plain-text format used by JetBrains IDEs and VS Code REST Client. No Electron. No cloud sync. No account required. Just your requests, version-controlled in Git, executable from a TUI or CI pipeline.

[![CI](https://github.com/shahadulhaider/restless/actions/workflows/ci.yml/badge.svg)](https://github.com/shahadulhaider/restless/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/shahadulhaider/restless)](https://goreportcard.com/report/github.com/shahadulhaider/restless)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

## Why restless?

Most API tools want you to live inside their app. Restless takes the opposite approach:

- **Plain text files** — `.http` files are readable, diffable, and belong in your repo
- **Terminal native** — works over SSH, in tmux, on headless servers
- **Zero lock-in** — your requests are JetBrains-compatible; switch tools anytime
- **Fast** — starts instantly, no splash screens, no update prompts

If you've ever wished Postman was a CLI tool that understood Git, this is it.

## Install

**Homebrew**
```bash
brew tap shahadulhaider/tap
brew install restless
```

**Go**
```bash
go install github.com/shahadulhaider/restless/cmd/restless@latest
```

**Binary**

Grab the latest from [Releases](https://github.com/shahadulhaider/restless/releases) — available for Linux, macOS, and Windows.

## 30-Second Demo

```bash
# Create a request file
cat > api.http << 'EOF'
# @name health
GET https://httpbin.org/get
Accept: application/json

###

# @name echo
POST https://httpbin.org/post
Content-Type: application/json

{"message": "hello from restless"}
EOF

# Launch the TUI
restless .

# Or run headless
restless run api.http
```

## Features

### Interactive TUI

Browse your collection, send requests, inspect responses — all from the keyboard.

- **Split-pane layout** — collection browser on the left, request/response detail on the right
- **Fuzzy search** — press `/` to find any request by name, method, or URL
- **Request/Response viewer** — toggle between request and response views (`r`/`s`), both with collapsible accordion sections, JSON pretty-printing, XML/HTML indentation, line numbers, word wrap, search, vim-style fold commands (`zo`/`zc`/`zR`/`zM`)
- **Response history** — every response is saved; browse past responses with `h`, diff any two with `d`
- **Timing waterfall** — color-coded DNS, TCP, TLS, TTFB, and body transfer breakdown

### Request CRUD

Create, edit, and manage requests without leaving the terminal.

- **Inline editor** — press `n` to create, `E` to edit. Full form with method selector, URL, headers, body, and metadata fields
- **$EDITOR support** — press `ctrl+e` to open the `.http` file in your preferred editor (vim, nvim, code, etc.)
- **Duplicate** — `Y` to clone a request
- **Delete** — `D` with confirmation dialog

### Collection Management

Organize your requests into files and folders.

- `N` — create a new `.http` file
- `F` — create a new folder
- `R` — rename a file or folder
- `M` — move a file or folder
- `D` — delete with confirmation

### Environments

Switch between dev, staging, and production with a single keystroke.

```json
// http-client.env.json
{
  "$shared": {
    "baseUrl": "https://api.example.com"
  },
  "dev": {
    "token": "dev-secret"
  },
  "prod": {
    "token": "prod-secret"
  }
}
```

Press `ctrl+e` in the TUI to switch. Variables are expanded as `{{token}}` in your requests.

### Request Chaining

Pass data between requests using response references:

```http
# @name login
POST {{baseUrl}}/auth/login
Content-Type: application/json

{"email": "user@example.com", "password": "secret"}

###

# @name getProfile
GET {{baseUrl}}/me
Authorization: Bearer {{login.response.body.token}}
```

### Copy as curl

Press `y` on any request to copy it as a `curl` command to your clipboard. Paste it into Slack, docs, or a shell.

### Import From Anywhere

Already have a collection? Bring it over:

```bash
restless import postman   collection.json     # Postman v2.1
restless import insomnia  export.json          # Insomnia v4
restless import bruno     ./my-collection/     # Bruno directory
restless import curl      "curl -X POST ..."   # curl command
restless import openapi   spec.yaml            # OpenAPI 3.x / Swagger 2.0
```

All importers produce standard `.http` files. Environments are converted to `http-client.env.json`.

### Headless Runner

Run requests in CI/CD pipelines or scripts:

```bash
# Run all requests in a file
restless run api.http --env production

# Fail fast on first error
restless run api.http --env staging --fail-fast
```

### Cookie Jar

Cookies persist automatically per environment. Login once, and subsequent requests carry the session — just like a browser.

### Detailed Timing

Every response includes a timing waterfall: DNS, TCP connect, TLS handshake, time-to-first-byte, and body transfer. See exactly where your latency comes from.

## Keyboard Reference

### Global

| Key | Action |
|-----|--------|
| `Tab` | Switch between browser and detail panes |
| `/` | Fuzzy search requests |
| `ctrl+e` | Switch environment |
| `n` | Create new request |
| `e` | Edit in `$EDITOR` (falls back to internal editor) |
| `E` | Edit in internal form editor |
| `D` | Delete (with confirmation) |
| `Y` | Duplicate request |
| `yb` | Copy response body to clipboard |
| `yh` | Copy response headers to clipboard |
| `ya` | Copy full response to clipboard |
| `yc` | Copy request as curl to clipboard |
| `q` / `ctrl+c` | Quit |

### Browser Pane

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate |
| `Enter` | Select / expand |
| `N` | New `.http` file |
| `F` | New folder |
| `R` | Rename |
| `M` | Move |

### Detail Pane

| Key | Action |
|-----|--------|
| `r` | Switch to request view |
| `s` | Switch to response view |
| `Enter` / `ctrl+r` | Send request |
| `Space` | Toggle fold section under cursor |
| `1` / `2` / `3` | Toggle sections (Body/Headers/Metadata or Body/Headers/Timing) |
| `zo` | Expand section under cursor |
| `zc` | Collapse section under cursor |
| `zR` | Expand all sections |
| `zM` | Collapse all sections |
| `p` | Toggle pretty-print / raw |
| `j` / `k` | Scroll line by line |
| `ctrl+d` / `ctrl+u` | Scroll half page down / up |
| `g` / `G` | Jump to top / bottom |
| `f` | Search within response body |
| `n` / `N` | Next / previous search match |
| `w` | Toggle word wrap |
| `l` | Toggle line numbers |
| `h` | Toggle response history |
| `d` | Diff two history entries |

### Editor Overlay

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Navigate fields |
| `←` / `→` | Cycle method (on method field) |
| `Enter` | Edit value / add header row |
| `ctrl+d` | Delete header row |
| `ctrl+w` | Delete word backward |
| `ctrl+u` | Clear field |
| `ctrl+s` | Save |
| `Esc` | Cancel |

## CLI Reference

```
restless [directory]                               Launch TUI
restless run <file.http> [--env name] [--fail-fast] Run requests headlessly
restless import postman  <file> [--output dir]      Import Postman collection
restless import insomnia <file> [--output dir]      Import Insomnia export
restless import bruno    <dir>  [--output dir]      Import Bruno collection
restless import curl     <cmd>  [--output dir]      Import curl command
restless import openapi  <spec> [--output dir]      Import OpenAPI/Swagger spec
restless version                                    Print version
```

## `.http` File Format

```http
# @name requestName
# @no-redirect
# @timeout 30
GET https://api.example.com/users/{{userId}} HTTP/1.1
Authorization: Bearer {{token}}
Accept: application/json

###

# @name createUser
POST https://api.example.com/users
Content-Type: application/json

{
  "name": "Alice",
  "email": "alice@example.com"
}

###

# File body reference
PUT https://api.example.com/config
Content-Type: application/json

< ./payload.json
```

### Metadata Tags

| Tag | Effect |
|-----|--------|
| `# @name <name>` | Name the request (used in chaining and display) |
| `# @no-redirect` | Don't follow redirects |
| `# @no-cookie-jar` | Don't send/store cookies |
| `# @timeout <seconds>` | Request timeout |
| `# @connection-timeout <seconds>` | Connection timeout |

Compatible with [JetBrains HTTP Client](https://www.jetbrains.com/help/idea/http-client-in-product-code-editor.html) and [VS Code REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client).

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
