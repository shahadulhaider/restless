# restless — environment switching
#
# The request uses {{region}}, {{tier}} and {{apiKey}}, which are only defined
# per-environment in restless.env.json. Ctrl+E picks an environment, then the
# same request resolves and runs.
#
# Prerequisites (see docs/demo/README.md):
#   go build -o /tmp/restless ./cmd/restless
#   go run docs/demo/stub/server.go &
# Run from the repository root:
#   vhs docs/demo/envs.tape

Output "/tmp/restless-demo/envs.gif"

Set Shell "bash"
Set Theme "Catppuccin Mocha"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 16
Set Width 1000
Set Height 600
Set Padding 20
Set TypingSpeed 90ms

# Open regions.http -> activeSession off-camera.
Hide
Type `export PS1="❯ " && export PATH=/tmp:$PATH && cd docs/demo/collection && clear`
Enter
Sleep 1s
Type "restless ."
Enter
Sleep 2500ms
Down
Enter
Sleep 800ms
Down
Enter
Sleep 1200ms
Show

# The request is full of {{...}} placeholders and the status bar reads env:none.
Sleep 1800ms

# Pick an environment.
Ctrl+E
Sleep 1600ms
Down
Sleep 550ms
Down
Sleep 1000ms
Enter
Sleep 1700ms

# Send it. The stub echoes back whatever the environment resolved to.
Tab
Sleep 600ms
Enter
Sleep 2600ms
