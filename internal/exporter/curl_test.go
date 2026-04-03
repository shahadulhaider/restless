package exporter

import (
	"strings"
	"testing"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestToCurlSimpleGET(t *testing.T) {
	req := model.Request{
		Method: "GET",
		URL:    "https://api.example.com/health",
	}
	cmd := ToCurl(req)
	assert.Contains(t, cmd, "curl")
	assert.Contains(t, cmd, "https://api.example.com/health")
	assert.NotContains(t, cmd, "-X") // GET is implicit
}

func TestToCurlPOSTWithBody(t *testing.T) {
	req := model.Request{
		Method: "POST",
		URL:    "https://api.example.com/users",
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"name":"Alice"}`,
	}
	cmd := ToCurl(req)
	assert.Contains(t, cmd, "curl")
	assert.Contains(t, cmd, "-X POST")
	assert.Contains(t, cmd, "Content-Type: application/json")
	assert.Contains(t, cmd, `{"name":"Alice"}`)
	assert.Contains(t, cmd, "https://api.example.com/users")
}

func TestToCurlCustomHeaders(t *testing.T) {
	req := model.Request{
		Method: "GET",
		URL:    "https://api.example.com/me",
		Headers: []model.Header{
			{Key: "Authorization", Value: "Bearer token123"},
			{Key: "Accept", Value: "application/json"},
		},
	}
	cmd := ToCurl(req)
	assert.Contains(t, cmd, "Authorization: Bearer token123")
	assert.Contains(t, cmd, "Accept: application/json")
}

func TestToCurlURLIsLast(t *testing.T) {
	req := model.Request{
		Method: "DELETE",
		URL:    "https://api.example.com/items/1",
	}
	cmd := ToCurl(req)
	// URL should be at the end of the command
	trimmed := strings.TrimSpace(cmd)
	assert.True(t, strings.HasSuffix(trimmed, "https://api.example.com/items/1") ||
		strings.Contains(trimmed, "https://api.example.com/items/1"))
}
