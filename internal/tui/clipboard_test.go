package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestCopyCmdBatchesClipboardWrites(t *testing.T) {
	cmd := copyCmd("payload", "body")
	assert.NotNil(t, cmd)
	batch, ok := cmd().(tea.BatchMsg)
	assert.True(t, ok)
	assert.Len(t, batch, 2)
}
