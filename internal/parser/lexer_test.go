package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func tokenTypes(tokens []Token) []TokenType {
	var types []TokenType
	for _, t := range tokens {
		types = append(types, t.Type)
	}
	return types
}

func TestLexerSimpleGET(t *testing.T) {
	input := "GET https://example.com/users\n"
	tokens := NewLexer([]byte(input)).Tokenize()
	types := tokenTypes(tokens)
	assert.Equal(t, []TokenType{TokenMethod, TokenURL, TokenEOF}, types)
	assert.Equal(t, "GET", tokens[0].Value)
	assert.Equal(t, "https://example.com/users", tokens[1].Value)
}

func TestLexerPOSTWithBody(t *testing.T) {
	input := `POST https://example.com/users
Content-Type: application/json

{"name": "Alice"}
`
	tokens := NewLexer([]byte(input)).Tokenize()
	types := tokenTypes(tokens)
	assert.Equal(t, []TokenType{
		TokenMethod, TokenURL,
		TokenHeaderKey, TokenHeaderValue,
		TokenBlankLine,
		TokenBody,
		TokenEOF,
	}, types)
	assert.Equal(t, "POST", tokens[0].Value)
	assert.Equal(t, "Content-Type", tokens[2].Value)
	assert.Equal(t, "application/json", tokens[3].Value)
	assert.Equal(t, `{"name": "Alice"}`, tokens[5].Value)
}

func TestLexerMultipleRequests(t *testing.T) {
	input := `GET https://example.com/users

###

POST https://example.com/users
Content-Type: application/json

{}
`
	tokens := NewLexer([]byte(input)).Tokenize()
	methods := []string{}
	for _, t := range tokens {
		if t.Type == TokenMethod {
			methods = append(methods, t.Value)
		}
	}
	assert.Equal(t, []string{"GET", "POST"}, methods)

	separators := 0
	for _, t := range tokens {
		if t.Type == TokenRequestSeparator {
			separators++
		}
	}
	assert.Equal(t, 1, separators)
}

func TestLexerMetadata(t *testing.T) {
	input := `# @name getUser
# @no-redirect
# @timeout 30
GET https://example.com/users/1
`
	tokens := NewLexer([]byte(input)).Tokenize()
	metaTokens := []Token{}
	for _, t := range tokens {
		if t.Type == TokenMetadata {
			metaTokens = append(metaTokens, t)
		}
	}
	assert.Len(t, metaTokens, 3)
	assert.Contains(t, metaTokens[0].Value, "@name getUser")
	assert.Contains(t, metaTokens[1].Value, "@no-redirect")
	assert.Contains(t, metaTokens[2].Value, "@timeout 30")
}

func TestLexerFileRef(t *testing.T) {
	input := `POST https://example.com/users
Content-Type: application/json

< ./body.json
`
	tokens := NewLexer([]byte(input)).Tokenize()
	var fileRef *Token
	for i := range tokens {
		if tokens[i].Type == TokenFileRef {
			fileRef = &tokens[i]
			break
		}
	}
	assert.NotNil(t, fileRef)
	assert.Equal(t, "./body.json", fileRef.Value)
}

func TestLexerVariableInURL(t *testing.T) {
	input := "GET {{base_url}}/users\n"
	tokens := NewLexer([]byte(input)).Tokenize()
	var urlToken *Token
	for i := range tokens {
		if tokens[i].Type == TokenURL {
			urlToken = &tokens[i]
			break
		}
	}
	assert.NotNil(t, urlToken)
	assert.Equal(t, "{{base_url}}/users", urlToken.Value)
}

func TestLexerEmptyFile(t *testing.T) {
	tokens := NewLexer([]byte("")).Tokenize()
	assert.Equal(t, []TokenType{TokenEOF}, tokenTypes(tokens))
}

func TestLexerBodyWithBlankLines(t *testing.T) {
	input := `POST https://example.com/data
Content-Type: application/json

line1

line3
`
	tokens := NewLexer([]byte(input)).Tokenize()
	bodyTokens := []Token{}
	for _, t := range tokens {
		if t.Type == TokenBody {
			bodyTokens = append(bodyTokens, t)
		}
	}
	assert.Len(t, bodyTokens, 3, "blank lines inside body are body tokens")
}

func TestLexerNoTrailingSeparator(t *testing.T) {
	input := `GET https://example.com/users`
	tokens := NewLexer([]byte(input)).Tokenize()
	assert.Equal(t, TokenMethod, tokens[0].Type)
	assert.Equal(t, TokenURL, tokens[1].Type)
	assert.Equal(t, TokenEOF, tokens[len(tokens)-1].Type)
}

func TestLexerCRLF(t *testing.T) {
	input := "GET https://example.com/users\r\nContent-Type: application/json\r\n"
	tokens := NewLexer([]byte(input)).Tokenize()
	assert.Equal(t, TokenMethod, tokens[0].Type)
	assert.Equal(t, TokenHeaderKey, tokens[2].Type)
	assert.Equal(t, "Content-Type", tokens[2].Value)
}
