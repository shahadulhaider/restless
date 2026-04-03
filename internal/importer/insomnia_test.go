package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const simpleInsomniaExport = `{
  "__export_type": "insomnia",
  "resources": [
    {"_type":"workspace","_id":"wrk_1","name":"My API","parentId":""},
    {
      "_type":"request","_id":"req_1","parentId":"wrk_1",
      "name":"Health Check","method":"GET","url":"https://api.example.com/health",
      "headers":[],"body":{},"authentication":{}
    },
    {
      "_type":"request","_id":"req_2","parentId":"wrk_1",
      "name":"Create User","method":"POST","url":"https://api.example.com/users",
      "headers":[{"name":"Content-Type","value":"application/json","disabled":false}],
      "body":{"mimeType":"application/json","text":"{\"name\":\"Alice\"}"},
      "authentication":{}
    }
  ]
}`

const folderInsomniaExport = `{
  "__export_type": "insomnia",
  "resources": [
    {"_type":"workspace","_id":"wrk_1","name":"Grouped API","parentId":""},
    {"_type":"request_group","_id":"grp_1","parentId":"wrk_1","name":"Users"},
    {
      "_type":"request","_id":"req_1","parentId":"grp_1",
      "name":"List Users","method":"GET","url":"https://api.example.com/users",
      "headers":[],"body":{},"authentication":{}
    },
    {
      "_type":"request","_id":"req_2","parentId":"grp_1",
      "name":"Get User","method":"GET","url":"https://api.example.com/users/1",
      "headers":[],"body":{},"authentication":{}
    }
  ]
}`

const authInsomniaExport = `{
  "__export_type": "insomnia",
  "resources": [
    {"_type":"workspace","_id":"wrk_1","name":"Auth API","parentId":""},
    {
      "_type":"request","_id":"req_1","parentId":"wrk_1",
      "name":"Protected","method":"GET","url":"https://api.example.com/protected",
      "headers":[],"body":{},
      "authentication":{"type":"bearer","token":"my-secret-token"}
    }
  ]
}`

const varInsomniaExport = `{
  "__export_type": "insomnia",
  "resources": [
    {"_type":"workspace","_id":"wrk_1","name":"Var API","parentId":""},
    {
      "_type":"request","_id":"req_1","parentId":"wrk_1",
      "name":"Get User","method":"GET","url":"{{ _.baseUrl }}/users/{{ _.userId }}",
      "headers":[{"name":"Authorization","value":"Bearer {{ _.token }}","disabled":false}],
      "body":{},"authentication":{}
    }
  ]
}`

func TestImportInsomnia(t *testing.T) {
	col := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(col, []byte(simpleInsomniaExport), 0644))

	out := t.TempDir()
	err := ImportInsomnia(col, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(out, "my_api.http"))
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "GET https://api.example.com/health")
	assert.Contains(t, s, "POST https://api.example.com/users")
	assert.Contains(t, s, "# @name Health Check")
	assert.Contains(t, s, `{"name":"Alice"}`)

	// Verify files parse cleanly
	reqs, err := parser.ParseFile(filepath.Join(out, "my_api.http"))
	require.NoError(t, err)
	require.Len(t, reqs, 2)
}

func TestImportInsomniaFolders(t *testing.T) {
	col := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(col, []byte(folderInsomniaExport), 0644))

	out := t.TempDir()
	err := ImportInsomnia(col, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	// Should create users/ folder
	usersFile := filepath.Join(out, "users", "users.http")
	require.FileExists(t, usersFile)

	content, err := os.ReadFile(usersFile)
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "GET https://api.example.com/users")
	assert.Contains(t, s, "GET https://api.example.com/users/1")

	// Verify parseable
	reqs, err := parser.ParseFile(usersFile)
	require.NoError(t, err)
	assert.Len(t, reqs, 2)
}

func TestImportInsomniaAuth(t *testing.T) {
	col := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(col, []byte(authInsomniaExport), 0644))

	out := t.TempDir()
	require.NoError(t, ImportInsomnia(col, ImportOptions{OutputDir: out}))

	content, err := os.ReadFile(filepath.Join(out, "auth_api.http"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "Authorization: Bearer my-secret-token")
}

func TestImportInsomniaVarConversion(t *testing.T) {
	col := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(col, []byte(varInsomniaExport), 0644))

	out := t.TempDir()
	require.NoError(t, ImportInsomnia(col, ImportOptions{OutputDir: out}))

	content, err := os.ReadFile(filepath.Join(out, "var_api.http"))
	require.NoError(t, err)
	s := string(content)
	// {{ _.baseUrl }} should become {{baseUrl}}
	assert.Contains(t, s, "{{baseUrl}}")
	assert.Contains(t, s, "{{userId}}")
	assert.Contains(t, s, "{{token}}")
	assert.NotContains(t, s, "{{ _.")
}
