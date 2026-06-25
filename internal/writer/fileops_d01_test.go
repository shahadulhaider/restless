package writer

import (
	"os"
	"testing"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const d01FileVarsContent = "@baseUrl = http://127.0.0.1:8799\n@token = secret-abc-123\n\n# @name existing-get\nGET {{baseUrl}}/get\nAuthorization: Bearer {{token}}\n\n# @assert status == 200"

func assertFileVarsPreserved(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(raw)
	assert.Contains(t, content, "@baseUrl = http://127.0.0.1:8799")
	assert.Contains(t, content, "@token = secret-abc-123")
}

func TestFileOpsInsertPreservesFileVars(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", d01FileVarsContent)

	req := model.Request{Method: "POST", URL: "http://127.0.0.1:8799/anything", Name: "added"}
	require.NoError(t, InsertRequest(path, req))

	assertFileVarsPreserved(t, path)
	require.Len(t, parseReqs(t, path), 2)
}

func TestFileOpsUpdatePreservesFileVars(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", d01FileVarsContent)

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 1)
	updated := reqs[0]
	updated.URL = "{{baseUrl}}/get-updated"
	require.NoError(t, UpdateRequest(path, reqs[0], updated))

	assertFileVarsPreserved(t, path)
	after := parseReqs(t, path)
	require.Len(t, after, 1)
	assert.Equal(t, "{{baseUrl}}/get-updated", after[0].URL)
}

func TestFileOpsDeletePreservesFileVars(t *testing.T) {
	dir := t.TempDir()
	content := d01FileVarsContent + "\n\n###\n\n# @name second\nGET {{baseUrl}}/two"
	path := writeHTTP(t, dir, "col.http", content)

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 2)
	require.NoError(t, DeleteRequest(path, reqs[1]))

	assertFileVarsPreserved(t, path)
	require.Len(t, parseReqs(t, path), 1)
}

func TestFileOpsDuplicatePreservesFileVars(t *testing.T) {
	dir := t.TempDir()
	path := writeHTTP(t, dir, "col.http", d01FileVarsContent)

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 1)
	require.NoError(t, DuplicateRequest(reqs[0], path))

	assertFileVarsPreserved(t, path)
	require.Len(t, parseReqs(t, path), 2)
}

func TestFileOpsInsertPreservesLeadingComment(t *testing.T) {
	dir := t.TempDir()
	content := "# Collection header comment\n@baseUrl = http://x\n\n# @name a\nGET {{baseUrl}}/a"
	path := writeHTTP(t, dir, "col.http", content)

	require.NoError(t, InsertRequest(path, model.Request{Method: "POST", URL: "http://x/b"}))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "# Collection header comment")
	assert.Contains(t, string(raw), "@baseUrl = http://x")
}

func TestFileOpsUpdatePreservesMidFileVars(t *testing.T) {
	dir := t.TempDir()
	content := "@leading = http://lead\n\n# @name first\nGET {{leading}}/a\n\n###\n\n@token = secret-xyz\n\n# @name second\nGET {{leading}}/b\nAuthorization: Bearer {{token}}"
	path := writeHTTP(t, dir, "col.http", content)

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 2)
	updated := reqs[1]
	updated.URL = "{{leading}}/b-updated"
	require.NoError(t, UpdateRequest(path, reqs[1], updated))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	got := string(raw)
	assert.Contains(t, got, "@leading = http://lead")
	assert.Contains(t, got, "@token = secret-xyz")
}

func TestFileOpsDeletePreservesMidFileVars(t *testing.T) {
	dir := t.TempDir()
	content := "@leading = http://lead\n\n# @name first\nGET {{leading}}/a\n\n###\n\n@token = secret-xyz\n\n# @name second\nGET {{leading}}/b"
	path := writeHTTP(t, dir, "col.http", content)

	reqs := parseReqs(t, path)
	require.Len(t, reqs, 2)
	require.NoError(t, DeleteRequest(path, reqs[0]))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	got := string(raw)
	assert.Contains(t, got, "@leading = http://lead")
	assert.Contains(t, got, "@token = secret-xyz")
}
