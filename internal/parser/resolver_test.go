package parser

import (
	"testing"
	"time"

	asrt "github.com/shahadulhaider/restless/internal/assert"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRequestChainStatusDurationSize(t *testing.T) {
	chainCtx := NewChainContext()
	chainCtx.StoreResponse("login", &model.Response{
		StatusCode: 201,
		Body:       []byte(`{"ok":true}`),
		Timing:     model.ResponseTiming{Total: 150 * time.Millisecond},
	})

	req := &model.Request{
		Method: "GET",
		URL:    "https://api/{{login.response.status}}",
		Headers: []model.Header{
			{Key: "X-Duration", Value: "{{login.response.duration}}"},
			{Key: "X-Size", Value: "{{login.response.size}}"},
		},
	}

	resolved, err := ResolveRequest(req, nil, chainCtx)
	require.NoError(t, err)
	assert.Equal(t, "https://api/201", resolved.URL)
	assert.Equal(t, "150", resolved.Headers[0].Value)
	assert.Equal(t, "11", resolved.Headers[1].Value)
}

func TestResolveRequestInterpolatesAssertionExpected(t *testing.T) {
	vars := map[string]string{"expectedToken": "jwt-abc"}
	req := &model.Request{
		Method: "GET",
		URL:    "https://api/login",
		Assertions: []model.Assertion{
			{Target: "body.$.token", Operator: "==", Expected: "{{expectedToken}}", Raw: "body.$.token == {{expectedToken}}"},
			{Target: "status", Operator: "==", Expected: "200", Raw: "status == 200"},
		},
	}

	resolved, err := ResolveRequest(req, vars, nil)
	require.NoError(t, err)

	assert.Equal(t, "jwt-abc", resolved.Assertions[0].Expected)
	assert.Equal(t, "200", resolved.Assertions[1].Expected)
	assert.Equal(t, "{{expectedToken}}", req.Assertions[0].Expected)

	resp := &model.Response{StatusCode: 200, Body: []byte(`{"token":"jwt-abc"}`)}
	res := asrt.Evaluate(resolved.Assertions[0], resp)
	assert.True(t, res.Passed)
}
