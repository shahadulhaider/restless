package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shahadulhaider/restless/internal/history"
	"github.com/shahadulhaider/restless/internal/model"
)

func TestHistoryService_List_Empty(t *testing.T) {
	svc := &HistoryService{}
	dir := t.TempDir()

	req := &model.Request{Method: "GET", URL: "https://example.com"}
	entries, err := svc.List(dir, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestHistoryService_List_WithEntries(t *testing.T) {
	svc := &HistoryService{}
	dir := t.TempDir()

	req := &model.Request{Method: "GET", URL: "https://example.com"}
	resp := &model.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"ok":true}`)}

	if err := history.Save(dir, req, resp, "test"); err != nil {
		t.Fatalf("save: %v", err)
	}

	entries, err := svc.List(dir, req)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Environment != "test" {
		t.Errorf("expected env 'test', got %q", entries[0].Environment)
	}
}

func TestHistoryService_Diff(t *testing.T) {
	svc := &HistoryService{}

	a := &history.HistoryEntry{
		Request:   &model.Request{Method: "GET", URL: "https://example.com"},
		Response:  &model.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"v":1}`)},
		Timestamp: time.Now(),
	}
	b := &history.HistoryEntry{
		Request:   &model.Request{Method: "GET", URL: "https://example.com"},
		Response:  &model.Response{StatusCode: 201, Status: "201 Created", Body: []byte(`{"v":2}`)},
		Timestamp: time.Now(),
	}

	result := svc.Diff(a, b)
	if result == "" {
		t.Fatal("expected non-empty diff")
	}
	if !contains(result, "- Status: 200") || !contains(result, "+ Status: 201") {
		t.Errorf("diff missing status change: %s", result)
	}
}

func TestHistoryService_List_FiltersCorrectly(t *testing.T) {
	svc := &HistoryService{}
	dir := t.TempDir()

	reqA := &model.Request{Method: "GET", URL: "https://a.com"}
	reqB := &model.Request{Method: "POST", URL: "https://b.com"}
	resp := &model.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{}`)}

	writeHistoryEntry(t, dir, reqA, resp, "env1")
	writeHistoryEntry(t, dir, reqB, resp, "env2")

	entries, err := svc.List(dir, reqA)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for reqA, got %d", len(entries))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func writeHistoryEntry(t *testing.T, rootDir string, req *model.Request, resp *model.Response, env string) {
	t.Helper()
	histDir := filepath.Join(rootDir, ".restless", "history")
	if err := os.MkdirAll(histDir, 0755); err != nil {
		t.Fatal(err)
	}

	entry := history.HistoryEntry{
		Request:     req,
		Response:    resp,
		Environment: env,
		Timestamp:   time.Now().UTC(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}

	name := entry.Timestamp.Format("20060102T150405.000000000Z") + "_" + req.Method + ".json"
	if err := os.WriteFile(filepath.Join(histDir, name), data, 0644); err != nil {
		t.Fatal(err)
	}
}
