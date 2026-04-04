package writer

import (
	"fmt"
	"strings"
	"time"

	"github.com/shahadulhaider/restless/internal/model"
)

// SerializeRequest converts a model.Request to .http file format text.
// It does not include a trailing newline after the last line.
func SerializeRequest(req model.Request) string {
	var sb strings.Builder

	// Metadata tags (before request line)
	if req.Name != "" {
		sb.WriteString(fmt.Sprintf("# @name %s\n", req.Name))
	}
	if req.Metadata.NoRedirect {
		sb.WriteString("# @no-redirect\n")
	}
	if req.Metadata.NoCookieJar {
		sb.WriteString("# @no-cookie-jar\n")
	}
	if req.Metadata.Timeout > 0 {
		sb.WriteString(fmt.Sprintf("# @timeout %d\n", int(req.Metadata.Timeout/time.Second)))
	}
	if req.Metadata.ConnTimeout > 0 {
		sb.WriteString(fmt.Sprintf("# @connection-timeout %d\n", int(req.Metadata.ConnTimeout/time.Second)))
	}
	if req.Metadata.Insecure {
		sb.WriteString("# @insecure\n")
	}
	if req.Metadata.Proxy != "" {
		sb.WriteString(fmt.Sprintf("# @proxy %s\n", req.Metadata.Proxy))
	}

	// Request line: METHOD URL [HTTP/version]
	if req.HTTPVersion != "" {
		sb.WriteString(fmt.Sprintf("%s %s %s\n", req.Method, req.URL, req.HTTPVersion))
	} else {
		sb.WriteString(fmt.Sprintf("%s %s\n", req.Method, req.URL))
	}

	// Headers
	for _, h := range req.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\n", h.Key, h.Value))
	}

	// Body (only if present)
	hasBody := req.Body != "" || req.BodyFile != ""
	if hasBody {
		sb.WriteString("\n")
		if req.BodyFile != "" {
			sb.WriteString(fmt.Sprintf("< %s\n", req.BodyFile))
		} else {
			sb.WriteString(req.Body)
			if !strings.HasSuffix(req.Body, "\n") {
				sb.WriteString("\n")
			}
		}
	}

	result := sb.String()
	// Trim trailing newline for cleanliness (SerializeRequests adds separators)
	return strings.TrimRight(result, "\n")
}

// SerializeRequests serializes multiple requests, joining them with the ### separator.
func SerializeRequests(reqs []model.Request) string {
	parts := make([]string, 0, len(reqs))
	for _, r := range reqs {
		parts = append(parts, SerializeRequest(r))
	}
	return strings.Join(parts, "\n\n###\n\n")
}
