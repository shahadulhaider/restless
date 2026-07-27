# Demo recordings

The `.tape` files here are [VHS](https://github.com/charmbracelet/vhs) scripts that
record the GIFs used in the project README.

The GIFs themselves are **not** committed — they are written to `/tmp/restless-demo/`.
Only the tapes and the demo collection live in git.

| Tape | Size | Shows |
|------|------|-------|
| `hero.tape` | 1200×750 | Launch, browse the collection, send a request, fold JSON, Headers tab |
| `folding.tape` | 1000×600 | `za` / `zM` / `zR` folding and the `1`/`2`/`3` tabs |
| `yank.tape` | 1000×600 | `v` / `V` visual selection, `y`, plus `yb` and `yc` |
| `codegen.tape` | 1000×600 | The `yg` which-key popup, then `ygp` and `ygj` |
| `envs.tape` | 1000×600 | `Ctrl+E` environment switching, with a variable resolving |

## Requirements

```bash
brew install vhs gifsicle          # ffmpeg comes with vhs
```

A monospace Nerd Font is required for the file-tree glyphs and the `▾`/`▸` fold
markers. The tapes use `JetBrainsMono Nerd Font Mono` with the `Catppuccin Mocha`
theme, which matches the palette in `internal/tui/styles.go`.

## Recording

Everything below runs from the **repository root**.

```bash
# 1. Build the binary the tapes invoke (they prepend /tmp to $PATH).
go build -o /tmp/restless ./cmd/restless

# 2. Start the local stub API. Recordings never touch the public network.
go run docs/demo/stub/server.go &

# 3. Make sure the output directory exists.
mkdir -p /tmp/restless-demo

# 4. Record.
vhs docs/demo/hero.tape
vhs docs/demo/folding.tape
vhs docs/demo/yank.tape
vhs docs/demo/codegen.tape
vhs docs/demo/envs.tape

# 5. Shrink.
for g in hero folding yank codegen envs; do
  gifsicle -O3 "/tmp/restless-demo/$g.gif" -o "/tmp/restless-demo/$g.opt.gif"
  mv "/tmp/restless-demo/$g.opt.gif" "/tmp/restless-demo/$g.gif"
done

# 6. Stop the stub.
kill %1
```

## The stub server

`stub/server.go` carries a `//go:build ignore` tag, so it is invisible to
`go build ./...`, `go vet ./...` and `go test ./...`; run it with `go run`.

It listens on `:4010` (override with `-addr`) and serves fixed, hand-written JSON
so that every recording renders identical pixels:

| Route | Returns |
|-------|---------|
| `GET /health` | Service health with nested per-dependency checks |
| `POST /auth/login` | A token plus a nested account/plan object |
| `GET /users` | Paginated users with nested `profile`, `links` and `teams` |
| `GET /users/{id}` | One deeply nested user |
| `POST /users` | `201` with a nested invitation object |
| `GET /session` | Echoes the `region`, `tier` and API key it received |

`/session` is the one dynamic route: it reflects back whatever the request sent,
which is what lets `envs.tape` show an environment variable actually resolving.

## The demo collection

`collection/` is a small but realistic collection, and doubles as a set of
examples. Every request asserts on its response, so it is also a quick smoke
test of the stub:

```bash
cd docs/demo/collection
restless run auth.http
restless run users.http
restless run regions.http --env staging
```

`restless.env.json` defines `local` and `staging`. Both inherit
`baseUrl = http://localhost:4010` from `$shared` — deliberately, so no recording
can ever reach the public internet — and differ in `region`, `tier` and `apiKey`.

## Notes when editing tapes

- **Never split prefix keystrokes.** Write `Type@15ms "za"`, not `Type "z"` +
  `Sleep` + `Type "a"`. The app draws its which-key popup the instant it sees the
  prefix, and that popup *replaces* the detail pane, so any gap between the two
  keys shows up as a blank pane. At the default 90 ms `TypingSpeed` even
  `Type "za"` leaves a visible flash; `Type@15ms` collapses it to a single 0.04 s
  frame. The same applies to `zM`, `zR`, `yb` and `yc`.
- **`codegen.tape` deliberately does the opposite**, splitting `y` → `g` → `p`
  with long sleeps, because there the popup is the thing being demonstrated.
- **Nothing is expanded on launch.** The browser opens on three collapsed files.
  `Enter` toggles a file open or closed, and only one file stays expanded at a
  time. The cursor keeps its row index when a file expands, so the first request
  in a file sits one `Down` below it.
- **`Enter` in the browser selects; `Enter` in the detail pane sends.** Use `Tab`
  to move focus between them.
- **The request view shows the raw `{{...}}` source**, never the resolved value.
  To show a variable resolving you have to send the request and read the
  response, which is why `envs.tape` targets `/session`.
- **Long lines wrap.** At 1000×600 the detail pane fits roughly 60 characters, so
  keep stub JSON values short or they fold onto the next line.
- **The environment list is unordered.** `envswitch.go` iterates a Go map, so
  `local` and `staging` swap places between runs. `envs.tape` works either way.
