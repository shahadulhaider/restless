package writer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDirectory(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, CreateDirectory(root, "api"))
	info, err := os.Stat(filepath.Join(root, "api"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCreateHTTPFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, CreateHTTPFile(root, "requests.http"))
	content, err := os.ReadFile(filepath.Join(root, "requests.http"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "#")
}

func TestCreateHTTPFileCreatesParentDir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, CreateHTTPFile(root, "api/users.http"))
	assert.FileExists(t, filepath.Join(root, "api", "users.http"))
}

func TestRenameEntry(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "old.http"), []byte("GET https://example.com\n"), 0644))
	require.NoError(t, RenameEntry(root, "old.http", "new.http"))
	assert.FileExists(t, filepath.Join(root, "new.http"))
	assert.NoFileExists(t, filepath.Join(root, "old.http"))
}

func TestMoveEntry(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "file.http"), []byte("GET https://example.com\n"), 0644))
	require.NoError(t, MoveEntry(root, "file.http", "subdir/file.http"))
	assert.FileExists(t, filepath.Join(root, "subdir", "file.http"))
	assert.NoFileExists(t, filepath.Join(root, "file.http"))
}

func TestDeleteEntryFile(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "delete-me.http")
	require.NoError(t, os.WriteFile(fp, []byte(""), 0644))
	require.NoError(t, DeleteEntry(root, "delete-me.http"))
	assert.NoFileExists(t, fp)
}

func TestDeleteEntryDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "mydir")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "req.http"), []byte(""), 0644))
	require.NoError(t, DeleteEntry(root, "mydir"))
	assert.NoDirExists(t, dir)
}

func TestPathTraversalBlocked(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath(root, "../escape")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "traverses outside")
}

func TestPathTraversalBlockedDeep(t *testing.T) {
	root := t.TempDir()
	err := ValidatePath(root, "a/b/../../../../../../etc/passwd")
	assert.Error(t, err)
}

func TestIsHTTPFile(t *testing.T) {
	assert.True(t, IsHTTPFile("requests.http"))
	assert.True(t, IsHTTPFile("/path/to/api/users.http"))
	assert.False(t, IsHTTPFile("collection.json"))
	assert.False(t, IsHTTPFile("readme.md"))
	assert.False(t, IsHTTPFile(".http"))
}
