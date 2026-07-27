# restless — code generation
#
# yg opens a which-key popup listing every supported target; the next key
# copies the current request as code. Here: Python, then JavaScript.
#
# Prerequisites (see docs/demo/README.md):
#   go build -o /tmp/restless ./cmd/restless
#   go run docs/demo/stub/server.go &
# Run from the repository root:
#   vhs docs/demo/codegen.tape

Output "/tmp/restless-demo/codegen.gif"

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
Show

Sleep 1200ms

# y opens the yank menu...
Type "y"
Sleep 1400ms
# ...g narrows it to the code generators.
Type "g"
Sleep 1900ms
# p copies the request as Python.
Type "p"
Sleep 2000ms

# Same prefix, different target: JavaScript.
Type "y"
Sleep 1000ms
Type "g"
Sleep 1500ms
Type "j"
Sleep 2000ms
