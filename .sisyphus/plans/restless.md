# restless — TUI HTTP Client

## TL;DR

> **Quick Summary**: Build a full-featured TUI HTTP client in Go + Bubbletea v2 that uses the open `.http` file format (JetBrains spec), syncs collections via Git, and replaces Postman/Insomnia for terminal-native developers.
> 
> **Deliverables**:
> - `.http` file parser (JetBrains EBNF spec compliant)
> - Interactive TUI with split-pane layout (collection browser + request/response viewer)
> - Environment management (`http-client.env.json` with `$shared` support)
> - Response history with diff
> - Request chaining via `{{name.response.body.path}}` syntax
> - Postman collection import (v2.1 → .http)
> - Cookie jar per environment
> - Single binary, zero dependencies
> 
> **Estimated Effort**: Large (4 weeks)
> **Parallel Execution**: YES — 4 waves
> **Critical Path**: Types → Parser → HTTP Engine → TUI Shell → Browser Pane → Detail Pane → Wire Together → Features

---

## Context

### Original Request
User wants to build a TUI-based Postman replacement using open standards — `.http` files for requests, Git for sync, plain text everywhere. Named "restless" (REST + restless).

### Interview Summary
**Key Discussions**:
- **Scope**: Full v1 — core TUI + response history + request chaining + Postman import + cookies
- **Auth**: Static tokens only (Bearer, API keys, Basic auth via environment variables)
- **Protocol**: HTTP/1.1 and HTTP/2 only. No WebSocket, gRPC, SSE.
- **Tests**: After implementation, Go built-in testing
- **Format**: `.http` files (JetBrains spec) — NOT Bruno's `.bru`, NOT Hurl's format

**Research Findings**:
- JetBrains published formal EBNF grammar: `https://github.com/JetBrains/http-request-in-editor-spec`
- Bubbletea v2 (Feb 2026): `View()` returns `tea.View`, import paths `charm.land/*`, `KeyMsg` → `KeyPressMsg`
- No Go `.http` parser exists — greenfield opportunity
- Request chaining is declarative in the spec: `{{name.response.body.path}}`
- `$shared` environment key applies to ALL environments
- `http-client.private.env.json` overrides public env vars
- Key Go libraries: `theory/jsonpath` (RFC 9535), `rbretecher/go-postman-collection`, `tidwall/pretty`
- Posting (Python, 11.6k stars) is closest competitor but slow, Python dependency

### Metis Review
**Identified Gaps** (addressed):
- Metadata tags (`@no-redirect`, `@timeout`) added to v1 scope — trivial to implement, users expect them
- `$shared` environment key — must merge correctly with per-env vars
- `http-client.private.env.json` — private vars override public, must be `.gitignore`d
- File body references (`< ./path`) — in core spec, essential for v1
- Body end detection — body ends at `###` or EOF, empty lines within body are content
- URL encoding — must not double-encode already-encoded `%20`
- ANSI escape codes in response bodies — must be escaped for display

---

## Work Objectives

### Core Objective
Build a production-quality TUI HTTP client that reads `.http` files, sends requests, and displays responses in an interactive terminal interface with environment management, response history, and request chaining.

### Concrete Deliverables
- `restless` binary (Go, single binary, `go install` compatible)
- `.http` file parser following JetBrains spec
- Interactive TUI with collection browser, request editor, response viewer
- Environment switcher with `http-client.env.json` support
- Response history in `.restless/history/`
- Request chaining via `{{name.response.body.path}}`
- `restless import postman <file>` command
- Automatic cookie jar per environment

### Definition of Done
- [ ] `restless` opens TUI, browses `.http` files in current directory
- [ ] Can send any HTTP/1.1 request and display response with timing
- [ ] Environment switching works with variable substitution
- [ ] Response history stores and diffs responses
- [ ] Request chaining resolves `{{name.response.body.path}}` across requests
- [ ] `restless import postman collection.json` produces valid `.http` files
- [ ] All tests pass (`go test ./...`)

### Must Have
- JetBrains `.http` spec compliance (core subset: methods, headers, body, `###` separators, `{{variables}}`, `# @name`, `< ./file`, `@no-redirect`, `@timeout`)
- Split-pane TUI (collection browser left, request/response right)
- Fuzzy search across all requests in collection
- Environment file support with `$shared` key and private env file merge
- Response body syntax highlighting (JSON at minimum)
- Request timing display (DNS, connect, TLS, TTFB, total)
- Single binary distribution

### Must NOT Have (Guardrails)
- **NO response handler scripts** (`> {% %}`) — defer to v2
- **NO VS Code-specific extensions** (`$processEnv`, `$dotenv`, `$aadToken`, `$oidcAccessToken`)
- **NO `<@` variable-substitution file references** — only plain `< ./path`
- **NO OAuth2 flows, WebSocket, gRPC, SSE**
- **NO plugin system or extensibility hooks**
- **NO GUI or web interface** — TUI only
- **NO cloud sync** — Git is the sync mechanism
- **NO over-abstraction** — no interfaces "for future providers", no config framework. Direct code.
- **NO Electron, no CGo** — pure Go, single binary

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: NO (new project)
- **Automated tests**: YES — tests after implementation
- **Framework**: Go built-in `testing` package + `testify` for assertions
- **Setup**: `go test ./...` from project root

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Parser**: Bash — run Go test with table-driven cases
- **HTTP Engine**: Bash — start local test server, send requests, assert responses
- **TUI**: interactive_bash (tmux) — launch restless, send keystrokes, capture output
- **Import**: Bash — run import command, diff output files

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — all independent, start immediately):
├── Task 1: Project scaffolding (go mod, directory structure, Makefile) [quick]
├── Task 2: Core types (Request, Response, Environment, Collection models) [quick]
├── Task 3: .http file lexer (tokenize .http files per JetBrains spec) [deep]
└── Task 4: Environment parser (http-client.env.json + private + $shared merge) [quick]

Wave 2 (Parser + Engine — after Wave 1):
├── Task 5: .http file parser (tokens → Request structs, handle body/headers/metadata) [deep]
├── Task 6: Variable resolver ({{var}} substitution from environment + chaining context) [unspecified-high]
├── Task 7: HTTP execution engine (send request, capture response with timing) [unspecified-high]
└── Task 8: File body loader (< ./path resolution, relative path handling) [quick]

Wave 3 (TUI + Features — after Wave 2):
├── Task 9: TUI shell (Bubbletea v2 app skeleton, alt screen, quit, window resize) [unspecified-high]
├── Task 10: Collection browser pane (directory tree, .http files, request list) [deep]
├── Task 11: Request/Response detail pane (headers, body with syntax highlight, timing) [visual-engineering]
├── Task 12: Fuzzy search overlay (search across all requests by name/URL/method) [quick]
├── Task 13: Environment switcher overlay (list envs, select, show active) [quick]
├── Task 14: Response history (store responses, list history, diff between runs) [unspecified-high]
├── Task 15: Request chaining engine (resolve {{name.response.body.path}} from history) [deep]
├── Task 16: Cookie jar (per-environment, automatic, respect @no-cookie-jar) [quick]
└── Task 17: Postman import command (collection.json v2.1 → .http files + env.json) [unspecified-high]

Wave 4 (Integration + Polish — after Wave 3):
├── Task 18: Wire everything together (browser → parser → engine → detail, keyboard nav) [deep]
├── Task 19: CLI entry point (cobra: `restless` TUI, `restless import postman <file>`, `restless run <file>`) [quick]
├── Task 20: Integration tests (end-to-end: parse → send → display → history) [unspecified-high]
└── Task 21: Build & release setup (Makefile, goreleaser config, README) [quick]

Wave FINAL (After ALL tasks — 4 parallel reviews, then user okay):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay

Critical Path: T1 → T2 → T3 → T5 → T6 → T7 → T9 → T10 → T11 → T18 → T19 → F1-F4
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 4 (Waves 1, 3)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|-----------|--------|
| T1 | — | T2-T21 |
| T2 | T1 | T3, T4, T5, T6, T7, T8, T14, T15, T16, T17 |
| T3 | T2 | T5 |
| T4 | T2 | T6 |
| T5 | T3 | T6, T8, T18 |
| T6 | T4, T5 | T7, T15, T18 |
| T7 | T2, T6 | T14, T15, T16, T18 |
| T8 | T5 | T18 |
| T9 | T1 | T10, T11, T12, T13, T18 |
| T10 | T5, T9 | T18 |
| T11 | T7, T9 | T18 |
| T12 | T9, T10 | T18 |
| T13 | T4, T9 | T18 |
| T14 | T7 | T15, T18 |
| T15 | T6, T7, T14 | T18 |
| T16 | T7 | T18 |
| T17 | T2, T5, T4 | T20 |
| T18 | T10, T11, T12, T13, T14, T15, T16 | T19, T20 |
| T19 | T18 | T20 |
| T20 | T18, T19, T17 | F1-F4 |
| T21 | T19 | F1-F4 |

### Agent Dispatch Summary

- **Wave 1**: 4 tasks — T1 `quick`, T2 `quick`, T3 `deep`, T4 `quick`
- **Wave 2**: 4 tasks — T5 `deep`, T6 `unspecified-high`, T7 `unspecified-high`, T8 `quick`
- **Wave 3**: 9 tasks — T9 `unspecified-high`, T10 `deep`, T11 `visual-engineering`, T12 `quick`, T13 `quick`, T14 `unspecified-high`, T15 `deep`, T16 `quick`, T17 `unspecified-high`
- **Wave 4**: 4 tasks — T18 `deep`, T19 `quick`, T20 `unspecified-high`, T21 `quick`
- **FINAL**: 4 tasks — F1 `oracle`, F2 `unspecified-high`, F3 `unspecified-high`, F4 `deep`

---

## TODOs

- [x] 1. Project Scaffolding

  **What to do**:
  - Initialize Go module: `go mod init github.com/shahadulhaider/restless`
  - Create directory structure:
    ```
    restless/
    ├── cmd/restless/main.go      # CLI entry point (minimal, just calls root cmd)
    ├── internal/
    │   ├── model/                # Core types
    │   ├── parser/               # .http lexer, parser, env parser
    │   ├── engine/               # HTTP execution, cookie jar
    │   ├── chain/                # Request chaining resolution
    │   ├── history/              # Response history storage
    │   ├── importer/             # Postman import
    │   └── tui/                  # All Bubbletea TUI code
    │       ├── app.go            # Root model
    │       ├── browser.go        # Collection browser pane
    │       ├── detail.go         # Request/response detail pane
    │       ├── search.go         # Fuzzy search overlay
    │       └── envswitch.go      # Environment switcher overlay
    ├── Makefile
    ├── .gitignore
    └── README.md
    ```
  - Add Makefile with targets: `build`, `test`, `vet`, `lint`, `run`
  - Add `.gitignore` (Go defaults + `.restless/` local data dir)
  - Install core dependencies: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`
  - Verify `go build ./...` succeeds with empty main.go

  **Must NOT do**:
  - Don't add cobra yet (Task 19)
  - Don't add any application logic — just empty packages with package declarations
  - Don't add CI/CD config yet (Task 21)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4)
  - **Blocks**: All subsequent tasks
  - **Blocked By**: None

  **References**:
  **Pattern References**:
  - User's existing Go projects at `/Users/msh/code/pp/ports/` and `/Users/msh/code/pp/envdiff/` — check their `go.mod`, directory structure, Makefile patterns
  
  **External References**:
  - Bubbletea v2 import paths: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`
  - JetBrains .http spec: `https://github.com/JetBrains/http-request-in-editor-spec`

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] Directory structure matches spec above
  - [ ] `go mod tidy` produces no changes (all deps resolved)

  **QA Scenarios**:
  ```
  Scenario: Project builds cleanly
    Tool: Bash
    Steps:
      1. cd /Users/msh/code/pp/restless && go build ./...
      2. go mod tidy && git diff --exit-code go.mod go.sum
      3. ls cmd/restless/main.go internal/model internal/parser internal/engine internal/tui
    Expected Result: All commands exit 0
    Evidence: .sisyphus/evidence/task-1-build.txt
  ```

  **Commit**: YES
  - Message: `chore(restless): init project scaffolding`
  - Pre-commit: `go build ./...`

- [x] 2. Core Types

  **What to do**:
  - Create `internal/model/request.go`:
    ```go
    type Request struct {
        Name        string            // from # @name tag
        Method      string            // GET, POST, etc.
        URL         string            // raw URL with {{variables}}
        HTTPVersion string            // HTTP/1.1, HTTP/2 (optional)
        Headers     []Header          // ordered key-value pairs
        Body        string            // raw body text
        BodyFile    string            // from < ./path reference
        Metadata    RequestMetadata   // @no-redirect, @timeout, etc.
        SourceFile  string            // which .http file this came from
        SourceLine  int               // line number in source file
    }
    type Header struct { Key, Value string }
    type RequestMetadata struct {
        NoRedirect      bool
        NoCookieJar     bool
        Timeout         time.Duration
        ConnTimeout     time.Duration
    }
    ```
  - Create `internal/model/response.go`:
    ```go
    type Response struct {
        StatusCode    int
        Status        string
        Headers       []Header
        Body          []byte
        ContentType   string
        Timing        ResponseTiming
        Request       *Request         // back-reference
        Timestamp     time.Time
    }
    type ResponseTiming struct {
        DNS       time.Duration
        Connect   time.Duration
        TLS       time.Duration
        TTFB      time.Duration
        Total     time.Duration
        BodyRead  time.Duration
    }
    ```
  - Create `internal/model/environment.go`:
    ```go
    type Environment struct {
        Name      string
        Variables map[string]string
    }
    type EnvironmentFile struct {
        Shared       map[string]string      // $shared key
        Environments map[string]Environment
    }
    ```
  - Create `internal/model/collection.go`:
    ```go
    type Collection struct {
        RootDir  string
        Files    []HTTPFile
    }
    type HTTPFile struct {
        Path     string
        Requests []Request
    }
    ```

  **Must NOT do**:
  - No interfaces — concrete structs only
  - No JSON tags yet (add when needed by history/import)
  - No methods beyond basic constructors if needed

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4)
  - **Blocks**: T3, T4, T5, T6, T7, T8, T14, T15, T16, T17
  - **Blocked By**: T1 (needs go.mod)

  **References**:
  **External References**:
  - JetBrains .http spec metadata tags: `https://github.com/JetBrains/http-request-in-editor-spec/blob/master/spec.md` — defines `@name`, `@no-redirect`, `@timeout`, `@connection-timeout`
  - Go `net/http` response fields — align Response struct fields with what `http.Response` provides

  **Acceptance Criteria**:
  - [ ] `go build ./internal/model/...` succeeds
  - [ ] All struct fields have appropriate Go types (no `interface{}`)
  - [ ] Types cover: Request (with metadata), Response (with timing), Environment (with $shared), Collection

  **QA Scenarios**:
  ```
  Scenario: Types compile and are usable
    Tool: Bash
    Steps:
      1. cd /Users/msh/code/pp/restless && go build ./internal/model/...
      2. go vet ./internal/model/...
    Expected Result: Exit 0, no warnings
    Evidence: .sisyphus/evidence/task-2-types.txt
  ```

  **Commit**: YES
  - Message: `feat(model): add core types for request, response, environment`
  - Pre-commit: `go build ./internal/model/...`

- [x] 3. .http File Lexer

  **What to do**:
  - Create `internal/parser/lexer.go` — tokenize `.http` files into a token stream
  - Token types needed:
    ```
    TokenRequestSeparator  // ###
    TokenMethod            // GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
    TokenURL               // everything after method until newline or HTTP/
    TokenHTTPVersion       // HTTP/1.1, HTTP/2
    TokenHeaderKey         // key before :
    TokenHeaderValue       // value after : (trimmed)
    TokenBody              // everything between headers and next ### or EOF
    TokenComment           // # comment line (not metadata)
    TokenMetadata          // # @name, # @no-redirect, # @timeout
    TokenVariable          // {{variable_name}} (within URL, headers, body)
    TokenFileRef           // < ./path/to/file
    TokenNewline           // significant newline (separates headers from body)
    TokenEOF
    ```
  - Lexer must handle:
    - `###` as request separator (with optional comment text after)
    - `# @name requestName` — metadata extraction
    - `# @no-redirect`, `# @no-cookie-jar`, `# @timeout N`, `# @connection-timeout N m`
    - Empty line between headers and body (blank line = body starts)
    - Body continues until `###` or EOF (empty lines within body are body content)
    - `< ./relative/path` file body references
    - `{{variable}}` detection (but NOT resolution — that's Task 6)
    - Lines starting with `#` or `//` are comments (skip unless metadata)
  - Create `internal/parser/lexer_test.go` — table-driven tests:
    - Simple GET request
    - POST with JSON body
    - Multiple requests separated by `###`
    - Request with metadata tags
    - Request with file body reference
    - Request with `{{variables}}` in URL and headers
    - Edge case: empty body, body with blank lines, request at EOF without trailing `###`

  **Must NOT do**:
  - Don't resolve variables (Task 6)
  - Don't parse into Request structs (Task 5)
  - Don't handle response handlers (`> {% %}`) — guardrail

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
    - Reason: Parser/lexer requires careful grammar handling and edge case coverage

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4)
  - **Blocks**: T5
  - **Blocked By**: T2 (needs model types for token metadata)

  **References**:
  **External References**:
  - JetBrains EBNF grammar: `https://github.com/JetBrains/http-request-in-editor-spec/blob/master/spec.md` — the authoritative grammar to implement against
  - RFC 7230 Section 3.1.1 — HTTP request line format: `method SP request-target SP HTTP-version CRLF`
  - Go lexer patterns: standard `text/scanner` or hand-written rune-by-rune (prefer hand-written for control)

  **Acceptance Criteria**:
  - [ ] Lexer tokenizes all example .http files from JetBrains spec without errors
  - [ ] All token types listed above are implemented
  - [ ] Tests cover: simple GET, POST with body, multi-request file, metadata, file refs, variables, edge cases
  - [ ] `go test ./internal/parser/... -run TestLexer` passes

  **QA Scenarios**:
  ```
  Scenario: Lexer handles multi-request .http file
    Tool: Bash
    Steps:
      1. Create test file with 3 requests (GET, POST with body, DELETE with metadata)
      2. Run go test ./internal/parser/... -v -run TestLexer
      3. Verify token counts and types match expectations
    Expected Result: All lexer tests pass, token stream correct
    Evidence: .sisyphus/evidence/task-3-lexer.txt

  Scenario: Lexer handles edge cases
    Tool: Bash
    Steps:
      1. Run go test ./internal/parser/... -v -run TestLexerEdge
      2. Cases: empty file, file without ###, body with blank lines, UTF-8 in body
    Expected Result: No panics, graceful handling
    Evidence: .sisyphus/evidence/task-3-lexer-edge.txt
  ```

  **Commit**: YES (groups with T4)
  - Message: `feat(parser): add .http lexer and environment parser`
  - Pre-commit: `go test ./internal/parser/...`

- [x] 4. Environment Parser

  **What to do**:
  - Create `internal/parser/envparser.go` — parse environment files
  - Parse `http-client.env.json`:
    ```json
    {
      "$shared": { "base_url": "https://api.example.com" },
      "dev": { "token": "dev-123", "base_url": "http://localhost:3000" },
      "prod": { "token": "prod-abc" }
    }
    ```
  - Parse `http-client.private.env.json` (same format, overrides public)
  - Merge logic (priority order, highest wins):
    1. Private env-specific vars
    2. Public env-specific vars
    3. Private `$shared` vars
    4. Public `$shared` vars
  - Support built-in dynamic variables:
    - `{{$uuid}}` → UUID v4
    - `{{$timestamp}}` → Unix timestamp
    - `{{$randomInt}}` → random integer
    - `{{$datetime "format"}}` → formatted date (Go time format)
  - Function: `LoadEnvironments(dir string) (*EnvironmentFile, error)` — finds and parses env files in directory
  - Function: `ResolveEnvironment(envFile *EnvironmentFile, envName string) (map[string]string, error)` — returns merged vars for a specific environment
  - Create `internal/parser/envparser_test.go`:
    - Basic env parsing
    - `$shared` merge with env-specific override
    - Private env override of public
    - Missing env file (not an error — just empty)
    - Dynamic variable resolution (`$uuid`, `$timestamp`)

  **Must NOT do**:
  - Don't support VS Code-specific vars (`$processEnv`, `$dotenv`, `$aadToken`)
  - Don't support nested objects in env values — flat string map only
  - Don't watch for file changes (static load)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3)
  - **Blocks**: T6, T13
  - **Blocked By**: T2 (needs Environment types)

  **References**:
  **External References**:
  - IntelliJ HTTP Client environment docs: `https://www.jetbrains.com/help/idea/exploring-http-syntax.html#environment-variables`
  - `$shared` key behavior: applies to all environments, overridden by env-specific values of same name
  - `http-client.private.env.json`: same structure, values override public file, should be in `.gitignore`

  **Acceptance Criteria**:
  - [ ] Parses `http-client.env.json` with `$shared` and multiple environments
  - [ ] Merges private env file correctly (private overrides public)
  - [ ] Dynamic variables (`$uuid`, `$timestamp`, `$randomInt`) resolve to correct types
  - [ ] Missing env files return empty environment (no error)
  - [ ] `go test ./internal/parser/... -run TestEnv` passes

  **QA Scenarios**:
  ```
  Scenario: Environment merge priority
    Tool: Bash
    Steps:
      1. Create http-client.env.json with $shared base_url and dev-specific base_url
      2. Create http-client.private.env.json with dev-specific token
      3. Run test that resolves "dev" env, assert: base_url from dev (not $shared), token from private
    Expected Result: Merge order correct — private env > public env > private shared > public shared
    Evidence: .sisyphus/evidence/task-4-envparser.txt

  Scenario: Missing env files don't error
    Tool: Bash
    Steps:
      1. Run LoadEnvironments on directory with no env files
      2. Assert returns empty EnvironmentFile, nil error
    Expected Result: Graceful handling, no panic
    Evidence: .sisyphus/evidence/task-4-envparser-missing.txt
  ```

  **Commit**: YES (groups with T3)
  - Message: `feat(parser): add .http lexer and environment parser`
  - Pre-commit: `go test ./internal/parser/...`

- [ ] 5. .http File Parser

  **What to do**:
  - Create `internal/parser/parser.go` — consume token stream from lexer, produce `[]model.Request`
  - Main function: `ParseFile(path string) ([]model.Request, error)`
  - Also: `ParseBytes(content []byte, sourcePath string) ([]model.Request, error)` for testing
  - Parser responsibilities:
    - Consume lexer tokens sequentially
    - Build `model.Request` for each request block (between `###` separators)
    - Assign method, URL, HTTP version from request line
    - Collect headers as ordered `[]model.Header`
    - Detect body start (first blank line after headers)
    - Collect body text (preserve formatting, whitespace, blank lines within body)
    - Extract metadata from `# @name`, `# @no-redirect`, `# @timeout N`, `# @connection-timeout N m`
    - Detect `< ./path` file body references → set `Request.BodyFile`
    - Track source file and line number for each request
    - Handle edge cases: request without body, request without headers, first request without leading `###`
  - Error handling:
    - Return parse errors with file:line context
    - Skip malformed requests with warning (don't fail entire file)
    - Validate HTTP method is known (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE, CONNECT)
  - Create `internal/parser/parser_test.go` — table-driven tests:
    - Parse single GET request
    - Parse POST with JSON body and Content-Type header
    - Parse multi-request file (3+ requests)
    - Parse request with all metadata tags
    - Parse request with `< ./body.json` file reference
    - Parse request with `{{variables}}` preserved in URL, headers, body
    - Error case: invalid method
    - Edge case: no trailing newline, Windows line endings (CRLF)

  **Must NOT do**:
  - Don't resolve `{{variables}}` — leave as literal strings (Task 6 handles resolution)
  - Don't read file body contents (Task 8) — just record the path
  - Don't execute requests — just parse structure

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
    - Reason: Parser needs careful spec compliance and extensive edge case handling

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (sequential dependency on T3)
  - **Blocks**: T6, T8, T17, T18
  - **Blocked By**: T3 (needs lexer token stream)

  **References**:
  **Pattern References**:
  - `internal/parser/lexer.go` (Task 3) — token types and lexer API to consume

  **External References**:
  - JetBrains EBNF grammar: `https://github.com/JetBrains/http-request-in-editor-spec/blob/master/spec.md`
  - HTTP methods: RFC 9110 Section 9 — standard method definitions

  **Acceptance Criteria**:
  - [ ] Parses all JetBrains spec examples into correct `model.Request` structs
  - [ ] Request.Name populated from `# @name` metadata
  - [ ] Request.Metadata populated from `@no-redirect`, `@timeout`, `@connection-timeout`
  - [ ] Request.BodyFile populated from `< ./path`
  - [ ] `{{variables}}` preserved as literal strings in URL, headers, body
  - [ ] Parse errors include file:line context
  - [ ] `go test ./internal/parser/... -run TestParser` passes

  **QA Scenarios**:
  ```
  Scenario: Parse complex multi-request .http file
    Tool: Bash
    Steps:
      1. Create .http file with: GET with headers, POST with JSON body, DELETE with @name and @no-redirect
      2. Run go test ./internal/parser/... -v -run TestParser
      3. Assert: 3 requests parsed, correct methods, URLs, headers, body, metadata
    Expected Result: All parser tests pass
    Evidence: .sisyphus/evidence/task-5-parser.txt

  Scenario: Graceful error handling
    Tool: Bash
    Steps:
      1. Create .http file with one valid request and one malformed (invalid method "FOOBAR")
      2. Parse file, assert: valid request returned, error/warning for malformed
    Expected Result: Partial parse succeeds, malformed request skipped with error
    Evidence: .sisyphus/evidence/task-5-parser-error.txt
  ```

  **Commit**: YES (groups with T6)
  - Message: `feat(parser): add .http parser with variable resolution`
  - Pre-commit: `go test ./internal/parser/...`

- [ ] 6. Variable Resolver

  **What to do**:
  - Create `internal/parser/resolver.go` — resolve `{{variable}}` placeholders in requests
  - Main function: `ResolveRequest(req *model.Request, vars map[string]string, chainCtx *ChainContext) (*model.Request, error)`
  - Returns a NEW request with all `{{variables}}` replaced (don't mutate original)
  - Resolution sources (priority order):
    1. Chain context variables (`{{requestName.response.body.path}}`, `{{requestName.response.headers.Name}}`)
    2. Environment variables (from `ResolveEnvironment()`)
    3. Dynamic variables (`{{$uuid}}`, `{{$timestamp}}`, `{{$randomInt}}`)
  - Resolve in: URL, header values, body text
  - Do NOT resolve in: header keys, method, HTTP version
  - `ChainContext` type:
    ```go
    type ChainContext struct {
        Responses map[string]*model.Response  // keyed by request @name
    }
    ```
  - For chain variables like `{{login.response.body.token}}`:
    - Parse the variable reference: `requestName.response.body.jsonpath` or `requestName.response.headers.headerName`
    - Body paths use dot notation for JSON access (use `gjson` or `theory/jsonpath`)
    - Header access: case-insensitive header name lookup
  - Handle unresolved variables:
    - If a `{{variable}}` has no value in any source, leave it as-is and emit a warning (don't error)
    - This allows partial resolution (e.g., some vars from env, others from chain context added later)
  - Create `internal/parser/resolver_test.go`:
    - Simple variable substitution in URL
    - Variable in header value
    - Variable in body
    - Chain context resolution (`{{login.response.body.token}}`)
    - Chain context header resolution (`{{login.response.headers.X-Request-Id}}`)
    - Unresolved variable left as-is
    - Dynamic variables resolve to expected formats
    - Multiple variables in same string

  **Must NOT do**:
  - Don't resolve VS Code-specific variables (`$processEnv`, etc.)
  - Don't support nested variable references (`{{{{var}}}}`)
  - Don't support expressions or arithmetic in variables

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T7, T8 once T5 is done)
  - **Parallel Group**: Wave 2
  - **Blocks**: T7, T15, T18
  - **Blocked By**: T4 (env parser), T5 (request parser)

  **References**:
  **Pattern References**:
  - `internal/parser/envparser.go` (Task 4) — `ResolveEnvironment()` provides the variable map
  - `internal/model/request.go` (Task 2) — Request struct to resolve against

  **External References**:
  - JetBrains spec variable syntax: `{{variableName}}`, `{{requestName.response.body.jsonpath}}`
  - `tidwall/gjson` — fast JSON path access for response body extraction (simpler than full JSONPath)
  - `theory/jsonpath` — RFC 9535 compliant alternative (more correct, heavier)

  **Acceptance Criteria**:
  - [ ] `{{var}}` in URL, headers, body resolved from environment
  - [ ] `{{name.response.body.path}}` resolved from chain context using JSON access
  - [ ] `{{name.response.headers.Name}}` resolved from chain context headers
  - [ ] Unresolved variables left as-is with warning
  - [ ] Dynamic variables (`$uuid`, `$timestamp`) generate correct formats
  - [ ] `go test ./internal/parser/... -run TestResolver` passes

  **QA Scenarios**:
  ```
  Scenario: Full variable resolution pipeline
    Tool: Bash
    Steps:
      1. Create Request with {{base_url}}/users/{{user_id}} URL, {{token}} in Authorization header
      2. Create env vars: base_url=http://localhost, user_id=42, token=abc123
      3. Resolve and assert URL = "http://localhost/users/42", Auth header = "abc123"
    Expected Result: All variables resolved correctly
    Evidence: .sisyphus/evidence/task-6-resolver.txt

  Scenario: Chain context resolution
    Tool: Bash
    Steps:
      1. Create ChainContext with "login" response containing JSON body {"token": "jwt-xyz"}
      2. Create Request with Authorization: Bearer {{login.response.body.token}}
      3. Resolve and assert header = "Bearer jwt-xyz"
    Expected Result: Chain variable correctly extracted from JSON response
    Evidence: .sisyphus/evidence/task-6-resolver-chain.txt
  ```

  **Commit**: YES (groups with T5)
  - Message: `feat(parser): add .http parser with variable resolution`
  - Pre-commit: `go test ./internal/parser/...`

- [ ] 7. HTTP Execution Engine

  **What to do**:
  - Create `internal/engine/engine.go` — execute HTTP requests and capture responses
  - Main function: `Execute(req *model.Request) (*model.Response, error)`
  - Use `net/http` with custom transport for timing:
    - Hook into `httptrace.ClientTrace` for detailed timing:
      - `DNSStart`/`DNSDone` → DNS duration
      - `ConnectStart`/`ConnectDone` → Connect duration
      - `TLSHandshakeStart`/`TLSHandshakeDone` → TLS duration
      - `GotFirstResponseByte` → TTFB
    - Total time: from request start to body fully read
  - Request construction from `model.Request`:
    - Build `http.Request` from method, URL, headers, body
    - Set `Content-Type` from headers (or detect from body if JSON)
    - Handle `@no-redirect`: set `CheckRedirect` to return `http.ErrUseLastResponse`
    - Handle `@timeout`: set `http.Client.Timeout`
    - Handle `@connection-timeout`: set `net.Dialer.Timeout`
  - Response capture into `model.Response`:
    - Read full body into `[]byte`
    - Detect content type from `Content-Type` header
    - Populate all timing fields
    - Set timestamp to `time.Now()`
  - HTTP/2 support: enable by default via `http.Transport.ForceAttemptHTTP2 = true`
  - Create `internal/engine/engine_test.go`:
    - Use `net/http/httptest.NewServer` for testing
    - Test GET request with JSON response
    - Test POST with body
    - Test redirect following (default)
    - Test `@no-redirect` stops redirect
    - Test `@timeout` causes timeout error
    - Test timing fields are populated (non-zero)
    - Test response body capture

  **Must NOT do**:
  - Don't handle cookies here (Task 16 wraps engine with cookie jar)
  - Don't handle variable resolution (already done in Task 6)
  - Don't handle file body loading (Task 8)
  - Don't add retry logic

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T6, T8)
  - **Parallel Group**: Wave 2
  - **Blocks**: T11, T14, T15, T16, T18
  - **Blocked By**: T2 (needs model types), T6 (needs resolved requests)

  **References**:
  **External References**:
  - Go `net/http/httptrace` package: `https://pkg.go.dev/net/http/httptrace` — ClientTrace hooks for timing
  - Go `net/http/httptest` package — for test HTTP servers
  - HTTP/2 in Go: `ForceAttemptHTTP2` on `http.Transport`

  **Acceptance Criteria**:
  - [ ] Sends GET/POST/PUT/DELETE/PATCH requests via `net/http`
  - [ ] Captures full response: status, headers, body, timing
  - [ ] Timing fields (DNS, Connect, TLS, TTFB, Total) populated correctly
  - [ ] `@no-redirect` prevents following redirects
  - [ ] `@timeout` causes request to fail after specified duration
  - [ ] `go test ./internal/engine/... -run TestEngine` passes

  **QA Scenarios**:
  ```
  Scenario: Execute GET and capture timing
    Tool: Bash
    Steps:
      1. Start httptest server returning JSON {"status": "ok"}
      2. Execute GET request against test server
      3. Assert: StatusCode=200, Body contains "ok", Timing.Total > 0
    Expected Result: Response captured with all fields
    Evidence: .sisyphus/evidence/task-7-engine.txt

  Scenario: @no-redirect prevents following
    Tool: Bash
    Steps:
      1. Start httptest server that returns 302 redirect
      2. Execute request with Metadata.NoRedirect=true
      3. Assert: StatusCode=302 (not the redirect target)
    Expected Result: Redirect not followed
    Evidence: .sisyphus/evidence/task-7-engine-noredirect.txt
  ```

  **Commit**: YES (groups with T8)
  - Message: `feat(engine): add HTTP execution engine with file body support`
  - Pre-commit: `go test ./internal/engine/...`

- [ ] 8. File Body Loader

  **What to do**:
  - Create `internal/parser/fileloader.go` — resolve `< ./path` file body references
  - Main function: `LoadFileBody(req *model.Request) (*model.Request, error)`
  - If `req.BodyFile` is set (from `< ./path` in .http file):
    - Resolve path relative to the `.http` file's directory (not CWD)
    - Read file contents into `req.Body`
    - Clear `req.BodyFile` after loading
    - If file doesn't exist, return error with context (which .http file, which line, which path)
  - If `req.BodyFile` is empty, return request unchanged
  - Path security: don't allow `../../../etc/passwd` style traversal outside collection root
    - Resolve absolute path, check it's within collection root directory
  - Create `internal/parser/fileloader_test.go`:
    - Load JSON file body
    - Load XML file body
    - Relative path resolution (relative to .http file, not CWD)
    - Missing file error with context
    - Path traversal blocked

  **Must NOT do**:
  - Don't process variables inside file body (`<@` syntax — excluded by guardrail)
  - Don't cache file contents
  - Don't support HTTP URLs as file references

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T6, T7)
  - **Parallel Group**: Wave 2
  - **Blocks**: T18
  - **Blocked By**: T5 (needs parser to populate BodyFile)

  **References**:
  **Pattern References**:
  - `internal/model/request.go` (Task 2) — `BodyFile` field on Request struct

  **External References**:
  - JetBrains spec file body syntax: `< ./relative/path` — reads file content as request body
  - Go `filepath.Rel` and `filepath.Abs` for secure path resolution

  **Acceptance Criteria**:
  - [ ] `< ./body.json` loads file content into Request.Body
  - [ ] Path resolved relative to .http file location
  - [ ] Path traversal outside collection root is blocked
  - [ ] Missing file returns descriptive error
  - [ ] `go test ./internal/parser/... -run TestFileLoader` passes

  **QA Scenarios**:
  ```
  Scenario: Load file body from relative path
    Tool: Bash
    Steps:
      1. Create /tmp/test-collection/api/create-user.http with POST and < ./body.json
      2. Create /tmp/test-collection/api/body.json with {"name": "test"}
      3. Run LoadFileBody, assert Body = {"name": "test"}
    Expected Result: File loaded relative to .http file
    Evidence: .sisyphus/evidence/task-8-fileloader.txt

  Scenario: Path traversal blocked
    Tool: Bash
    Steps:
      1. Create request with BodyFile = "../../../etc/passwd"
      2. Set collection root to /tmp/test-collection
      3. Run LoadFileBody, assert error returned
    Expected Result: Error about path traversal, no file read
    Evidence: .sisyphus/evidence/task-8-fileloader-traversal.txt
  ```

  **Commit**: YES (groups with T7)
  - Message: `feat(engine): add HTTP execution engine with file body support`
  - Pre-commit: `go test ./internal/engine/... && go test ./internal/parser/...`

- [ ] 9. TUI Shell (Bubbletea v2 App Skeleton)

  **What to do**:
  - Create `internal/tui/app.go` — root Bubbletea v2 model
  - CRITICAL: Use Bubbletea v2 API (NOT v1):
    - Import: `charm.land/bubbletea/v2` (NOT `github.com/charmbracelet/bubbletea`)
    - `View()` returns `tea.View` (NOT `string`)
    - Use `tea.NewView(s)` to create views
    - Set `v.AltScreen = true` via View fields (NOT `tea.WithAltScreen()`)
    - Use `tea.KeyPressMsg` (NOT `tea.KeyMsg`)
    - Use `charm.land/lipgloss/v2` for styling
    - Use `charm.land/bubbles/v2` for components
  - Root model structure:
    ```go
    type App struct {
        width, height int
        focus         Pane          // which pane has focus (Browser or Detail)
        browser       BrowserModel  // left pane (Task 10)
        detail        DetailModel   // right pane (Task 11)
        search        SearchModel   // overlay (Task 12)
        envSwitch     EnvModel      // overlay (Task 13)
        showSearch    bool
        showEnvSwitch bool
        // Will be wired in Task 18
    }
    type Pane int
    const (PaneBrowser Pane = iota; PaneDetail)
    ```
  - Implement:
    - `Init() tea.Cmd` — return `tea.WindowSize` command
    - `Update(msg tea.Msg) (tea.Model, tea.Cmd)` — handle:
      - `tea.WindowSizeMsg` → store dimensions, propagate to child models
      - `tea.KeyPressMsg` → global keys: `q`/`ctrl+c` quit, `tab` switch pane focus, `/` open search, `e` open env switcher
      - Delegate other keys to focused pane
    - `View() tea.View` — layout with lipgloss:
      - `lipgloss.JoinHorizontal()` for browser (30%) + detail (70%)
      - Status bar at bottom: current env name, request count, help text
      - Search/env overlays rendered on top when active
  - For now, browser and detail can be placeholder models that just show "Browser" / "Detail" text
  - Create `internal/tui/styles.go` — shared lipgloss styles:
    - Border styles for panes
    - Active vs inactive pane border colors
    - Status bar style
    - Keep minimal — just enough for structure

  **Must NOT do**:
  - Don't implement browser or detail content (Tasks 10, 11)
  - Don't connect to parser or engine yet (Task 18)
  - Don't add cobra CLI (Task 19) — just a `RunApp()` function
  - Don't use deprecated v1 API patterns

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T10-T17 after Wave 2)
  - **Parallel Group**: Wave 3 (but T10, T11 depend on this)
  - **Blocks**: T10, T11, T12, T13, T18
  - **Blocked By**: T1 (needs project structure)

  **References**:
  **External References**:
  - Bubbletea v2 split-editors example — reference architecture for dual-pane layout with focus tracking
  - Bubbletea v2 migration: `View()` returns `tea.View`, import `charm.land/bubbletea/v2`
  - Lipgloss v2: `charm.land/lipgloss/v2` — `JoinHorizontal`, `Place`, `NewStyle`

  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds with Bubbletea v2 imports
  - [ ] TUI launches in alt screen mode
  - [ ] Window resize updates layout
  - [ ] `q` / `ctrl+c` quits cleanly
  - [ ] Tab switches focus between panes (visual indicator changes)
  - [ ] Status bar shows at bottom

  **QA Scenarios**:
  ```
  Scenario: TUI launches and quits
    Tool: interactive_bash (tmux)
    Steps:
      1. Create tmux session: new-session -d -s restless-test
      2. send-keys "cd /Users/msh/code/pp/restless && go run ./cmd/restless" Enter
      3. Wait 2s for TUI to render
      4. Capture pane: capture-pane -t restless-test -p
      5. Assert output contains "Browser" and "Detail" placeholder text
      6. send-keys "q" to quit
      7. Assert process exited cleanly
    Expected Result: TUI renders dual-pane layout, quits on 'q'
    Evidence: .sisyphus/evidence/task-9-tui-shell.txt

  Scenario: Tab switches focus
    Tool: interactive_bash (tmux)
    Steps:
      1. Launch TUI in tmux
      2. Capture pane, note active pane indicator
      3. send-keys Tab
      4. Capture pane, assert active indicator moved
    Expected Result: Focus visually switches between panes
    Evidence: .sisyphus/evidence/task-9-tui-focus.txt
  ```

  **Commit**: YES
  - Message: `feat(tui): add Bubbletea v2 app shell`
  - Pre-commit: `go build ./...`

- [ ] 10. Collection Browser Pane

  **What to do**:
  - Create `internal/tui/browser.go` — left pane showing collection tree
  - Display hierarchy: directory → .http files → individual requests within each file
  - Tree structure:
    ```
    📁 api/
      📄 auth.http
        GET /login
        POST /register
      📄 users.http
        GET /users
        GET /users/{id}
        POST /users
    📁 webhooks/
      📄 stripe.http
        POST /webhook
    ```
  - Features:
    - Navigate with `j`/`k` or arrow keys (up/down)
    - `Enter` on directory → expand/collapse
    - `Enter` on request → select it (sends message to detail pane)
    - Visual indicators: method color-coding (GET=green, POST=blue, PUT=yellow, DELETE=red, PATCH=orange)
    - Show request name (from `# @name`) if available, otherwise show `METHOD /path`
    - Currently selected request highlighted
    - Scrollable if list exceeds pane height
  - Data loading:
    - Function `LoadCollection(rootDir string) (*model.Collection, error)` — walks directory, parses all .http files
    - Use `parser.ParseFile()` from Task 5 for each .http file
    - Sort: directories first, then files alphabetically
  - Model:
    ```go
    type BrowserModel struct {
        collection *model.Collection
        items      []BrowserItem      // flattened tree for display
        cursor     int                // current position
        expanded   map[string]bool    // which directories are expanded
        selected   *model.Request     // currently selected request
        height     int                // available height for scrolling
        offset     int                // scroll offset
    }
    type BrowserItem struct {
        Type    ItemType  // Dir, File, Request
        Depth   int       // indentation level
        Label   string    // display text
        Path    string    // for Dir/File
        Request *model.Request  // for Request items
    }
    ```

  **Must NOT do**:
  - Don't execute requests from browser (detail pane handles that)
  - Don't implement file watching / auto-reload
  - Don't add drag-and-drop or reordering

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
    - Reason: Tree navigation with expand/collapse, scrolling, and method color-coding requires careful TUI work

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T11-T17)
  - **Parallel Group**: Wave 3
  - **Blocks**: T12, T18
  - **Blocked By**: T5 (parser for loading), T9 (TUI shell)

  **References**:
  **Pattern References**:
  - `internal/parser/parser.go` (Task 5) — `ParseFile()` for loading .http files
  - `internal/model/collection.go` (Task 2) — Collection and HTTPFile types

  **External References**:
  - Bubbles v2 list component: `charm.land/bubbles/v2/list` — may be useful for scrollable list, or hand-roll for tree
  - Lipgloss v2 color: method color-coding with `lipgloss.Color()`

  **Acceptance Criteria**:
  - [ ] Displays directory tree with .http files and their requests
  - [ ] j/k and arrow keys navigate
  - [ ] Enter expands/collapses directories
  - [ ] Enter on request selects it
  - [ ] Method names color-coded (GET=green, POST=blue, etc.)
  - [ ] Scrolls when list exceeds pane height
  - [ ] Request names shown from `# @name` metadata

  **QA Scenarios**:
  ```
  Scenario: Browse collection tree
    Tool: interactive_bash (tmux)
    Steps:
      1. Create sample collection: api/auth.http (2 requests), api/users.http (3 requests)
      2. Launch restless in tmux pointing at sample collection
      3. Capture pane, assert tree shows directories and files
      4. send-keys "j" 3 times, then Enter to select a request
      5. Capture pane, assert request is highlighted
    Expected Result: Tree renders correctly, navigation works
    Evidence: .sisyphus/evidence/task-10-browser.txt

  Scenario: Method color coding visible
    Tool: interactive_bash (tmux)
    Steps:
      1. Launch with collection containing GET, POST, DELETE requests
      2. Capture pane output
      3. Assert GET, POST, DELETE labels appear with distinct styling
    Expected Result: Methods are visually distinguished
    Evidence: .sisyphus/evidence/task-10-browser-colors.txt
  ```

  **Commit**: YES (groups with T11-T13)
  - Message: `feat(tui): add collection browser, detail pane, search, env switcher`
  - Pre-commit: `go build ./...`

- [ ] 11. Request/Response Detail Pane

  **What to do**:
  - Create `internal/tui/detail.go` — right pane showing request details and response
  - Two modes: **Request View** (before send) and **Response View** (after send)
  - Request View:
    - Show method + URL (with resolved variables highlighted differently from literal text)
    - Show headers as key: value list
    - Show body (with syntax highlighting for JSON)
    - Show metadata tags (`@no-redirect`, `@timeout`)
    - Action: `Enter` or `ctrl+r` sends the request
  - Response View (after sending):
    - Tab bar: `Headers` | `Body` | `Timing` — switch with `1`, `2`, `3` keys
    - **Headers tab**: Status line (e.g., `HTTP/1.1 200 OK`), then response headers as key: value
    - **Body tab**: Response body with syntax highlighting:
      - JSON: pretty-print with `tidwall/pretty`, color syntax (keys, strings, numbers, booleans)
      - HTML/XML: basic tag highlighting
      - Other: raw text
    - **Timing tab**: Waterfall-style display:
      ```
      DNS        ████░░░░░░░░░░░░  12ms
      Connect    ░░░░████░░░░░░░░   8ms
      TLS        ░░░░░░░░████░░░░  15ms
      TTFB       ░░░░░░░░░░░░████  22ms
      Body       ░░░░░░░░░░░░░░██   3ms
      Total      ████████████████  60ms
      ```
    - Status code color: 2xx=green, 3xx=yellow, 4xx=orange, 5xx=red
    - Body scrollable if longer than pane height
  - Use `tidwall/pretty` for JSON formatting
  - Use lipgloss for all styling

  **Must NOT do**:
  - Don't implement request editing (read-only display of .http file content)
  - Don't implement response saving (Task 14)
  - Don't implement request chaining display (Task 15)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: []
    - Reason: Heavy UI work — syntax highlighting, timing waterfall, tab switching, scroll

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T10, T12-T17)
  - **Parallel Group**: Wave 3
  - **Blocks**: T18
  - **Blocked By**: T7 (engine for response), T9 (TUI shell)

  **References**:
  **Pattern References**:
  - `internal/engine/engine.go` (Task 7) — Response struct with timing fields
  - `internal/model/response.go` (Task 2) — ResponseTiming struct for waterfall

  **External References**:
  - `tidwall/pretty` — JSON pretty-printing and coloring
  - Lipgloss v2 tables/alignment for timing waterfall
  - Bubbletea viewport for scrollable content: `charm.land/bubbles/v2/viewport`

  **Acceptance Criteria**:
  - [ ] Shows request details (method, URL, headers, body) before sending
  - [ ] Enter/ctrl+r sends request and switches to response view
  - [ ] Response shows status with color coding (2xx green, 4xx orange, 5xx red)
  - [ ] Tab switching between Headers/Body/Timing with 1/2/3 keys
  - [ ] JSON body pretty-printed with syntax coloring
  - [ ] Timing waterfall displays all timing phases
  - [ ] Body scrollable when longer than pane

  **QA Scenarios**:
  ```
  Scenario: Send request and view response
    Tool: interactive_bash (tmux)
    Steps:
      1. Create .http file with GET https://httpbin.org/json
      2. Launch restless, navigate to request, press Enter
      3. Wait for response, capture pane
      4. Assert: status "200 OK" visible, JSON body rendered
      5. Press "3" for timing tab, capture pane
      6. Assert: timing waterfall visible with DNS, Connect, etc.
    Expected Result: Full request/response cycle works in TUI
    Evidence: .sisyphus/evidence/task-11-detail.txt

  Scenario: JSON syntax highlighting
    Tool: interactive_bash (tmux)
    Steps:
      1. Send request that returns JSON body
      2. Press "2" for body tab
      3. Capture pane, assert body is pretty-printed (indented, not single line)
    Expected Result: JSON formatted with indentation
    Evidence: .sisyphus/evidence/task-11-detail-json.txt
  ```

  **Commit**: YES (groups with T10, T12, T13)
  - Message: `feat(tui): add collection browser, detail pane, search, env switcher`
  - Pre-commit: `go build ./...`

- [ ] 12. Fuzzy Search Overlay

  **What to do**:
  - Create `internal/tui/search.go` — search overlay for finding requests
  - Activated by pressing `/` in the main TUI
  - Search input at top, filtered results below
  - Search matches against: request name (`@name`), method, URL path
  - Fuzzy matching: characters don't need to be contiguous (e.g., "gus" matches "GET /users")
  - Results update as user types (live filtering)
  - Enter on a result → navigate to that request in browser + select in detail
  - Escape closes overlay
  - Model:
    ```go
    type SearchModel struct {
        input     textinput.Model  // from bubbles/v2/textinput
        results   []SearchResult
        allItems  []SearchResult   // all searchable items
        cursor    int
        height    int
    }
    type SearchResult struct {
        Request  *model.Request
        File     string
        MatchStr string  // highlighted match display
    }
    ```
  - Use simple fuzzy matching algorithm (score by character position distance)
  - Display: `METHOD  URL  (filename)` for each result

  **Must NOT do**:
  - Don't use external fuzzy matching library — implement simple scoring
  - Don't persist search history
  - Don't search response bodies

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T10, T11, T13-T17)
  - **Parallel Group**: Wave 3
  - **Blocks**: T18
  - **Blocked By**: T9 (TUI shell), T10 (needs collection items)

  **References**:
  **External References**:
  - Bubbles v2 textinput: `charm.land/bubbles/v2/textinput` — input field component
  - Simple fuzzy matching: score based on character index distance in match string

  **Acceptance Criteria**:
  - [ ] `/` opens search overlay
  - [ ] Typing filters results live
  - [ ] Fuzzy matching works ("gus" matches "GET /users")
  - [ ] Enter selects result and navigates to it
  - [ ] Escape closes search
  - [ ] Results show method, URL, filename

  **QA Scenarios**:
  ```
  Scenario: Fuzzy search finds request
    Tool: interactive_bash (tmux)
    Steps:
      1. Create collection with 10+ requests across multiple files
      2. Launch restless, press "/"
      3. Type "pos us" (fuzzy for "POST /users")
      4. Assert: filtered results show POST /users request
      5. Press Enter, assert browser cursor moved to that request
    Expected Result: Search filters and navigates correctly
    Evidence: .sisyphus/evidence/task-12-search.txt
  ```

  **Commit**: YES (groups with T10, T11, T13)
  - Message: `feat(tui): add collection browser, detail pane, search, env switcher`
  - Pre-commit: `go build ./...`

- [ ] 13. Environment Switcher Overlay

  **What to do**:
  - Create `internal/tui/envswitch.go` — environment selection overlay
  - Activated by pressing `e` in the main TUI
  - Shows list of available environments from `http-client.env.json`
  - Current environment highlighted with indicator (e.g., `● dev  ○ staging  ○ prod`)
  - Navigate with j/k, Enter to select, Escape to cancel
  - When environment changes:
    - Send message to app model with new env name
    - App re-resolves all variables with new environment
    - Status bar updates to show current env name
  - Also show "No Environment" option (use no variables)
  - Model:
    ```go
    type EnvModel struct {
        envFile   *model.EnvironmentFile
        envNames  []string
        current   string  // currently active env name
        cursor    int
    }
    ```

  **Must NOT do**:
  - Don't allow editing environment values in TUI
  - Don't show environment variable contents (just names)
  - Don't auto-detect environment from git branch or hostname

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T10-T12, T14-T17)
  - **Parallel Group**: Wave 3
  - **Blocks**: T18
  - **Blocked By**: T4 (env parser), T9 (TUI shell)

  **References**:
  **Pattern References**:
  - `internal/parser/envparser.go` (Task 4) — `LoadEnvironments()` provides env names

  **Acceptance Criteria**:
  - [ ] `e` opens environment switcher overlay
  - [ ] Lists all environments from env file + "No Environment"
  - [ ] Current environment marked with indicator
  - [ ] Enter selects environment
  - [ ] Escape cancels without changing
  - [ ] Status bar reflects current environment

  **QA Scenarios**:
  ```
  Scenario: Switch environment
    Tool: interactive_bash (tmux)
    Steps:
      1. Create http-client.env.json with "dev" and "prod" environments
      2. Launch restless, assert status bar shows "dev" (or no env)
      3. Press "e", assert overlay shows "dev", "prod", "No Environment"
      4. Navigate to "prod", press Enter
      5. Assert status bar now shows "prod"
    Expected Result: Environment switches and status bar updates
    Evidence: .sisyphus/evidence/task-13-envswitch.txt
  ```

  **Commit**: YES (groups with T10, T11, T12)
  - Message: `feat(tui): add collection browser, detail pane, search, env switcher`
  - Pre-commit: `go build ./...`

- [ ] 14. Response History

  **What to do**:
  - Create `internal/history/history.go` — store and retrieve response history
  - Storage location: `.restless/history/` in collection root directory
  - File format: one JSON file per response, named `{timestamp}_{method}_{url-slug}.json`
    ```json
    {
      "request": { "method": "GET", "url": "...", "headers": [...] },
      "response": { "status_code": 200, "headers": [...], "body": "...", "timing": {...} },
      "environment": "dev",
      "timestamp": "2026-03-28T10:30:00Z"
    }
    ```
  - Functions:
    - `Save(req *model.Request, resp *model.Response, envName string) error`
    - `List(req *model.Request) ([]HistoryEntry, error)` — list all history for a specific request (matched by method+URL)
    - `Load(path string) (*HistoryEntry, error)` — load a specific history entry
    - `Diff(a, b *HistoryEntry) string` — text diff of two responses (body diff + header changes + status change)
  - For diff: simple line-by-line diff of pretty-printed JSON bodies. Use a basic diff algorithm (Myers or patience) — don't pull a heavy library.
  - Create `.restless/` directory automatically on first save
  - Add `.restless/` to generated `.gitignore` (response history is local, not shared)
  - History model for TUI:
    - Provide `HistoryEntry` struct with all data needed for display
    - Keep sorted by timestamp (newest first)
    - Limit to last 100 entries per request (configurable later)

  **Must NOT do**:
  - Don't build TUI for browsing history (Task 18 wires it in)
  - Don't compress or deduplicate entries
  - Don't sync history via git

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T9-T13, T15-T17)
  - **Parallel Group**: Wave 3
  - **Blocks**: T15, T18
  - **Blocked By**: T7 (needs Response type with real data)

  **References**:
  **Pattern References**:
  - `internal/model/response.go` (Task 2) — Response struct to serialize

  **External References**:
  - Myers diff algorithm — for text diffing of response bodies
  - Go `encoding/json` — serialize/deserialize history entries

  **Acceptance Criteria**:
  - [ ] Saves response to `.restless/history/` as JSON
  - [ ] Lists history entries for a request (by method+URL match)
  - [ ] Loads individual history entry
  - [ ] Diff shows body changes, header changes, status changes between two entries
  - [ ] Auto-creates `.restless/` directory
  - [ ] `go test ./internal/history/...` passes

  **QA Scenarios**:
  ```
  Scenario: Save and retrieve response history
    Tool: Bash
    Steps:
      1. Create request and response objects
      2. Call Save() twice with different response bodies
      3. Call List() for that request, assert 2 entries returned (newest first)
      4. Call Diff() on the two entries, assert diff shows body changes
    Expected Result: History round-trips correctly, diff works
    Evidence: .sisyphus/evidence/task-14-history.txt

  Scenario: History directory auto-created
    Tool: Bash
    Steps:
      1. Remove .restless/ directory if exists
      2. Call Save() with a response
      3. Assert .restless/history/ directory was created
      4. Assert history file exists with correct JSON
    Expected Result: Directory created automatically
    Evidence: .sisyphus/evidence/task-14-history-autodir.txt
  ```

  **Commit**: YES (groups with T15, T16)
  - Message: `feat(engine): add response history, request chaining, cookie jar`
  - Pre-commit: `go test ./internal/...`

- [ ] 15. Request Chaining Engine

  **What to do**:
  - Create `internal/chain/chain.go` — manage request chaining context
  - Core concept: when a named request (`# @name login`) is executed, its response is stored in a `ChainContext` so subsequent requests can reference it via `{{login.response.body.token}}`
  - Functions:
    - `NewChainContext() *ChainContext`
    - `(ctx *ChainContext) StoreResponse(name string, resp *model.Response)` — store response keyed by @name
    - `(ctx *ChainContext) Resolve(varRef string) (string, error)` — resolve a chain variable reference
  - Variable reference parsing:
    - `{{requestName.response.body.jsonpath}}` → extract value from JSON response body
    - `{{requestName.response.headers.Header-Name}}` → extract response header value
    - Parse format: split by `.`, first segment = request name, then `response`, then `body`/`headers`, then path
  - For body extraction:
    - Use `tidwall/gjson` for JSON path access (simple dot notation: `body.users.0.id`)
    - Non-JSON responses: treat body as plain text (only full body accessible, no path)
  - For header extraction:
    - Case-insensitive header name lookup
    - Return first matching header value
  - Chain context persists for the duration of a TUI session
  - When user runs requests in sequence, context accumulates
  - Create `internal/chain/chain_test.go`:
    - Store response, resolve body path
    - Store response, resolve header
    - Resolve nested JSON path (`body.data.items.0.id`)
    - Resolve non-existent request name → error
    - Resolve non-existent body path → error with context
    - Multiple responses in context, resolve from correct one

  **Must NOT do**:
  - Don't persist chain context to disk (session-only)
  - Don't auto-execute chained requests (user triggers each)
  - Don't support request ordering / sequencing

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
    - Reason: Variable reference parsing + JSON path extraction + error handling needs careful implementation

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T9-T14, T16-T17)
  - **Parallel Group**: Wave 3
  - **Blocks**: T18
  - **Blocked By**: T6 (resolver), T7 (engine), T14 (history for stored responses)

  **References**:
  **Pattern References**:
  - `internal/parser/resolver.go` (Task 6) — ChainContext type definition, integration point

  **External References**:
  - `tidwall/gjson` — `gjson.Get(json, "path.to.value")` for JSON extraction
  - JetBrains spec chaining: `{{requestName.response.body.jsonpath}}`, `{{requestName.response.headers.Name}}`

  **Acceptance Criteria**:
  - [ ] Store named response and resolve `{{name.response.body.path}}`
  - [ ] Resolve `{{name.response.headers.Header-Name}}`
  - [ ] Nested JSON paths work (`body.data.items.0.id`)
  - [ ] Missing request name returns descriptive error
  - [ ] Missing body path returns descriptive error
  - [ ] `go test ./internal/chain/...` passes

  **QA Scenarios**:
  ```
  Scenario: Chain requests with body extraction
    Tool: Bash
    Steps:
      1. Create ChainContext
      2. Store "login" response with body {"access_token": "jwt-123", "user": {"id": 42}}
      3. Resolve "login.response.body.access_token" → "jwt-123"
      4. Resolve "login.response.body.user.id" → "42"
    Expected Result: JSON paths correctly extracted
    Evidence: .sisyphus/evidence/task-15-chain.txt

  Scenario: Chain with header extraction
    Tool: Bash
    Steps:
      1. Store response with header X-Request-Id: abc-123
      2. Resolve "name.response.headers.X-Request-Id" → "abc-123"
      3. Resolve "name.response.headers.x-request-id" → "abc-123" (case insensitive)
    Expected Result: Headers resolved case-insensitively
    Evidence: .sisyphus/evidence/task-15-chain-headers.txt
  ```

  **Commit**: YES (groups with T14, T16)
  - Message: `feat(engine): add response history, request chaining, cookie jar`
  - Pre-commit: `go test ./internal/...`

- [ ] 16. Cookie Jar

  **What to do**:
  - Create `internal/engine/cookies.go` — per-environment cookie management
  - Wrap `net/http/cookiejar` with environment awareness:
    ```go
    type CookieManager struct {
        jars map[string]*cookiejar.Jar  // keyed by env name
    }
    ```
  - Functions:
    - `NewCookieManager() *CookieManager`
    - `(cm *CookieManager) JarForEnv(envName string) http.CookieJar` — get or create jar for environment
    - `(cm *CookieManager) ClearEnv(envName string)` — clear cookies for an environment
    - `(cm *CookieManager) ClearAll()` — clear all cookies
  - Integration with engine:
    - When executing a request, set `http.Client.Jar` to the jar for current environment
    - If request has `@no-cookie-jar` metadata, use a nil jar (no cookies sent/received)
  - Cookie jars are in-memory only (session duration), not persisted to disk
  - Create `internal/engine/cookies_test.go`:
    - Cookies persist between requests in same environment
    - Cookies isolated between environments (dev cookies don't leak to prod)
    - `@no-cookie-jar` prevents cookie use
    - ClearEnv removes only that env's cookies

  **Must NOT do**:
  - Don't persist cookies to disk
  - Don't show cookies in TUI (just silent background management)
  - Don't implement cookie editing

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T9-T15, T17)
  - **Parallel Group**: Wave 3
  - **Blocks**: T18
  - **Blocked By**: T7 (engine)

  **References**:
  **External References**:
  - Go `net/http/cookiejar` package: `https://pkg.go.dev/net/http/cookiejar`
  - JetBrains `@no-cookie-jar` metadata tag

  **Acceptance Criteria**:
  - [ ] Cookies persist between requests in same environment
  - [ ] Cookies isolated between environments
  - [ ] `@no-cookie-jar` disables cookies for that request
  - [ ] ClearEnv/ClearAll work correctly
  - [ ] `go test ./internal/engine/... -run TestCookie` passes

  **QA Scenarios**:
  ```
  Scenario: Cookie isolation between environments
    Tool: Bash
    Steps:
      1. Create CookieManager
      2. Execute request in "dev" env that sets cookie
      3. Execute request in "prod" env, assert cookie NOT sent
      4. Execute request in "dev" env again, assert cookie IS sent
    Expected Result: Cookies isolated per environment
    Evidence: .sisyphus/evidence/task-16-cookies.txt
  ```

  **Commit**: YES (groups with T14, T15)
  - Message: `feat(engine): add response history, request chaining, cookie jar`
  - Pre-commit: `go test ./internal/...`

- [ ] 17. Postman Import

  **What to do**:
  - Create `internal/importer/postman.go` — convert Postman Collection v2.1 JSON → .http files
  - Use `rbretecher/go-postman-collection` to parse Postman format
  - Conversion rules:
    - Collection name → root directory name
    - Folders → subdirectories
    - Requests → grouped into .http files by folder (one .http file per folder)
    - Request name → `# @name` tag
    - Method + URL → request line
    - Headers → header lines
    - Body (raw JSON, form-data, urlencoded) → body section
    - Auth (bearer token, basic auth) → Authorization header
    - Variables → `{{variable}}` syntax (same format)
  - Environment conversion:
    - Postman environment export → `http-client.env.json`
    - Postman globals → `$shared` key in env file
  - CLI usage: `restless import postman <collection.json> [--env <environment.json>] [--output <dir>]`
  - Create `internal/importer/postman_test.go`:
    - Import simple collection (3 requests, no folders)
    - Import collection with folders → subdirectories
    - Import with bearer token auth → Authorization header
    - Import with body (JSON) → body section
    - Import with environment → env.json
    - Verify generated .http files are valid (can be parsed by Task 5's parser)

  **Must NOT do**:
  - Don't support Postman Collection v1 (ancient format)
  - Don't support Postman pre-request scripts or test scripts → skip silently
  - Don't support Postman's proprietary variable scopes (collection, global, environment → all become flat env vars)
  - Don't import Postman responses/examples

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T9-T16)
  - **Parallel Group**: Wave 3
  - **Blocks**: T20
  - **Blocked By**: T2 (model types), T5 (parser for validation), T4 (env format)

  **References**:
  **External References**:
  - `rbretecher/go-postman-collection` — Go library for parsing Postman Collection v2.1 JSON
  - Postman Collection v2.1 schema: `https://schema.getpostman.com/json/collection/v2.1.0/collection.json`
  - JetBrains .http format for output generation

  **Acceptance Criteria**:
  - [ ] Converts Postman Collection v2.1 JSON → .http files
  - [ ] Folders become directories
  - [ ] Request names become `# @name` tags
  - [ ] Auth tokens converted to Authorization headers
  - [ ] Body preserved correctly
  - [ ] Generated .http files parse successfully with restless parser
  - [ ] `go test ./internal/importer/...` passes

  **QA Scenarios**:
  ```
  Scenario: Import Postman collection
    Tool: Bash
    Steps:
      1. Create sample Postman collection JSON with folder "Auth" (login, register) and folder "Users" (list, get, create)
      2. Run: go run ./cmd/restless import postman sample.json --output /tmp/imported
      3. Assert: /tmp/imported/Auth/auth.http exists with 2 requests
      4. Assert: /tmp/imported/Users/users.http exists with 3 requests
      5. Parse generated files with restless parser, assert no errors
    Expected Result: Valid .http files generated from Postman collection
    Evidence: .sisyphus/evidence/task-17-postman-import.txt

  Scenario: Import with environment
    Tool: Bash
    Steps:
      1. Create Postman environment JSON with "dev" vars (base_url, token)
      2. Run import with --env flag
      3. Assert http-client.env.json created with correct structure
    Expected Result: Environment file generated correctly
    Evidence: .sisyphus/evidence/task-17-postman-env.txt
  ```

  **Commit**: YES
  - Message: `feat(import): add Postman collection import`
  - Pre-commit: `go test ./internal/importer/...`

- [ ] 18. Wire Everything Together

  **What to do**:
  - This is the integration task — connect all modules into a working application
  - In `internal/tui/app.go`, wire:
    - On startup: `LoadCollection(rootDir)` → populate browser with parsed .http files
    - On startup: `LoadEnvironments(rootDir)` → populate env switcher
    - Browser request selection → populate detail pane with request
    - Detail pane "send" action → resolve variables (Task 6) → load file body (Task 8) → execute (Task 7 + Task 16 cookies) → display response in detail → save to history (Task 14) → store in chain context (Task 15)
    - Environment switch → update resolver context, re-resolve displayed request
    - Search selection → navigate browser to that request → update detail
    - History view: `h` key in detail pane shows history list for current request, navigate to see past responses, `d` to diff two selected entries
  - Keyboard navigation (full map):
    ```
    Global:
      q / ctrl+c  → quit
      tab         → switch pane focus
      /           → open search
      e           → open env switcher
    
    Browser (when focused):
      j/k / ↑/↓   → navigate
      Enter        → expand dir / select request
      h            → collapse dir (or go to parent)
    
    Detail (when focused):
      Enter / ctrl+r → send request
      1/2/3          → switch response tab (headers/body/timing)
      j/k / ↑/↓      → scroll body
      h              → show history for current request
      d              → diff mode (select two history entries)
      y              → copy response body to clipboard
    
    Overlays:
      Escape         → close overlay
      Enter          → select
      j/k / ↑/↓      → navigate
    ```
  - Loading states:
    - While request is executing: show spinner in detail pane
    - While collection is loading: show "Loading..." in browser
  - Error handling:
    - Parse errors: show in browser with warning icon
    - Network errors: show in detail pane with error message
    - Unresolved variables: show warning but still allow sending

  **Must NOT do**:
  - Don't add new features beyond wiring existing modules
  - Don't refactor module internals
  - Don't add command-line flags (Task 19)

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
    - Reason: Complex integration of 10+ modules, message passing, state management

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (sequential — needs all Wave 3 tasks)
  - **Blocks**: T19, T20
  - **Blocked By**: T10, T11, T12, T13, T14, T15, T16

  **References**:
  **Pattern References**:
  - ALL internal packages from Tasks 2-17 — this task integrates them all
  - `internal/tui/app.go` (Task 9) — root model to extend

  **External References**:
  - Bubbletea v2 command pattern: return `tea.Cmd` from Update for async operations (HTTP requests)
  - Bubbletea v2 `tea.Batch()` for combining commands

  **Acceptance Criteria**:
  - [ ] Launch restless → collection loads → browser shows tree
  - [ ] Select request → detail shows request info
  - [ ] Send request → spinner → response displayed with timing
  - [ ] Response saved to history automatically
  - [ ] Named request response stored in chain context
  - [ ] Switch environment → variables re-resolved
  - [ ] Search finds and navigates to requests
  - [ ] History view shows past responses
  - [ ] All keyboard shortcuts functional

  **QA Scenarios**:
  ```
  Scenario: Full workflow — browse, send, chain
    Tool: interactive_bash (tmux)
    Steps:
      1. Create collection:
         - api/auth.http: POST /login with body (# @name login)
         - api/users.http: GET /users with Authorization: Bearer {{login.response.body.token}}
      2. Create http-client.env.json with "dev" env (base_url=http://localhost:3000)
      3. Start a mock HTTP server that returns {"token": "test-jwt"} for POST /login
      4. Launch restless
      5. Navigate to POST /login, press Enter to send
      6. Assert: response shows 200, body shows token
      7. Navigate to GET /users, assert: Authorization header shows "Bearer test-jwt" (from chain)
      8. Press Enter to send GET /users
      9. Assert: response displayed
    Expected Result: Full request chaining workflow works end-to-end
    Evidence: .sisyphus/evidence/task-18-integration.txt

  Scenario: Environment switching updates variables
    Tool: interactive_bash (tmux)
    Steps:
      1. Create env with "dev" (base_url=http://localhost) and "prod" (base_url=https://api.example.com)
      2. Launch restless, select a request with {{base_url}}
      3. Assert detail shows http://localhost
      4. Press "e", switch to "prod"
      5. Assert detail now shows https://api.example.com
    Expected Result: Variables re-resolve on env switch
    Evidence: .sisyphus/evidence/task-18-envswitch.txt

  Scenario: Error handling — network failure
    Tool: interactive_bash (tmux)
    Steps:
      1. Create request targeting http://localhost:99999 (no server)
      2. Send request
      3. Assert: detail pane shows error message (connection refused)
    Expected Result: Graceful error display, no crash
    Evidence: .sisyphus/evidence/task-18-error.txt
  ```

  **Commit**: YES (groups with T19)
  - Message: `feat(tui): wire all components, add CLI entry point`
  - Pre-commit: `go build ./... && go test ./...`

- [ ] 19. CLI Entry Point

  **What to do**:
  - Create `cmd/restless/main.go` — CLI with cobra
  - Add `github.com/spf13/cobra` dependency
  - Commands:
    ```
    restless                           # Launch TUI (default, no subcommand)
    restless [directory]               # Launch TUI with collection from directory
    restless import postman <file>     # Import Postman collection
      --env <file>                     # Also import Postman environment
      --output <dir>                   # Output directory (default: current dir)
    restless run <file.http>           # Run all requests in file sequentially (headless)
      --env <name>                     # Use specific environment
      --fail-fast                      # Stop on first error
    restless version                   # Print version
    ```
  - Root command (no subcommand) → launch TUI via `tui.RunApp(dir)`
  - `import postman` → call `importer.ImportPostman()`
  - `run` → parse file, resolve vars, execute each request, print results to stdout (simple, no TUI)
  - Version: use `ldflags` at build time (`-X main.version=...`)
  - Global flags:
    - `--no-color` → disable color output
    - `--dir <path>` → override collection root directory

  **Must NOT do**:
  - Don't add config file support (no `.restlessrc`)
  - Don't add shell completion generation (future)
  - Don't add `watch` or `serve` commands

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (needs T18)
  - **Parallel Group**: Wave 4 (after T18)
  - **Blocks**: T20, T21
  - **Blocked By**: T18

  **References**:
  **Pattern References**:
  - User's existing Go projects (`ports`, `envdiff`) — check if they use cobra for CLI patterns

  **External References**:
  - `github.com/spf13/cobra` — CLI framework
  - Go ldflags for version injection: `-ldflags "-X main.version=1.0.0"`

  **Acceptance Criteria**:
  - [ ] `restless` launches TUI
  - [ ] `restless ./api` launches TUI with collection from `./api`
  - [ ] `restless import postman collection.json` works
  - [ ] `restless run file.http --env dev` executes requests headlessly
  - [ ] `restless version` prints version
  - [ ] `restless --help` shows all commands

  **QA Scenarios**:
  ```
  Scenario: CLI commands work
    Tool: Bash
    Steps:
      1. go build -o restless ./cmd/restless
      2. ./restless version → assert prints version string
      3. ./restless --help → assert shows "import", "run", "version" subcommands
      4. ./restless import postman sample.json --output /tmp/test → assert .http files created
      5. ./restless run /tmp/test/api.http --env dev → assert requests executed, output printed
    Expected Result: All CLI commands functional
    Evidence: .sisyphus/evidence/task-19-cli.txt
  ```

  **Commit**: YES (groups with T18)
  - Message: `feat(tui): wire all components, add CLI entry point`
  - Pre-commit: `go build ./... && go test ./...`

- [ ] 20. Integration Tests

  **What to do**:
  - Create `tests/integration_test.go` — end-to-end tests
  - Test scenarios:
    1. **Parse → Execute → History**: Parse .http file, execute request against httptest server, verify response saved to history
    2. **Multi-request chaining**: Parse file with 2 named requests, execute first, verify chain context resolves in second
    3. **Environment resolution**: Load env file, parse .http with variables, resolve, verify URL/headers correct
    4. **Postman import → Parse**: Import Postman collection, parse generated .http files, verify round-trip integrity
    5. **Cookie persistence**: Execute request that sets cookie, execute second request, verify cookie sent
    6. **File body loading**: Parse request with `< ./body.json`, load file body, execute
    7. **Error scenarios**: Invalid .http syntax, missing env file, network timeout
  - Use `httptest.NewServer` for all HTTP tests (no real network calls)
  - Create test fixtures in `tests/fixtures/`:
    - `simple.http` — basic GET/POST requests
    - `chained.http` — requests with `# @name` and `{{name.response.body.path}}`
    - `with-vars.http` — requests with `{{base_url}}` etc.
    - `http-client.env.json` — test environments
    - `postman-sample.json` — sample Postman collection for import testing
    - `body.json` — sample body file for `< ./path` testing

  **Must NOT do**:
  - Don't test TUI rendering (too fragile for CI)
  - Don't test against real external APIs
  - Don't add benchmarks (future)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO (needs T18, T19)
  - **Parallel Group**: Wave 4 (after T18, T19)
  - **Blocks**: F1-F4
  - **Blocked By**: T18, T19, T17

  **References**:
  **Pattern References**:
  - All `internal/` packages — integration tests exercise full pipeline

  **External References**:
  - Go `testing` package with `httptest.NewServer`
  - `testify/assert` for cleaner assertions

  **Acceptance Criteria**:
  - [ ] All 7 integration test scenarios pass
  - [ ] Test fixtures created and valid
  - [ ] No real network calls (all use httptest)
  - [ ] `go test ./tests/...` passes
  - [ ] Tests complete in < 30 seconds

  **QA Scenarios**:
  ```
  Scenario: Integration tests pass
    Tool: Bash
    Steps:
      1. cd /Users/msh/code/pp/restless
      2. go test ./tests/... -v
      3. Assert all tests pass, no failures
      4. go test ./... -count=1 (full suite)
      5. Assert all tests pass
    Expected Result: Full test suite green
    Evidence: .sisyphus/evidence/task-20-integration.txt
  ```

  **Commit**: YES (groups with T21)
  - Message: `feat(restless): add integration tests and release config`
  - Pre-commit: `go test ./...`

- [ ] 21. Build & Release Setup

  **What to do**:
  - Update `Makefile` with complete targets:
    ```makefile
    VERSION ?= $(shell git describe --tags --always --dirty)
    LDFLAGS = -ldflags "-X main.version=$(VERSION)"
    
    build:
        go build $(LDFLAGS) -o bin/restless ./cmd/restless
    
    test:
        go test ./... -count=1
    
    vet:
        go vet ./...
    
    lint:
        staticcheck ./...
    
    run:
        go run $(LDFLAGS) ./cmd/restless
    
    install:
        go install $(LDFLAGS) ./cmd/restless
    
    clean:
        rm -rf bin/ .restless/
    
    .PHONY: build test vet lint run install clean
    ```
  - Create `.goreleaser.yaml` for cross-platform binary releases:
    - Build for: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64
    - Archive format: tar.gz (unix), zip (windows)
    - Homebrew tap generation
  - Update `README.md` with:
    - Project description (1 paragraph)
    - Installation (go install, homebrew, binary download)
    - Quick start (create .http file, run restless)
    - Key features list
    - Keyboard shortcuts reference
    - Format compatibility (.http spec link)
    - License (MIT)
  - Add `.github/workflows/ci.yml`:
    - On push/PR: `go test ./...`, `go vet ./...`, `go build ./...`
    - On tag: run goreleaser
  - Ensure `go install github.com/shahadulhaider/restless/cmd/restless@latest` works

  **Must NOT do**:
  - Don't add complex CI pipelines
  - Don't set up homebrew tap repo (just goreleaser config)
  - Don't add Docker image

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with T20)
  - **Parallel Group**: Wave 4
  - **Blocks**: F1-F4
  - **Blocked By**: T19

  **References**:
  **Pattern References**:
  - User's `ports` repo — check for existing goreleaser config, CI workflow, README patterns

  **External References**:
  - GoReleaser docs: `https://goreleaser.com/quick-start/`
  - GitHub Actions Go workflow

  **Acceptance Criteria**:
  - [ ] `make build` produces binary in `bin/restless`
  - [ ] `make test` runs all tests
  - [ ] `.goreleaser.yaml` is valid (`goreleaser check`)
  - [ ] README covers: install, quick start, features, keyboard shortcuts
  - [ ] CI workflow file is valid YAML

  **QA Scenarios**:
  ```
  Scenario: Build and install
    Tool: Bash
    Steps:
      1. cd /Users/msh/code/pp/restless
      2. make build
      3. ./bin/restless version → assert prints version
      4. make test → assert all pass
      5. make vet → assert no issues
    Expected Result: Build pipeline works end-to-end
    Evidence: .sisyphus/evidence/task-21-build.txt
  ```

  **Commit**: YES (groups with T20)
  - Message: `feat(restless): add integration tests and release config`
  - Pre-commit: `go test ./...`

---

## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `staticcheck ./...` + `go test ./...`. Review all files for: empty error handling, unused imports, `interface{}` where concrete types work, commented-out code. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Create sample .http files, env files. Launch `restless`. Test: browse collections, send requests, switch environments, view history, chain requests, import Postman collection. Capture screenshots via tmux.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual code. Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| After Task(s) | Commit Message | Pre-commit Check |
|---|---|---|
| T1 | `chore(restless): init project scaffolding` | `go build ./...` |
| T2 | `feat(model): add core types for request, response, environment` | `go test ./internal/model/...` |
| T3-T4 | `feat(parser): add .http lexer and environment parser` | `go test ./internal/parser/...` |
| T5-T6 | `feat(parser): add .http parser with variable resolution` | `go test ./internal/parser/...` |
| T7-T8 | `feat(engine): add HTTP execution engine with file body support` | `go test ./internal/engine/...` |
| T9 | `feat(tui): add Bubbletea v2 app shell` | `go build ./...` |
| T10-T13 | `feat(tui): add collection browser, detail pane, search, env switcher` | `go build ./...` |
| T14-T16 | `feat(engine): add response history, request chaining, cookie jar` | `go test ./internal/...` |
| T17 | `feat(import): add Postman collection import` | `go test ./internal/importer/...` |
| T18-T19 | `feat(tui): wire all components, add CLI entry point` | `go build ./... && go test ./...` |
| T20-T21 | `feat(restless): add integration tests and release config` | `go test ./...` |

---

## Success Criteria

### Verification Commands
```bash
go build ./...              # Expected: clean build, single binary
go test ./...               # Expected: all tests pass
go vet ./...                # Expected: no issues
./restless                  # Expected: TUI opens, shows collection browser
./restless import postman test.json  # Expected: generates .http files
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Single binary, no CGo, no external dependencies at runtime
- [ ] .http files from VS Code REST Client / IntelliJ parse correctly
- [ ] Response history persists across sessions
- [ ] Request chaining resolves variables from prior responses
