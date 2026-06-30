package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnvFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadEnvironments(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "restless.env.json", `{
		"$shared": {"baseUrl": "http://localhost"},
		"dev":     {"port": "3000"},
		"prod":    {"port": "443"}
	}`)

	svc := &EnvironmentService{}
	envFile, err := svc.LoadEnvironments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if envFile.Shared["baseUrl"] != "http://localhost" {
		t.Errorf("shared baseUrl = %q, want %q", envFile.Shared["baseUrl"], "http://localhost")
	}
	if len(envFile.Environments) != 2 {
		t.Errorf("got %d environments, want 2", len(envFile.Environments))
	}
}

func TestLoadEnvironments_NoFile(t *testing.T) {
	dir := t.TempDir()
	svc := &EnvironmentService{}
	envFile, err := svc.LoadEnvironments(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(envFile.Environments) != 0 {
		t.Errorf("got %d environments, want 0", len(envFile.Environments))
	}
}

func TestListEnvironmentNames(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "restless.env.json", `{
		"$shared": {},
		"beta":    {"key": "b"},
		"alpha":   {"key": "a"},
		"gamma":   {"key": "g"}
	}`)

	svc := &EnvironmentService{}
	names, err := svc.ListEnvironmentNames(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "beta", "gamma"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestResolveVars(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "restless.env.json", `{
		"$shared": {"baseUrl": "http://shared"},
		"dev":     {"port": "3000", "baseUrl": "http://dev"}
	}`)

	svc := &EnvironmentService{}
	vars, err := svc.ResolveVars(dir, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if vars["baseUrl"] != "http://dev" {
		t.Errorf("baseUrl = %q, want %q", vars["baseUrl"], "http://dev")
	}
	if vars["port"] != "3000" {
		t.Errorf("port = %q, want %q", vars["port"], "3000")
	}
}

func TestResolveVars_SharedOnly(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "restless.env.json", `{
		"$shared": {"key": "value"},
		"dev":     {}
	}`)

	svc := &EnvironmentService{}
	vars, err := svc.ResolveVars(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if vars["key"] != "value" {
		t.Errorf("key = %q, want %q", vars["key"], "value")
	}
}

func TestResolveVars_NotFound(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "restless.env.json", `{"dev": {"k": "v"}}`)

	svc := &EnvironmentService{}
	_, err := svc.ResolveVars(dir, "staging")
	if err == nil {
		t.Fatal("expected error for missing environment")
	}
}

func TestGetSetCurrentEnv(t *testing.T) {
	svc := &EnvironmentService{}
	if got := svc.GetCurrentEnv(); got != "" {
		t.Errorf("initial currentEnv = %q, want empty", got)
	}
	svc.SetCurrentEnv("production")
	if got := svc.GetCurrentEnv(); got != "production" {
		t.Errorf("currentEnv = %q, want %q", got, "production")
	}
}
