package runner

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSingleRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	content := "GET " + srv.URL + "/health\nAccept: application/json\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	var out, errOut bytes.Buffer
	result, err := Run(RunConfig{
		FilePath:  httpFile,
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalRequests)
	assert.Equal(t, 1, result.PassedRequests)
	assert.False(t, result.AnyFailed)
	assert.Contains(t, out.String(), "200")
}

func TestRunWithAssertions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"id":42}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	content := "# @name test\nGET " + srv.URL + "/api\n\n# @assert status == 200\n# @assert body.$.id == 42\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	var out, errOut bytes.Buffer
	result, err := Run(RunConfig{
		FilePath:  httpFile,
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalAssertions)
	assert.Equal(t, 2, result.PassedAssertions)
	assert.False(t, result.AnyFailed)
}

func TestRunWithDataCSV(t *testing.T) {
	var receivedNames []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	_ = receivedNames

	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	content := "GET " + srv.URL + "/users?name={{name}}\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	csvFile := filepath.Join(dir, "data.csv")
	require.NoError(t, os.WriteFile(csvFile, []byte("name\nAlice\nBob\nCharlie\n"), 0644))

	var out, errOut bytes.Buffer
	result, err := Run(RunConfig{
		FilePath:  httpFile,
		DataFile:  csvFile,
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalIterations)
	assert.Equal(t, 3, result.TotalRequests)
	assert.False(t, result.AnyFailed)
	assert.Contains(t, out.String(), "Iteration 1/3")
	assert.Contains(t, out.String(), "Iteration 3/3")
}

func TestRunFailFast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`error`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	content := "# @name req1\nGET " + srv.URL + "/a\n\n# @assert status == 200\n\n###\n\n# @name req2\nGET " + srv.URL + "/b\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	var out, errOut bytes.Buffer
	result, err := Run(RunConfig{
		FilePath:  httpFile,
		FailFast:  true,
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.NoError(t, err)
	assert.True(t, result.AnyFailed)
	// Should stop after first failure — only 1 request executed
	assert.Equal(t, 1, result.TotalRequests)
}
