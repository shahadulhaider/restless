package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/pretty"
)

func TestJSONFolds(t *testing.T) {
	pj := string(pretty.Pretty([]byte(`{"a":1,"b":{"c":2},"d":[1,2]}`)))
	folds := jsonFolds(pj)
	assert.Contains(t, folds, 0)
	for open, close := range folds {
		assert.Greater(t, close, open+1)
	}
}

func TestJSONFoldsSingleLineNotFoldable(t *testing.T) {
	folds := jsonFolds(`{}`)
	assert.Empty(t, folds)
}

func TestBuildVisibleHidesCollapsedChildren(t *testing.T) {
	// 6 lines, body starts at full line 1; a fold opens at json line 1 (full 2),
	// closes at json line 3 (full 4).
	jf := map[int]int{1: 3}
	collapsed := map[int]bool{1: true}
	vis := buildVisible(6, 1, jf, collapsed)
	// hidden full lines: 3,4 (bodyStart+jOpen+1=3 .. bodyStart+jClose=4)
	assert.Equal(t, []int{0, 1, 2, 5}, vis)
}

func TestBuildVisibleExpanded(t *testing.T) {
	jf := map[int]int{1: 3}
	vis := buildVisible(6, 1, jf, map[int]bool{})
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, vis)
}

func TestVisIndexOf(t *testing.T) {
	vis := []int{0, 1, 2, 5, 6}
	assert.Equal(t, 0, visIndexOf(vis, 0))
	assert.Equal(t, 3, visIndexOf(vis, 5))
	assert.Equal(t, 3, visIndexOf(vis, 3)) // first visible >= 3 is 5 (index 3)
	assert.Equal(t, 4, visIndexOf(vis, 6))
}
