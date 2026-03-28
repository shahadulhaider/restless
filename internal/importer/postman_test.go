package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const simpleCollection = `{
  "info": { "name": "My API", "_postman_id": "abc", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json" },
  "item": [
    { "name": "Get Users", "request": { "method": "GET", "url": { "raw": "https://api.example.com/users" }, "header": [] } },
    { "name": "Create User", "request": { "method": "POST", "url": { "raw": "https://api.example.com/users" }, "header": [{"key":"Content-Type","value":"application/json"}], "body": { "mode": "raw", "raw": "{\"name\":\"Alice\"}" } } }
  ]
}`

const folderCollection = `{
  "info": { "name": "Grouped API", "_postman_id": "xyz", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json" },
  "item": [
    { "name": "Auth", "item": [
      { "name": "Login", "request": { "method": "POST", "url": { "raw": "https://api.example.com/login" }, "header": [] } }
    ]},
    { "name": "Users", "item": [
      { "name": "List Users", "request": { "method": "GET", "url": { "raw": "https://api.example.com/users" }, "header": [] } },
      { "name": "Get User", "request": { "method": "GET", "url": { "raw": "https://api.example.com/users/1" }, "header": [] } }
    ]}
  ]
}`

func TestImportSimpleCollection(t *testing.T) {
	col := t.TempDir() + "/simple.json"
	require.NoError(t, os.WriteFile(col, []byte(simpleCollection), 0644))

	out := t.TempDir()
	err := ImportPostman(col, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	httpFile := filepath.Join(out, "my_api.http")
	require.FileExists(t, httpFile)

	content, err := os.ReadFile(httpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "GET https://api.example.com/users")
	assert.Contains(t, string(content), "POST https://api.example.com/users")
	assert.Contains(t, string(content), `{"name":"Alice"}`)
}

func TestImportWithFolders(t *testing.T) {
	col := t.TempDir() + "/grouped.json"
	require.NoError(t, os.WriteFile(col, []byte(folderCollection), 0644))

	out := t.TempDir()
	err := ImportPostman(col, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	authFile := filepath.Join(out, "auth", "auth.http")
	usersFile := filepath.Join(out, "users", "users.http")
	require.FileExists(t, authFile)
	require.FileExists(t, usersFile)

	authContent, _ := os.ReadFile(authFile)
	assert.Contains(t, string(authContent), "POST https://api.example.com/login")

	usersContent, _ := os.ReadFile(usersFile)
	assert.Contains(t, string(usersContent), "GET https://api.example.com/users")
	assert.Contains(t, string(usersContent), "GET https://api.example.com/users/1")
}

func TestImportRequestName(t *testing.T) {
	col := t.TempDir() + "/simple.json"
	require.NoError(t, os.WriteFile(col, []byte(simpleCollection), 0644))

	out := t.TempDir()
	require.NoError(t, ImportPostman(col, ImportOptions{OutputDir: out}))

	content, _ := os.ReadFile(filepath.Join(out, "my_api.http"))
	assert.Contains(t, string(content), "# @name Get Users")
	assert.Contains(t, string(content), "# @name Create User")
}

func TestSanitizeName(t *testing.T) {
	assert.Equal(t, "my_api", sanitizeName("My API"))
	assert.Equal(t, "collection", sanitizeName(""))
	assert.Equal(t, "hello_world", sanitizeName("hello world"))
}
