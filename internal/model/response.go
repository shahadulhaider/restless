package model

import "time"

type Response struct {
	StatusCode       int
	Status           string
	Headers          []Header
	Body             []byte
	ContentType      string
	Timing           ResponseTiming
	Request          *Request
	Timestamp        time.Time
	AssertionResults []AssertionResult
}

// AssertionResult holds the outcome of evaluating one assertion.
type AssertionResult struct {
	Assertion Assertion
	Passed    bool
	Actual    string // the value that was tested
	Error     string // non-empty if evaluation itself failed
}

type ResponseTiming struct {
	DNS      time.Duration
	Connect  time.Duration
	TLS      time.Duration
	TTFB     time.Duration
	Total    time.Duration
	BodyRead time.Duration
}
