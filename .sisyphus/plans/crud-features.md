# CRUD, Collection Management, Importers & Copy-as-Curl — restless

## TL;DR

> **Quick Summary**: Add full CRUD for requests/collections in the TUI (inline forms + $EDITOR), 4 new importers (Insomnia, Bruno, curl, OpenAPI/Swagger), and copy-as-curl — transforming restless from a read-only viewer into a full-featured HTTP client.
> 
> **Deliverables**:
> - Request serializer (model → .http text) and file-level CRUD operations
> - TUI request editor with method selector, URL, headers, body, name, metadata fields
> - $EDITOR integration for full editing
> - Collection management in browser: create/rename/move/delete files and folders, duplicate requests
> - 4 importers: Insomnia, Bruno, curl commands, OpenAPI/Swagger
> - curl generator with clipboard copy
> - CLI commands for all importers
> 
> **Estimated Effort**: Large
> **Parallel Execution**: YES — 4 waves
> **Critical Path**: T1 (serializer) → T7 (file CRUD) → T11 (wire editor) → T14 (keybindings)

---

## Context

### Original Request
User wants restless to be "nearly feature complete like Postman, Insomnia, Bruno." Currently the TUI is read-only — cannot create, edit, or delete requests. Only Postman import exists. No way to create collections or manage files from the TUI.

### Interview Summary
**Key Discussions**:
- **Priority**: CRUD first, then imports. No WebSocket/GraphQL/scripting for now.
- **Editing UX**: Both inline forms AND $EDITOR support (Option C)
- **Inline form fields**: method, URL, headers (add/remove), body, request name, metadata (@no-redirect, @timeout, etc.)
- **Collection management**: Create/rename/move/delete .http files and folders, duplicate requests
- **Delete behavior**: Hard delete (rewrite file), no soft delete
- **Import scope**: All four — Insomnia, Bruno, curl, OpenAPI/Swagger
- **Copy as curl**: Yes, include it

### Research Findings
- `.http` file format uses `###` separators between requests — editing requires rewriting file sections
- `model.Request` has `SourceFile` and `SourceLine` fields — can locate requests in files for editing
- `charm.land/bubbles/v2` already an indirect dep — provides text inputs for TUI forms
- `gopkg.in/yaml.v3` already indirect dep — useful for OpenAPI YAML parsing
- Existing Postman importer at `internal/importer/postman.go` provides a pattern for new importers

---

## Work Objectives

### Core Objective
Transform restless from a read-only .http file viewer into a full CRUD HTTP client with import capabilities comparable to Postman, Insomnia, and Bruno.

### Concrete Deliverables
- `internal/writer/` — serializer + file CRUD + directory operations
- `internal/tui/editor.go` — request editor form component
- `internal/tui/confirm.go` — confirmation dialog component
- `internal/tui/manager.go` — collection management (file/folder operations in browser)
- `internal/importer/insomnia.go` — Insomnia JSON import
- `internal/importer/bruno.go` — Bruno .bru file import
- `internal/importer/curl.go` — curl command ↔ .http conversion
- `internal/importer/openapi.go` — OpenAPI/Swagger import
- `internal/exporter/curl.go` — Request → curl command string
- Updated CLI commands in `cmd/restless/main.go`
- Updated TUI with all new keybindings

### Definition of Done
- [ ] Can create a new request from TUI and it persists to .http file
- [ ] Can edit an existing request inline and changes persist
- [ ] Can open $EDITOR on a request file and changes reload
- [ ] Can delete a request from TUI and it's removed from file
- [ ] Can create/rename/delete .http files and folders from TUI
- [ ] Can duplicate a request within or across files
- [ ] `restless import insomnia <file>` converts Insomnia export to .http files
- [ ] `restless import bruno <dir>` converts Bruno collection to .http files
- [ ] `restless import curl "<command>"` converts curl to .http
- [ ] `restless import openapi <spec>` converts OpenAPI/Swagger to .http files
- [ ] Can copy request as curl from TUI
- [ ] All existing tests still pass

### Must Have
- Inline form editing with all fields (method, URL, headers, body, name, metadata)
- $EDITOR fallback for full editing
- Hard delete with file rewrite
- All 4 importers producing valid .http files
- Copy-as-curl from TUI

### Must NOT Have (Guardrails)
- Do NOT add WebSocket or GraphQL support
- Do NOT add pre/post request scripting
- Do NOT add OAuth2 or auth helper flows
- Do NOT add export to Postman/Insomnia/Bruno formats (only import FROM them)
- Do NOT modify the .http parser (lexer.go, parser.go) — build the writer as a separate module
- Do NOT change the existing request execution engine
- Do NOT break the existing headless `run` command
- Do NOT introduce CGo dependencies (keep cross-compilation clean)

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (go test)
- **Automated tests**: YES (TDD for writer + importers, tests-after for TUI integration)
- **Framework**: `go test ./...`

### QA Policy
Every task includes agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Unit tests**: `go test ./internal/writer/... ./internal/importer/... ./internal/exporter/...`
- **CLI verification**: Bash — run import commands, verify output files
- **TUI verification**: interactive_bash (tmux) — launch TUI, send keystrokes, verify behavior

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — foundation + independent importers, 6 parallel):
├── Task 1: Request serializer [deep]
├── Task 2: Directory operations module [quick]
├── Task 3: Insomnia importer [unspecified-high]
├── Task 4: Bruno importer [unspecified-high]
├── Task 5: curl parser + generator [unspecified-high]
└── Task 6: OpenAPI/Swagger importer [deep]

Wave 2 (After Wave 1 — file CRUD + TUI components, 5 parallel):
├── Task 7: File-level request CRUD (depends: T1) [deep]
├── Task 8: TUI request editor component (depends: T1) [deep]
├── Task 9: TUI confirmation dialog component [quick]
├── Task 10: Import CLI commands (depends: T3,T4,T5,T6) [unspecified-high]
└── Task 11: curl export + copy-as-curl (depends: T5) [quick]  ← RENAMED from T10

Wave 3 (After Wave 2 — full TUI integration, 4 parallel):
├── Task 12: Wire create/edit request flows into TUI app (depends: T7,T8) [deep]
├── Task 13: Wire collection management into browser (depends: T2,T7,T9) [deep]
├── Task 14: $EDITOR integration (depends: T7) [unspecified-high]
└── Task 15: Wire curl export into TUI (depends: T11) [quick]

Wave 4 (After Wave 3 — polish + integration):
├── Task 16: Update status bar, keybindings, help text (depends: T12,T13,T14,T15) [quick]
└── Task 17: Integration tests (depends: all) [deep]

Wave FINAL (After ALL tasks — 4 parallel reviews):
├── F1: Plan compliance audit (oracle)
├── F2: Code quality review (unspecified-high)
├── F3: Real manual QA (unspecified-high)
└── F4: Scope fidelity check (deep)
-> Present results -> Get explicit user okay
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|-----------|--------|
| T1 | — | T7, T8 |
| T2 | — | T13 |
| T3 | — | T10 |
| T4 | — | T10 |
| T5 | — | T10, T11 |
| T6 | — | T10 |
| T7 | T1 | T12, T13, T14 |
| T8 | T1 | T12 |
| T9 | — | T13 |
| T10 | T3,T4,T5,T6 | T17 |
| T11 | T5 | T15 |
| T12 | T7,T8 | T16 |
| T13 | T2,T7,T9 | T16 |
| T14 | T7 | T16 |
| T15 | T11 | T16 |
| T16 | T12,T13,T14,T15 | T17 |
| T17 | T16 | — |

### Agent Dispatch Summary

- **Wave 1**: **6** — T1 → `deep`, T2 → `quick`, T3 → `unspecified-high`, T4 → `unspecified-high`, T5 → `unspecified-high`, T6 → `deep`
- **Wave 2**: **5** — T7 → `deep`, T8 → `deep`, T9 → `quick`, T10 → `unspecified-high`, T11 → `quick`
- **Wave 3**: **4** — T12 → `deep`, T13 → `deep`, T14 → `unspecified-high`, T15 → `quick`
- **Wave 4**: **2** — T16 → `quick`, T17 → `deep`
- **FINAL**: **4** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. Request Serializer — model.Request → .http format text

  **What to do**:
  - Create `internal/writer/serializer.go` with function `SerializeRequest(req *model.Request) string`
  - Must produce valid .http format: `# @name Name\nMETHOD URL\nHeader: Value\n\nbody`
  - Handle all metadata: `@name`, `@no-redirect`, `@no-cookie-jar`, `@timeout N`, `@connection-timeout N`
  - Handle file body references: `< ./path`
  - Handle HTTP version if set: `GET https://url HTTP/1.1`
  - Add `SerializeRequests(reqs []model.Request) string` that joins with `\n###\n\n`
  - Write comprehensive tests in `internal/writer/serializer_test.go`
  - Round-trip test: parse a .http file → serialize → parse again → compare

  **Must NOT do**:
  - Do NOT modify the parser (lexer.go, parser.go)
  - Do NOT change model types
  - Keep serializer stateless — no file I/O

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Core module requiring precise format compliance and thorough round-trip testing
  - **Skills**: []
  - **Skills Evaluated but Omitted**:
    - `playwright`: No UI work
    - `git-master`: Standard git operations only

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T2, T3, T4, T5, T6)
  - **Blocks**: T7, T8
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/parser/parser.go:1-180` — Parser produces `model.Request` from tokens. Serializer must produce text that round-trips through this parser.
  - `internal/parser/lexer.go:1-150` — Token types and .http format rules. Serializer output must match what lexer expects.
  - `internal/importer/postman.go:77-115` — `convertItem()` already does basic serialization for Postman import. Use as starting point but make it comprehensive.

  **API/Type References**:
  - `internal/model/request.go:1-28` — `Request`, `Header`, `RequestMetadata` structs. These are the inputs.
  - `internal/model/request.go:20-28` — `RequestMetadata` with `NoRedirect`, `NoCookieJar`, `Timeout`, `ConnTimeout` fields.

  **Test References**:
  - `internal/parser/parser_test.go` — Shows how parser tests are structured. Match this style.
  - `internal/parser/lexer_test.go` — Token-level tests show exact format expectations.

  **WHY Each Reference Matters**:
  - `convertItem()` in postman.go is the closest existing serializer — but it's incomplete (no metadata, no HTTP version). Start from its pattern.
  - Parser + lexer define the canonical .http format — serializer must produce text that survives a round-trip.

  **Acceptance Criteria**:
  - [ ] `go test ./internal/writer/... -v` passes with ≥10 test cases
  - [ ] Round-trip: parse → serialize → parse produces identical Request structs
  - [ ] Handles all metadata tags correctly
  - [ ] Handles empty body, JSON body, file ref body

  **QA Scenarios**:

  ```
  Scenario: Round-trip serialization of a complex request
    Tool: Bash
    Preconditions: serializer_test.go exists
    Steps:
      1. Run `go test ./internal/writer/ -run TestRoundTrip -v`
      2. Assert exit code 0 and "PASS" in output
    Expected Result: All round-trip tests pass
    Failure Indicators: Any FAIL, parse errors, field mismatches
    Evidence: .sisyphus/evidence/task-1-roundtrip.txt

  Scenario: Serializer handles all metadata types
    Tool: Bash
    Preconditions: tests cover @name, @no-redirect, @no-cookie-jar, @timeout, @connection-timeout
    Steps:
      1. Run `go test ./internal/writer/ -run TestSerializeMetadata -v`
      2. Assert all metadata tags appear in output
    Expected Result: All metadata serialized correctly
    Failure Indicators: Missing @ tags, wrong format
    Evidence: .sisyphus/evidence/task-1-metadata.txt
  ```

  **Commit**: YES
  - Message: `feat(writer): add request serializer for .http format`
  - Files: `internal/writer/serializer.go`, `internal/writer/serializer_test.go`
  - Pre-commit: `go test ./internal/writer/...`

- [x] 2. Directory Operations Module

  **What to do**:
  - Create `internal/writer/dirops.go` with functions:
    - `CreateDirectory(path string) error`
    - `CreateHTTPFile(path string) error` — creates empty .http file with a comment header
    - `RenameEntry(oldPath, newPath string) error` — works for both files and dirs
    - `MoveEntry(src, dst string) error` — move file or dir
    - `DeleteEntry(path string) error` — remove file or dir (with `os.RemoveAll` for dirs)
    - `IsHTTPFile(path string) bool` — checks .http extension
  - All operations should validate paths are within the collection root (prevent path traversal)
  - Write tests in `internal/writer/dirops_test.go` using temp directories

  **Must NOT do**:
  - Do NOT do recursive operations without path validation
  - Do NOT delete non-.http files or non-empty directories without the recursive flag

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Straightforward file system operations with standard Go os package
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T3, T4, T5, T6)
  - **Blocks**: T13
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/parser/fileloader.go:14-22` — Path traversal validation pattern. Reuse this approach for all path operations.
  - `internal/tui/browser.go:63-80` — `LoadCollection()` walks directories to find .http files. Dir ops must maintain this structure.

  **WHY Each Reference Matters**:
  - fileloader.go has the path traversal guard pattern — reuse it to prevent escaping collection root.
  - browser.go shows the directory structure convention — new dirs/files must be compatible.

  **Acceptance Criteria**:
  - [ ] `go test ./internal/writer/ -run TestDirOps -v` passes
  - [ ] Path traversal attempts return error
  - [ ] Create/rename/move/delete all work for files and dirs

  **QA Scenarios**:

  ```
  Scenario: Create and delete directory
    Tool: Bash
    Preconditions: Test exists
    Steps:
      1. Run `go test ./internal/writer/ -run TestCreateDirectory -v`
      2. Assert PASS
    Expected Result: Directory created and cleaned up
    Evidence: .sisyphus/evidence/task-2-dirops.txt

  Scenario: Path traversal blocked
    Tool: Bash
    Preconditions: Test for path traversal exists
    Steps:
      1. Run `go test ./internal/writer/ -run TestPathTraversal -v`
      2. Assert PASS — operations with `../` are rejected
    Expected Result: Error returned for path traversal attempts
    Evidence: .sisyphus/evidence/task-2-traversal.txt
  ```

  **Commit**: YES
  - Message: `feat(writer): add directory operations module`
  - Files: `internal/writer/dirops.go`, `internal/writer/dirops_test.go`
  - Pre-commit: `go test ./internal/writer/...`

- [x] 3. Insomnia Collection Importer

  **What to do**:
  - Create `internal/importer/insomnia.go` with `ImportInsomnia(path string, opts ImportOptions) error`
  - Parse Insomnia v4 JSON export format (array of resources with `_type` field)
  - Handle resource types: `request`, `request_group` (folders), `environment`
  - Map Insomnia request fields: method, url, headers, body (JSON, form-data, raw), authentication
  - Convert authentication (bearer, basic) to Authorization headers
  - Create folder structure matching Insomnia request groups
  - Convert Insomnia environments to `http-client.env.json` format
  - Handle Insomnia template variables `{{ _.varName }}` → `{{varName}}`
  - Write tests with sample Insomnia export JSON

  **Must NOT do**:
  - Do NOT support Insomnia plugins or custom auth types
  - Do NOT import Insomnia unit tests or test suites

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Requires understanding Insomnia's JSON structure and careful field mapping
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T4, T5, T6)
  - **Blocks**: T10
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/importer/postman.go:1-190` — Complete Postman importer. Follow same structure: parse JSON, walk items, convert to .http files, handle auth, handle env.
  - `internal/importer/postman.go:30-50` — `ImportPostman()` function signature and flow. Mirror for Insomnia.
  - `internal/importer/postman.go:60-80` — `writeItems()` creating folder structure. Reuse pattern.
  - `internal/importer/postman.go:100-130` — `convertAuth()` for bearer/basic. Adapt for Insomnia auth format.

  **Test References**:
  - `internal/importer/postman_test.go` — Test structure for importers. Follow same pattern.

  **External References**:
  - Insomnia v4 export format: resources array with `_type: "request"`, `_type: "request_group"`, `_type: "environment"`

  **WHY Each Reference Matters**:
  - postman.go is the template — same flow (parse JSON → walk tree → write .http files), just different input schema.

  **Acceptance Criteria**:
  - [ ] `go test ./internal/importer/ -run TestInsomnia -v` passes
  - [ ] Produces valid .http files parseable by existing parser
  - [ ] Folder structure matches Insomnia request groups
  - [ ] Environments converted to http-client.env.json

  **QA Scenarios**:

  ```
  Scenario: Import Insomnia collection with nested groups
    Tool: Bash
    Preconditions: Test fixture JSON exists in test
    Steps:
      1. Run `go test ./internal/importer/ -run TestImportInsomnia -v`
      2. Assert PASS
      3. Verify test checks: .http files created, parseable, correct method/URL/headers/body
    Expected Result: All Insomnia requests converted to valid .http format
    Evidence: .sisyphus/evidence/task-3-insomnia.txt

  Scenario: Insomnia environment conversion
    Tool: Bash
    Preconditions: Test fixture includes environment resources
    Steps:
      1. Run `go test ./internal/importer/ -run TestInsomniaEnv -v`
      2. Assert PASS — env vars mapped to http-client.env.json format
    Expected Result: Environment file created with correct variable mapping
    Evidence: .sisyphus/evidence/task-3-insomnia-env.txt
  ```

  **Commit**: YES
  - Message: `feat(importer): add Insomnia collection import`
  - Files: `internal/importer/insomnia.go`, `internal/importer/insomnia_test.go`
  - Pre-commit: `go test ./internal/importer/...`

- [x] 4. Bruno Collection Importer

  **What to do**:
  - Create `internal/importer/bruno.go` with `ImportBruno(dirPath string, opts ImportOptions) error`
  - Parse Bruno's `.bru` file format (custom DSL with sections: `meta`, `get`/`post`/etc., `headers`, `body:json`, `vars`, `auth`)
  - Walk Bruno collection directory structure (each .bru file = one request, `bruno.json` = collection metadata)
  - Convert Bruno's directory hierarchy to restless folder structure with .http files
  - Map Bruno environment files (`environments/*.bru`) to `http-client.env.json`
  - Handle Bruno variables `{{varName}}` (same syntax as restless — pass through)
  - Write a simple .bru parser (line-oriented, section-based — no need for full grammar)
  - Write tests with sample .bru files

  **Must NOT do**:
  - Do NOT support Bruno scripting blocks (`script:pre-request`, `script:post-response`)
  - Do NOT support Bruno assertions or tests blocks
  - Do NOT import Bruno's `docs` blocks

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Requires parsing Bruno's custom .bru format and understanding its directory conventions
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T5, T6)
  - **Blocks**: T10
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/importer/postman.go:30-80` — ImportPostman flow: parse → walk → write. Follow same structure.
  - `internal/parser/lexer.go:1-150` — Lexer pattern for line-oriented parsing. The .bru parser needs a similar approach (read lines, detect section headers, collect content).

  **External References**:
  - Bruno .bru format: sections like `meta { }`, `get { url: ... }`, `headers { }`, `body:json { }`, `auth:bearer { }`, `vars:pre-request { }`

  **WHY Each Reference Matters**:
  - postman.go provides the importer structure. lexer.go shows how to build a simple line parser.

  **Acceptance Criteria**:
  - [ ] `go test ./internal/importer/ -run TestBruno -v` passes
  - [ ] Produces valid .http files from .bru files
  - [ ] Directory structure preserved
  - [ ] Bruno environments converted to http-client.env.json

  **QA Scenarios**:

  ```
  Scenario: Import Bruno collection directory
    Tool: Bash
    Preconditions: Test fixture with .bru files
    Steps:
      1. Run `go test ./internal/importer/ -run TestImportBruno -v`
      2. Assert PASS — .http files created matching .bru requests
    Expected Result: All Bruno requests converted to valid .http format
    Evidence: .sisyphus/evidence/task-4-bruno.txt
  ```

  **Commit**: YES
  - Message: `feat(importer): add Bruno collection import`
  - Files: `internal/importer/bruno.go`, `internal/importer/bruno_test.go`
  - Pre-commit: `go test ./internal/importer/...`

- [x] 5. curl Command Parser and Generator

  **What to do**:
  - Create `internal/importer/curl.go` with:
    - `ParseCurl(command string) (*model.Request, error)` — parse a curl command into a Request
    - `GenerateCurl(req *model.Request) string` — generate curl command from a Request
    - `ImportCurl(command string, opts ImportOptions) error` — parse curl → write .http file
  - Parse curl flags: `-X` (method), `-H` (headers), `-d`/`--data` (body), `-u` (basic auth), `--url`, `-k` (insecure), `-L` (follow redirects), `-b` (cookies), `--compressed`
  - Handle quoted strings and escaped characters in curl commands
  - Generator must produce a single-line curl command suitable for copy-paste
  - Write comprehensive tests for both directions

  **Must NOT do**:
  - Do NOT handle curl file uploads (`-F`, `--form`) in this task — only raw body data
  - Do NOT support curl's full flag set (100+ flags) — cover the common ones listed above

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Bidirectional conversion with tricky string parsing (quoted args, escapes)
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T4, T6)
  - **Blocks**: T10, T11
  - **Blocked By**: None

  **References**:

  **API/Type References**:
  - `internal/model/request.go:1-28` — Request struct that ParseCurl produces and GenerateCurl consumes

  **Pattern References**:
  - `internal/importer/postman.go:77-115` — `convertItem()` builds .http text from request data. GenerateCurl does similar but outputs curl syntax.

  **WHY Each Reference Matters**:
  - model.Request is the intermediate format — curl ↔ Request ↔ .http

  **Acceptance Criteria**:
  - [ ] `go test ./internal/importer/ -run TestCurl -v` passes
  - [ ] ParseCurl handles: GET, POST with body, custom headers, basic auth, multiple -H flags
  - [ ] GenerateCurl produces valid curl commands
  - [ ] Round-trip: GenerateCurl(ParseCurl(cmd)) preserves method, URL, headers, body

  **QA Scenarios**:

  ```
  Scenario: Parse complex curl command
    Tool: Bash
    Preconditions: Tests exist
    Steps:
      1. Run `go test ./internal/importer/ -run TestParseCurl -v`
      2. Assert PASS — handles POST with JSON body, bearer token, multiple headers
    Expected Result: All curl flags correctly mapped to Request fields
    Evidence: .sisyphus/evidence/task-5-curl-parse.txt

  Scenario: Generate curl from Request
    Tool: Bash
    Preconditions: Tests exist
    Steps:
      1. Run `go test ./internal/importer/ -run TestGenerateCurl -v`
      2. Assert PASS — produces valid curl command string
    Expected Result: Generated curl is valid and includes all request details
    Evidence: .sisyphus/evidence/task-5-curl-generate.txt
  ```

  **Commit**: YES
  - Message: `feat(importer): add curl command parser and generator`
  - Files: `internal/importer/curl.go`, `internal/importer/curl_test.go`
  - Pre-commit: `go test ./internal/importer/...`

- [x] 6. OpenAPI/Swagger Importer

  **What to do**:
  - Create `internal/importer/openapi.go` with `ImportOpenAPI(specPath string, opts ImportOptions) error`
  - Support both OpenAPI 3.x (JSON/YAML) and Swagger 2.0 (JSON/YAML)
  - Parse spec manually using `encoding/json` and `gopkg.in/yaml.v3` (both already available) — no heavy OAS library needed
  - For each path + operation: generate a request in .http format
  - Map: servers[0].url as base URL, path parameters as `{{paramName}}`, request body schema as example JSON
  - Group endpoints by tag into separate .http files (one file per tag, or one file per path group)
  - Generate example request bodies from schema (use default values, type-based placeholders)
  - Generate `http-client.env.json` with server URLs as environments (dev/staging/prod if multiple servers)
  - Handle path parameters: `/users/{id}` → `{{baseUrl}}/users/{{id}}`
  - Handle query parameters: add as `?key={{key}}` with defaults where available
  - Write tests with sample OpenAPI 3.0 and Swagger 2.0 specs

  **Must NOT do**:
  - Do NOT add a heavy OpenAPI library dependency (parse manually from JSON/YAML maps)
  - Do NOT generate requests for deprecated endpoints
  - Do NOT handle callback URLs or webhooks

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Most complex importer — needs to handle two spec versions, schema-to-example generation, multiple output files
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with T1, T2, T3, T4, T5)
  - **Blocks**: T10
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/importer/postman.go:30-80` — Importer flow pattern. Follow same structure.
  - `internal/importer/postman.go:133-160` — `ImportPostmanEnv()` for environment file generation. Reuse for server URL → env mapping.

  **External References**:
  - OpenAPI 3.0 spec structure: `openapi`, `info`, `servers`, `paths`, `components/schemas`
  - Swagger 2.0 spec structure: `swagger`, `info`, `host`, `basePath`, `paths`, `definitions`

  **WHY Each Reference Matters**:
  - postman.go importer pattern is the template. The key difference is that OpenAPI is a spec (not a collection of executed requests), so we generate example requests.

  **Acceptance Criteria**:
  - [ ] `go test ./internal/importer/ -run TestOpenAPI -v` passes
  - [ ] Handles OpenAPI 3.0 JSON and YAML
  - [ ] Handles Swagger 2.0 JSON
  - [ ] Produces valid .http files grouped by tag
  - [ ] Generates example request bodies from schemas
  - [ ] Creates http-client.env.json from server URLs

  **QA Scenarios**:

  ```
  Scenario: Import OpenAPI 3.0 spec
    Tool: Bash
    Preconditions: Test fixture with petstore-style OpenAPI 3.0 spec
    Steps:
      1. Run `go test ./internal/importer/ -run TestImportOpenAPI3 -v`
      2. Assert PASS — .http files created per tag, example bodies generated
    Expected Result: Valid .http files with correct endpoints and example bodies
    Evidence: .sisyphus/evidence/task-6-openapi3.txt

  Scenario: Import Swagger 2.0 spec
    Tool: Bash
    Preconditions: Test fixture with Swagger 2.0 spec
    Steps:
      1. Run `go test ./internal/importer/ -run TestImportSwagger2 -v`
      2. Assert PASS — .http files created, host+basePath correctly combined
    Expected Result: Valid .http files from Swagger 2.0 format
    Evidence: .sisyphus/evidence/task-6-swagger2.txt
  ```

  **Commit**: YES
  - Message: `feat(importer): add OpenAPI/Swagger import`
  - Files: `internal/importer/openapi.go`, `internal/importer/openapi_test.go`
  - Pre-commit: `go test ./internal/importer/...`

- [ ] 7. File-Level Request CRUD Operations

  **What to do**:
  - Create `internal/writer/fileops.go` with functions:
    - `InsertRequest(filePath string, req *model.Request) error` — append a new request to end of .http file (with `###` separator)
    - `UpdateRequest(filePath string, oldReq *model.Request, newReq *model.Request) error` — find request by SourceLine, replace its section
    - `DeleteRequest(filePath string, req *model.Request) error` — find request by SourceLine, remove its section (including separator)
    - `DuplicateRequest(srcFile string, req *model.Request, dstFile string) error` — copy request to end of target file
  - Uses the serializer from T1 to generate .http text
  - For update/delete: parse the file, identify the byte range of the target request (using SourceLine), rewrite the file
  - Handle edge cases: single request in file (delete = delete file?), first request (no leading separator), last request (no trailing separator)
  - Write thorough tests — this is the most critical module for data integrity

  **Must NOT do**:
  - Do NOT modify the parser module
  - Do NOT hold file locks — simple read-modify-write is fine for a single-user TUI
  - When deleting the last request in a file, leave the file empty (don't delete the file — let the user do that explicitly)

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: File manipulation with precise byte-range editing. Bugs here corrupt user data.
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T8, T9, T10, T11)
  - **Blocks**: T12, T13, T14
  - **Blocked By**: T1

  **References**:

  **Pattern References**:
  - `internal/writer/serializer.go` (T1 output) — Used to generate .http text for insert/update/duplicate
  - `internal/parser/parser.go:40-70` — `parseTokens()` shows how requests are identified by token boundaries. FileOps needs to map these back to line ranges.
  - `internal/parser/lexer.go:85-155` — `Tokenize()` produces tokens with `Line` numbers. These line numbers are the key to locating requests in files.

  **API/Type References**:
  - `internal/model/request.go:8-10` — `SourceFile` and `SourceLine` fields identify where a request lives in a file

  **WHY Each Reference Matters**:
  - The serializer produces text; fileops handles the file I/O and section replacement.
  - SourceLine is the anchor — fileops uses it to find the start of a request, then scans forward to find the end (next `###` or EOF).

  **Acceptance Criteria**:
  - [ ] `go test ./internal/writer/ -run TestFileOps -v` passes with ≥12 test cases
  - [ ] Insert appends correctly with separator
  - [ ] Update replaces exactly one request, preserving others
  - [ ] Delete removes request and its separator cleanly
  - [ ] Duplicate copies to same or different file
  - [ ] Edge case: operations on single-request files work correctly

  **QA Scenarios**:

  ```
  Scenario: Insert, update, and delete round-trip
    Tool: Bash
    Preconditions: Tests exist
    Steps:
      1. Run `go test ./internal/writer/ -run TestFileOps -v`
      2. Assert all sub-tests pass: TestInsert, TestUpdate, TestDelete, TestDuplicate
    Expected Result: All CRUD operations produce valid .http files
    Evidence: .sisyphus/evidence/task-7-fileops.txt

  Scenario: Update preserves other requests in file
    Tool: Bash
    Preconditions: Test creates a 3-request file, updates the middle one
    Steps:
      1. Run `go test ./internal/writer/ -run TestUpdateMiddleRequest -v`
      2. Assert first and third requests unchanged, middle request updated
    Expected Result: Only target request modified
    Evidence: .sisyphus/evidence/task-7-update-preserve.txt
  ```

  **Commit**: YES
  - Message: `feat(writer): add file-level request CRUD operations`
  - Files: `internal/writer/fileops.go`, `internal/writer/fileops_test.go`
  - Pre-commit: `go test ./internal/writer/...`

- [ ] 8. TUI Request Editor Form Component

  **What to do**:
  - Create `internal/tui/editor.go` with `EditorModel` bubbletea component
  - Full-screen overlay form with fields:
    - **Method selector**: cycle through GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS with left/right arrows
    - **URL input**: single-line text input with `bubbles/v2/textinput`
    - **Request name**: optional single-line text input
    - **Headers editor**: dynamic list of key-value pairs. Tab between key/value. Enter to add new row. Ctrl+D to delete row.
    - **Body editor**: multi-line text area with `bubbles/v2/textarea`
    - **Metadata toggles**: checkboxes for @no-redirect, @no-cookie-jar. Number inputs for @timeout, @connection-timeout
  - Navigation: Tab/Shift+Tab to move between fields. Ctrl+S to save. Esc to cancel.
  - Two modes: "Create" (empty form) and "Edit" (pre-populated from existing Request)
  - Expose `EditorModel.Request() *model.Request` to extract the form data as a Request
  - Emit messages: `EditorSaved{Request}` and `EditorCancelled{}`
  - Style consistently with existing TUI (use styles from `styles.go`)

  **Must NOT do**:
  - Do NOT do any file I/O in the editor component — it only manages form state
  - Do NOT add request execution from the editor — save first, then send from detail view
  - Do NOT build a full text editor — body is a simple textarea, not vim

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex multi-field TUI component with navigation, two modes, and bubbletea message passing
  - **Skills**: [`playwright`]
    - `playwright`: For visual verification of TUI rendering in terminal

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T7, T9, T10, T11)
  - **Blocks**: T12
  - **Blocked By**: T1 (needs to know Request struct for form ↔ model mapping)

  **References**:

  **Pattern References**:
  - `internal/tui/search.go:1-120` — SearchModel is the closest existing overlay component. Follow its pattern: separate model, Init/Update/View, message types.
  - `internal/tui/envswitch.go:1-100` — Another overlay with cursor navigation. Shows how overlays integrate with app.go.
  - `internal/tui/detail.go:1-50` — DetailModel shows how to hold and display a Request. Editor is the writable counterpart.
  - `internal/tui/styles.go` — Existing style definitions. Use these for consistency.

  **API/Type References**:
  - `internal/model/request.go:1-28` — All fields the editor must expose
  - `charm.land/bubbles/v2` — `textinput` and `textarea` components (already indirect dep in go.mod)

  **WHY Each Reference Matters**:
  - search.go and envswitch.go are the overlay patterns to follow — the editor is a more complex version of the same idea.
  - bubbles/v2 provides the input widgets so we don't build from scratch.

  **Acceptance Criteria**:
  - [ ] EditorModel compiles and integrates with bubbletea
  - [ ] Create mode: all fields start empty, method defaults to GET
  - [ ] Edit mode: pre-populates all fields from existing Request
  - [ ] Tab navigation moves between fields correctly
  - [ ] Headers: can add, edit, and remove header rows
  - [ ] Ctrl+S emits EditorSaved with populated Request
  - [ ] Esc emits EditorCancelled

  **QA Scenarios**:

  ```
  Scenario: Create new request via editor form
    Tool: interactive_bash (tmux)
    Preconditions: TUI wired with editor (after T12), test .http collection exists
    Steps:
      1. Launch `restless .` in tmux
      2. Press `n` (new request keybinding)
      3. Verify editor overlay appears with empty fields
      4. Type URL: `https://httpbin.org/get`
      5. Tab to headers, type `Accept`, tab, type `application/json`
      6. Press Ctrl+S
      7. Verify request appears in browser and .http file on disk
    Expected Result: New request created and persisted
    Evidence: .sisyphus/evidence/task-8-editor-create.txt

  Scenario: Edit existing request
    Tool: interactive_bash (tmux)
    Preconditions: TUI running with existing requests
    Steps:
      1. Select a request in browser, press `E` (edit keybinding)
      2. Verify editor opens with pre-populated fields
      3. Change the URL
      4. Press Ctrl+S
      5. Verify changes persisted to .http file
    Expected Result: Request updated in file
    Evidence: .sisyphus/evidence/task-8-editor-edit.txt
  ```

  **Commit**: YES
  - Message: `feat(tui): add request editor form component`
  - Files: `internal/tui/editor.go`

- [ ] 9. TUI Confirmation Dialog Component

  **What to do**:
  - Create `internal/tui/confirm.go` with `ConfirmModel` bubbletea component
  - Simple overlay: message text + [Yes] / [No] buttons
  - Left/Right arrows to select, Enter to confirm
  - Emit `ConfirmResult{Confirmed bool, Context interface{}}` message
  - Reusable for: delete request, delete file, delete folder confirmations
  - Style: centered overlay with border, matching existing theme

  **Must NOT do**:
  - Do NOT add complex dialog features (input fields, dropdowns). Keep it minimal.

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple UI component, minimal logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T7, T8, T10, T11)
  - **Blocks**: T13
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - `internal/tui/envswitch.go:40-75` — Simple overlay with cursor + Enter. Confirmation dialog is even simpler.
  - `internal/tui/styles.go` — Style consistency

  **Acceptance Criteria**:
  - [ ] ConfirmModel compiles
  - [ ] Shows message with Yes/No options
  - [ ] Enter emits correct ConfirmResult

  **QA Scenarios**:

  ```
  Scenario: Confirmation dialog works
    Tool: interactive_bash (tmux)
    Preconditions: TUI wired with confirm dialog (after T13)
    Steps:
      1. Launch TUI, select a request, press `D` (delete)
      2. Verify confirmation dialog appears: "Delete this request?"
      3. Press right arrow to select "No", press Enter
      4. Verify request still exists
      5. Press `D` again, select "Yes", press Enter
      6. Verify request removed
    Expected Result: Dialog controls delete behavior correctly
    Evidence: .sisyphus/evidence/task-9-confirm.txt
  ```

  **Commit**: YES
  - Message: `feat(tui): add confirmation dialog component`
  - Files: `internal/tui/confirm.go`

- [ ] 10. Import CLI Commands

  **What to do**:
  - Update `cmd/restless/main.go` to add 3 new subcommands under `import`:
    - `restless import insomnia <file.json> [--output dir] [--env file]`
    - `restless import bruno <directory> [--output dir]`
    - `restless import curl "<curl command>" [--output dir]`
    - `restless import openapi <spec.json|spec.yaml> [--output dir]`
  - Each command calls the corresponding importer from `internal/importer/`
  - Print summary after import: number of requests imported, files created, env file created
  - Follow the existing `postman` subcommand pattern exactly

  **Must NOT do**:
  - Do NOT modify the existing `postman` subcommand
  - Do NOT add interactive prompts — CLI only

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Wiring 4 importers with cobra commands, straightforward but needs care with flags
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T7, T8, T9, T11)
  - **Blocks**: T17
  - **Blocked By**: T3, T4, T5, T6

  **References**:

  **Pattern References**:
  - `cmd/restless/main.go:47-80` — Existing `postman` subcommand. Replicate this exact pattern for each new importer.

  **API/Type References**:
  - `internal/importer/insomnia.go` (T3) — `ImportInsomnia(path, opts)`
  - `internal/importer/bruno.go` (T4) — `ImportBruno(dirPath, opts)`
  - `internal/importer/curl.go` (T5) — `ImportCurl(command, opts)`
  - `internal/importer/openapi.go` (T6) — `ImportOpenAPI(specPath, opts)`

  **Acceptance Criteria**:
  - [ ] `restless import --help` lists all 5 importers (postman, insomnia, bruno, curl, openapi)
  - [ ] Each subcommand has `--output` flag
  - [ ] Each subcommand prints success summary

  **QA Scenarios**:

  ```
  Scenario: All import subcommands registered
    Tool: Bash
    Preconditions: Binary built
    Steps:
      1. Run `go build -o /tmp/restless ./cmd/restless/`
      2. Run `/tmp/restless import --help`
      3. Assert output contains: postman, insomnia, bruno, curl, openapi
    Expected Result: All 5 import subcommands visible in help
    Evidence: .sisyphus/evidence/task-10-import-help.txt
  ```

  **Commit**: YES
  - Message: `feat(cli): add import commands for insomnia, bruno, curl, openapi`
  - Files: `cmd/restless/main.go`
  - Pre-commit: `go build ./cmd/restless/`

- [ ] 11. curl Export and Copy-to-Clipboard

  **What to do**:
  - Create `internal/exporter/curl.go` with:
    - `ToCurl(req *model.Request) string` — convert resolved Request to curl command
    - `CopyToClipboard(text string) error` — copy string to system clipboard
  - curl output format: `curl -X METHOD 'URL' -H 'Key: Value' -d 'body'`
  - Properly escape single quotes in values
  - Clipboard: use `pbcopy` on macOS, `xclip -selection clipboard` on Linux, `clip.exe` on Windows (shell out via `exec.Command`)
  - Write tests for ToCurl (clipboard is platform-specific, skip in CI)

  **Must NOT do**:
  - Do NOT add CGo clipboard library — shell out to system tools
  - Do NOT fail if clipboard tool is unavailable — just return an error

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple string formatting + platform exec
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with T7, T8, T9, T10)
  - **Blocks**: T15
  - **Blocked By**: T5 (curl format knowledge, but can use model.Request directly)

  **References**:

  **Pattern References**:
  - `internal/importer/curl.go` (T5) — `GenerateCurl()` already does Request → curl. ToCurl may be a thin wrapper or import from there.

  **API/Type References**:
  - `internal/model/request.go:1-28` — Request struct input

  **WHY Each Reference Matters**:
  - T5's GenerateCurl does the heavy lifting. ToCurl may just call it + add clipboard.

  **Acceptance Criteria**:
  - [ ] `go test ./internal/exporter/ -v` passes
  - [ ] ToCurl produces valid curl command for GET, POST with body, custom headers
  - [ ] CopyToClipboard works on macOS (dev platform)

  **QA Scenarios**:

  ```
  Scenario: Generate curl from request
    Tool: Bash
    Preconditions: Tests exist
    Steps:
      1. Run `go test ./internal/exporter/ -run TestToCurl -v`
      2. Assert PASS
    Expected Result: Valid curl commands generated
    Evidence: .sisyphus/evidence/task-11-curl-export.txt
  ```

  **Commit**: YES
  - Message: `feat(exporter): add curl export and copy-to-clipboard`
  - Files: `internal/exporter/curl.go`, `internal/exporter/curl_test.go`
  - Pre-commit: `go test ./internal/exporter/...`

- [ ] 12. Wire Create/Edit Request Flows into TUI App

  **What to do**:
  - Update `internal/tui/app.go` to integrate the EditorModel (T8) with file operations (T7)
  - **Create flow**: keybinding `n` → open EditorModel in create mode → on EditorSaved, call `writer.InsertRequest()` → reload collection → select new request
  - **Edit flow**: keybinding `E` (shift+e) on selected request → open EditorModel in edit mode, pre-populated → on EditorSaved, call `writer.UpdateRequest()` → reload collection
  - Add `showEditor bool` and `editor EditorModel` fields to App struct
  - Handle EditorSaved and EditorCancelled messages in App.Update()
  - After save: re-run `LoadCollection()` to refresh the browser, re-select the affected request
  - **Create file prompt**: If creating a request and no .http file exists in current directory, create a default `requests.http` file

  **Must NOT do**:
  - Do NOT auto-send requests after creation
  - Do NOT modify EditorModel itself — only wire it into the app

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex state management — editor overlay, file operations, collection reload, re-selection
  - **Skills**: [`playwright`]
    - `playwright`: For visual TUI verification

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T13, T14, T15)
  - **Blocks**: T16
  - **Blocked By**: T7, T8

  **References**:

  **Pattern References**:
  - `internal/tui/app.go:80-105` — How search overlay is shown/hidden. Editor follows same pattern.
  - `internal/tui/app.go:55-70` — `collectionLoaded` handler shows how to refresh browser after file changes.
  - `internal/tui/app.go:107-120` — `SearchSelected` message handling. `EditorSaved` follows same pattern.

  **API/Type References**:
  - `internal/tui/editor.go` (T8) — `EditorModel`, `EditorSaved`, `EditorCancelled` messages
  - `internal/writer/fileops.go` (T7) — `InsertRequest()`, `UpdateRequest()`

  **WHY Each Reference Matters**:
  - The search overlay pattern in app.go is the exact template for how to integrate the editor overlay.

  **Acceptance Criteria**:
  - [ ] `n` opens create editor, `E` opens edit editor
  - [ ] Saving from create mode writes new request to file
  - [ ] Saving from edit mode updates existing request in file
  - [ ] Browser refreshes after save
  - [ ] Esc cancels without changes

  **QA Scenarios**:

  ```
  Scenario: Full create-edit cycle
    Tool: interactive_bash (tmux)
    Preconditions: restless built and running in test directory
    Steps:
      1. Launch `restless /tmp/test-crud`
      2. Press `n` to create new request
      3. Set method to POST, URL to `https://httpbin.org/post`, add header
      4. Press Ctrl+S
      5. Verify request appears in browser
      6. Verify .http file on disk contains the request
      7. Select the request, press `E`
      8. Change URL to `https://httpbin.org/put`, method to PUT
      9. Press Ctrl+S
      10. Verify changes reflected in browser and file
    Expected Result: Create and edit both persist correctly
    Evidence: .sisyphus/evidence/task-12-create-edit.txt
  ```

  **Commit**: YES
  - Message: `feat(tui): wire create/edit request flows`
  - Files: `internal/tui/app.go`, `internal/tui/detail.go`

- [ ] 13. Wire Collection Management into Browser

  **What to do**:
  - Update `internal/tui/browser.go` and `internal/tui/app.go` to add collection management keybindings:
    - `N` (shift+n): Create new .http file — prompt for filename → `writer.CreateHTTPFile()`
    - `F` (shift+f): Create new folder — prompt for name → `writer.CreateDirectory()`
    - `R`: Rename selected file/folder — prompt for new name → `writer.RenameEntry()`
    - `D`: Delete selected item — show ConfirmModel (T9) → `writer.DeleteRequest()` (for requests) or `writer.DeleteEntry()` (for files/folders)
    - `Y`: Duplicate selected request — `writer.DuplicateRequest()`
    - `M`: Move selected file/folder — prompt for destination path → `writer.MoveEntry()`
  - Add a simple text input prompt overlay for filename/path entry (reuse textinput from bubbles)
  - After each operation: reload collection to refresh browser
  - Handle errors: show error message briefly in status bar

  **Must NOT do**:
  - Do NOT allow deleting non-empty folders without confirmation
  - Do NOT allow operations outside the collection root directory

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Many keybindings, multiple operation types, overlay prompts, error handling
  - **Skills**: [`playwright`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T12, T14, T15)
  - **Blocks**: T16
  - **Blocked By**: T2, T7, T9

  **References**:

  **Pattern References**:
  - `internal/tui/browser.go:150-200` — Existing keybinding handling in browser. Add new keybindings here.
  - `internal/tui/app.go:130-165` — Key event routing between overlays and panes. Collection management keys route through here.
  - `internal/tui/search.go:60-85` — Text input handling. Reuse for filename prompts.

  **API/Type References**:
  - `internal/writer/dirops.go` (T2) — `CreateDirectory()`, `CreateHTTPFile()`, `RenameEntry()`, `MoveEntry()`, `DeleteEntry()`
  - `internal/writer/fileops.go` (T7) — `DeleteRequest()`, `DuplicateRequest()`
  - `internal/tui/confirm.go` (T9) — `ConfirmModel` for delete confirmations

  **Acceptance Criteria**:
  - [ ] All 6 keybindings functional: N, F, R, D, Y, M
  - [ ] File/folder creation works with name prompt
  - [ ] Delete shows confirmation dialog
  - [ ] Duplicate creates copy in same file
  - [ ] Browser refreshes after each operation

  **QA Scenarios**:

  ```
  Scenario: Create folder, create file, delete both
    Tool: interactive_bash (tmux)
    Preconditions: restless running in test directory
    Steps:
      1. Press `F` to create folder, type "api", Enter
      2. Verify folder appears in browser
      3. Navigate into folder, press `N` to create file, type "users.http", Enter
      4. Verify file appears
      5. Select file, press `D`, confirm Yes
      6. Verify file removed
      7. Navigate to folder, press `D`, confirm Yes
      8. Verify folder removed
    Expected Result: All collection management operations work
    Evidence: .sisyphus/evidence/task-13-collection-mgmt.txt

  Scenario: Duplicate request
    Tool: interactive_bash (tmux)
    Steps:
      1. Select a request, press `Y`
      2. Verify duplicated request appears in browser
      3. Verify .http file contains both original and copy
    Expected Result: Request duplicated in file
    Evidence: .sisyphus/evidence/task-13-duplicate.txt
  ```

  **Commit**: YES
  - Message: `feat(tui): wire collection management into browser`
  - Files: `internal/tui/browser.go`, `internal/tui/app.go`

- [ ] 14. $EDITOR Integration

  **What to do**:
  - Create `internal/tui/editor_external.go` with function to open a file in the user's editor
  - Detect editor: `$VISUAL` → `$EDITOR` → `vim` fallback
  - Keybinding `O` (shift+o) on selected request: open its `SourceFile` in $EDITOR at `SourceLine`
  - Use `tea.ExecProcess()` from bubbletea v2 to suspend TUI, run editor, resume TUI
  - Pass line number to editor: `vim +LINE file`, `nano +LINE file`, `code -g file:LINE` (handle common editors)
  - After editor returns: reload collection, re-select the request that was being edited
  - Handle case where editor is not found: show error in status bar

  **Must NOT do**:
  - Do NOT implement an in-TUI text editor
  - Do NOT watch the file for changes — just reload when editor exits

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
    - Reason: Process suspension/resumption with bubbletea, editor detection
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T12, T13, T15)
  - **Blocks**: T16
  - **Blocked By**: T7

  **References**:

  **Pattern References**:
  - `internal/tui/app.go:55-70` — `collectionLoaded` handler for reloading after changes
  - bubbletea v2 `tea.ExecProcess()` API — suspends TUI, runs external process, resumes

  **External References**:
  - bubbletea exec: `tea.ExecProcess(exec.Command(editor, args...), func(err error) tea.Msg { ... })`

  **WHY Each Reference Matters**:
  - `tea.ExecProcess` is the key API — it handles terminal state save/restore automatically.

  **Acceptance Criteria**:
  - [ ] `O` opens the .http file in $EDITOR at correct line
  - [ ] TUI suspends cleanly and resumes after editor exits
  - [ ] Collection reloads after editor closes
  - [ ] Works with vim, nano, and $VISUAL override

  **QA Scenarios**:

  ```
  Scenario: Open request in $EDITOR
    Tool: interactive_bash (tmux)
    Preconditions: EDITOR=nano set in environment
    Steps:
      1. Launch restless, select a request
      2. Press `O`
      3. Verify nano opens the .http file
      4. Make a small change (add a comment), save, exit nano
      5. Verify TUI resumes and shows updated content
    Expected Result: Editor opens, changes persist, TUI resumes
    Evidence: .sisyphus/evidence/task-14-editor.txt
  ```

  **Commit**: YES
  - Message: `feat(tui): add $EDITOR integration`
  - Files: `internal/tui/editor_external.go`, `internal/tui/app.go`

- [ ] 15. Wire curl Export into TUI

  **What to do**:
  - Add keybinding `c` in detail view: copy current request as curl to clipboard
  - Resolve variables first (using current env vars + chain context) before generating curl
  - Show brief status message: "Copied as curl" in status bar for 2 seconds
  - Use `exporter.ToCurl()` from T11 and `exporter.CopyToClipboard()` for clipboard

  **Must NOT do**:
  - Do NOT add a separate pane or overlay — just copy and show status message

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple wiring — one keybinding, two function calls, status message
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with T12, T13, T14)
  - **Blocks**: T16
  - **Blocked By**: T11

  **References**:

  **Pattern References**:
  - `internal/tui/detail.go:90-110` — Existing keybinding handling in detail view. Add `c` here.
  - `internal/tui/detail.go:100-110` — Request resolution flow (envVars, chainCtx). Reuse for curl generation.

  **API/Type References**:
  - `internal/exporter/curl.go` (T11) — `ToCurl()`, `CopyToClipboard()`

  **Acceptance Criteria**:
  - [ ] `c` in detail view copies curl to clipboard
  - [ ] Status bar shows "Copied as curl" briefly
  - [ ] Variables are resolved before generating curl

  **QA Scenarios**:

  ```
  Scenario: Copy request as curl
    Tool: interactive_bash (tmux)
    Preconditions: restless running with a request selected
    Steps:
      1. Select a request, press Enter to view details
      2. Press `c`
      3. Verify status bar shows "Copied as curl"
      4. Run `pbpaste` in another tmux pane
      5. Assert clipboard contains valid curl command starting with `curl`
    Expected Result: Valid curl command in clipboard
    Evidence: .sisyphus/evidence/task-15-copy-curl.txt
  ```

  **Commit**: YES
  - Message: `feat(tui): wire curl export with keybinding`
  - Files: `internal/tui/detail.go`, `internal/tui/app.go`

- [ ] 16. Update Status Bar, Keybindings, and Help Text

  **What to do**:
  - Update status bar in `internal/tui/app.go` to show new keybindings contextually:
    - **Browser pane focused**: `n:new │ E:edit │ D:delete │ Y:dup │ N:new file │ F:new folder │ R:rename │ O:editor`
    - **Detail pane focused**: `Enter:send │ c:curl │ h:history │ 1-3:tabs`
    - **Editor overlay**: `Ctrl+S:save │ Esc:cancel │ Tab:next field`
  - Status bar should be contextual — show only relevant bindings for current state
  - Add a temporary message system: show "Copied as curl", "Request saved", "Request deleted" etc. for 2 seconds then revert to default status bar
  - Update README.md keyboard shortcuts table with all new keybindings

  **Must NOT do**:
  - Do NOT add a help overlay/modal — just update status bar
  - Do NOT change existing keybindings — only add new ones

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Status bar text updates, simple timer for temp messages
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with T17)
  - **Blocks**: T17
  - **Blocked By**: T12, T13, T14, T15

  **References**:

  **Pattern References**:
  - `internal/tui/app.go:165-175` — Current status bar rendering. Extend this.
  - `README.md:60-75` — Keyboard shortcuts table to update.

  **Acceptance Criteria**:
  - [ ] Status bar shows context-appropriate keybindings
  - [ ] Temporary messages display and auto-clear
  - [ ] README updated with all new keybindings

  **QA Scenarios**:

  ```
  Scenario: Context-sensitive status bar
    Tool: interactive_bash (tmux)
    Steps:
      1. Launch restless, focus browser pane
      2. Verify status bar shows browser keybindings (n, E, D, etc.)
      3. Press Tab to switch to detail pane
      4. Verify status bar shows detail keybindings (c, h, etc.)
    Expected Result: Status bar changes based on focused pane
    Evidence: .sisyphus/evidence/task-16-statusbar.txt
  ```

  **Commit**: YES
  - Message: `feat(tui): update status bar and keybindings`
  - Files: `internal/tui/app.go`, `internal/tui/styles.go`, `README.md`

- [ ] 17. Integration Tests

  **What to do**:
  - Add integration tests in `tests/integration_test.go` covering:
    - **Serializer round-trip**: parse real .http files → serialize → parse → compare
    - **Import pipeline**: run each importer on sample data → verify output parses correctly
    - **File CRUD**: create request → update → delete → verify file state at each step
    - **curl round-trip**: Request → curl → parse curl → compare Request
  - Use real .http files from the existing test fixtures or create minimal ones
  - Each test should be self-contained with temp directories

  **Must NOT do**:
  - Do NOT test TUI interactions in integration tests — those are covered by QA scenarios
  - Do NOT add external service dependencies — all tests must work offline

  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Cross-module testing requiring understanding of all new modules
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with T16)
  - **Blocks**: None
  - **Blocked By**: T16 (all features must be complete)

  **References**:

  **Pattern References**:
  - `tests/integration_test.go` — Existing integration test file. Add to it.

  **Acceptance Criteria**:
  - [ ] `go test ./tests/ -v` passes all new integration tests
  - [ ] ≥8 test functions covering all new modules
  - [ ] All tests work offline with no network calls

  **QA Scenarios**:

  ```
  Scenario: Full integration test suite passes
    Tool: Bash
    Steps:
      1. Run `go test ./... -count=1 -v`
      2. Assert exit code 0
      3. Assert all new test functions pass
    Expected Result: All tests pass
    Evidence: .sisyphus/evidence/task-17-integration.txt
  ```

  **Commit**: YES
  - Message: `test: add integration tests for CRUD and import flows`
  - Files: `tests/integration_test.go`
  - Pre-commit: `go test ./...`

---

## Final Verification Wave

> After ALL tasks complete, run these 4 verification agents in PARALLEL.
> ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet ./...` + `go test ./...`. Review all changed/new files for: unchecked errors, unused imports, dead code, empty catches. Check for AI slop: excessive comments, over-abstraction, generic variable names. Verify no CGo dependencies introduced.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Launch TUI, create a new request, edit it, send it, delete it. Create a folder, create a file in it. Import an Insomnia collection, import an OpenAPI spec. Copy a request as curl. Open $EDITOR on a request. Verify all flows end-to-end. Save evidence screenshots/terminal output.
  Output: `CRUD [N/N pass] | Collection Mgmt [N/N] | Import [N/N] | Curl [PASS/FAIL] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual implementation. Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance. Detect scope creep (WebSocket, GraphQL, scripting, auth, CGo). Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Scope [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Task | Commit Message | Files |
|------|---------------|-------|
| T1 | `feat(writer): add request serializer for .http format` | `internal/writer/serializer.go`, `internal/writer/serializer_test.go` |
| T2 | `feat(writer): add directory operations module` | `internal/writer/dirops.go`, `internal/writer/dirops_test.go` |
| T3 | `feat(importer): add Insomnia collection import` | `internal/importer/insomnia.go`, `internal/importer/insomnia_test.go` |
| T4 | `feat(importer): add Bruno collection import` | `internal/importer/bruno.go`, `internal/importer/bruno_test.go` |
| T5 | `feat(importer): add curl command parser and generator` | `internal/importer/curl.go`, `internal/importer/curl_test.go` |
| T6 | `feat(importer): add OpenAPI/Swagger import` | `internal/importer/openapi.go`, `internal/importer/openapi_test.go` |
| T7 | `feat(writer): add file-level request CRUD operations` | `internal/writer/fileops.go`, `internal/writer/fileops_test.go` |
| T8 | `feat(tui): add request editor form component` | `internal/tui/editor.go` |
| T9 | `feat(tui): add confirmation dialog component` | `internal/tui/confirm.go` |
| T10 | `feat(cli): add import commands for insomnia, bruno, curl, openapi` | `cmd/restless/main.go` |
| T11 | `feat(exporter): add curl export and copy-to-clipboard` | `internal/exporter/curl.go`, `internal/exporter/curl_test.go` |
| T12 | `feat(tui): wire create/edit request flows` | `internal/tui/app.go`, `internal/tui/detail.go` |
| T13 | `feat(tui): wire collection management into browser` | `internal/tui/browser.go`, `internal/tui/app.go` |
| T14 | `feat(tui): add $EDITOR integration` | `internal/tui/editor_external.go`, `internal/tui/app.go` |
| T15 | `feat(tui): wire curl export with keybinding` | `internal/tui/detail.go`, `internal/tui/app.go` |
| T16 | `feat(tui): update status bar and keybindings` | `internal/tui/app.go`, `internal/tui/styles.go` |
| T17 | `test: add integration tests for CRUD and import flows` | `tests/integration_test.go` |

---

## Success Criteria

### Verification Commands
```bash
go test ./... -count=1                          # Expected: all pass
go vet ./...                                    # Expected: clean
restless import insomnia test.json --output /tmp/test  # Expected: .http files created
restless import openapi spec.yaml --output /tmp/test   # Expected: .http files created
restless import curl "curl -X GET https://example.com" --output /tmp/test  # Expected: .http file
```

### Final Checklist
- [ ] All "Must Have" features present and working
- [ ] All "Must NOT Have" items absent from codebase
- [ ] All existing tests still pass
- [ ] No CGo dependencies introduced
- [ ] Headless `run` command still works
- [ ] All 4 importers produce valid .http files parseable by existing parser
