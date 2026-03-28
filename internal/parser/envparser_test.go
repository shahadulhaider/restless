package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeEnvFile(t *testing.T, dir, name string, content map[string]map[string]string) {
	t.Helper()
	data, err := json.Marshal(content)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0600))
}

func TestEnvBasicParsing(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "http-client.env.json", map[string]map[string]string{
		"$shared": {"base_url": "https://api.example.com"},
		"dev":     {"token": "dev-123", "base_url": "http://localhost:3000"},
		"prod":    {"token": "prod-abc"},
	})

	envFile, err := LoadEnvironments(dir)
	require.NoError(t, err)
	assert.Contains(t, envFile.Environments, "dev")
	assert.Contains(t, envFile.Environments, "prod")
}

func TestEnvSharedMerge(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "http-client.env.json", map[string]map[string]string{
		"$shared": {"base_url": "https://shared.example.com", "timeout": "30"},
		"dev":     {"base_url": "http://localhost:3000"},
	})

	envFile, err := LoadEnvironments(dir)
	require.NoError(t, err)

	vars, err := ResolveEnvironment(envFile, "dev")
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:3000", vars["base_url"], "env-specific should override shared")
	assert.Equal(t, "30", vars["timeout"], "shared var should be present when not overridden")
}

func TestEnvPrivateOverride(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "http-client.env.json", map[string]map[string]string{
		"dev": {"token": "public-token", "base_url": "http://localhost"},
	})
	writeEnvFile(t, dir, "http-client.private.env.json", map[string]map[string]string{
		"dev": {"token": "private-token"},
	})

	envFile, err := LoadEnvironments(dir)
	require.NoError(t, err)

	vars, err := ResolveEnvironment(envFile, "dev")
	require.NoError(t, err)

	assert.Equal(t, "private-token", vars["token"], "private env vars should override public")
	assert.Equal(t, "http://localhost", vars["base_url"], "public var present when not in private")
}

func TestEnvMissingFiles(t *testing.T) {
	dir := t.TempDir()

	envFile, err := LoadEnvironments(dir)
	require.NoError(t, err, "missing env files should not error")
	assert.NotNil(t, envFile)
	assert.Empty(t, envFile.Environments)
}

func TestEnvDynamicVars(t *testing.T) {
	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	uuid := ResolveDynamicVars("{{$uuid}}")
	assert.Regexp(t, uuidRe, uuid, "uuid should match UUID format")

	ts := ResolveDynamicVars("{{$timestamp}}")
	n, err := strconv.ParseInt(ts, 10, 64)
	require.NoError(t, err, "timestamp should be numeric")
	assert.Greater(t, n, int64(0))

	ri := ResolveDynamicVars("{{$randomInt}}")
	rn, err := strconv.Atoi(ri)
	require.NoError(t, err, "randomInt should be an integer")
	assert.GreaterOrEqual(t, rn, 0)
	assert.Less(t, rn, 1000)
}
