package tui

import (
	"unicode"
)

// lineEdit is a single-line text buffer with cursor position.
// Supports readline/emacs keybindings.
type lineEdit struct {
	text []rune
	pos  int // cursor position: 0..len(text)
}

func newLineEdit(s string) lineEdit {
	r := []rune(s)
	return lineEdit{text: r, pos: len(r)}
}

func (l *lineEdit) String() string {
	return string(l.text)
}

func (l *lineEdit) Len() int {
	return len(l.text)
}

// Insert inserts a rune at cursor position.
func (l *lineEdit) Insert(r rune) {
	l.text = append(l.text, 0)
	copy(l.text[l.pos+1:], l.text[l.pos:])
	l.text[l.pos] = r
	l.pos++
}

// Backspace deletes the rune before the cursor.
func (l *lineEdit) Backspace() {
	if l.pos == 0 {
		return
	}
	l.text = append(l.text[:l.pos-1], l.text[l.pos:]...)
	l.pos--
}

// Delete deletes the rune at the cursor (ctrl+d in emacs).
func (l *lineEdit) Delete() {
	if l.pos >= len(l.text) {
		return
	}
	l.text = append(l.text[:l.pos], l.text[l.pos+1:]...)
}

// Left moves cursor one position left.
func (l *lineEdit) Left() {
	if l.pos > 0 {
		l.pos--
	}
}

// Right moves cursor one position right.
func (l *lineEdit) Right() {
	if l.pos < len(l.text) {
		l.pos++
	}
}

// Home moves cursor to start of line (ctrl+a).
func (l *lineEdit) Home() {
	l.pos = 0
}

// End moves cursor to end of line (ctrl+e).
func (l *lineEdit) End() {
	l.pos = len(l.text)
}

// KillToEnd deletes from cursor to end of line (ctrl+k).
func (l *lineEdit) KillToEnd() {
	l.text = l.text[:l.pos]
}

// KillToStart deletes from start to cursor (ctrl+u).
func (l *lineEdit) KillToStart() {
	l.text = l.text[l.pos:]
	l.pos = 0
}

// DeleteWordBackward deletes the word before the cursor (ctrl+w).
func (l *lineEdit) DeleteWordBackward() {
	if l.pos == 0 {
		return
	}
	end := l.pos
	// Skip spaces
	for l.pos > 0 && l.text[l.pos-1] == ' ' {
		l.pos--
	}
	// Skip word chars
	for l.pos > 0 && l.text[l.pos-1] != ' ' {
		l.pos--
	}
	l.text = append(l.text[:l.pos], l.text[end:]...)
}

// ForwardWord moves cursor forward one word (alt+f / meta+f).
func (l *lineEdit) ForwardWord() {
	// Skip current word chars
	for l.pos < len(l.text) && !unicode.IsSpace(l.text[l.pos]) {
		l.pos++
	}
	// Skip spaces
	for l.pos < len(l.text) && unicode.IsSpace(l.text[l.pos]) {
		l.pos++
	}
}

// BackwardWord moves cursor backward one word (alt+b / meta+b).
func (l *lineEdit) BackwardWord() {
	// Skip spaces
	for l.pos > 0 && unicode.IsSpace(l.text[l.pos-1]) {
		l.pos--
	}
	// Skip word chars
	for l.pos > 0 && !unicode.IsSpace(l.text[l.pos-1]) {
		l.pos--
	}
}

// Set replaces the entire content and moves cursor to end.
func (l *lineEdit) Set(s string) {
	l.text = []rune(s)
	l.pos = len(l.text)
}

// View returns the text with a cursor block character inserted at pos.
func (l *lineEdit) View(focused bool) string {
	if !focused {
		return string(l.text)
	}
	before := string(l.text[:l.pos])
	after := string(l.text[l.pos:])
	return before + "█" + after
}

// HandleKey processes a keystroke and returns true if the key was consumed.
func (l *lineEdit) HandleKey(key string) bool {
	switch key {
	case "left":
		l.Left()
	case "right":
		l.Right()
	case "ctrl+a", "home":
		l.Home()
	case "ctrl+e", "end":
		l.End()
	case "ctrl+k":
		l.KillToEnd()
	case "ctrl+u":
		l.KillToStart()
	case "ctrl+w":
		l.DeleteWordBackward()
	case "ctrl+f":
		l.Right()
	case "ctrl+b":
		l.Left()
	case "alt+f":
		l.ForwardWord()
	case "alt+b":
		l.BackwardWord()
	case "backspace":
		l.Backspace()
	case "delete", "ctrl+d":
		l.Delete()
	default:
		r := []rune(key)
		if len(r) == 1 {
			l.Insert(r[0])
		} else {
			return false // key not consumed
		}
	}
	return true
}

// HandleKeyFiltered is like HandleKey but only inserts runes that pass the filter.
func (l *lineEdit) HandleKeyFiltered(key string, filter func(rune) bool) bool {
	switch key {
	case "left", "right", "ctrl+a", "home", "ctrl+e", "end",
		"ctrl+k", "ctrl+u", "ctrl+w", "ctrl+f", "ctrl+b",
		"alt+f", "alt+b", "backspace", "delete", "ctrl+d":
		return l.HandleKey(key)
	default:
		r := []rune(key)
		if len(r) == 1 && filter(r[0]) {
			l.Insert(r[0])
			return true
		}
		return false
	}
}
