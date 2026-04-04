package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestParseAssertions(t *testing.T) {
	content := []byte(`# @name test
GET https://example.com/api
# @assert status == 200
# @assert body.$.id != null
# @assert header.Content-Type contains json
# @assert duration < 2000
# @assert body contains "hello"
# @assert header.X-Id exists
`)
	reqs, err := ParseBytes(content, "test.http")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Len(t, reqs[0].Assertions, 6)

	a := reqs[0].Assertions
	assert.Equal(t, "status", a[0].Target)
	assert.Equal(t, "==", a[0].Operator)
	assert.Equal(t, "200", a[0].Expected)

	assert.Equal(t, "body.$.id", a[1].Target)
	assert.Equal(t, "!=", a[1].Operator)
	assert.Equal(t, "null", a[1].Expected)

	assert.Equal(t, "header.Content-Type", a[2].Target)
	assert.Equal(t, "contains", a[2].Operator)
	assert.Equal(t, "json", a[2].Expected)

	assert.Equal(t, "duration", a[3].Target)
	assert.Equal(t, "<", a[3].Operator)

	assert.Equal(t, "header.X-Id", a[5].Target)
	assert.Equal(t, "exists", a[5].Operator)
}

func TestParseScriptBlocks(t *testing.T) {
	content := []byte(`# @name test
# @pre-request {
#   setHeader("X-Timestamp", String(timestamp()));
#   var sig = hmac_sha256(env.secret, request.body);
#   setHeader("X-Sig", sig);
# }
POST https://example.com/api
Content-Type: application/json

{"data": "value"}

# @post-response {
#   if (response.status === 200) {
#     setVar("token", response.body.token);
#   }
# }
# @assert status == 200
`)
	reqs, err := ParseBytes(content, "test.http")
	require.NoError(t, err)
	require.Len(t, reqs, 1)

	assert.NotEmpty(t, reqs[0].PreRequestScript, "pre-request script should be parsed")
	assert.Contains(t, reqs[0].PreRequestScript, "setHeader")
	assert.Contains(t, reqs[0].PreRequestScript, "hmac_sha256")

	assert.NotEmpty(t, reqs[0].PostResponseScript, "post-response script should be parsed")
	assert.Contains(t, reqs[0].PostResponseScript, "setVar")
	assert.Contains(t, reqs[0].PostResponseScript, "response.body.token")

	assert.Len(t, reqs[0].Assertions, 1)
}
