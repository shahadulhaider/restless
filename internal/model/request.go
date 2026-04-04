package model

import "time"

type Request struct {
	Name        string
	Method      string
	URL         string
	HTTPVersion string
	Headers     []Header
	Body        string
	BodyFile    string
	Metadata    RequestMetadata
	Assertions  []Assertion
	SourceFile  string
	SourceLine  int
}

// Assertion defines a response assertion declared via # @assert.
type Assertion struct {
	Target   string // "status", "body.$.id", "header.Content-Type", "duration", "size"
	Operator string // "==", "!=", "<", ">", "<=", ">=", "contains", "matches", "exists", "!exists"
	Expected string // "201", "Alice", "json"
	Raw      string // original text: "status == 201"
}

type Header struct {
	Key   string
	Value string
}

type RequestMetadata struct {
	NoRedirect  bool
	NoCookieJar bool
	Timeout     time.Duration
	ConnTimeout time.Duration
	Insecure    bool   // skip TLS verification
	Proxy       string // proxy URL (http://proxy:port)
}
