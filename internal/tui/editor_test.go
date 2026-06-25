package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shahadulhaider/restless/internal/model"
)

func TestEditorModelPreservesCarriedFieldsOnSave(t *testing.T) {
	req := model.Request{
		Name:        "rich",
		Method:      "POST",
		URL:         "https://api.example.com/old",
		HTTPVersion: "HTTP/1.1",
		Headers:     []model.Header{{Key: "Accept", Value: "application/json"}},
		Body:        `{"a":1}`,
		Assertions: []model.Assertion{
			{Target: "status", Operator: "==", Expected: "200", Raw: "status == 200"},
		},
		PreRequestScript:   `request.setHeader("X-Pre", "v");`,
		PostResponseScript: `client.log("done");`,
		Metadata: model.RequestMetadata{
			NoRedirect:  true,
			Timeout:     30 * time.Second,
			ConnTimeout: 5 * time.Second,
			Insecure:    true,
			Proxy:       "http://127.0.0.1:9999",
		},
	}

	m := NewEditorModelFromRequest(req)
	m.focus = fieldURL
	for _, r := range "/v2" {
		m.url.HandleKey(string(r))
	}

	saved := m.Request()

	assert.Equal(t, "https://api.example.com/old/v2", saved.URL)
	assert.Equal(t, "rich", saved.Name)

	assert.Len(t, saved.Assertions, 1)
	assert.Equal(t, "status == 200", saved.Assertions[0].Raw)
	assert.Equal(t, req.PreRequestScript, saved.PreRequestScript)
	assert.Equal(t, req.PostResponseScript, saved.PostResponseScript)
	assert.Equal(t, "HTTP/1.1", saved.HTTPVersion)
	assert.True(t, saved.Metadata.Insecure)
	assert.Equal(t, "http://127.0.0.1:9999", saved.Metadata.Proxy)
	assert.Equal(t, 5*time.Second, saved.Metadata.ConnTimeout)
	assert.True(t, saved.Metadata.NoRedirect)
	assert.Equal(t, 30*time.Second, saved.Metadata.Timeout)
}

func TestEditorModelPreservesBodyFileWhenNoInlineBodyTyped(t *testing.T) {
	req := model.Request{
		Name:     "filed",
		Method:   "POST",
		URL:      "https://api.example.com/old",
		BodyFile: "./payload.json",
	}

	m := NewEditorModelFromRequest(req)
	m.focus = fieldURL
	for _, r := range "/v2" {
		m.url.HandleKey(string(r))
	}

	saved := m.Request()

	assert.Equal(t, "https://api.example.com/old/v2", saved.URL)
	assert.Equal(t, "./payload.json", saved.BodyFile)
	assert.Equal(t, "", saved.Body)
}

func TestEditorModelInlineBodyClearsBodyFile(t *testing.T) {
	req := model.Request{
		Name:     "filed",
		Method:   "POST",
		URL:      "https://api.example.com",
		BodyFile: "./payload.json",
	}

	m := NewEditorModelFromRequest(req)
	m.focus = fieldBody
	for _, r := range `{"typed":true}` {
		m.body[m.bodyCursor].HandleKey(string(r))
	}

	saved := m.Request()

	assert.Equal(t, `{"typed":true}`, saved.Body)
	assert.Equal(t, "", saved.BodyFile)
}

func TestEditorModelSavesFormControlledMetadataChanges(t *testing.T) {
	req := model.Request{
		Method:   "GET",
		URL:      "https://api.example.com",
		Metadata: model.RequestMetadata{NoRedirect: false},
	}

	m := NewEditorModelFromRequest(req)
	m.noRedirect = true
	m.focus = fieldTimeout
	for _, r := range "45" {
		m.timeoutSecs.HandleKeyFiltered(string(r), isDigit)
	}

	saved := m.Request()

	assert.True(t, saved.Metadata.NoRedirect)
	assert.Equal(t, 45*time.Second, saved.Metadata.Timeout)
}
