package gui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shahadulhaider/restless/internal/model"
)

func writeHTTPFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const sampleHTTP = `# @name health
GET https://httpbin.org/get
Accept: application/json

###

# @name echo
POST https://httpbin.org/post
Content-Type: application/json

{"msg": "hello"}
`

func TestLoadCollection(t *testing.T) {
	dir := t.TempDir()
	writeHTTPFile(t, dir, "api.http", sampleHTTP)
	writeHTTPFile(t, dir, "sub/other.http", "# @name sub\nGET https://example.com\n")

	svc := &CollectionService{}
	col, err := svc.LoadCollection(dir)
	if err != nil {
		t.Fatal(err)
	}
	if col.RootDir != dir {
		t.Errorf("RootDir = %q, want %q", col.RootDir, dir)
	}
	if len(col.Files) != 2 {
		t.Fatalf("got %d files, want 2", len(col.Files))
	}
}

func TestLoadCollection_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	svc := &CollectionService{}
	col, err := svc.LoadCollection(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Files) != 0 {
		t.Errorf("got %d files, want 0", len(col.Files))
	}
}

func TestGetRequests(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTPFile(t, dir, "api.http", sampleHTTP)

	svc := &CollectionService{}
	reqs, err := svc.GetRequests(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 2 {
		t.Fatalf("got %d requests, want 2", len(reqs))
	}
	if reqs[0].Name != "health" {
		t.Errorf("reqs[0].Name = %q, want %q", reqs[0].Name, "health")
	}
	if reqs[1].Method != "POST" {
		t.Errorf("reqs[1].Method = %q, want %q", reqs[1].Method, "POST")
	}
}

func TestCreateRequest(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTPFile(t, dir, "api.http", "# @name first\nGET https://example.com\n")

	svc := &CollectionService{}
	err := svc.CreateRequest(path, model.Request{
		Name:   "second",
		Method: "POST",
		URL:    "https://example.com/data",
	})
	if err != nil {
		t.Fatal(err)
	}

	reqs, err := svc.GetRequests(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 2 {
		t.Fatalf("got %d requests, want 2", len(reqs))
	}
	if reqs[1].Name != "second" {
		t.Errorf("reqs[1].Name = %q, want %q", reqs[1].Name, "second")
	}
}

func TestDeleteRequest(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTPFile(t, dir, "api.http", sampleHTTP)

	svc := &CollectionService{}
	reqs, err := svc.GetRequests(path)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.DeleteRequest(path, reqs[0])
	if err != nil {
		t.Fatal(err)
	}

	remaining, err := svc.GetRequests(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 {
		t.Fatalf("got %d requests after delete, want 1", len(remaining))
	}
	if remaining[0].Name != "echo" {
		t.Errorf("remaining[0].Name = %q, want %q", remaining[0].Name, "echo")
	}
}

func TestCreateFile(t *testing.T) {
	dir := t.TempDir()
	svc := &CollectionService{}

	err := svc.CreateFile(dir, "nested/new.http")
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "nested", "new.http")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("file was not created")
	}

	reqs, err := svc.GetRequests(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 0 {
		t.Errorf("new file has %d requests, want 0", len(reqs))
	}
}

func TestCreateDirectory(t *testing.T) {
	dir := t.TempDir()
	svc := &CollectionService{}

	err := svc.CreateDirectory(dir, "subdir")
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dir, "subdir"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("created entry is not a directory")
	}
}

func TestDeleteEntry_File(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTPFile(t, dir, "del.http", "GET https://example.com\n")

	svc := &CollectionService{}
	err := svc.DeleteEntry(dir, "del.http")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file still exists after delete")
	}
}

func TestDeleteEntry_Directory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "todelete")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeHTTPFile(t, dir, "todelete/inner.http", "GET https://example.com\n")

	svc := &CollectionService{}
	err := svc.DeleteEntry(dir, "todelete")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(subdir); !os.IsNotExist(err) {
		t.Error("directory still exists after delete")
	}
}

func TestRenameEntry(t *testing.T) {
	dir := t.TempDir()
	writeHTTPFile(t, dir, "old.http", "GET https://example.com\n")

	svc := &CollectionService{}
	err := svc.RenameEntry(dir, "old.http", "new.http")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "old.http")); !os.IsNotExist(err) {
		t.Error("old file still exists")
	}
	if _, err := os.Stat(filepath.Join(dir, "new.http")); os.IsNotExist(err) {
		t.Error("new file does not exist")
	}
}

func TestDuplicateRequest(t *testing.T) {
	dir := t.TempDir()
	writeHTTPFile(t, dir, "src.http", "# @name orig\nGET https://example.com\n")
	dstPath := filepath.Join(dir, "dst.http")

	svc := &CollectionService{}
	srcReqs, err := svc.GetRequests(filepath.Join(dir, "src.http"))
	if err != nil {
		t.Fatal(err)
	}

	err = svc.DuplicateRequest(srcReqs[0], dstPath)
	if err != nil {
		t.Fatal(err)
	}

	dstReqs, err := svc.GetRequests(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(dstReqs) != 1 {
		t.Fatalf("got %d requests in dst, want 1", len(dstReqs))
	}
	if dstReqs[0].Name != "orig" {
		t.Errorf("duplicated name = %q, want %q", dstReqs[0].Name, "orig")
	}
}
