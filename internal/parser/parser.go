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
			pendingMeta = append(pendingMeta, tok)
			i++

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
			applyMetadataSingle(&req, t)
			i++
			consumed++
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
			return req, consumed, nil
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
		case TokenRequestSeparator, TokenEOF:
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
	}
}
