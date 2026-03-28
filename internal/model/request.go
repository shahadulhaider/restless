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
	SourceFile  string
	SourceLine  int
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
}
