package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/history"
	"github.com/shahadulhaider/restless/internal/importer"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

func fixturesDir() string {
	return filepath.Join("fixtures")
}

func TestParseExecuteHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	httpFile := filepath.Join(tmp, "test.http")
	content := "GET " + srv.URL + "/json\nAccept: application/json\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	reqs, err := parser.ParseFile(httpFile)
	require.NoError(t, err)
	require.Len(t, reqs, 1)

	resp, err := engine.Execute(&reqs[0])
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, string(resp.Body), "ok")
	assert.GreaterOrEqual(t, resp.Timing.Total.Nanoseconds(), int64(0))

	require.NoError(t, history.Save(tmp, &reqs[0], resp, "test"))
	entries, err := history.List(tmp, &reqs[0])
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, 200, entries[0].Response.StatusCode)
}

func TestRequestChaining(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login":
			w.WriteHeader(200)
			w.Write([]byte(`{"token":"jwt-test-123"}`))
		case "/profile":
			auth := r.Header.Get("Authorization")
			if auth == "Bearer jwt-test-123" {
				w.WriteHeader(200)
				w.Write([]byte(`{"user":"alice"}`))
			} else {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":"unauthorized"}`))
			}
		}
	}))
	defer srv.Close()

	tmp := t.TempDir()
	httpFile := filepath.Join(tmp, "chained.http")
	content := "# @name login\nPOST " + srv.URL + "/login\nContent-Type: application/json\n\n{\"username\":\"alice\"}\n\n###\n\nGET " + srv.URL + "/profile\nAuthorization: Bearer {{login.response.body.token}}\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	reqs, err := parser.ParseFile(httpFile)
	require.NoError(t, err)
	require.Len(t, reqs, 2)

	chainCtx := parser.NewChainContext()

	resp1, err := engine.Execute(&reqs[0])
	require.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)
	chainCtx.StoreResponse(reqs[0].Name, resp1)

	resolved, err := parser.ResolveRequest(&reqs[1], nil, chainCtx)
	require.NoError(t, err)

	authHeader := ""
	for _, h := range resolved.Headers {
		if strings.EqualFold(h.Key, "authorization") {
			authHeader = h.Value
		}
	}
	assert.Equal(t, "Bearer jwt-test-123", authHeader)

	resp2, err := engine.Execute(resolved)
	require.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)
}

func TestEnvironmentResolution(t *testing.T) {
	fixDir := fixturesDir()
	envFile, err := parser.LoadEnvironments(fixDir)
	require.NoError(t, err)
	require.NotNil(t, envFile)

	vars, err := parser.ResolveEnvironment(envFile, "dev")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", vars["base_url"])
	assert.Equal(t, "dev-key-456", vars["api_key"])

	reqs, err := parser.ParseFile(filepath.Join(fixDir, "with-vars.http"))
	require.NoError(t, err)
	require.NotEmpty(t, reqs)

	resolved, err := parser.ResolveRequest(&reqs[0], vars, nil)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/status", resolved.URL)

	authKey := ""
	for _, h := range resolved.Headers {
		if strings.EqualFold(h.Key, "x-api-key") {
			authKey = h.Value
		}
	}
	assert.Equal(t, "dev-key-456", authKey)
}

func TestPostmanImportAndParse(t *testing.T) {
	tmp := t.TempDir()
	err := importer.ImportPostman(
		filepath.Join(fixturesDir(), "postman-sample.json"),
		importer.ImportOptions{OutputDir: tmp},
	)
	require.NoError(t, err)

	httpFiles, err := filepath.Glob(filepath.Join(tmp, "*.http"))
	require.NoError(t, err)
	require.NotEmpty(t, httpFiles)

	var allReqs []model.Request
	for _, f := range httpFiles {
		reqs, parseErr := parser.ParseFile(f)
		require.NoError(t, parseErr, "generated file %s should parse cleanly", f)
		allReqs = append(allReqs, reqs...)
	}
	assert.GreaterOrEqual(t, len(allReqs), 2)

	methods := make(map[string]bool)
	for _, r := range allReqs {
		methods[r.Method] = true
	}
	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
}

func TestCookiePersistence(t *testing.T) {
	var cookieReceived string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/set-cookie":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "sess-abc", Path: "/"})
			w.WriteHeader(200)
			w.Write([]byte(`{}`))
		case "/check-cookie":
			c, err := r.Cookie("session")
			if err == nil {
				cookieReceived = c.Value
			}
			w.WriteHeader(200)
			w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	cm := engine.NewCookieManager()
	jar := cm.JarForEnv("test")

	req1 := &model.Request{Method: "GET", URL: srv.URL + "/set-cookie"}
	_, err := engine.ExecuteWithJar(req1, jar)
	require.NoError(t, err)

	req2 := &model.Request{Method: "GET", URL: srv.URL + "/check-cookie"}
	_, err = engine.ExecuteWithJar(req2, jar)
	require.NoError(t, err)
	assert.Equal(t, "sess-abc", cookieReceived)
}

func TestFileBodyLoading(t *testing.T) {
	tmp := t.TempDir()
	bodyFile := filepath.Join(tmp, "body.json")
	require.NoError(t, os.WriteFile(bodyFile, []byte(`{"loaded":true}`), 0644))

	httpFile := filepath.Join(tmp, "test.http")
	content := "POST http://example.com/api\nContent-Type: application/json\n\n< ./body.json\n"
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	reqs, err := parser.ParseFile(httpFile)
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "./body.json", reqs[0].BodyFile)

	loaded, err := parser.LoadFileBody(&reqs[0], tmp)
	require.NoError(t, err)
	assert.Equal(t, `{"loaded":true}`, strings.TrimSpace(loaded.Body))
	assert.Empty(t, loaded.BodyFile)
}

func TestErrorScenarios(t *testing.T) {
	t.Run("invalid http syntax", func(t *testing.T) {
		tmp := t.TempDir()
		badFile := filepath.Join(tmp, "bad.http")
		require.NoError(t, os.WriteFile(badFile, []byte("FOOBAR http://example.com\n"), 0644))
		reqs, _ := parser.ParseFile(badFile)
		assert.Empty(t, reqs)
	})

	t.Run("missing env file returns empty", func(t *testing.T) {
		tmp := t.TempDir()
		ef, err := parser.LoadEnvironments(tmp)
		require.NoError(t, err)
		assert.NotNil(t, ef)
		assert.Empty(t, ef.Environments)
	})

	t.Run("network timeout", func(t *testing.T) {
		req := &model.Request{
			Method: "GET",
			URL:    "http://localhost:19999",
			Metadata: model.RequestMetadata{
				Timeout: 100,
			},
		}
		_, err := engine.Execute(req)
		assert.Error(t, err)
	})
}

func TestSharedEnvKey(t *testing.T) {
	tmp := t.TempDir()
	envContent := `{"$shared":{"host":"shared.example.com"},"dev":{"token":"dev-tok"}}`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "http-client.env.json"), []byte(envContent), 0644))

	ef, err := parser.LoadEnvironments(tmp)
	require.NoError(t, err)

	vars, err := parser.ResolveEnvironment(ef, "dev")
	require.NoError(t, err)
	assert.Equal(t, "shared.example.com", vars["host"])
	assert.Equal(t, "dev-tok", vars["token"])
}

func TestPostmanEnvImport(t *testing.T) {
	tmp := t.TempDir()
	envJSON := `{
		"name": "Dev Environment",
		"values": [
			{"key": "base_url", "value": "http://localhost:3000", "enabled": true},
			{"key": "token", "value": "dev-secret", "enabled": true}
		]
	}`
	envFile := filepath.Join(tmp, "env.json")
	require.NoError(t, os.WriteFile(envFile, []byte(envJSON), 0644))

	require.NoError(t, importer.ImportPostmanEnv(envFile, tmp))

	data, err := os.ReadFile(filepath.Join(tmp, "http-client.env.json"))
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.NotEmpty(t, parsed)
}
