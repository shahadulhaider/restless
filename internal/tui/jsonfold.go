package tui

import "strings"

// foldState captures everything the renderer and scroll handlers need to fold
// JSON body nodes. bodyStart is the full rendered line index of the first body
// JSON line, so JSON-line j maps to full line bodyStart+j. It is -1 when folding
// is inactive (non-JSON body, wrap/select/search mode, or a non-Body tab).
type foldState struct {
	visible   []int       // full line indices in display order (hidden lines removed)
	jsonFolds map[int]int // JSON-line open index -> matching close index
	bodyStart int
}

// jsonFolds maps each multi-line object/array opener line to its matching closer
// line within pretty-printed JSON. Only nodes spanning more than one line (i.e.
// having children) are foldable. Detection relies on pretty.Pretty's layout
// where a structural opener ends a line with '{' or '[' and a closer begins a
// line with '}' or ']'; string values can't false-trigger because they end with
// a quote or comma.
func jsonFolds(prettyJSON string) map[int]int {
	lines := strings.Split(prettyJSON, "\n")
	folds := map[int]int{}
	var stack []int
	for i, line := range lines {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasSuffix(t, "{"), strings.HasSuffix(t, "["):
			stack = append(stack, i)
		case strings.HasPrefix(t, "}"), strings.HasPrefix(t, "]"):
			if len(stack) > 0 {
				open := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if i > open+1 {
					folds[open] = i
				}
			}
		}
	}
	return folds
}

// buildVisible returns the full line indices that remain visible after hiding the
// children of every collapsed fold. collapsed is keyed by JSON-line index.
func buildVisible(total, bodyStart int, jsonFolds map[int]int, collapsed map[int]bool) []int {
	hidden := make(map[int]bool)
	for jOpen, jClose := range jsonFolds {
		if collapsed[jOpen] {
			for k := bodyStart + jOpen + 1; k <= bodyStart+jClose && k < total; k++ {
				hidden[k] = true
			}
		}
	}
	vis := make([]int, 0, total)
	for i := 0; i < total; i++ {
		if !hidden[i] {
			vis = append(vis, i)
		}
	}
	return vis
}

// visIndexOf returns the index of the first visible line >= target (lower bound),
// clamped to the slice bounds.
func visIndexOf(visible []int, target int) int {
	lo, hi := 0, len(visible)
	for lo < hi {
		mid := (lo + hi) / 2
		if visible[mid] < target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo >= len(visible) && len(visible) > 0 {
		return len(visible) - 1
	}
	return lo
}
