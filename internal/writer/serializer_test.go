package writer

import (
	"testing"
	"time"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializeSimpleGET(t *testing.T) {
	req := model.Request{Method: "GET", URL: "https://example.com/users"}
	out := SerializeRequest(req)
	assert.Equal(t, "GET https://example.com/users", out)
}

func TestSerializePOSTWithBody(t *testing.T) {
	req := model.Request{
		Method: "POST",
		URL:    "https://api.example.com/users",
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"name": "Alice"}`,
	}
	out := SerializeRequest(req)
	assert.Contains(t, out, "POST https://api.example.com/users")
	assert.Contains(t, out, "Content-Type: application/json")
	assert.Contains(t, out, `{"name": "Alice"}`)
	// blank line before body
	assert.Contains(t, out, "\n\n")
}

func TestSerializeWithMetadata(t *testing.T) {
	req := model.Request{
		Name:   "getUser",
		Method: "GET",
		URL:    "https://example.com/users/1",
		Metadata: model.RequestMetadata{
			NoRedirect:  true,
			NoCookieJar: true,
			Timeout:     30 * time.Second,
			ConnTimeout: 10 * time.Second,
		},
	}
	out := SerializeRequest(req)
	assert.Contains(t, out, "# @name getUser")
	assert.Contains(t, out, "# @no-redirect")
	assert.Contains(t, out, "# @no-cookie-jar")
	assert.Contains(t, out, "# @timeout 30")
	assert.Contains(t, out, "# @connection-timeout 10")
	assert.Contains(t, out, "GET https://example.com/users/1")
}

func TestSerializeFileBody(t *testing.T) {
	req := model.Request{
		Method:   "POST",
		URL:      "https://example.com/upload",
		BodyFile: "./payload.bin",
	}
	out := SerializeRequest(req)
	assert.Contains(t, out, "POST https://example.com/upload")
	assert.Contains(t, out, "< ./payload.bin")
}

func TestSerializeHTTPVersion(t *testing.T) {
	req := model.Request{
		Method:      "GET",
		URL:         "https://example.com",
		HTTPVersion: "HTTP/1.1",
	}
	out := SerializeRequest(req)
	assert.Equal(t, "GET https://example.com HTTP/1.1", out)
}

func TestSerializeNoBodyNoBlankLine(t *testing.T) {
	req := model.Request{
		Method: "GET",
		URL:    "https://example.com",
		Headers: []model.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}
	out := SerializeRequest(req)
	assert.Contains(t, out, "Accept: application/json")
	// No double newline if no body
	assert.NotContains(t, out, "\n\n")
}

func TestSerializeMultipleRequests(t *testing.T) {
	reqs := []model.Request{
		{Method: "GET", URL: "https://example.com/a"},
		{Method: "POST", URL: "https://example.com/b"},
		{Method: "DELETE", URL: "https://example.com/c"},
	}
	out := SerializeRequests(reqs)
	assert.Contains(t, out, "GET https://example.com/a")
	assert.Contains(t, out, "POST https://example.com/b")
	assert.Contains(t, out, "DELETE https://example.com/c")
	// Two ### separators for 3 requests
	assert.Equal(t, 2, countOccurrences(out, "###"))
}

func TestRoundTripGET(t *testing.T) {
	req := model.Request{
		Name:   "getUsers",
		Method: "GET",
		URL:    "https://api.example.com/users",
		Headers: []model.Header{
			{Key: "Accept", Value: "application/json"},
		},
	}
	serialized := SerializeRequest(req)
	parsed, err := parser.ParseBytes([]byte(serialized), "test.http")
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	assert.Equal(t, req.Name, parsed[0].Name)
	assert.Equal(t, req.Method, parsed[0].Method)
	assert.Equal(t, req.URL, parsed[0].URL)
	assert.Equal(t, req.Headers[0].Key, parsed[0].Headers[0].Key)
	assert.Equal(t, req.Headers[0].Value, parsed[0].Headers[0].Value)
}

func TestRoundTripPOST(t *testing.T) {
	req := model.Request{
		Method: "POST",
		URL:    "https://api.example.com/users",
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"name": "Alice", "age": 30}`,
	}
	serialized := SerializeRequest(req)
	parsed, err := parser.ParseBytes([]byte(serialized), "test.http")
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	assert.Equal(t, "POST", parsed[0].Method)
	assert.Equal(t, req.URL, parsed[0].URL)
	assert.Contains(t, parsed[0].Body, `"Alice"`)
}

func TestRoundTripMetadata(t *testing.T) {
	req := model.Request{
		Name:   "testReq",
		Method: "GET",
		URL:    "https://example.com",
		Metadata: model.RequestMetadata{
			NoRedirect:  true,
			NoCookieJar: true,
			Timeout:     15 * time.Second,
			ConnTimeout: 5 * time.Second,
		},
	}
	serialized := SerializeRequest(req)
	parsed, err := parser.ParseBytes([]byte(serialized), "test.http")
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	assert.Equal(t, "testReq", parsed[0].Name)
	assert.True(t, parsed[0].Metadata.NoRedirect)
	assert.True(t, parsed[0].Metadata.NoCookieJar)
	assert.Equal(t, 15*time.Second, parsed[0].Metadata.Timeout)
	assert.Equal(t, 5*time.Second, parsed[0].Metadata.ConnTimeout)
}

func TestRoundTripMultiRequest(t *testing.T) {
	reqs := []model.Request{
		{Method: "GET", URL: "https://example.com/a"},
		{Method: "POST", URL: "https://example.com/b", Body: `{"key":"val"}`},
		{Name: "third", Method: "DELETE", URL: "https://example.com/c/1"},
	}
	serialized := SerializeRequests(reqs)
	parsed, err := parser.ParseBytes([]byte(serialized), "test.http")
	require.NoError(t, err)
	require.Len(t, parsed, 3)
	assert.Equal(t, "GET", parsed[0].Method)
	assert.Equal(t, "POST", parsed[1].Method)
	assert.Equal(t, "DELETE", parsed[2].Method)
	assert.Equal(t, "third", parsed[2].Name)
}

func countOccurrences(s, sub string) int {
	count := 0
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			count++
		}
	}
	return count
}
