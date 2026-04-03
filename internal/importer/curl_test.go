package importer

import (
	"testing"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCurlSimpleGET(t *testing.T) {
	req, err := ParseCurlCommand("curl https://api.example.com/health")
	require.NoError(t, err)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "https://api.example.com/health", req.URL)
	assert.Empty(t, req.Headers)
	assert.Empty(t, req.Body)
}

func TestParseCurlPOSTWithBody(t *testing.T) {
	cmd := `curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"Alice"}'`
	req, err := ParseCurlCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "https://api.example.com/users", req.URL)
	assert.Equal(t, `{"name":"Alice"}`, req.Body)
	require.Len(t, req.Headers, 1)
	assert.Equal(t, "Content-Type", req.Headers[0].Key)
	assert.Equal(t, "application/json", req.Headers[0].Value)
}

func TestParseCurlWithHeaders(t *testing.T) {
	cmd := `curl -H "Authorization: Bearer token123" -H "Accept: application/json" https://api.example.com/me`
	req, err := ParseCurlCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "GET", req.Method)
	require.Len(t, req.Headers, 2)
	assert.Equal(t, "Authorization", req.Headers[0].Key)
	assert.Equal(t, "Bearer token123", req.Headers[0].Value)
	assert.Equal(t, "Accept", req.Headers[1].Key)
	assert.Equal(t, "application/json", req.Headers[1].Value)
}

func TestParseCurlBasicAuth(t *testing.T) {
	cmd := `curl -u admin:secret https://api.example.com/admin`
	req, err := ParseCurlCommand(cmd)
	require.NoError(t, err)
	require.Len(t, req.Headers, 1)
	assert.Equal(t, "Authorization", req.Headers[0].Key)
	assert.Equal(t, "Basic admin:secret", req.Headers[0].Value)
}

func TestParseCurlDataImpliesPost(t *testing.T) {
	cmd := `curl https://api.example.com/data -d "payload"`
	req, err := ParseCurlCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "payload", req.Body)
}

func TestParseCurlLineContinuation(t *testing.T) {
	cmd := "curl \\\n  -X DELETE \\\n  https://api.example.com/items/1"
	req, err := ParseCurlCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "https://api.example.com/items/1", req.URL)
}

func TestParseCurlURLFlag(t *testing.T) {
	cmd := `curl --url https://api.example.com/resource -X PATCH`
	req, err := ParseCurlCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, "PATCH", req.Method)
	assert.Equal(t, "https://api.example.com/resource", req.URL)
}

func TestParseCurlInvalidNotCurl(t *testing.T) {
	_, err := ParseCurlCommand("wget https://example.com")
	assert.Error(t, err)
}

func TestParseCurlMissingURL(t *testing.T) {
	_, err := ParseCurlCommand("curl -X GET")
	assert.Error(t, err)
}

func TestGenerateCurl(t *testing.T) {
	req := model.Request{
		Method: "POST",
		URL:    "https://api.example.com/users",
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"name":"Alice"}`,
	}
	cmd := GenerateCurl(req)
	assert.Contains(t, cmd, "curl")
	assert.Contains(t, cmd, "-X POST")
	assert.Contains(t, cmd, "Content-Type: application/json")
	assert.Contains(t, cmd, `{"name":"Alice"}`)
	assert.Contains(t, cmd, "https://api.example.com/users")
}

func TestGenerateCurlGET(t *testing.T) {
	req := model.Request{
		Method: "GET",
		URL:    "https://api.example.com/health",
	}
	cmd := GenerateCurl(req)
	assert.Contains(t, cmd, "curl")
	assert.NotContains(t, cmd, "-X")
	assert.Contains(t, cmd, "https://api.example.com/health")
}

func TestGenerateCurlRoundTrip(t *testing.T) {
	original := model.Request{
		Method: "PUT",
		URL:    "https://api.example.com/items/42",
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "Authorization", Value: "Bearer my-token"},
		},
		Body: `{"status":"active"}`,
	}
	cmd := GenerateCurl(original)

	parsed, err := ParseCurlCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, original.Method, parsed.Method)
	assert.Equal(t, original.URL, parsed.URL)
	assert.Equal(t, original.Body, parsed.Body)
	require.Len(t, parsed.Headers, 2)
}
