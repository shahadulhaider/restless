package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rlBin is the path to a freshly built restless binary, used by the
// subprocess exit-code tests (technique B). It is built once in TestMain
// into an OS temp dir (never the repo) and removed afterward.
var rlBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "rl-cli-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mktemp: %v\n", err)
		os.Exit(2)
	}
	rlBin = filepath.Join(dir, "restless-test-bin")
	build := exec.Command("go", "build", "-o", rlBin, "github.com/shahadulhaider/restless/cmd/restless")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build test binary: %v\n", err)
		os.RemoveAll(dir)
		os.Exit(2)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// captureStdout is required because versionCmd and the import RunE write
// directly to os.Stdout, bypassing the cobra SetOut buffer.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	return <-done
}

// writeEnvFile writes a restless.env.json with a "dev" env pointing at the
// live local API, into dir (where the .http file's loader looks for it).
func writeEnvFile(t *testing.T, dir string) {
	t.Helper()
	env := `{"dev":{"baseUrl":"http://localhost:8000"}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "restless.env.json"), []byte(env), 0644))
}

// requireLiveAPI skips the calling test unless http://localhost:8000/health
// answers 200. The subprocess exit-code tests depend on it.
func requireLiveAPI(t *testing.T) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:8000/health")
	if err != nil {
		t.Skipf("live API not reachable on localhost:8000: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Skipf("live API /health returned %d, want 200", resp.StatusCode)
	}
}

// runBinary executes the built binary and returns combined output + exit code.
func runBinary(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(rlBin, args...)
	out, _ := cmd.CombinedOutput()
	code := -1
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	return string(out), code
}

// assertHasHTTPFile walks dir recursively (each importer lays out files
// differently) and asserts at least one .http file was produced.
func assertHasHTTPFile(t *testing.T, dir string) {
	t.Helper()
	var found bool
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".http") {
			found = true
		}
		return nil
	})
	assert.True(t, found, "expected at least one .http file under %s", dir)
}

const minimalOpenAPISpec = `{
  "openapi": "3.0.0",
  "info": {"title": "Test API", "version": "1.0.0"},
  "servers": [{"url": "https://api.example.com"}],
  "paths": {
    "/ping": {
      "get": {"operationId": "ping", "tags": ["health"], "responses": {}}
    }
  }
}`

const minimalPostmanCollection = `{
  "info": {"name": "Demo API", "_postman_id": "abc", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
  "item": [
    {"name": "Get Users", "request": {"method": "GET", "url": {"raw": "https://api.example.com/users"}, "header": []}}
  ]
}`

const minimalInsomniaExport = `{
  "__export_type": "insomnia",
  "resources": [
    {"_type":"workspace","_id":"wrk_1","name":"Demo API","parentId":""},
    {"_type":"request","_id":"req_1","parentId":"wrk_1","name":"Health Check","method":"GET","url":"https://api.example.com/health","headers":[],"body":{},"authentication":{}}
  ]
}`

const minimalBru = `meta {
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

func TestVersionCommand(t *testing.T) {
	out := captureStdout(t, func() {
		c := versionCmd()
		c.SetArgs([]string{})
		require.NoError(t, c.Execute())
	})
	assert.Equal(t, "restless "+version, strings.TrimSpace(out))
}

func TestVersionRejectsExtraArgs(t *testing.T) {
	c := rootCmd()
	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"version", "foo", "bar"})

	err := c.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestRootHelpListsSubcommands(t *testing.T) {
	c := rootCmd()
	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--help"})
	require.NoError(t, c.Execute())

	s := buf.String()
	for _, sub := range []string{"import", "run", "version"} {
		assert.Contains(t, s, sub, "root help should list %q subcommand", sub)
	}
}

func TestRunHelpListsAllFlags(t *testing.T) {
	c := rootCmd()
	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"run", "--help"})
	require.NoError(t, c.Execute())

	s := buf.String()
	for _, flag := range []string{
		"--env", "--fail-fast", "--insecure", "--proxy",
		"--data", "--iterations", "--delay", "--format",
	} {
		assert.Contains(t, s, flag, "run --help should list %q", flag)
	}
	assert.Contains(t, s, `(default "text")`, "run --help should show format default text")
}

func TestRunFlagDefaults(t *testing.T) {
	f := runCmd().Flags()
	assert.Equal(t, "text", f.Lookup("format").DefValue)
	assert.Equal(t, "false", f.Lookup("fail-fast").DefValue)
	assert.Equal(t, "false", f.Lookup("insecure").DefValue)
	assert.Equal(t, "0", f.Lookup("iterations").DefValue)
	assert.Equal(t, "0", f.Lookup("delay").DefValue)
	assert.Equal(t, "", f.Lookup("env").DefValue)
	assert.Equal(t, "", f.Lookup("proxy").DefValue)
	assert.Equal(t, "", f.Lookup("data").DefValue)
}

func TestRunFlagParsingForwardsValues(t *testing.T) {
	c := runCmd()
	err := c.ParseFlags([]string{
		"--env", "prod",
		"--fail-fast",
		"--insecure",
		"--proxy", "http://proxy:8080",
		"--data", "users.csv",
		"--iterations", "5",
		"--delay", "250",
		"--format", "json",
	})
	require.NoError(t, err)

	f := c.Flags()
	assert.Equal(t, "prod", f.Lookup("env").Value.String())
	assert.Equal(t, "true", f.Lookup("fail-fast").Value.String())
	assert.Equal(t, "true", f.Lookup("insecure").Value.String())
	assert.Equal(t, "http://proxy:8080", f.Lookup("proxy").Value.String())
	assert.Equal(t, "users.csv", f.Lookup("data").Value.String())
	assert.Equal(t, "5", f.Lookup("iterations").Value.String())
	assert.Equal(t, "250", f.Lookup("delay").Value.String())
	assert.Equal(t, "json", f.Lookup("format").Value.String())
}

func TestRootUnknownFlagErrors(t *testing.T) {
	c := rootCmd()
	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"--definitely-not-a-flag"})

	err := c.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag")
}

func TestRunUnknownFlagErrors(t *testing.T) {
	err := runCmd().ParseFlags([]string{"--nope"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag")
}

func TestRunRejectsWrongArgCount(t *testing.T) {
	c := rootCmd()
	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"run"})
	require.Error(t, c.Execute())

	// ExactArgs(1) rejects 2 args during validation, before RunE runs, so
	// this path never reaches runner.Run or its os.Exit(1).
	c = rootCmd()
	buf.Reset()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"run", "a.http", "b.http"})
	require.Error(t, c.Execute())
}

// TestUnknownImportSubcommand documents cobra's behavior for an unknown
// subcommand of the non-runnable `import` parent: it prints usage/help and
// does NOT dispatch to any importer. (cobra v1.10.2 returns no error here;
// the error-path coverage for bad invocations is in the unknown-flag and
// wrong-arg-count tests above.)
func TestUnknownImportSubcommand(t *testing.T) {
	c := rootCmd()
	var buf bytes.Buffer
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"import", "no-such-importer"})

	err := c.Execute()
	if err != nil {
		assert.Contains(t, strings.ToLower(err.Error()), "unknown command")
		return
	}
	assert.Contains(t, buf.String(), "Usage")
	assert.Contains(t, buf.String(), "import")
}

func TestImportOpenAPIDispatchWritesHTTP(t *testing.T) {
	specDir := t.TempDir()
	spec := filepath.Join(specDir, "spec.json")
	require.NoError(t, os.WriteFile(spec, []byte(minimalOpenAPISpec), 0644))

	outDir := t.TempDir()
	c := rootCmd()
	c.SetArgs([]string{"import", "openapi", spec, "--output", outDir})

	out := captureStdout(t, func() {
		require.NoError(t, c.Execute())
	})
	assert.Contains(t, out, "Imported to")
	assertHasHTTPFile(t, outDir)
}

func TestImportCurlDispatchWritesHTTP(t *testing.T) {
	outDir := t.TempDir()
	c := rootCmd()
	c.SetArgs([]string{"import", "curl", "curl https://api.example.com/ping", "--output", outDir})

	out := captureStdout(t, func() {
		require.NoError(t, c.Execute())
	})
	assert.Contains(t, out, "Imported to")
	assertHasHTTPFile(t, outDir)
}

func TestImportPostmanDispatchWritesHTTP(t *testing.T) {
	col := filepath.Join(t.TempDir(), "collection.json")
	require.NoError(t, os.WriteFile(col, []byte(minimalPostmanCollection), 0644))

	outDir := t.TempDir()
	c := rootCmd()
	c.SetArgs([]string{"import", "postman", col, "--output", outDir})

	out := captureStdout(t, func() {
		require.NoError(t, c.Execute())
	})
	assert.Contains(t, out, "Imported to")
	assertHasHTTPFile(t, outDir)
}

func TestImportInsomniaDispatchWritesHTTP(t *testing.T) {
	exp := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(exp, []byte(minimalInsomniaExport), 0644))

	outDir := t.TempDir()
	c := rootCmd()
	c.SetArgs([]string{"import", "insomnia", exp, "--output", outDir})

	out := captureStdout(t, func() {
		require.NoError(t, c.Execute())
	})
	assert.Contains(t, out, "Imported to")
	assertHasHTTPFile(t, outDir)
}

func TestImportBrunoDispatchWritesHTTP(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "health.bru"), []byte(minimalBru), 0644))

	outDir := t.TempDir()
	c := rootCmd()
	c.SetArgs([]string{"import", "bruno", srcDir, "--output", outDir})

	out := captureStdout(t, func() {
		require.NoError(t, c.Execute())
	})
	assert.Contains(t, out, "Imported to")
	assertHasHTTPFile(t, outDir)
}

// TestRunCommandPassesInProcess exercises the `run` RunE happy path
// in-process: all assertions pass => AnyFailed is false => RunE returns nil
// (the os.Exit(1) branch is never taken). Requires the live API.
func TestRunCommandPassesInProcess(t *testing.T) {
	requireLiveAPI(t)

	dir := t.TempDir()
	writeEnvFile(t, dir)
	f := filepath.Join(dir, "pass.http")
	body := "# @name health\nGET {{baseUrl}}/health\n\n# @assert status == 200\n"
	require.NoError(t, os.WriteFile(f, []byte(body), 0644))

	c := rootCmd()
	c.SetArgs([]string{"run", f, "--env", "dev"})

	var execErr error
	_ = captureStdout(t, func() { execErr = c.Execute() })
	require.NoError(t, execErr)
}

func TestExitCodeZeroWhenAssertionsPass(t *testing.T) {
	requireLiveAPI(t)

	dir := t.TempDir()
	writeEnvFile(t, dir)
	f := filepath.Join(dir, "pass.http")
	body := "# @name health\nGET {{baseUrl}}/health\n\n# @assert status == 200\n"
	require.NoError(t, os.WriteFile(f, []byte(body), 0644))

	out, code := runBinary(t, "run", f, "--env", "dev")
	assert.Equal(t, 0, code, "expected exit 0 on all-pass; output:\n%s", out)
}

func TestExitCodeOneWhenAssertionFails(t *testing.T) {
	requireLiveAPI(t)

	dir := t.TempDir()
	writeEnvFile(t, dir)
	f := filepath.Join(dir, "fail.http")
	// /health returns 200; asserting 500 forces an assertion failure => exit 1.
	body := "# @name health\nGET {{baseUrl}}/health\n\n# @assert status == 500\n"
	require.NoError(t, os.WriteFile(f, []byte(body), 0644))

	out, code := runBinary(t, "run", f, "--env", "dev")
	assert.Equal(t, 1, code, "expected exit 1 on assertion failure; output:\n%s", out)
}

func TestVersionSubprocess(t *testing.T) {
	out, code := runBinary(t, "version")
	assert.Equal(t, 0, code)
	assert.Equal(t, "restless "+version, strings.TrimSpace(out))
}
