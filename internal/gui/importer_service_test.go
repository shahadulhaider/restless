package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportCurl(t *testing.T) {
	dir := t.TempDir()
	svc := &ImporterService{}

	err := svc.ImportCurl(`curl -X GET https://httpbin.org/get -H "Accept: application/json"`, dir)
	if err != nil {
		t.Fatalf("ImportCurl failed: %v", err)
	}

	out := filepath.Join(dir, "imported.http")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Fatal("output file is empty")
	}
}

func TestImportCurl_InvalidCommand(t *testing.T) {
	dir := t.TempDir()
	svc := &ImporterService{}

	err := svc.ImportCurl("not-curl http://example.com", dir)
	if err == nil {
		t.Fatal("expected error for non-curl command")
	}
}

func TestImportPostman_MissingFile(t *testing.T) {
	dir := t.TempDir()
	svc := &ImporterService{}

	err := svc.ImportPostman("/nonexistent/collection.json", dir)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
