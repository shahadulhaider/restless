package chain

import (
	"testing"
	"time"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loginResponse() *model.Response {
	return &model.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       []byte(`{"access_token":"jwt-123","user":{"id":42,"name":"Alice"}}`),
		Headers:    []model.Header{{Key: "X-Request-Id", Value: "abc-123"}, {Key: "Content-Type", Value: "application/json"}},
		Timestamp:  time.Now(),
	}
}

func TestStoreAndResolveBodyPath(t *testing.T) {
	ctx := NewChainContext()
	ctx.StoreResponse("login", loginResponse())

	val, err := ctx.Resolve("login.response.body.access_token")
	require.NoError(t, err)
	assert.Equal(t, "jwt-123", val)
}

func TestResolveNestedBodyPath(t *testing.T) {
	ctx := NewChainContext()
	ctx.StoreResponse("login", loginResponse())

	val, err := ctx.Resolve("login.response.body.user.id")
	require.NoError(t, err)
	assert.Equal(t, "42", val)
}

func TestResolveHeader(t *testing.T) {
	ctx := NewChainContext()
	ctx.StoreResponse("login", loginResponse())

	val, err := ctx.Resolve("login.response.headers.X-Request-Id")
	require.NoError(t, err)
	assert.Equal(t, "abc-123", val)
}

func TestResolveHeaderCaseInsensitive(t *testing.T) {
	ctx := NewChainContext()
	ctx.StoreResponse("login", loginResponse())

	val, err := ctx.Resolve("login.response.headers.x-request-id")
	require.NoError(t, err)
	assert.Equal(t, "abc-123", val)
}

func TestResolveUnknownRequest(t *testing.T) {
	ctx := NewChainContext()

	_, err := ctx.Resolve("missing.response.body.token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestResolveUnknownBodyPath(t *testing.T) {
	ctx := NewChainContext()
	ctx.StoreResponse("login", loginResponse())

	_, err := ctx.Resolve("login.response.body.nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestResolveMultipleResponses(t *testing.T) {
	ctx := NewChainContext()
	ctx.StoreResponse("login", loginResponse())
	ctx.StoreResponse("profile", &model.Response{
		Body: []byte(`{"username":"bob"}`),
	})

	token, err := ctx.Resolve("login.response.body.access_token")
	require.NoError(t, err)
	assert.Equal(t, "jwt-123", token)

	name, err := ctx.Resolve("profile.response.body.username")
	require.NoError(t, err)
	assert.Equal(t, "bob", name)
}

func TestHasResponse(t *testing.T) {
	ctx := NewChainContext()
	assert.False(t, ctx.HasResponse("login"))
	ctx.StoreResponse("login", loginResponse())
	assert.True(t, ctx.HasResponse("login"))
}
