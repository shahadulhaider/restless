package parser

import (
	"strings"
)

// TokenType represents the type of a lexer token.
type TokenType int

const (
	TokenRequestSeparator TokenType = iota // ###
	TokenMethod                            // GET, POST, PUT, DELETE, etc.
	TokenURL                               // URL after method
	TokenHTTPVersion                       // HTTP/1.1, HTTP/2
	TokenHeaderKey                         // header key before :
	TokenHeaderValue                       // header value after :
	TokenBody                              // body content line
	TokenComment                           // # comment line (non-metadata)
	TokenMetadata                          // # @name, # @no-redirect, etc.
	TokenFileRef                           // < ./path
	TokenBlankLine                         // blank line (separates headers from body)
	TokenEOF
)

// Token is a single lexer token.
type Token struct {
	Type  TokenType
	Value string
	Line  int
}

// Lexer tokenizes .http files per the JetBrains HTTP spec.
type Lexer struct {
	lines []string
}

// NewLexer creates a new Lexer from raw bytes.
func NewLexer(input []byte) *Lexer {
	// Normalize CRLF to LF
	normalized := strings.ReplaceAll(string(input), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	// Remove trailing empty line from Split artifact
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return &Lexer{lines: lines}
}

var httpMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true, "TRACE": true, "CONNECT": true,
}

// Tokenize scans the input and returns a slice of tokens.
func (l *Lexer) Tokenize() []Token {
	if len(l.lines) == 0 {
		return []Token{{Type: TokenEOF, Line: 1}}
	}

	var tokens []Token
	lineNum := 1

	// State machine
	type state int
	const (
		stateStart   state = iota // looking for request line
		stateHeaders              // reading headers
		stateBody                 // reading body
	)

	cur := stateStart

	for _, line := range l.lines {
		trimmed := strings.TrimSpace(line)

		switch cur {
		case stateStart:
			if strings.HasPrefix(trimmed, "@") && strings.Contains(trimmed, "=") {
				// Inline file variable: @varName = value — skip (handled by ExtractFileVariables)
			} else if strings.HasPrefix(trimmed, "###") {
				tokens = append(tokens, Token{Type: TokenRequestSeparator, Value: trimmed, Line: lineNum})
				// stays in stateStart looking for next request line
			} else if strings.HasPrefix(trimmed, "# @") || strings.HasPrefix(trimmed, "// @") {
				tokens = append(tokens, Token{Type: TokenMetadata, Value: trimmed, Line: lineNum})
			} else if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
				tokens = append(tokens, Token{Type: TokenComment, Value: trimmed, Line: lineNum})
			} else if trimmed == "" {
				// blank lines before request line are ignored
			} else {
				// This should be a request line: METHOD URL [HTTP/version]
				parts := strings.Fields(trimmed)
				if len(parts) >= 2 && httpMethods[parts[0]] {
					tokens = append(tokens, Token{Type: TokenMethod, Value: parts[0], Line: lineNum})
					// Check if last part is HTTP version
					if len(parts) >= 3 && strings.HasPrefix(parts[len(parts)-1], "HTTP/") {
						url := strings.Join(parts[1:len(parts)-1], " ")
						tokens = append(tokens, Token{Type: TokenURL, Value: url, Line: lineNum})
						tokens = append(tokens, Token{Type: TokenHTTPVersion, Value: parts[len(parts)-1], Line: lineNum})
					} else {
						url := strings.Join(parts[1:], " ")
						tokens = append(tokens, Token{Type: TokenURL, Value: url, Line: lineNum})
					}
					cur = stateHeaders
				} else {
					// Unknown line in start state — treat as comment
					tokens = append(tokens, Token{Type: TokenComment, Value: trimmed, Line: lineNum})
				}
			}

		case stateHeaders:
			if strings.HasPrefix(trimmed, "###") {
				tokens = append(tokens, Token{Type: TokenRequestSeparator, Value: trimmed, Line: lineNum})
				cur = stateStart
			} else if trimmed == "" {
				tokens = append(tokens, Token{Type: TokenBlankLine, Value: "", Line: lineNum})
				cur = stateBody
			} else if strings.HasPrefix(trimmed, "# @") || strings.HasPrefix(trimmed, "// @") {
				tokens = append(tokens, Token{Type: TokenMetadata, Value: trimmed, Line: lineNum})
			} else if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
				tokens = append(tokens, Token{Type: TokenComment, Value: trimmed, Line: lineNum})
			} else {
				// Parse header: Key: Value
				idx := strings.Index(line, ":")
				if idx > 0 {
					key := strings.TrimSpace(line[:idx])
					value := strings.TrimSpace(line[idx+1:])
					tokens = append(tokens, Token{Type: TokenHeaderKey, Value: key, Line: lineNum})
					tokens = append(tokens, Token{Type: TokenHeaderValue, Value: value, Line: lineNum})
				} else {
					tokens = append(tokens, Token{Type: TokenComment, Value: trimmed, Line: lineNum})
				}
			}

		case stateBody:
			if strings.HasPrefix(trimmed, "###") {
				tokens = append(tokens, Token{Type: TokenRequestSeparator, Value: trimmed, Line: lineNum})
				cur = stateStart
			} else if strings.HasPrefix(trimmed, "# @") || strings.HasPrefix(trimmed, "// @") {
				// Metadata after body (e.g. @assert, @post-response)
				tokens = append(tokens, Token{Type: TokenMetadata, Value: trimmed, Line: lineNum})
			} else if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
				// Comment in body area (could be script block content)
				tokens = append(tokens, Token{Type: TokenComment, Value: trimmed, Line: lineNum})
			} else if strings.HasPrefix(trimmed, "< ") {
				// File reference
				path := strings.TrimSpace(trimmed[2:])
				tokens = append(tokens, Token{Type: TokenFileRef, Value: path, Line: lineNum})
			} else {
				// Body line (preserve original, not trimmed, except for empty)
				tokens = append(tokens, Token{Type: TokenBody, Value: line, Line: lineNum})
			}
		}

		lineNum++
	}

	tokens = append(tokens, Token{Type: TokenEOF, Line: lineNum})
	return tokens
}
