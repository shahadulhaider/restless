package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLineEditInsert(t *testing.T) {
	l := newLineEdit("")
	l.Insert('h')
	l.Insert('i')
	assert.Equal(t, "hi", l.String())
	assert.Equal(t, 2, l.pos)
}

func TestLineEditBackspace(t *testing.T) {
	l := newLineEdit("hello")
	l.Backspace()
	assert.Equal(t, "hell", l.String())
	assert.Equal(t, 4, l.pos)
}

func TestLineEditLeftRight(t *testing.T) {
	l := newLineEdit("abc")
	assert.Equal(t, 3, l.pos)
	l.Left()
	assert.Equal(t, 2, l.pos)
	l.Left()
	l.Left()
	assert.Equal(t, 0, l.pos)
	l.Left() // should not go below 0
	assert.Equal(t, 0, l.pos)
	l.Right()
	assert.Equal(t, 1, l.pos)
}

func TestLineEditHomeEnd(t *testing.T) {
	l := newLineEdit("hello world")
	l.Home()
	assert.Equal(t, 0, l.pos)
	l.End()
	assert.Equal(t, 11, l.pos)
}

func TestLineEditInsertMiddle(t *testing.T) {
	l := newLineEdit("hllo")
	l.Home()
	l.Right() // pos=1, between h and l
	l.Insert('e')
	assert.Equal(t, "hello", l.String())
	assert.Equal(t, 2, l.pos)
}

func TestLineEditKillToEnd(t *testing.T) {
	l := newLineEdit("hello world")
	l.Home()
	for i := 0; i < 5; i++ {
		l.Right()
	}
	l.KillToEnd()
	assert.Equal(t, "hello", l.String())
}

func TestLineEditKillToStart(t *testing.T) {
	l := newLineEdit("hello world")
	l.Home()
	for i := 0; i < 6; i++ {
		l.Right()
	}
	l.KillToStart()
	assert.Equal(t, "world", l.String())
	assert.Equal(t, 0, l.pos)
}

func TestLineEditDeleteWordBackward(t *testing.T) {
	l := newLineEdit("hello world")
	l.DeleteWordBackward()
	assert.Equal(t, "hello ", l.String())

	l.DeleteWordBackward()
	assert.Equal(t, "", l.String())
}

func TestLineEditDelete(t *testing.T) {
	l := newLineEdit("abc")
	l.Home()
	l.Delete()
	assert.Equal(t, "bc", l.String())
	assert.Equal(t, 0, l.pos)
}

func TestLineEditForwardBackwardWord(t *testing.T) {
	l := newLineEdit("hello world foo")
	l.Home()
	l.ForwardWord()
	assert.Equal(t, 6, l.pos) // after "hello "
	l.ForwardWord()
	assert.Equal(t, 12, l.pos) // after "world "
	l.BackwardWord()
	assert.Equal(t, 6, l.pos)
	l.BackwardWord()
	assert.Equal(t, 0, l.pos)
}

func TestLineEditView(t *testing.T) {
	l := newLineEdit("abc")
	assert.Equal(t, "abc", l.View(false))
	assert.Equal(t, "abc█", l.View(true)) // cursor at end

	l.Home()
	assert.Equal(t, "█abc", l.View(true)) // cursor at start

	l.Right()
	assert.Equal(t, "a█bc", l.View(true)) // cursor in middle
}

func TestLineEditHandleKey(t *testing.T) {
	l := newLineEdit("")
	l.HandleKey("h")
	l.HandleKey("e")
	l.HandleKey("l")
	l.HandleKey("l")
	l.HandleKey("o")
	assert.Equal(t, "hello", l.String())

	l.HandleKey("ctrl+a")
	assert.Equal(t, 0, l.pos)

	l.HandleKey("ctrl+e")
	assert.Equal(t, 5, l.pos)

	l.HandleKey("ctrl+w")
	assert.Equal(t, "", l.String())
}
