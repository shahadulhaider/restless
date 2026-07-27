package tui

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

//go:generate go test -run TestKeybindingsDocInSync -update

var updateKeybindingsDoc = flag.Bool("update", false,
	"rewrite docs/keybindings.md from the Go keybinding reference")

func keybindingsDocPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve the path of keyref_test.go")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "docs", "keybindings.md")
}

func TestKeybindingsDocInSync(t *testing.T) {
	path := keybindingsDocPath(t)
	want := renderKeybindingsMarkdown()

	if *updateKeybindingsDoc {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		t.Logf("regenerated %s", path)
		return
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Error("docs/keybindings.md is out of sync with keyReference in internal/tui/keyref.go.\n" +
			"Edit keyReference, never the markdown, then regenerate with:\n" +
			"    go test ./internal/tui -run TestKeybindingsDocInSync -update")
	}
}
