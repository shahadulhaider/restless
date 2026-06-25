package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shahadulhaider/restless/internal/model"
)

func TestSearchModelResetClearsQueryAndSelection(t *testing.T) {
	items := []SearchResult{
		{Request: &model.Request{Method: "GET", URL: "https://api/alpha", Name: "alpha"}, File: "a.http"},
		{Request: &model.Request{Method: "POST", URL: "https://api/beta", Name: "beta"}, File: "b.http"},
		{Request: &model.Request{Method: "GET", URL: "https://api/gamma", Name: "gamma"}, File: "c.http"},
	}
	m := NewSearchModel()
	m.SetItems(items)

	m.input = "a"
	m.results = m.filter(m.input)
	m.cursor = 2

	m.Reset()

	assert.Equal(t, "", m.input)
	assert.Equal(t, 0, m.cursor)
	assert.Equal(t, len(items), len(m.results))
}
