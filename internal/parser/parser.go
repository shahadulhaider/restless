package parser

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shahadulhaider/restless/internal/model"
)

var knownMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true, "TRACE": true, "CONNECT": true,
}

func ParseFile(path string) ([]model.Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return ParseBytes(data, path)
}

func ParseBytes(content []byte, sourcePath string) ([]model.Request, error) {
	lexer := NewLexer(content)
	tokens := lexer.Tokenize()
	return parseTokens(tokens, sourcePath)
}

func parseTokens(tokens []Token, sourcePath string) ([]model.Request, error) {
	var requests []model.Request
	var pendingMeta []Token

	i := 0
	for i < len(tokens) {
		tok := tokens[i]

		switch tok.Type {
		case TokenEOF:
			return requests, nil

		case TokenRequestSeparator:
			pendingMeta = nil
			i++

		case TokenMetadata:
			// Check for script block start
			val := extractMetaValue(tok.Value)
			if strings.HasPrefix(val, "pre-request {") || strings.HasPrefix(val, "post-response {") {
				scriptLines, consumed := collectScriptBlock(tokens, i)
				scriptType := "pre-request"
				if strings.HasPrefix(val, "post-response") {
					scriptType = "post-response"
				}
				// Store as a special metadata token with the full script
				pendingMeta = append(pendingMeta, Token{
					Type:  TokenMetadata,
					Value: "# @" + scriptType + "-script " + strings.Join(scriptLines, "\n"),
					Line:  tok.Line,
				})
				i += consumed
			} else {
				pendingMeta = append(pendingMeta, tok)
				i++
			}

		case TokenComment:
			i++

		case TokenMethod:
			req, consumed, err := parseRequest(tokens, i, sourcePath, pendingMeta)
			pendingMeta = nil
			if err != nil {
				i++
				continue
			}
			requests = append(requests, req)
			i += consumed

		case TokenBlankLine:
			i++

		default:
			i++
		}
	}

	return requests, nil
}

// extractMetaValue strips the comment prefix and @ from a metadata token value.
func extractMetaValue(s string) string {
	s = strings.TrimPrefix(s, "# ")
	s = strings.TrimPrefix(s, "// ")
	s = strings.TrimPrefix(s, "@")
	return s
}

// collectScriptBlock collects lines of a script block from # @pre-request { to # }
// Returns the script lines (with # prefix stripped) and total tokens consumed.
// Works with both metadata and comment tokens (script content in body state is comments).
func collectScriptBlock(tokens []Token, startIdx int) ([]string, int) {
	var lines []string
	i := startIdx + 1 // skip the opening line
	consumed := 1

	for i < len(tokens) {
		tok := tokens[i]
		if tok.Type == TokenRequestSeparator || tok.Type == TokenEOF {
			break
		}
		if tok.Type == TokenMetadata || tok.Type == TokenComment {
			line := tok.Value
			// Strip comment prefix
			if strings.HasPrefix(line, "# ") {
				line = line[2:]
			} else if strings.HasPrefix(line, "// ") {
				line = line[3:]
			} else if line == "#" {
				line = ""
			}
			trimmed := strings.TrimSpace(line)

			// Check for closing brace
			if trimmed == "}" {
				i++
				consumed++
				break
			}
			lines = append(lines, line)
		}
		// Skip body tokens between script lines (e.g. empty lines)
		i++
		consumed++
	}
	return lines, consumed
}

func parseRequest(tokens []Token, start int, sourcePath string, meta []Token) (model.Request, int, error) {
	if start >= len(tokens) || tokens[start].Type != TokenMethod {
		return model.Request{}, 0, fmt.Errorf("expected method token")
	}

	method := tokens[start].Value
	if !knownMethods[method] {
		return model.Request{}, 1, fmt.Errorf("%s:%d: unknown HTTP method %q", sourcePath, tokens[start].Line, method)
	}

	req := model.Request{
		Method:     method,
		SourceFile: sourcePath,
		SourceLine: tokens[start].Line,
	}

	applyMetadata(&req, meta)

	i := start + 1
	consumed := 1

	if i < len(tokens) && tokens[i].Type == TokenURL {
		req.URL = tokens[i].Value
		i++
		consumed++
	}

	if i < len(tokens) && tokens[i].Type == TokenHTTPVersion {
		req.HTTPVersion = tokens[i].Value
		i++
		consumed++
	}

	for i < len(tokens) {
		t := tokens[i]
		switch t.Type {
		case TokenHeaderKey:
			if i+1 < len(tokens) && tokens[i+1].Type == TokenHeaderValue {
				req.Headers = append(req.Headers, model.Header{
					Key:   t.Value,
					Value: tokens[i+1].Value,
				})
				i += 2
				consumed += 2
			} else {
				i++
				consumed++
			}
		case TokenMetadata:
			val := extractMetaValue(t.Value)
			if strings.HasPrefix(val, "pre-request {") || strings.HasPrefix(val, "post-response {") {
				scriptType := "pre-request"
				if strings.HasPrefix(val, "post-response") {
					scriptType = "post-response"
				}
				scriptLines, sc := collectScriptBlock(tokens, i)
				scriptText := strings.Join(scriptLines, "\n")
				if scriptType == "pre-request" {
					req.PreRequestScript = scriptText
				} else {
					req.PostResponseScript = scriptText
				}
				i += sc
				consumed += sc
			} else {
				applyMetadataSingle(&req, t)
				i++
				consumed++
			}
		case TokenComment:
			i++
			consumed++
		case TokenBlankLine:
			i++
			consumed++
			bodyLines, bodyConsumed, fileRef := collectBody(tokens, i)
			req.Body = strings.Join(bodyLines, "\n")
			req.BodyFile = fileRef
			i += bodyConsumed
			consumed += bodyConsumed
			// Continue to process metadata after body (e.g. @assert, @post-response)
		case TokenRequestSeparator, TokenEOF:
			return req, consumed, nil
		default:
			i++
			consumed++
		}
	}

	return req, consumed, nil
}

func collectBody(tokens []Token, start int) (lines []string, consumed int, fileRef string) {
	i := start
	for i < len(tokens) {
		t := tokens[i]
		switch t.Type {
		case TokenRequestSeparator, TokenEOF, TokenMetadata:
			return lines, consumed, fileRef
		case TokenFileRef:
			fileRef = t.Value
			i++
			consumed++
		case TokenBody:
			lines = append(lines, t.Value)
			i++
			consumed++
		default:
			i++
			consumed++
		}
	}
	return lines, consumed, fileRef
}

// parseAssertion parses an assertion string like "status == 200" into an Assertion.
func parseAssertion(raw string) (model.Assertion, bool) {
	a := model.Assertion{Raw: raw}

	// Handle unary operators first: "exists" and "!exists"
	if strings.HasSuffix(raw, " exists") {
		a.Target = strings.TrimSuffix(raw, " exists")
		a.Operator = "exists"
		return a, true
	}
	if strings.HasSuffix(raw, " !exists") {
		a.Target = strings.TrimSuffix(raw, " !exists")
		a.Operator = "!exists"
		return a, true
	}

	// Binary operators (ordered longest first to avoid partial matches)
	operators := []string{">=", "<=", "!=", "==", ">", "<", "contains", "matches"}
	for _, op := range operators {
		idx := strings.Index(raw, " "+op+" ")
		if idx >= 0 {
			a.Target = strings.TrimSpace(raw[:idx])
			a.Operator = op
			a.Expected = strings.TrimSpace(raw[idx+len(op)+2:])
			// Strip surrounding quotes from expected value
			if len(a.Expected) >= 2 && a.Expected[0] == '"' && a.Expected[len(a.Expected)-1] == '"' {
				a.Expected = a.Expected[1 : len(a.Expected)-1]
			}
			return a, true
		}
	}

	return a, false
}

func applyMetadata(req *model.Request, meta []Token) {
	for _, m := range meta {
		applyMetadataSingle(req, m)
	}
}

func applyMetadataSingle(req *model.Request, m Token) {
	val := strings.TrimPrefix(m.Value, "# ")
	val = strings.TrimPrefix(val, "// ")
	val = strings.TrimPrefix(val, "@")

	switch {
	case strings.HasPrefix(val, "name "):
		req.Name = strings.TrimSpace(strings.TrimPrefix(val, "name "))
	case val == "no-redirect":
		req.Metadata.NoRedirect = true
	case val == "no-cookie-jar":
		req.Metadata.NoCookieJar = true
	case strings.HasPrefix(val, "timeout "):
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(val, "timeout ")))
		if err == nil {
			req.Metadata.Timeout = time.Duration(n) * time.Second
		}
	case strings.HasPrefix(val, "connection-timeout "):
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(val, "connection-timeout ")))
		if err == nil {
			req.Metadata.ConnTimeout = time.Duration(n) * time.Second
		}
	case val == "insecure":
		req.Metadata.Insecure = true
	case strings.HasPrefix(val, "proxy "):
		req.Metadata.Proxy = strings.TrimSpace(strings.TrimPrefix(val, "proxy "))
	case strings.HasPrefix(val, "assert "):
		raw := strings.TrimSpace(strings.TrimPrefix(val, "assert "))
		if a, ok := parseAssertion(raw); ok {
			req.Assertions = append(req.Assertions, a)
		}
	case strings.HasPrefix(val, "pre-request-script "):
		req.PreRequestScript = strings.TrimPrefix(val, "pre-request-script ")
	case strings.HasPrefix(val, "post-response-script "):
		req.PostResponseScript = strings.TrimPrefix(val, "post-response-script ")
	case strings.HasPrefix(val, "pre-request {"):
		// Script block start — handled by collectScriptBlock in parseTokens
	case strings.HasPrefix(val, "post-response {"):
		// Script block start — handled by collectScriptBlock in parseTokens
	}
}
