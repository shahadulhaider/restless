# Getting Started

## Installation

### Homebrew (macOS/Linux)
```bash
brew tap shahadulhaider/tap
brew install restless
```

### Go
```bash
go install github.com/shahadulhaider/restless/cmd/restless@latest
```

### Binary Download
Download from [Releases](https://github.com/shahadulhaider/restless/releases).

## Your First Collection

### 1. Create a .http file

```http
@baseUrl = https://httpbin.org

# @name getRequest
GET {{baseUrl}}/get
Accept: application/json

###

# @name postRequest
POST {{baseUrl}}/post
Content-Type: application/json

{
  "message": "Hello from restless!",
  "timestamp": "{{$isoTimestamp}}"
}

# @assert status == 200
# @assert body.$.json.message == "Hello from restless!"
```

### 2. Launch the TUI

```bash
restless .
```

### 3. Navigate and Send

- Use `j`/`k` to navigate the request list
- Press `Enter` to expand files and select requests
- Press `Enter` again (or `Ctrl+R`) to send the request
- The response appears immediately with body expanded

### 4. Explore the Response

- Press `1`/`2`/`3`/`4` to toggle sections (Body, Headers, Timing, Assertions)
- Press `Space` to fold/unfold the section under your cursor
- Press `p` to toggle pretty-print vs raw
- Press `f` to search in the response body

### 5. Copy and Generate Code

- Press `yb` to copy the response body
- Press `yc` to copy the request as a curl command
- Press `ygp` to generate Python code, `ygj` for JavaScript, etc.

## Environments

Create `http-client.env.json` in your project root:

```json
{
  "$shared": {
    "baseUrl": "https://api.example.com"
  },
  "dev": {
    "baseUrl": "http://localhost:8000",
    "token": "dev-secret"
  },
  "prod": {
    "token": "prod-secret"
  }
}
```

Press `Ctrl+E` in the TUI to switch environments. Variables like `{{baseUrl}}` and `{{token}}` are expanded automatically.

## Running in CI/CD

```bash
restless run api.http --env production --fail-fast
```

This executes all requests, runs assertions, and exits with code 1 if anything fails.
