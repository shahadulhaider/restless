package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeBruFile writes a .bru file to path, creating parent dirs.
func writeBruFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

const simpleGetBru = `meta {
  name: Health Check
  type: http
  seq: 1
}

get {
  url: https://api.example.com/health
  body: none
  auth: none
}
`

const postWithBodyBru = `meta {
  name: Create User
  type: http
  seq: 2
}

post {
  url: https://api.example.com/users
  body: json
  auth: none
}

headers {
  Content-Type: application/json
}

body:json {
  {"name":"Alice","email":"alice@example.com"}
}
`

const bearerAuthBru = `meta {
  name: Protected Resource
  type: http
  seq: 1
}

get {
  url: https://api.example.com/protected
  body: none
  auth: bearer
}

auth:bearer {
  token: my-secret-token
}
`

const basicAuthBru = `meta {
  name: Basic Auth Request
  type: http
  seq: 1
}

get {
  url: https://api.example.com/admin
  body: none
  auth: basic
}

auth:basic {
  username: admin
  password: secret123
}
`

const envDevBru = `vars {
  baseUrl: https://dev.example.com
  apiKey: dev-key-123
}
`

const envProdBru = `vars {
  baseUrl: https://prod.example.com
  apiKey: prod-key-456
}
`

func TestImportBrunoSingleFile(t *testing.T) {
	src := t.TempDir()
	writeBruFile(t, filepath.Join(src, "health.bru"), simpleGetBru)

	out := t.TempDir()
	err := ImportBruno(src, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	// Should produce a .http file named after the source directory
	files, err := filepath.Glob(filepath.Join(out, "*.http"))
	require.NoError(t, err)
	require.Len(t, files, 1)

	content, err := os.ReadFile(files[0])
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "GET https://api.example.com/health")
	assert.Contains(t, s, "# @name Health Check")

	// Should parse cleanly
	reqs, err := parser.ParseFile(files[0])
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "GET", reqs[0].Method)
}

func TestImportBrunoFolderStructure(t *testing.T) {
	src := t.TempDir()
	// Root level request
	writeBruFile(t, filepath.Join(src, "health.bru"), simpleGetBru)
	// Subfolder with requests
	writeBruFile(t, filepath.Join(src, "users", "create.bru"), postWithBodyBru)

	out := t.TempDir()
	err := ImportBruno(src, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	// Root .http file should exist
	rootFiles, err := filepath.Glob(filepath.Join(out, "*.http"))
	require.NoError(t, err)
	assert.Len(t, rootFiles, 1)

	// Subfolder .http file should exist under users/
	subFiles, err := filepath.Glob(filepath.Join(out, "users", "*.http"))
	require.NoError(t, err)
	assert.Len(t, subFiles, 1)

	content, err := os.ReadFile(subFiles[0])
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "POST https://api.example.com/users")
	assert.Contains(t, s, "# @name Create User")
}

func TestImportBrunoAuth(t *testing.T) {
	src := t.TempDir()
	writeBruFile(t, filepath.Join(src, "bearer.bru"), bearerAuthBru)
	writeBruFile(t, filepath.Join(src, "basic.bru"), basicAuthBru)

	out := t.TempDir()
	err := ImportBruno(src, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(out, "*.http"))
	require.NoError(t, err)
	require.Len(t, files, 1)

	content, err := os.ReadFile(files[0])
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "Authorization: Bearer my-secret-token")
	assert.Contains(t, s, "Authorization: Basic admin:secret123")
}

func TestImportBrunoBody(t *testing.T) {
	src := t.TempDir()
	writeBruFile(t, filepath.Join(src, "create.bru"), postWithBodyBru)

	out := t.TempDir()
	err := ImportBruno(src, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(out, "*.http"))
	require.NoError(t, err)
	require.Len(t, files, 1)

	content, err := os.ReadFile(files[0])
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "POST https://api.example.com/users")
	assert.Contains(t, s, "Content-Type: application/json")
	assert.Contains(t, s, `{"name":"Alice"`)

	// Should parse cleanly
	reqs, err := parser.ParseFile(files[0])
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "POST", reqs[0].Method)
	assert.NotEmpty(t, reqs[0].Body)
}

func TestImportBrunoEnvironments(t *testing.T) {
	src := t.TempDir()
	writeBruFile(t, filepath.Join(src, "health.bru"), simpleGetBru)
	writeBruFile(t, filepath.Join(src, "environments", "dev.bru"), envDevBru)
	writeBruFile(t, filepath.Join(src, "environments", "prod.bru"), envProdBru)

	out := t.TempDir()
	err := ImportBruno(src, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	// http-client.env.json should be written
	envFile := filepath.Join(out, "http-client.env.json")
	require.FileExists(t, envFile)

	content, err := os.ReadFile(envFile)
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "dev")
	assert.Contains(t, s, "prod")
	assert.Contains(t, s, "baseUrl")
	assert.Contains(t, s, "https://dev.example.com")
	assert.Contains(t, s, "https://prod.example.com")
}
