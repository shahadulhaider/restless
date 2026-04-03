package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/writer"
)

// ParseCurlCommand parses a curl command string and returns a model.Request.
// Supports: -X/-request, -H/--header, -d/--data/--data-raw, -u/--user,
// -b/--cookie, -L (follow redirects), --url, and positional URL.
func ParseCurlCommand(cmd string) (*model.Request, error) {
	tokens, err := tokenizeCurl(cmd)
	if err != nil {
		return nil, fmt.Errorf("tokenize: %w", err)
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	if strings.ToLower(tokens[0]) != "curl" {
		return nil, fmt.Errorf("not a curl command")
	}

	req := &model.Request{
		Method: "GET",
	}

	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		switch {
		case tok == "-X" || tok == "--request":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("missing value for %s", tok)
			}
			i++
			req.Method = strings.ToUpper(tokens[i])

		case tok == "-H" || tok == "--header":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("missing value for %s", tok)
			}
			i++
			k, v, ok := strings.Cut(tokens[i], ":")
			if !ok {
				return nil, fmt.Errorf("invalid header: %s", tokens[i])
			}
			req.Headers = append(req.Headers, model.Header{
				Key:   strings.TrimSpace(k),
				Value: strings.TrimSpace(v),
			})

		case tok == "-d" || tok == "--data" || tok == "--data-raw" || tok == "--data-binary":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("missing value for %s", tok)
			}
			i++
			req.Body = tokens[i]
			// Infer POST if method not explicitly set
			if req.Method == "GET" {
				req.Method = "POST"
			}

		case tok == "-u" || tok == "--user":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("missing value for %s", tok)
			}
			i++
			req.Headers = append(req.Headers, model.Header{
				Key:   "Authorization",
				Value: "Basic " + tokens[i],
			})

		case tok == "-b" || tok == "--cookie":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("missing value for %s", tok)
			}
			i++
			req.Headers = append(req.Headers, model.Header{
				Key:   "Cookie",
				Value: tokens[i],
			})

		case tok == "--url":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("missing value for %s", tok)
			}
			i++
			req.URL = tokens[i]

		case tok == "-L" || tok == "--location":
			// -L means follow redirects — in restless, default is to follow.
			// No metadata flag needed; ignore.

		case tok == "-s" || tok == "--silent",
			tok == "-S" || tok == "--show-error",
			tok == "-v" || tok == "--verbose",
			tok == "-i" || tok == "--include",
			tok == "-I" || tok == "--head",
			tok == "-k" || tok == "--insecure",
			tok == "--compressed",
			tok == "-f" || tok == "--fail":
			// Flags with no value argument — skip

		case tok == "-o" || tok == "--output",
			tok == "--connect-timeout",
			tok == "--max-time",
			tok == "-m",
			tok == "-A" || tok == "--user-agent",
			tok == "-e" || tok == "--referer",
			tok == "--proxy",
			tok == "-x":
			// Flags that consume one value — skip both
			i++

		case strings.HasPrefix(tok, "-"):
			// Unknown flag — if it looks like it takes a value, try to skip it
			// For safety, just skip the flag token itself.

		default:
			// Positional argument — the URL
			if req.URL == "" {
				req.URL = tok
			}
		}
	}

	if req.URL == "" {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	return req, nil
}

// ImportCurl parses a curl command string and writes a .http file to opts.OutputDir.
func ImportCurl(cmd string, opts ImportOptions) error {
	req, err := ParseCurlCommand(cmd)
	if err != nil {
		return fmt.Errorf("parse curl: %w", err)
	}
	outDir := opts.OutputDir
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	outPath := filepath.Join(outDir, "imported.http")
	return writer.InsertRequest(outPath, *req)
}

// GenerateCurl produces a curl command string from a model.Request.
func GenerateCurl(req model.Request) string {
	var parts []string
	parts = append(parts, "curl")

	if req.Method != "" && req.Method != "GET" {
		parts = append(parts, "-X", req.Method)
	}

	for _, h := range req.Headers {
		parts = append(parts, "-H", shellQuote(h.Key+": "+h.Value))
	}

	if req.Body != "" {
		parts = append(parts, "--data-raw", shellQuote(req.Body))
	}

	if req.Metadata.NoRedirect {
		// No -L flag means curl won't follow redirects by default
	} else {
		parts = append(parts, "-L")
	}

	parts = append(parts, shellQuote(req.URL))

	return strings.Join(parts, " ")
}

// shellQuote wraps s in single quotes, escaping any single quotes within.
func shellQuote(s string) string {
	if !strings.ContainsAny(s, " \t\n\"'\\{}$&|;<>()#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// tokenizeCurl splits a curl command respecting single- and double-quoted strings
// and backslash-escaped newlines (line continuations).
func tokenizeCurl(cmd string) ([]string, error) {
	// Normalize line continuations: "\ \n" → " "
	cmd = strings.ReplaceAll(cmd, "\\\n", " ")

	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	i := 0
	runes := []rune(cmd)

	for i < len(runes) {
		ch := runes[i]

		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			} else {
				cur.WriteRune(ch)
			}
			i++

		case inDouble:
			if ch == '"' {
				inDouble = false
				i++
			} else if ch == '\\' && i+1 < len(runes) {
				next := runes[i+1]
				switch next {
				case '"', '\\', '$', '`', '\n':
					cur.WriteRune(next)
				default:
					cur.WriteRune(ch)
					cur.WriteRune(next)
				}
				i += 2
			} else {
				cur.WriteRune(ch)
				i++
			}

		case ch == '\'':
			inSingle = true
			i++

		case ch == '"':
			inDouble = true
			i++

		case ch == '\\' && i+1 < len(runes):
			cur.WriteRune(runes[i+1])
			i += 2

		case unicode.IsSpace(ch):
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			i++

		default:
			cur.WriteRune(ch)
			i++
		}
	}

	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote")
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}
