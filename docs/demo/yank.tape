# restless — visual selection and yanking
#
# v starts a character selection, V a line selection; vim motions move the
# cursor and y copies exactly what is highlighted. yb and yc copy the whole
# body or the request as a curl command.
#
# Prerequisites (see docs/demo/README.md):
#   go build -o /tmp/restless ./cmd/restless
#   go run docs/demo/stub/server.go &
# Run from the repository root:
#   vhs docs/demo/yank.tape

Output "/tmp/restless-demo/yank.gif"

Set Shell "bash"
Set Theme "Catppuccin Mocha"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 16
Set Width 1000
Set Height 600
Set Padding 20
Set TypingSpeed 90ms

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
Down 3
Show

Sleep 1000ms

# Character-wise selection.
Type "v"
Sleep 800ms
Type "l"
Type "l"
Type "l"
Type "l"
Type "l"
Type "l"
Type "l"
Type "l"
Sleep 700ms
Type "j"
Type "j"
Sleep 900ms
Type "y"
Sleep 1600ms

# Line-wise selection.
Type "V"
Sleep 800ms
Type "j"
Type "j"
Type "j"
Sleep 900ms
Type "y"
Sleep 1600ms

# Copy the whole body, then the request as a curl command. Each pair stays a
# single Type so the which-key popup never blanks the pane mid-demo.
Type@15ms "yb"
Sleep 1800ms

Type@15ms "yc"
Sleep 2200ms
