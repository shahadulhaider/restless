package model

import "time"

type Response struct {
	StatusCode  int
	Status      string
	Headers     []Header
	Body        []byte
	ContentType string
	Timing      ResponseTiming
	Request     *Request
	Timestamp   time.Time
}

type ResponseTiming struct {
	DNS      time.Duration
	Connect  time.Duration
	TLS      time.Duration
	TTFB     time.Duration
	Total    time.Duration
	BodyRead time.Duration
}
