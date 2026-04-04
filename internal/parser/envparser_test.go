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

	assert.Equal(t, "http://localhost:3000", vars["base_url"])
	assert.Equal(t, "30", vars["timeout"])
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

	assert.Equal(t, "private-token", vars["token"])
	assert.Equal(t, "http://localhost", vars["base_url"])
}

func TestEnvMissingFiles(t *testing.T) {
	dir := t.TempDir()

	envFile, err := LoadEnvironments(dir)
	require.NoError(t, err)
	assert.NotNil(t, envFile)
	assert.Empty(t, envFile.Environments)
}

func TestEnvDynamicVars(t *testing.T) {
	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	uuid := ResolveDynamicVars("{{$uuid}}")
	assert.Regexp(t, uuidRe, uuid)

	ts := ResolveDynamicVars("{{$timestamp}}")
	n, err := strconv.ParseInt(ts, 10, 64)
	require.NoError(t, err)
	assert.Greater(t, n, int64(0))

	ri := ResolveDynamicVars("{{$randomInt}}")
	rn, err := strconv.Atoi(ri)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, rn, 0)
	assert.Less(t, rn, 1000)

	iso := ResolveDynamicVars("{{$isoTimestamp}}")
	assert.Contains(t, iso, "T")

	date := ResolveDynamicVars("{{$date}}")
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, date)

	email := ResolveDynamicVars("{{$randomEmail}}")
	assert.Contains(t, email, "@example.com")

	name := ResolveDynamicVars("{{$randomName}}")
	assert.NotEmpty(t, name)

	rf := ResolveDynamicVars("{{$randomFloat}}")
	_, err = strconv.ParseFloat(rf, 64)
	require.NoError(t, err)

	rb := ResolveDynamicVars("{{$randomBool}}")
	assert.True(t, rb == "true" || rb == "false")
}

func TestSharedEnvKey(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "http-client.env.json", map[string]map[string]string{
		"$shared": {"host": "shared.example.com", "token": "shared-tok"},
		"dev":     {"token": "dev-tok"},
	})
	ef, err := LoadEnvironments(dir)
	require.NoError(t, err)

	vars, err := ResolveEnvironment(ef, "dev")
	require.NoError(t, err)
	assert.Equal(t, "shared.example.com", vars["host"])
	assert.Equal(t, "dev-tok", vars["token"])
}
