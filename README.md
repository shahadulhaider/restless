# restless

A terminal-native HTTP client that uses `.http` files — the same format supported by JetBrains IDEs, VS Code REST Client, and IntelliJ IDEA. Store your API requests in plain text files, version them with Git, and run them from a full-featured TUI or headless CLI.

## Installation

**Go install**
```bash
go install github.com/shahadulhaider/restless/cmd/restless@latest
```

**Homebrew**
```bash
brew tap shahadulhaider/tap
brew install restless
```

**Binary download**

Download the latest release for your platform from the [releases page](https://github.com/shahadulhaider/restless/releases).

## Quick Start

Create a `.http` file:

```http
# @name getUser
GET https://api.example.com/users/1
Accept: application/json

###

# @name createPost
POST https://api.example.com/posts
Content-Type: application/json
Authorization: Bearer {{token}}

{
  "title": "Hello",
  "body": "World"
}
```

Create an environment file `http-client.env.json`:

```json
{
  "$shared": {
    "baseUrl": "https://api.example.com"
  },
  "dev": {
    "token": "dev-secret-token"
  },
  "prod": {
    "token": "prod-secret-token"
  }
}
```

Launch the TUI:

```bash
restless .
```

Run headless:

```bash
restless run requests.http --env dev
```

## Features

- **`.http` file format** — JetBrains-compatible request syntax
- **Interactive TUI** — browse, search, and send requests from the terminal
- **Environment management** — switch between dev/staging/prod with `e`
- **Request chaining** — use `{{requestName.response.body.path}}` to pass data between requests
- **Response history** — every response is saved; browse and diff with `h`
- **Cookie jar** — cookies persist per environment automatically
- **Postman import** — convert existing Postman collections with `restless import postman`
- **File bodies** — reference external files with `< ./body.json`
- **Git-friendly** — plain text files, no proprietary format

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Navigate request list |
| `Enter` | Select request |
| `r` | Send request |
| `e` | Switch environment |
| `/` | Search requests |
| `h` | Toggle response history |
| `d` | Diff two history entries |
| `tab` | Switch between panes |
| `q` | Quit |

## CLI Reference

```
restless [directory]                          Launch TUI for directory
restless run <file.http> [--env <name>]       Run all requests headlessly
restless import postman <file> [--output dir] Import Postman collection
restless version                              Print version
```

## `.http` Format

```http
# @name requestName
METHOD https://url
Header-Name: {{variable}}

request body (optional)

###

# next request
```

Full spec: [JetBrains HTTP Client documentation](https://www.jetbrains.com/help/idea/http-client-in-product-code-editor.html)

## License

MIT
