package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequestZeroValue(t *testing.T) {
	var r Request

	assert.Equal(t, "", r.Name)
	assert.Equal(t, "", r.Method)
	assert.Equal(t, "", r.URL)
	assert.Equal(t, "", r.HTTPVersion)
	assert.Nil(t, r.Headers)
	assert.Empty(t, r.Headers)
	assert.Equal(t, "", r.Body)
	assert.Equal(t, "", r.BodyFile)
	assert.Equal(t, RequestMetadata{}, r.Metadata)
	assert.Nil(t, r.Assertions)
	assert.Equal(t, "", r.PreRequestScript)
	assert.Equal(t, "", r.PostResponseScript)
	assert.Equal(t, "", r.SourceFile)
	assert.Equal(t, 0, r.SourceLine)
}

func TestRequestFieldAssignment(t *testing.T) {
	r := Request{
		Name:        "create-user",
		Method:      "POST",
		URL:         "https://api.example.com/users",
		HTTPVersion: "HTTP/1.1",
		Headers: []Header{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "Authorization", Value: "Bearer xyz"},
		},
		Body:     `{"name":"Alice"}`,
		BodyFile: "body.json",
		Metadata: RequestMetadata{
			NoRedirect:  true,
			NoCookieJar: true,
			Timeout:     30 * time.Second,
			ConnTimeout: 5 * time.Second,
			Insecure:    true,
			Proxy:       "http://proxy:8080",
		},
		Assertions: []Assertion{
			{Target: "status", Operator: "==", Expected: "201", Raw: "status == 201"},
		},
		PreRequestScript:   "pre();",
		PostResponseScript: "post();",
		SourceFile:         "api.http",
		SourceLine:         12,
	}

	assert.Equal(t, "create-user", r.Name)
	assert.Equal(t, "POST", r.Method)
	assert.Equal(t, "https://api.example.com/users", r.URL)
	assert.Equal(t, "HTTP/1.1", r.HTTPVersion)
	assert.Len(t, r.Headers, 2)
	assert.Equal(t, "Content-Type", r.Headers[0].Key)
	assert.Equal(t, "application/json", r.Headers[0].Value)
	assert.Equal(t, "Authorization", r.Headers[1].Key)
	assert.Equal(t, `{"name":"Alice"}`, r.Body)
	assert.Equal(t, "body.json", r.BodyFile)
	assert.True(t, r.Metadata.NoRedirect)
	assert.True(t, r.Metadata.NoCookieJar)
	assert.Equal(t, 30*time.Second, r.Metadata.Timeout)
	assert.Equal(t, 5*time.Second, r.Metadata.ConnTimeout)
	assert.True(t, r.Metadata.Insecure)
	assert.Equal(t, "http://proxy:8080", r.Metadata.Proxy)
	assert.Len(t, r.Assertions, 1)
	assert.Equal(t, "status == 201", r.Assertions[0].Raw)
	assert.Equal(t, "pre();", r.PreRequestScript)
	assert.Equal(t, "post();", r.PostResponseScript)
	assert.Equal(t, "api.http", r.SourceFile)
	assert.Equal(t, 12, r.SourceLine)
}

func TestRequestDeepEqual(t *testing.T) {
	build := func() Request {
		return Request{
			Name:    "echo",
			Method:  "GET",
			URL:     "https://httpbin.org/get",
			Headers: []Header{{Key: "Accept", Value: "application/json"}},
			Metadata: RequestMetadata{
				Timeout: 15 * time.Second,
			},
			Assertions: []Assertion{
				{Target: "status", Operator: "==", Expected: "200", Raw: "status == 200"},
			},
			SourceLine: 3,
		}
	}

	a := build()
	b := build()
	assert.Equal(t, a, b)

	b.Headers[0].Value = "text/plain"
	assert.NotEqual(t, a, b)
}

func TestHeader(t *testing.T) {
	tests := []struct {
		name      string
		header    Header
		wantKey   string
		wantValue string
	}{
		{"typical", Header{Key: "Content-Type", Value: "application/json"}, "Content-Type", "application/json"},
		{"empty value", Header{Key: "X-Empty", Value: ""}, "X-Empty", ""},
		{"empty key", Header{Key: "", Value: "orphan"}, "", "orphan"},
		{"zero", Header{}, "", ""},
		{"whitespace preserved", Header{Key: "X-Pad", Value: "  spaced  "}, "X-Pad", "  spaced  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantKey, tt.header.Key)
			assert.Equal(t, tt.wantValue, tt.header.Value)
		})
	}
}

func TestHeaderComparable(t *testing.T) {
	a := Header{Key: "X", Value: "1"}
	b := Header{Key: "X", Value: "1"}
	c := Header{Key: "X", Value: "2"}

	assert.True(t, a == b)
	assert.False(t, a == c)

	m := map[Header]int{a: 10}
	assert.Equal(t, 10, m[b])
}

func TestHeadersPreserveOrderAndDuplicates(t *testing.T) {
	headers := []Header{
		{Key: "Set-Cookie", Value: "a=1"},
		{Key: "Set-Cookie", Value: "b=2"},
		{Key: "Accept", Value: "text/html"},
	}

	assert.Len(t, headers, 3)
	assert.Equal(t, "a=1", headers[0].Value)
	assert.Equal(t, "b=2", headers[1].Value)
	assert.Equal(t, "Accept", headers[2].Key)

	dupes := 0
	for _, h := range headers {
		if h.Key == "Set-Cookie" {
			dupes++
		}
	}
	assert.Equal(t, 2, dupes)
}

func TestAssertion(t *testing.T) {
	tests := []struct {
		name         string
		assertion    Assertion
		wantTarget   string
		wantOperator string
		wantExpected string
		wantRaw      string
	}{
		{
			name:         "status equals",
			assertion:    Assertion{Target: "status", Operator: "==", Expected: "201", Raw: "status == 201"},
			wantTarget:   "status",
			wantOperator: "==",
			wantExpected: "201",
			wantRaw:      "status == 201",
		},
		{
			name:         "body jsonpath contains",
			assertion:    Assertion{Target: "body.$.name", Operator: "contains", Expected: "Alice", Raw: "body.$.name contains Alice"},
			wantTarget:   "body.$.name",
			wantOperator: "contains",
			wantExpected: "Alice",
			wantRaw:      "body.$.name contains Alice",
		},
		{
			name:         "exists without expected",
			assertion:    Assertion{Target: "header.Content-Type", Operator: "exists", Raw: "header.Content-Type exists"},
			wantTarget:   "header.Content-Type",
			wantOperator: "exists",
			wantExpected: "",
			wantRaw:      "header.Content-Type exists",
		},
		{
			name:      "zero",
			assertion: Assertion{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantTarget, tt.assertion.Target)
			assert.Equal(t, tt.wantOperator, tt.assertion.Operator)
			assert.Equal(t, tt.wantExpected, tt.assertion.Expected)
			assert.Equal(t, tt.wantRaw, tt.assertion.Raw)
		})
	}
}

func TestAssertionComparable(t *testing.T) {
	a := Assertion{Target: "status", Operator: "==", Expected: "200", Raw: "status == 200"}
	b := Assertion{Target: "status", Operator: "==", Expected: "200", Raw: "status == 200"}
	c := Assertion{Target: "status", Operator: "==", Expected: "404", Raw: "status == 404"}

	assert.True(t, a == b)
	assert.False(t, a == c)
}

func TestRequestMetadata(t *testing.T) {
	tests := []struct {
		name            string
		meta            RequestMetadata
		wantNoRedirect  bool
		wantNoCookieJar bool
		wantTimeout     time.Duration
		wantConnTimeout time.Duration
		wantInsecure    bool
		wantProxy       string
	}{
		{name: "zero"},
		{
			name:            "timeouts only",
			meta:            RequestMetadata{Timeout: 10 * time.Second, ConnTimeout: 2 * time.Second},
			wantTimeout:     10 * time.Second,
			wantConnTimeout: 2 * time.Second,
		},
		{
			name:            "flags only",
			meta:            RequestMetadata{NoRedirect: true, NoCookieJar: true, Insecure: true},
			wantNoRedirect:  true,
			wantNoCookieJar: true,
			wantInsecure:    true,
		},
		{
			name:      "proxy only",
			meta:      RequestMetadata{Proxy: "http://localhost:8888"},
			wantProxy: "http://localhost:8888",
		},
		{
			name: "fully populated",
			meta: RequestMetadata{
				NoRedirect:  true,
				NoCookieJar: true,
				Timeout:     time.Minute,
				ConnTimeout: 3 * time.Second,
				Insecure:    true,
				Proxy:       "http://p:1",
			},
			wantNoRedirect:  true,
			wantNoCookieJar: true,
			wantTimeout:     time.Minute,
			wantConnTimeout: 3 * time.Second,
			wantInsecure:    true,
			wantProxy:       "http://p:1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantNoRedirect, tt.meta.NoRedirect)
			assert.Equal(t, tt.wantNoCookieJar, tt.meta.NoCookieJar)
			assert.Equal(t, tt.wantTimeout, tt.meta.Timeout)
			assert.Equal(t, tt.wantConnTimeout, tt.meta.ConnTimeout)
			assert.Equal(t, tt.wantInsecure, tt.meta.Insecure)
			assert.Equal(t, tt.wantProxy, tt.meta.Proxy)
		})
	}
}

func TestRequestMetadataComparable(t *testing.T) {
	a := RequestMetadata{Timeout: time.Second, Proxy: "http://p"}
	b := RequestMetadata{Timeout: time.Second, Proxy: "http://p"}
	c := RequestMetadata{Timeout: 2 * time.Second, Proxy: "http://p"}

	assert.True(t, a == b)
	assert.False(t, a == c)
	assert.Equal(t, time.Duration(0), RequestMetadata{}.Timeout)
}
