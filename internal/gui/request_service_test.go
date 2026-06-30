package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shahadulhaider/restless/internal/model"
)

func TestExecute_BasicGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	svc := NewRequestService()
	resp, err := svc.Execute(model.Request{
		Method: "GET",
		URL:    srv.URL,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("body = %q, want %q", string(resp.Body), `{"ok":true}`)
	}
}

func TestExecute_POST(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		received = string(buf[:n])
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	svc := NewRequestService()
	resp, err := svc.Execute(model.Request{
		Method: "POST",
		URL:    srv.URL,
		Body:   `{"name":"test"}`,
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	if received != `{"name":"test"}` {
		t.Errorf("received body = %q, want %q", received, `{"name":"test"}`)
	}
}

func TestExecute_ChainContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"token":"abc123"}`)
	}))
	defer srv.Close()

	svc := NewRequestService()

	_, err := svc.Execute(model.Request{
		Name:   "login",
		Method: "POST",
		URL:    srv.URL,
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	names := svc.GetChainVariables()
	if len(names) != 1 || names[0] != "login" {
		t.Errorf("chain variables = %v, want [login]", names)
	}
}

func TestExecute_NoChainForUnnamed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := NewRequestService()
	_, err := svc.Execute(model.Request{
		Method: "GET",
		URL:    srv.URL,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(svc.GetChainVariables()) != 0 {
		t.Error("unnamed request should not be stored in chain context")
	}
}

func TestExecute_Assertions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	}))
	defer srv.Close()

	svc := NewRequestService()
	resp, err := svc.Execute(model.Request{
		Method: "GET",
		URL:    srv.URL,
		Assertions: []model.Assertion{
			{Target: "status", Operator: "==", Expected: "200"},
		},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.AssertionResults) != 1 {
		t.Fatalf("got %d assertion results, want 1", len(resp.AssertionResults))
	}
	if !resp.AssertionResults[0].Passed {
		t.Errorf("assertion failed: actual=%q", resp.AssertionResults[0].Actual)
	}
}

func TestExecute_WithEnvVars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"url":%q}`, r.URL.Path)
	}))
	defer srv.Close()

	dir := t.TempDir()
	envData := fmt.Sprintf(`{"$shared":{"baseUrl":%q},"test":{}}`, srv.URL)
	if err := os.WriteFile(filepath.Join(dir, "restless.env.json"), []byte(envData), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewRequestService()
	svc.SetRootDir(dir)

	resp, err := svc.Execute(model.Request{
		Method: "GET",
		URL:    "{{baseUrl}}/api/test",
	}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestExecute_History(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dir := t.TempDir()
	svc := NewRequestService()
	svc.SetRootDir(dir)

	_, err := svc.Execute(model.Request{
		Method: "GET",
		URL:    srv.URL,
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	histDir := filepath.Join(dir, ".restless", "history")
	entries, err := os.ReadDir(histDir)
	if err != nil {
		t.Fatalf("history dir missing: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least 1 history entry")
	}
}

func TestResolvePreview(t *testing.T) {
	dir := t.TempDir()
	envData := `{"$shared":{"host":"example.com"},"dev":{}}`
	if err := os.WriteFile(filepath.Join(dir, "restless.env.json"), []byte(envData), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewRequestService()
	svc.SetRootDir(dir)

	resolved, err := svc.ResolvePreview(model.Request{
		Method: "GET",
		URL:    "https://{{host}}/api",
		Headers: []model.Header{
			{Key: "X-Custom", Value: "{{host}}"},
		},
	}, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.URL != "https://example.com/api" {
		t.Errorf("URL = %q, want %q", resolved.URL, "https://example.com/api")
	}
	if resolved.Headers[0].Value != "example.com" {
		t.Errorf("header = %q, want %q", resolved.Headers[0].Value, "example.com")
	}
}

func TestResolvePreview_WithChainVars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"token":"secret"}`)
	}))
	defer srv.Close()

	svc := NewRequestService()

	_, err := svc.Execute(model.Request{
		Name:   "auth",
		Method: "POST",
		URL:    srv.URL,
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := svc.ResolvePreview(model.Request{
		Method: "GET",
		URL:    srv.URL + "/protected",
		Headers: []model.Header{
			{Key: "Authorization", Value: "Bearer {{auth.response.body.token}}"},
		},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Headers[0].Value != "Bearer secret" {
		t.Errorf("header = %q, want %q", resolved.Headers[0].Value, "Bearer secret")
	}
}

func TestGetChainVariables_Empty(t *testing.T) {
	svc := NewRequestService()
	names := svc.GetChainVariables()
	if len(names) != 0 {
		t.Errorf("expected empty chain variables, got %v", names)
	}
}

func TestExecute_PreRequestScript(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("X-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := NewRequestService()
	_, err := svc.Execute(model.Request{
		Method:           "GET",
		URL:              srv.URL,
		PreRequestScript: `setHeader("X-Token", "generated-token")`,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if receivedAuth != "generated-token" {
		t.Errorf("X-Token = %q, want %q", receivedAuth, "generated-token")
	}
}

func TestExecute_PostResponseScript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":42}`)
	}))
	defer srv.Close()

	svc := NewRequestService()
	resp, err := svc.Execute(model.Request{
		Method:             "GET",
		URL:                srv.URL,
		PostResponseScript: `log("status: " + response.status)`,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestExecute_PostResponseScriptError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := NewRequestService()
	resp, err := svc.Execute(model.Request{
		Method:             "GET",
		URL:                srv.URL,
		PostResponseScript: `throw "boom"`,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.ScriptError == "" {
		t.Error("expected ScriptError to be set")
	}
}

func TestExecute_CookiePersistence(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "xyz"})
		}
		if callCount == 2 {
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value != "xyz" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := NewRequestService()

	_, err := svc.Execute(model.Request{Method: "GET", URL: srv.URL}, "test")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := svc.Execute(model.Request{Method: "GET", URL: srv.URL}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("second request status = %d, want 200 (cookie should persist)", resp.StatusCode)
	}
}

func TestExecute_FileVars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, _ := json.Marshal(map[string]string{"path": r.URL.Path})
		w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	httpContent := fmt.Sprintf("@apiBase = %s\n\nGET {{apiBase}}/items\n", srv.URL)
	httpFile := filepath.Join(dir, "test.http")
	if err := os.WriteFile(httpFile, []byte(httpContent), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewRequestService()
	svc.SetRootDir(dir)

	resp, err := svc.Execute(model.Request{
		Method:     "GET",
		URL:        "{{apiBase}}/items",
		SourceFile: httpFile,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
