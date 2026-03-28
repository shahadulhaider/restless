package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserSingleGET(t *testing.T) {
	input := "GET https://example.com/users\n"
	reqs, err := ParseBytes([]byte(input), "test.http")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "GET", reqs[0].Method)
	assert.Equal(t, "https://example.com/users", reqs[0].URL)
}

func TestParserPOSTWithBody(t *testing.T) {
	input := `POST https://example.com/users
Content-Type: application/json

{"name": "Alice"}
`
	reqs, err := ParseBytes([]byte(input), "test.http")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "POST", reqs[0].Method)
	assert.Equal(t, "application/json", reqs[0].Headers[0].Value)
	assert.Contains(t, reqs[0].Body, `"Alice"`)
}

func TestParserMultiRequest(t *testing.T) {
	input := `GET https://example.com/users

###

POST https://example.com/users
Content-Type: application/json

{}

###

DELETE https://example.com/users/1
`
	reqs, err := ParseBytes([]byte(input), "test.http")
	require.NoError(t, err)
	assert.Len(t, reqs, 3)
	assert.Equal(t, "GET", reqs[0].Method)
	assert.Equal(t, "POST", reqs[1].Method)
	assert.Equal(t, "DELETE", reqs[2].Method)
}

func TestParserMetadataTags(t *testing.T) {
	input := `# @name getUser
# @no-redirect
# @no-cookie-jar
# @timeout 30
# @connection-timeout 10
GET https://example.com/users/1
`
	reqs, err := ParseBytes([]byte(input), "test.http")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "getUser", reqs[0].Name)
	assert.True(t, reqs[0].Metadata.NoRedirect)
	assert.True(t, reqs[0].Metadata.NoCookieJar)
	assert.Equal(t, 30*time.Second, reqs[0].Metadata.Timeout)
	assert.Equal(t, 10*time.Second, reqs[0].Metadata.ConnTimeout)
}

func TestParserFileBodyRef(t *testing.T) {
	input := `POST https://example.com/users
Content-Type: application/json

< ./body.json
`
	reqs, err := ParseBytes([]byte(input), "test.http")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "./body.json", reqs[0].BodyFile)
}

func TestParserVariablesPreserved(t *testing.T) {
	input := `GET {{base_url}}/users/{{user_id}}
Authorization: Bearer {{token}}
`
	reqs, err := ParseBytes([]byte(input), "test.http")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "{{base_url}}/users/{{user_id}}", reqs[0].URL)
	assert.Equal(t, "Bearer {{token}}", reqs[0].Headers[0].Value)
}

func TestParserSourceTracking(t *testing.T) {
	input := "GET https://example.com\n"
	reqs, err := ParseBytes([]byte(input), "api/users.http")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "api/users.http", reqs[0].SourceFile)
}

func TestParserCRLF(t *testing.T) {
	input := "GET https://example.com/users\r\nContent-Type: application/json\r\n"
	reqs, err := ParseBytes([]byte(input), "test.http")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "Content-Type", reqs[0].Headers[0].Key)
}
