package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

func TestDetectFormat(t *testing.T) {
	assert.Equal(t, formatJSON, detectFormat("application/json", ""))
	assert.Equal(t, formatXML, detectFormat("application/xml", ""))
	assert.Equal(t, formatPlain, detectFormat("text/html", "<html></html>"))
	assert.Equal(t, formatJSON, detectFormat("", `{"a":1}`))
	assert.Equal(t, formatJSON, detectFormat("", `[1,2]`))
	assert.Equal(t, formatXML, detectFormat("", `<?xml version="1.0"?><a/>`))
	assert.Equal(t, formatPlain, detectFormat("", "<html>"))
	assert.Equal(t, formatPlain, detectFormat("", "plain text"))
}

func TestColorizeBodyJSON(t *testing.T) {
	out := colorizeBody(`{"a":1}`, "application/json")
	assert.Contains(t, out, "\x1b[")
	assert.Contains(t, stripANSI(out), `"a": 1`)
}

func TestColorizeBodyInvalidJSONUnchanged(t *testing.T) {
	body := `{"token": {{auth}}}`
	assert.Equal(t, body, colorizeBody(body, "application/json"))
}

func TestColorizeBodyHTMLNotGarbled(t *testing.T) {
	body := "<html><br><p>hi</html>"
	assert.Equal(t, body, colorizeBody(body, "text/html"))
}

func TestRequestContentType(t *testing.T) {
	req := &model.Request{Headers: []model.Header{{Key: "content-type", Value: "application/json"}}}
	assert.Equal(t, "application/json", requestContentType(req))
	assert.Equal(t, "", requestContentType(&model.Request{}))
}

func TestRequestBodyIsPrettyColorized(t *testing.T) {
	m := NewDetailModel("", parser.NewChainContext(), engine.NewCookieManager())
	m.width = 80
	m.request = &model.Request{
		Method:  "POST",
		URL:     "http://x",
		Headers: []model.Header{{Key: "Content-Type", Value: "application/json"}},
		Body:    `{"a":1}`,
	}
	content := m.buildRequestAccordion().content
	assert.True(t, strings.Contains(content, "\x1b["))
	assert.Contains(t, stripANSI(content), `"a": 1`)
}
