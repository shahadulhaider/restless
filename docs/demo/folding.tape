# restless — JSON folding + response tabs
#
# za folds the node under the cursor, zM closes every fold, zR opens them all.
# 1/2/3 jump between the Body, Headers and Timing tabs.
#
# Prerequisites (see docs/demo/README.md):
#   go build -o /tmp/restless ./cmd/restless
#   go run docs/demo/stub/server.go &
# Run from the repository root:
#   vhs docs/demo/folding.tape

Output "/tmp/restless-demo/folding.gif"

Set Shell "bash"
Set Theme "Catppuccin Mocha"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 16
Set Width 1000
Set Height 600
Set Padding 20
Set TypingSpeed 90ms

# Get to a rendered response off-camera.
Hide
Type `export PS1="❯ " && export PATH=/tmp:$PATH && cd docs/demo/collection && clear`
Enter
Sleep 1s
Type "restless ."
Enter
Sleep 2500ms
Down 2
Enter
Sleep 800ms
Down
Enter
Sleep 800ms
Tab
Sleep 400ms
Enter
Sleep 2500ms
Show

Sleep 1200ms

# Fold a single nested object under the cursor. Each z-pair stays a single Type:
# splitting it lets the which-key popup blank the pane between the two keys.
Down 8
Sleep 700ms
Type@15ms "za"
Sleep 1700ms

# ...and open it again.
Type@15ms "za"
Sleep 1600ms

# Collapse the whole document, then expand it.
Type@15ms "zM"
Sleep 1800ms
Type@15ms "zR"
Sleep 1800ms

# Tabs: Headers, Timing, back to Body.
Type "2"
Sleep 1400ms
Type "3"
Sleep 1400ms
Type "1"
Sleep 1600ms
