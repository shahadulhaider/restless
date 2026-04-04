package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.csv")
	content := "name,email,role\nAlice,alice@example.com,admin\nBob,bob@example.com,user\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	rows, err := LoadDataFile(path)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "Alice", rows[0]["name"])
	assert.Equal(t, "alice@example.com", rows[0]["email"])
	assert.Equal(t, "admin", rows[0]["role"])

	assert.Equal(t, "Bob", rows[1]["name"])
	assert.Equal(t, "user", rows[1]["role"])
}

func TestLoadCSVQuotedFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")
	content := "name,body\nAlice,\"{\"\"key\"\":\"\"value\"\"}\"\nBob,simple\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	rows, err := LoadDataFile(path)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Contains(t, rows[0]["body"], "key")
}

func TestLoadCSVHeaderOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")
	require.NoError(t, os.WriteFile(path, []byte("name,email\n"), 0644))

	_, err := LoadDataFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data rows")
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	content := `[
		{"name": "Alice", "email": "alice@example.com", "age": 30},
		{"name": "Bob", "email": "bob@example.com", "age": 25}
	]`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	rows, err := LoadDataFile(path)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "Alice", rows[0]["name"])
	assert.Equal(t, "30", rows[0]["age"])
	assert.Equal(t, "Bob", rows[1]["name"])
}

func TestLoadJSONEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0644))

	_, err := LoadDataFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data")
}

func TestLoadJSONNotArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"name":"Alice"}`), 0644))

	_, err := LoadDataFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "array")
}

func TestLoadUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.xml")
	require.NoError(t, os.WriteFile(path, []byte("<data/>"), 0644))

	_, err := LoadDataFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestLoadJSONWithNestedObjects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested.json")
	content := `[{"name": "Alice", "address": {"city": "NYC"}, "active": true}]`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	rows, err := LoadDataFile(path)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Alice", rows[0]["name"])
	assert.Contains(t, rows[0]["address"], "NYC") // serialized as JSON string
	assert.Equal(t, "true", rows[0]["active"])
}
