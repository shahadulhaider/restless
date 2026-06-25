package parser

import (
	"testing"
	"time"

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
