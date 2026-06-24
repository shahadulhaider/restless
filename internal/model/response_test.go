package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResponseZeroValue(t *testing.T) {
	var r Response

	assert.Equal(t, 0, r.StatusCode)
	assert.Equal(t, "", r.Status)
	assert.Nil(t, r.Headers)
	assert.Nil(t, r.Body)
	assert.Equal(t, "", r.ContentType)
	assert.Equal(t, ResponseTiming{}, r.Timing)
	assert.Nil(t, r.Request)
	assert.True(t, r.Timestamp.IsZero())
	assert.Nil(t, r.AssertionResults)
	assert.Equal(t, "", r.ScriptError)
}

func TestResponseFieldAssignment(t *testing.T) {
	ts := time.Date(2026, 6, 24, 10, 30, 0, 0, time.UTC)
	r := Response{
		StatusCode:  200,
		Status:      "200 OK",
		Headers:     []Header{{Key: "Content-Type", Value: "application/json"}},
		Body:        []byte(`{"ok":true}`),
		ContentType: "application/json",
		Timing:      ResponseTiming{Total: 120 * time.Millisecond},
		Timestamp:   ts,
		AssertionResults: []AssertionResult{
			{Assertion: Assertion{Target: "status", Operator: "==", Expected: "200"}, Passed: true, Actual: "200"},
		},
		ScriptError: "",
	}

	assert.Equal(t, 200, r.StatusCode)
	assert.Equal(t, "200 OK", r.Status)
	assert.Len(t, r.Headers, 1)
	assert.Equal(t, "application/json", r.Headers[0].Value)
	assert.Equal(t, []byte(`{"ok":true}`), r.Body)
	assert.Equal(t, `{"ok":true}`, string(r.Body))
	assert.Equal(t, "application/json", r.ContentType)
	assert.Equal(t, 120*time.Millisecond, r.Timing.Total)
	assert.Equal(t, ts, r.Timestamp)
	assert.False(t, r.Timestamp.IsZero())
	assert.Len(t, r.AssertionResults, 1)
	assert.True(t, r.AssertionResults[0].Passed)
	assert.Equal(t, "200", r.AssertionResults[0].Actual)
	assert.Equal(t, "", r.ScriptError)
}

func TestResponseRequestPointer(t *testing.T) {
	var r Response
	assert.Nil(t, r.Request)

	req := &Request{Name: "src", Method: "GET"}
	r.Request = req
	assert.Same(t, req, r.Request)
	assert.Equal(t, "src", r.Request.Name)
	assert.Equal(t, "GET", r.Request.Method)

	req.Method = "POST"
	assert.Equal(t, "POST", r.Request.Method)
}

func TestResponseBodyBytes(t *testing.T) {
	var r Response
	assert.Nil(t, r.Body)
	assert.Len(t, r.Body, 0)

	r.Body = []byte{}
	assert.NotNil(t, r.Body)
	assert.Len(t, r.Body, 0)
	assert.Equal(t, "", string(r.Body))

	r.Body = []byte("hello")
	assert.Equal(t, []byte("hello"), r.Body)
	assert.Equal(t, "hello", string(r.Body))
	assert.Len(t, r.Body, 5)
}

func TestResponseScriptError(t *testing.T) {
	var r Response
	assert.Empty(t, r.ScriptError)

	r.ScriptError = "ReferenceError: foo is not defined"
	assert.Equal(t, "ReferenceError: foo is not defined", r.ScriptError)
}

func TestAssertionResult(t *testing.T) {
	tests := []struct {
		name       string
		result     AssertionResult
		wantTarget string
		wantPassed bool
		wantActual string
		wantError  string
	}{
		{
			name:       "passed",
			result:     AssertionResult{Assertion: Assertion{Target: "status", Operator: "==", Expected: "200"}, Passed: true, Actual: "200"},
			wantTarget: "status",
			wantPassed: true,
			wantActual: "200",
		},
		{
			name:       "failed",
			result:     AssertionResult{Assertion: Assertion{Target: "status", Operator: "==", Expected: "200"}, Passed: false, Actual: "404"},
			wantTarget: "status",
			wantPassed: false,
			wantActual: "404",
		},
		{
			name:       "evaluation error",
			result:     AssertionResult{Assertion: Assertion{Target: "body.$.missing"}, Passed: false, Error: "path not found"},
			wantTarget: "body.$.missing",
			wantPassed: false,
			wantError:  "path not found",
		},
		{
			name:   "zero",
			result: AssertionResult{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantTarget, tt.result.Assertion.Target)
			assert.Equal(t, tt.wantPassed, tt.result.Passed)
			assert.Equal(t, tt.wantActual, tt.result.Actual)
			assert.Equal(t, tt.wantError, tt.result.Error)
		})
	}
}

func TestAssertionResultComparable(t *testing.T) {
	base := Assertion{Target: "status", Operator: "==", Expected: "200"}
	a := AssertionResult{Assertion: base, Passed: true, Actual: "200"}
	b := AssertionResult{Assertion: base, Passed: true, Actual: "200"}
	c := AssertionResult{Assertion: base, Passed: false, Actual: "500"}

	assert.True(t, a == b)
	assert.False(t, a == c)
}

func TestResponseTiming(t *testing.T) {
	tests := []struct {
		name         string
		timing       ResponseTiming
		wantDNS      time.Duration
		wantConnect  time.Duration
		wantTLS      time.Duration
		wantTTFB     time.Duration
		wantTotal    time.Duration
		wantBodyRead time.Duration
	}{
		{name: "zero"},
		{
			name: "fully populated",
			timing: ResponseTiming{
				DNS:      1 * time.Millisecond,
				Connect:  2 * time.Millisecond,
				TLS:      3 * time.Millisecond,
				TTFB:     4 * time.Millisecond,
				Total:    10 * time.Millisecond,
				BodyRead: 5 * time.Millisecond,
			},
			wantDNS:      1 * time.Millisecond,
			wantConnect:  2 * time.Millisecond,
			wantTLS:      3 * time.Millisecond,
			wantTTFB:     4 * time.Millisecond,
			wantTotal:    10 * time.Millisecond,
			wantBodyRead: 5 * time.Millisecond,
		},
		{
			name:      "total only (plain HTTP, no TLS)",
			timing:    ResponseTiming{Total: 50 * time.Millisecond},
			wantTotal: 50 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantDNS, tt.timing.DNS)
			assert.Equal(t, tt.wantConnect, tt.timing.Connect)
			assert.Equal(t, tt.wantTLS, tt.timing.TLS)
			assert.Equal(t, tt.wantTTFB, tt.timing.TTFB)
			assert.Equal(t, tt.wantTotal, tt.timing.Total)
			assert.Equal(t, tt.wantBodyRead, tt.timing.BodyRead)
		})
	}
}

func TestResponseTimingComparable(t *testing.T) {
	a := ResponseTiming{Total: time.Second}
	b := ResponseTiming{Total: time.Second}
	c := ResponseTiming{Total: 2 * time.Second}

	assert.True(t, a == b)
	assert.False(t, a == c)
}
