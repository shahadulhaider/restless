package engine

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/shahadulhaider/restless/internal/model"
)

func Execute(req *model.Request) (*model.Response, error) {
	return ExecuteWithJar(req, nil)
}

func ExecuteWithJar(req *model.Request, jar http.CookieJar) (*model.Response, error) {
	dialTimeout := 30 * time.Second
	if req.Metadata.ConnTimeout > 0 {
		dialTimeout = req.Metadata.ConnTimeout
	}

	transport := &http.Transport{
		ForceAttemptHTTP2: true,
		DialContext: (&net.Dialer{
			Timeout: dialTimeout,
		}).DialContext,
		TLSClientConfig: &tls.Config{},
	}

	clientTimeout := 0 * time.Second
	if req.Metadata.Timeout > 0 {
		clientTimeout = req.Metadata.Timeout
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   clientTimeout,
		Jar:       jar,
	}

	if req.Metadata.NoRedirect {
		client.CheckRedirect = func(r *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	var timing model.ResponseTiming
	var dnsStart, connectStart, tlsStart, firstByte time.Time
	start := time.Now()

	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { timing.DNS = time.Since(dnsStart) },
		ConnectStart:         func(_, _ string) { connectStart = time.Now() },
		ConnectDone:          func(_, _ string, _ error) { timing.Connect = time.Since(connectStart) },
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { timing.TLS = time.Since(tlsStart) },
		GotFirstResponseByte: func() { firstByte = time.Now(); timing.TTFB = time.Since(start) },
	}

	ctx := httptrace.WithClientTrace(context.Background(), trace)

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	for _, h := range req.Headers {
		httpReq.Header.Set(h.Key, h.Value)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	bodyStart := time.Now()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	timing.BodyRead = time.Since(bodyStart)

	if firstByte.IsZero() {
		timing.TTFB = time.Since(start)
	}
	timing.Total = time.Since(start)

	var headers []model.Header
	for key, vals := range resp.Header {
		for _, val := range vals {
			headers = append(headers, model.Header{Key: key, Value: val})
		}
	}

	contentType := resp.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	return &model.Response{
		StatusCode:  resp.StatusCode,
		Status:      resp.Status,
		Headers:     headers,
		Body:        body,
		ContentType: contentType,
		Timing:      timing,
		Request:     req,
		Timestamp:   time.Now(),
	}, nil
}

func HeaderValue(resp *model.Response, key string) string {
	for _, h := range resp.Headers {
		if strings.EqualFold(h.Key, key) {
			return h.Value
		}
	}
	return ""
}

func IsJSON(resp *model.Response) bool {
	ct := resp.ContentType
	return strings.Contains(ct, "application/json") || strings.Contains(ct, "+json")
}

func BodyString(resp *model.Response) string {
	return string(resp.Body)
}

var _ = bytes.NewReader
