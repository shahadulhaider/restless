package exporter

import (
	"testing"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/stretchr/testify/assert"
)

var simpleGET = model.Request{
	Method: "GET",
	URL:    "https://api.example.com/health",
}

var postJSON = model.Request{
	Method: "POST",
	URL:    "https://api.example.com/users",
	Headers: []model.Header{
		{Key: "Content-Type", Value: "application/json"},
		{Key: "Authorization", Value: "Bearer token123"},
	},
	Body: `{"name":"Alice"}`,
}

var deleteReq = model.Request{
	Method: "DELETE",
	URL:    "https://api.example.com/items/42",
}

func TestGeneratePython(t *testing.T) {
	code := GeneratePython(postJSON)
	assert.Contains(t, code, "import requests")
	assert.Contains(t, code, "requests.post(")
	assert.Contains(t, code, "https://api.example.com/users")
	assert.Contains(t, code, "Content-Type")
	assert.Contains(t, code, "Bearer token123")
	assert.Contains(t, code, "Alice")

	simple := GeneratePython(simpleGET)
	assert.Contains(t, simple, "requests.get(")
	assert.NotContains(t, simple, "data=")
	assert.NotContains(t, simple, "headers=")
}

func TestGenerateJavaScript(t *testing.T) {
	code := GenerateJavaScript(postJSON)
	assert.Contains(t, code, "fetch(")
	assert.Contains(t, code, `method: "POST"`)
	assert.Contains(t, code, "JSON.stringify")
	assert.Contains(t, code, "Content-Type")

	simple := GenerateJavaScript(simpleGET)
	assert.Contains(t, simple, "fetch(")
	assert.NotContains(t, simple, "method:")
}

func TestGenerateGo(t *testing.T) {
	code := GenerateGo(postJSON)
	assert.Contains(t, code, "http.NewRequest")
	assert.Contains(t, code, `"POST"`)
	assert.Contains(t, code, "strings.NewReader")
	assert.Contains(t, code, "Header.Set")
	assert.Contains(t, code, "Content-Type")

	simple := GenerateGo(simpleGET)
	assert.Contains(t, simple, `"GET"`)
	assert.Contains(t, simple, "nil)")
}

func TestGenerateJava(t *testing.T) {
	code := GenerateJava(postJSON)
	assert.Contains(t, code, "HttpClient")
	assert.Contains(t, code, "HttpRequest")
	assert.Contains(t, code, `URI.create("https://api.example.com/users")`)
	assert.Contains(t, code, "BodyPublishers.ofString")
	assert.Contains(t, code, ".header(")

	simple := GenerateJava(simpleGET)
	assert.Contains(t, simple, ".GET()")
}

func TestGenerateRuby(t *testing.T) {
	code := GenerateRuby(postJSON)
	assert.Contains(t, code, "Net::HTTP")
	assert.Contains(t, code, "Net::HTTP::Post")
	assert.Contains(t, code, "request.body")
	assert.Contains(t, code, "Content-Type")

	simple := GenerateRuby(simpleGET)
	assert.Contains(t, simple, "Net::HTTP::Get")
	assert.NotContains(t, simple, "request.body")
}

func TestGenerateHTTPie(t *testing.T) {
	code := GenerateHTTPie(postJSON)
	assert.Contains(t, code, "http")
	assert.Contains(t, code, "POST")
	assert.Contains(t, code, "https://api.example.com/users")

	simple := GenerateHTTPie(simpleGET)
	assert.Contains(t, simple, "http")
	assert.Contains(t, simple, "https://api.example.com/health")
	assert.NotContains(t, simple, "POST")
}

func TestGeneratePowerShell(t *testing.T) {
	code := GeneratePowerShell(postJSON)
	assert.Contains(t, code, "Invoke-RestMethod")
	assert.Contains(t, code, "-Method POST")
	assert.Contains(t, code, "$headers")
	assert.Contains(t, code, "$body")

	simple := GeneratePowerShell(simpleGET)
	assert.Contains(t, simple, "-Method GET")
	assert.NotContains(t, simple, "$body")
}

func TestGeneratorsRegistry(t *testing.T) {
	assert.Len(t, Generators, 8)
	for key, gen := range Generators {
		code := gen.Generate(postJSON)
		assert.NotEmpty(t, code, "generator %s (%s) should produce output", key, gen.Name)
	}
}
