package assert

import (
	"testing"
	"time"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/stretchr/testify/assert"
)

func makeResp(statusCode int, body string, headers map[string]string, totalMs int64) *model.Response {
	var hdrs []model.Header
	for k, v := range headers {
		hdrs = append(hdrs, model.Header{Key: k, Value: v})
	}
	return &model.Response{
		StatusCode:  statusCode,
		Status:      "",
		Headers:     hdrs,
		Body:        []byte(body),
		ContentType: headers["Content-Type"],
		Timing:      model.ResponseTiming{Total: time.Duration(totalMs) * time.Millisecond},
	}
}

func TestStatusEquals(t *testing.T) {
	resp := makeResp(200, "", nil, 100)
	r := Evaluate(model.Assertion{Target: "status", Operator: "==", Expected: "200"}, resp)
	assert.True(t, r.Passed)
	assert.Equal(t, "200", r.Actual)
}

func TestStatusNotEquals(t *testing.T) {
	resp := makeResp(404, "", nil, 100)
	r := Evaluate(model.Assertion{Target: "status", Operator: "==", Expected: "200"}, resp)
	assert.False(t, r.Passed)
	assert.Equal(t, "404", r.Actual)
}

func TestStatusRange(t *testing.T) {
	resp := makeResp(201, "", nil, 100)
	r := Evaluate(model.Assertion{Target: "status", Operator: ">=", Expected: "200"}, resp)
	assert.True(t, r.Passed)
	r = Evaluate(model.Assertion{Target: "status", Operator: "<", Expected: "300"}, resp)
	assert.True(t, r.Passed)
}

func TestBodyJsonPath(t *testing.T) {
	resp := makeResp(200, `{"id":42,"name":"Alice","nested":{"key":"val"}}`, nil, 100)

	r := Evaluate(model.Assertion{Target: "body.$.id", Operator: "==", Expected: "42"}, resp)
	assert.True(t, r.Passed)

	r = Evaluate(model.Assertion{Target: "body.$.name", Operator: "==", Expected: "Alice"}, resp)
	assert.True(t, r.Passed)

	r = Evaluate(model.Assertion{Target: "body.$.nested.key", Operator: "==", Expected: "val"}, resp)
	assert.True(t, r.Passed)
}

func TestBodyJsonPathNotNull(t *testing.T) {
	resp := makeResp(200, `{"id":42}`, nil, 100)
	r := Evaluate(model.Assertion{Target: "body.$.id", Operator: "exists"}, resp)
	assert.True(t, r.Passed)

	r = Evaluate(model.Assertion{Target: "body.$.missing", Operator: "exists"}, resp)
	assert.False(t, r.Passed)

	r = Evaluate(model.Assertion{Target: "body.$.missing", Operator: "!exists"}, resp)
	assert.True(t, r.Passed)
}

func TestBodyContains(t *testing.T) {
	resp := makeResp(200, `{"message":"hello world"}`, nil, 100)
	r := Evaluate(model.Assertion{Target: "body", Operator: "contains", Expected: "hello"}, resp)
	assert.True(t, r.Passed)

	r = Evaluate(model.Assertion{Target: "body", Operator: "contains", Expected: "goodbye"}, resp)
	assert.False(t, r.Passed)
}

func TestHeaderContains(t *testing.T) {
	resp := makeResp(200, "", map[string]string{"Content-Type": "application/json; charset=utf-8"}, 100)
	r := Evaluate(model.Assertion{Target: "header.Content-Type", Operator: "contains", Expected: "json"}, resp)
	assert.True(t, r.Passed)
}

func TestHeaderExists(t *testing.T) {
	resp := makeResp(200, "", map[string]string{"X-Request-Id": "abc123"}, 100)
	r := Evaluate(model.Assertion{Target: "header.X-Request-Id", Operator: "exists"}, resp)
	assert.True(t, r.Passed)

	r = Evaluate(model.Assertion{Target: "header.X-Missing", Operator: "exists"}, resp)
	assert.False(t, r.Passed)
}

func TestDuration(t *testing.T) {
	resp := makeResp(200, "", nil, 150)
	r := Evaluate(model.Assertion{Target: "duration", Operator: "<", Expected: "2000"}, resp)
	assert.True(t, r.Passed)
	assert.Equal(t, "150", r.Actual)

	r = Evaluate(model.Assertion{Target: "duration", Operator: "<", Expected: "100"}, resp)
	assert.False(t, r.Passed)
}

func TestSize(t *testing.T) {
	resp := makeResp(200, `{"id":1}`, nil, 100)
	r := Evaluate(model.Assertion{Target: "size", Operator: "<", Expected: "1024"}, resp)
	assert.True(t, r.Passed)
	assert.Equal(t, "8", r.Actual)
}

func TestBodyMatches(t *testing.T) {
	resp := makeResp(200, `{"id":42}`, nil, 100)
	r := Evaluate(model.Assertion{Target: "body", Operator: "matches", Expected: `"id":\d+`}, resp)
	assert.True(t, r.Passed)
}

func TestEvaluateAll(t *testing.T) {
	req := &model.Request{
		Assertions: []model.Assertion{
			{Target: "status", Operator: "==", Expected: "200"},
			{Target: "duration", Operator: "<", Expected: "5000"},
			{Target: "status", Operator: "==", Expected: "404"}, // should fail
		},
	}
	resp := makeResp(200, "", nil, 100)
	results := EvaluateAll(req, resp)
	assert.Len(t, results, 3)
	assert.Equal(t, 2, CountPassed(results))
	assert.False(t, AllPassed(results))
}

func TestUnknownTarget(t *testing.T) {
	resp := makeResp(200, "", nil, 100)
	r := Evaluate(model.Assertion{Target: "unknown", Operator: "==", Expected: "foo"}, resp)
	assert.False(t, r.Passed)
	assert.NotEmpty(t, r.Error)
}
