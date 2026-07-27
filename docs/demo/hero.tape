# restless — hero demo
#
# Launch the TUI on a collection, walk the tree, send a request, then explore
# the response: fold two JSON nodes, scroll, and jump to the Headers tab.
#
# Prerequisites (see docs/demo/README.md):
#   go build -o /tmp/restless ./cmd/restless
#   go run docs/demo/stub/server.go &
# Run from the repository root:
#   vhs docs/demo/hero.tape

Output "/tmp/restless-demo/hero.gif"

Set Shell "bash"
Set Theme "Catppuccin Mocha"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 17
Set Width 1200
Set Height 750
Set Padding 20
Set TypingSpeed 90ms

# Quiet prompt, demo collection, no build noise.
Hide
Type `export PS1="❯ " && export PATH=/tmp:$PATH && cd docs/demo/collection && clear`
Enter
Sleep 1s
Show

Sleep 500ms
Type "restless ."
Sleep 350ms
Enter
Sleep 1700ms

# Walk down to users.http and open it.
Down
Sleep 400ms
Down
Sleep 600ms
Enter
Sleep 1000ms

# Pick the listUsers request.
Down
Sleep 550ms
Enter
Sleep 1100ms

# Focus the detail pane and send.
Tab
Sleep 550ms
Enter
Sleep 2600ms

# Drop the cursor onto "profile" and fold it. Keep za a single Type: splitting
# it lets the which-key popup blank the pane between the two keys.
Down 8
Sleep 700ms
Type@15ms "za"
Sleep 1700ms

# The next visible node is "teams" — fold that too.
Down
Sleep 500ms
Type@15ms "za"
Sleep 1700ms

# Scroll through the rest of the payload.
Ctrl+D
Sleep 1500ms

# Response headers.
Type "2"
Sleep 2200ms
