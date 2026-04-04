package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractFileVariables(t *testing.T) {
	content := []byte(`@baseUrl = https://api.example.com
@token = my-secret-token
@port = 8080

# @name health
GET {{baseUrl}}:{{port}}/health
Authorization: Bearer {{token}}
`)

	vars := ExtractFileVariables(content)
	assert.Equal(t, "https://api.example.com", vars["baseUrl"])
	assert.Equal(t, "my-secret-token", vars["token"])
	assert.Equal(t, "8080", vars["port"])
	assert.Len(t, vars, 3)
}

func TestExtractFileVariablesEmpty(t *testing.T) {
	content := []byte(`GET https://example.com/health
`)
	vars := ExtractFileVariables(content)
	assert.Empty(t, vars)
}

func TestExtractFileVariablesSpacing(t *testing.T) {
	content := []byte(`@foo=bar
@baz =  qux
@key  =value
`)
	vars := ExtractFileVariables(content)
	assert.Equal(t, "bar", vars["foo"])
	assert.Equal(t, "qux", vars["baz"])
	assert.Equal(t, "value", vars["key"])
}

func TestLexerSkipsFileVariables(t *testing.T) {
	content := []byte(`@baseUrl = https://example.com

GET {{baseUrl}}/health
`)
	lexer := NewLexer(content)
	tokens := lexer.Tokenize()

	// Should not produce any token for the @baseUrl line
	for _, tok := range tokens {
		assert.NotContains(t, tok.Value, "@baseUrl")
	}
	// Should still parse the GET request
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenMethod && tok.Value == "GET" {
			found = true
		}
	}
	assert.True(t, found, "GET method should still be parsed")
}
