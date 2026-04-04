package script

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreRequestSetHeader(t *testing.T) {
	req := &model.Request{
		Method:  "POST",
		URL:     "https://api.example.com",
		Headers: []model.Header{{Key: "Content-Type", Value: "application/json"}},
		Body:    `{"key":"value"}`,
	}
	ctx := &ScriptContext{
		Request: req,
		EnvVars: map[string]string{"secret": "my-key"},
	}

	script := `setHeader("X-Custom", "hello"); setHeader("Content-Type", "text/plain");`
	err := RunPreRequest(script, ctx)
	require.NoError(t, err)

	// Should have added X-Custom and modified Content-Type
	found := map[string]string{}
	for _, h := range req.Headers {
		found[h.Key] = h.Value
	}
	assert.Equal(t, "hello", found["X-Custom"])
	assert.Equal(t, "text/plain", found["Content-Type"])
}

func TestPreRequestSetBody(t *testing.T) {
	req := &model.Request{
		Method: "POST",
		URL:    "https://api.example.com",
		Body:   `{"original": true}`,
	}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}}

	script := `
		var body = JSON.parse(request.body);
		body.timestamp = timestamp();
		body.nonce = uuid();
		setBody(JSON.stringify(body));
	`
	err := RunPreRequest(script, ctx)
	require.NoError(t, err)
	assert.Contains(t, req.Body, "timestamp")
	assert.Contains(t, req.Body, "nonce")
	assert.Contains(t, req.Body, "original")
}

func TestPreRequestSetUrl(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://old.example.com/path"}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}}

	err := RunPreRequest(`setUrl("https://new.example.com/path");`, ctx)
	require.NoError(t, err)
	assert.Equal(t, "https://new.example.com/path", req.URL)
}

func TestPreRequestRemoveHeader(t *testing.T) {
	req := &model.Request{
		Method: "GET",
		URL:    "https://api.example.com",
		Headers: []model.Header{
			{Key: "Authorization", Value: "Bearer old"},
			{Key: "Accept", Value: "application/json"},
		},
	}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}}

	err := RunPreRequest(`removeHeader("Authorization");`, ctx)
	require.NoError(t, err)
	assert.Len(t, req.Headers, 1)
	assert.Equal(t, "Accept", req.Headers[0].Key)
}

func TestPreRequestEnvAccess(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://api.example.com"}
	ctx := &ScriptContext{
		Request: req,
		EnvVars: map[string]string{"apiKey": "secret123"},
	}

	err := RunPreRequest(`setHeader("X-Api-Key", env.apiKey);`, ctx)
	require.NoError(t, err)
	assert.Equal(t, "secret123", req.Headers[0].Value)
}

func TestPreRequestHmac(t *testing.T) {
	req := &model.Request{Method: "POST", URL: "https://api.example.com", Body: "data"}
	ctx := &ScriptContext{
		Request: req,
		EnvVars: map[string]string{"secret": "key123"},
	}

	script := `
		var sig = hmac_sha256(env.secret, request.body);
		setHeader("X-Signature", sig);
	`
	err := RunPreRequest(script, ctx)
	require.NoError(t, err)
	assert.Len(t, req.Headers, 1)
	assert.Equal(t, "X-Signature", req.Headers[0].Key)
	assert.Len(t, req.Headers[0].Value, 64) // SHA256 hex = 64 chars
}

func TestPreRequestSetVar(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://api.example.com"}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}}

	err := RunPreRequest(`setVar("myVar", "myValue");`, ctx)
	require.NoError(t, err)
	assert.Equal(t, "myValue", ctx.SetVars["myVar"])
}

func TestPreRequestLog(t *testing.T) {
	var buf bytes.Buffer
	req := &model.Request{Method: "GET", URL: "https://api.example.com"}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}, LogOut: &buf}

	err := RunPreRequest(`log("hello from script");`, ctx)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "hello from script")
}

func TestPreRequestSyntaxError(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://api.example.com"}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}}

	err := RunPreRequest(`this is not valid javascript!!!`, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "script error")
}

func TestPreRequestRuntimeError(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://api.example.com"}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}}

	err := RunPreRequest(`undefinedFunction();`, ctx)
	assert.Error(t, err)
}

func TestPostResponseReadBody(t *testing.T) {
	req := &model.Request{Method: "POST", URL: "https://api.example.com"}
	resp := &model.Response{
		StatusCode: 200,
		Body:       []byte(`{"token":"abc123","user":{"id":42}}`),
		Headers:    []model.Header{{Key: "Content-Type", Value: "application/json"}},
	}
	ctx := &ScriptContext{Request: req, Response: resp, EnvVars: map[string]string{}}

	script := `
		setVar("token", response.body.token);
		setVar("userId", String(response.body.user.id));
		setVar("status", String(response.status));
	`
	err := RunPostResponse(script, ctx)
	require.NoError(t, err)
	assert.Equal(t, "abc123", ctx.SetVars["token"])
	assert.Equal(t, "42", ctx.SetVars["userId"])
	assert.Equal(t, "200", ctx.SetVars["status"])
}

func TestPostResponseHeaders(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://api.example.com"}
	resp := &model.Response{
		StatusCode: 200,
		Body:       []byte("ok"),
		Headers:    []model.Header{{Key: "X-Request-Id", Value: "req-456"}},
	}
	ctx := &ScriptContext{Request: req, Response: resp, EnvVars: map[string]string{}}

	err := RunPostResponse(`setVar("reqId", response.headers["X-Request-Id"]);`, ctx)
	require.NoError(t, err)
	assert.Equal(t, "req-456", ctx.SetVars["reqId"])
}

func TestPostResponseSetVarPropagates(t *testing.T) {
	req := &model.Request{Method: "POST", URL: "https://api.example.com"}
	resp := &model.Response{
		StatusCode: 200,
		Body:       []byte(`{"access_token":"tok123","refresh_token":"ref456"}`),
	}
	ctx := &ScriptContext{Request: req, Response: resp, EnvVars: map[string]string{}}

	script := `
		if (response.status === 200) {
			setVar("access_token", response.body.access_token);
			setVar("refresh_token", response.body.refresh_token);
		}
	`
	err := RunPostResponse(script, ctx)
	require.NoError(t, err)
	assert.Equal(t, "tok123", ctx.SetVars["access_token"])
	assert.Equal(t, "ref456", ctx.SetVars["refresh_token"])
}

func TestBuiltinBase64(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://api.example.com"}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}}

	script := `
		var encoded = base64Encode("hello:world");
		setVar("encoded", encoded);
		setVar("decoded", base64Decode(encoded));
	`
	err := RunPreRequest(script, ctx)
	require.NoError(t, err)
	assert.Equal(t, "aGVsbG86d29ybGQ=", ctx.SetVars["encoded"])
	assert.Equal(t, "hello:world", ctx.SetVars["decoded"])
}

func TestBuiltinSha256(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://api.example.com"}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}}

	err := RunPreRequest(`setVar("hash", sha256("test"));`, ctx)
	require.NoError(t, err)
	assert.Len(t, ctx.SetVars["hash"], 64)
	assert.True(t, strings.HasPrefix(ctx.SetVars["hash"], "9f86d"))
}

func TestBuiltinMd5(t *testing.T) {
	req := &model.Request{Method: "GET", URL: "https://api.example.com"}
	ctx := &ScriptContext{Request: req, EnvVars: map[string]string{}}

	err := RunPreRequest(`setVar("hash", md5("test"));`, ctx)
	require.NoError(t, err)
	assert.Equal(t, "098f6bcd4621d373cade4e832627b4f6", ctx.SetVars["hash"])
}
