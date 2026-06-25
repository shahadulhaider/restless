package runner

import (
	"bytes"
	"encoding/json"
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

func TestRunMissingBodyFile(t *testing.T) {
	serverHit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHit = true
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	content := "POST " + srv.URL + "/upload\nContent-Type: application/json\n\n< ./does-not-exist.json\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	var out, errOut bytes.Buffer
	result, err := Run(RunConfig{
		FilePath:  httpFile,
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.NoError(t, err)
	assert.True(t, result.AnyFailed)
	assert.Contains(t, errOut.String(), "cannot read body file")
	assert.False(t, serverHit, "request must not be sent when the body file is missing")
}

func TestRunUnknownFormatErrors(t *testing.T) {
	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	require.NoError(t, os.WriteFile(httpFile, []byte("GET http://127.0.0.1:1/health\n"), 0644))

	var out, errOut bytes.Buffer
	_, err := Run(RunConfig{
		FilePath:  httpFile,
		Format:    "bogus",
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown output format")
}

func TestRunUnknownEnvErrors(t *testing.T) {
	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	require.NoError(t, os.WriteFile(httpFile, []byte("GET http://127.0.0.1:1/health\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "restless.env.json"),
		[]byte(`{"dev":{"baseUrl":"http://localhost:8000"}}`), 0644))

	var out, errOut bytes.Buffer
	_, err := Run(RunConfig{
		FilePath:  httpFile,
		EnvName:   "doesNotExist",
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
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

func TestRunDataJSONIterationsField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	content := "GET " + srv.URL + "/users?name={{name}}\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	csvFile := filepath.Join(dir, "data.csv")
	require.NoError(t, os.WriteFile(csvFile, []byte("name\nAlice\nBob\nCharlie\n"), 0644))

	var out, errOut bytes.Buffer
	_, err := Run(RunConfig{
		FilePath:  httpFile,
		DataFile:  csvFile,
		Format:    FormatJSON,
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.NoError(t, err)

	var report jsonReport
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Len(t, report.Iterations, 3)
	assert.Equal(t, 1, report.Iterations[0].Index)
	assert.True(t, report.Iterations[0].Passed)
	assert.Equal(t, "Alice", report.Iterations[0].DataVars["name"])
	assert.Equal(t, 3, report.Iterations[2].Index)
	assert.Equal(t, "Charlie", report.Iterations[2].DataVars["name"])
}

func TestRunDataOverridesEnvVar(t *testing.T) {
	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.URL.Query().Get("token")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	content := "GET " + srv.URL + "/get?token={{token}}\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "restless.env.json"),
		[]byte(`{"dev":{"token":"env-token"}}`), 0644))
	csvFile := filepath.Join(dir, "data.csv")
	require.NoError(t, os.WriteFile(csvFile, []byte("token\ndata-token\n"), 0644))

	var out, errOut bytes.Buffer
	_, err := Run(RunConfig{
		FilePath:  httpFile,
		EnvName:   "dev",
		DataFile:  csvFile,
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.NoError(t, err)
	assert.Equal(t, "data-token", gotToken, "--data column must override env var of the same name")
}

func TestRunCookiePersistsAcrossIterations(t *testing.T) {
	var receivedSess []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("sess"); err == nil {
			receivedSess = append(receivedSess, c.Value)
		} else {
			receivedSess = append(receivedSess, "")
		}
		http.SetCookie(w, &http.Cookie{Name: "sess", Value: "v1", Path: "/"})
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	content := "GET " + srv.URL + "/track?row={{row}}\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	csvFile := filepath.Join(dir, "data.csv")
	require.NoError(t, os.WriteFile(csvFile, []byte("row\n1\n2\n"), 0644))

	var out, errOut bytes.Buffer
	_, err := Run(RunConfig{
		FilePath:  httpFile,
		DataFile:  csvFile,
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.NoError(t, err)
	require.Len(t, receivedSess, 2)
	assert.Equal(t, "", receivedSess[0], "iteration 1 sends no cookie yet")
	assert.Equal(t, "v1", receivedSess[1], "cookie from iteration 1 must persist into iteration 2")
}

func TestRunDataNestedDotPath(t *testing.T) {
	var gotRole, gotTag, gotWhole string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole = r.URL.Query().Get("role")
		gotTag = r.URL.Query().Get("tag")
		gotWhole = r.URL.Query().Get("whole")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	httpFile := filepath.Join(dir, "test.http")
	content := "GET " + srv.URL + "/get?role={{meta.role}}&tag={{meta.tags.0}}&whole={{name}}\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	jsonFile := filepath.Join(dir, "data.json")
	data := `[{"name": "alice", "meta": {"role": "admin", "tags": ["alpha", "beta"]}}]`
	require.NoError(t, os.WriteFile(jsonFile, []byte(data), 0644))

	var out, errOut bytes.Buffer
	_, err := Run(RunConfig{
		FilePath:  httpFile,
		DataFile:  jsonFile,
		Output:    &out,
		ErrOutput: &errOut,
	})
	require.NoError(t, err)
	assert.Equal(t, "admin", gotRole, "{{meta.role}} must resolve nested JSON object field")
	assert.Equal(t, "alpha", gotTag, "{{meta.tags.0}} must resolve nested JSON array index")
	assert.Equal(t, "alice", gotWhole, "scalar columns must still resolve")
}
