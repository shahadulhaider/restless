package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func writeHTTP(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func parseReqs(t *testing.T, path string) []model.Request {
	t.Helper()
	reqs, err := parser.ParseFile(path)
	require.NoError(t, err)
	return reqs
}

// --- TestFileOpsInsert ---

func TestFileOpsInsertIntoEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.http")
	req := model.Request{Method: "GET", URL: "https://example.com/a"}
	require.NoError(t, InsertRequest(path, req))

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 1)
	assert.Equal(t, "GET", reqs[0].Method)
	assert.Equal(t, "https://example.com/a", reqs[0].URL)
}

func TestFileOpsInsertAppends(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", "GET https://example.com/a")

	req := model.Request{Method: "POST", URL: "https://example.com/b"}
	require.NoError(t, InsertRequest(path, req))

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 2)
	assert.Equal(t, "GET", reqs[0].Method)
	assert.Equal(t, "POST", reqs[1].Method)
}

func TestFileOpsInsertPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	content := "# @name First\nGET https://example.com/first\n\n###\n\n# @name Second\nGET https://example.com/second"
	path := writeHTTP(t, dir, "col.http", content)

	req := model.Request{Method: "DELETE", URL: "https://example.com/third", Name: "Third"}
	require.NoError(t, InsertRequest(path, req))

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 3)
	assert.Equal(t, "First", reqs[0].Name)
	assert.Equal(t, "Second", reqs[1].Name)
	assert.Equal(t, "Third", reqs[2].Name)
}

func TestFileOpsInsertWithHeaders(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", "GET https://example.com/a")

	req := model.Request{
		Method: "POST",
		URL:    "https://example.com/b",
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"key":"value"}`,
	}
	require.NoError(t, InsertRequest(path, req))

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 2)
	assert.Equal(t, "POST", reqs[1].Method)
	require.Len(t, reqs[1].Headers, 1)
	assert.Equal(t, "Content-Type", reqs[1].Headers[0].Key)
	assert.NotEmpty(t, reqs[1].Body)
}

// --- TestFileOpsUpdate ---

func TestFileOpsUpdateURL(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", "GET https://old.example.com/path")

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 1)

	updated := reqs[0]
	updated.URL = "https://new.example.com/path"
	require.NoError(t, UpdateRequest(path, reqs[0], updated))

	after := parseReqs(t, path)
	require.Len(t, after, 1)
	assert.Equal(t, "https://new.example.com/path", after[0].URL)
}

func TestFileOpsUpdateMiddleRequest(t *testing.T) {
	dir := t.TempDir()
	content := "# @name First\nGET https://example.com/first\n\n###\n\n# @name Second\nGET https://example.com/second\n\n###\n\n# @name Third\nGET https://example.com/third"
	path := writeHTTP(t, dir, "col.http", content)

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 3)

	updated := reqs[1]
	updated.URL = "https://example.com/second-updated"
	updated.Name = "Second Updated"
	require.NoError(t, UpdateRequest(path, reqs[1], updated))

	after := parseReqs(t, path)
	require.Len(t, after, 3)
	assert.Equal(t, "First", after[0].Name)
	assert.Equal(t, "Second Updated", after[1].Name)
	assert.Equal(t, "https://example.com/second-updated", after[1].URL)
	assert.Equal(t, "Third", after[2].Name)
}

func TestFileOpsUpdateNotFound(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", "GET https://example.com/a")

	ghost := model.Request{Method: "GET", URL: "https://nowhere.com", SourceLine: 999}
	err := UpdateRequest(path, ghost, ghost)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- TestFileOpsDelete ---

func TestFileOpsDeleteOnlyRequest(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", "GET https://example.com/a")

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 1)

	require.NoError(t, DeleteRequest(path, reqs[0]))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(string(content)))
}

func TestFileOpsDeleteFirstRequest(t *testing.T) {
	dir := t.TempDir()
	content := "# @name First\nGET https://example.com/first\n\n###\n\n# @name Second\nGET https://example.com/second"
	path := writeHTTP(t, dir, "col.http", content)

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 2)

	require.NoError(t, DeleteRequest(path, reqs[0]))

	after := parseReqs(t, path)
	require.Len(t, after, 1)
	assert.Equal(t, "Second", after[0].Name)
}

func TestFileOpsDeleteLastRequest(t *testing.T) {
	dir := t.TempDir()
	content := "# @name First\nGET https://example.com/first\n\n###\n\n# @name Second\nGET https://example.com/second"
	path := writeHTTP(t, dir, "col.http", content)

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 2)

	require.NoError(t, DeleteRequest(path, reqs[1]))

	after := parseReqs(t, path)
	require.Len(t, after, 1)
	assert.Equal(t, "First", after[0].Name)
}

func TestFileOpsDeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", "GET https://example.com/a")

	ghost := model.Request{Method: "GET", URL: "https://nowhere.com", SourceLine: 999}
	err := DeleteRequest(path, ghost)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- TestFileOpsDuplicate ---

func TestFileOpsDuplicateSameFile(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", "# @name Original\nGET https://example.com/original")

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 1)

	require.NoError(t, DuplicateRequest(reqs[0], path))

	after := parseReqs(t, path)
	require.Len(t, after, 2)
	assert.Equal(t, "GET", after[1].Method)
	assert.Equal(t, "https://example.com/original", after[1].URL)
}

func TestFileOpsDuplicateDifferentFile(t *testing.T) {
	dir := t.TempDir()
	src := writeHTTP(t, dir, "src.http", "# @name SrcReq\nGET https://example.com/src")
	dst := filepath.Join(dir, "dst.http")

	reqs := parseReqs(t, src)
	require.Len(t, reqs, 1)

	require.NoError(t, DuplicateRequest(reqs[0], dst))

	after := parseReqs(t, dst)
	require.Len(t, after, 1)
	assert.Equal(t, "SrcReq", after[0].Name)
	assert.Equal(t, "https://example.com/src", after[0].URL)
}

func TestFileOpsDuplicateNewFileCreated(t *testing.T) {
	dir := t.TempDir()
	src := writeHTTP(t, dir, "src.http", "GET https://example.com/a")
	dst := filepath.Join(dir, "newfile.http")

	reqs := parseReqs(t, src)
	require.NoError(t, DuplicateRequest(reqs[0], dst))

	require.FileExists(t, dst)
	after := parseReqs(t, dst)
	require.Len(t, after, 1)
}
