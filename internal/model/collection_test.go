package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectionZeroValue(t *testing.T) {
	var c Collection

	assert.Equal(t, "", c.RootDir)
	assert.Nil(t, c.Files)
	assert.Empty(t, c.Files)
}

func TestHTTPFileZeroValue(t *testing.T) {
	var f HTTPFile

	assert.Equal(t, "", f.Path)
	assert.Nil(t, f.Requests)
	assert.Empty(t, f.Requests)
}

func TestHTTPFileEmptyRequests(t *testing.T) {
	f := HTTPFile{Path: "empty.http"}

	assert.Equal(t, "empty.http", f.Path)
	assert.Empty(t, f.Requests)
	assert.Len(t, f.Requests, 0)
}

func TestCollectionConstruction(t *testing.T) {
	c := Collection{
		RootDir: "/projects/api",
		Files: []HTTPFile{
			{
				Path: "users.http",
				Requests: []Request{
					{Name: "list", Method: "GET", URL: "/users"},
					{Name: "create", Method: "POST", URL: "/users"},
				},
			},
			{
				Path:     "health.http",
				Requests: []Request{{Name: "ping", Method: "GET", URL: "/health"}},
			},
		},
	}

	assert.Equal(t, "/projects/api", c.RootDir)
	assert.Len(t, c.Files, 2)

	assert.Equal(t, "users.http", c.Files[0].Path)
	assert.Len(t, c.Files[0].Requests, 2)
	assert.Equal(t, "list", c.Files[0].Requests[0].Name)
	assert.Equal(t, "create", c.Files[0].Requests[1].Name)
	assert.Equal(t, "POST", c.Files[0].Requests[1].Method)

	assert.Equal(t, "health.http", c.Files[1].Path)
	assert.Len(t, c.Files[1].Requests, 1)
	assert.Equal(t, "ping", c.Files[1].Requests[0].Name)
}

func TestCollectionDeepEqual(t *testing.T) {
	build := func() Collection {
		return Collection{
			RootDir: "/root",
			Files: []HTTPFile{
				{
					Path: "a.http",
					Requests: []Request{
						{
							Name:    "req",
							Method:  "GET",
							URL:     "https://example.com",
							Headers: []Header{{Key: "Accept", Value: "*/*"}},
							Assertions: []Assertion{
								{Target: "status", Operator: "==", Expected: "200", Raw: "status == 200"},
							},
						},
					},
				},
			},
		}
	}

	a := build()
	b := build()
	assert.Equal(t, a, b)

	b.Files[0].Requests[0].Headers[0].Value = "application/json"
	assert.NotEqual(t, a, b)
}
