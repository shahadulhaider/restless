package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const simpleOpenAPI3JSON = `{
  "openapi": "3.0.0",
  "info": {"title": "Pet Store", "version": "1.0.0"},
  "servers": [{"url": "https://api.petstore.example.com/v1"}],
  "paths": {
    "/pets": {
      "get": {
        "operationId": "listPets",
        "summary": "List all pets",
        "tags": ["pets"],
        "parameters": [],
        "responses": {}
      },
      "post": {
        "operationId": "createPet",
        "summary": "Create a pet",
        "tags": ["pets"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "name": {"type": "string"},
                  "age":  {"type": "integer"}
                }
              }
            }
          }
        },
        "responses": {}
      }
    }
  }
}`

const swagger2JSON = `{
  "swagger": "2.0",
  "info": {"title": "User API", "version": "1.0.0"},
  "host": "api.users.example.com",
  "basePath": "/v1",
  "schemes": ["https"],
  "paths": {
    "/users": {
      "get": {
        "operationId": "listUsers",
        "tags": ["users"],
        "parameters": [],
        "responses": {}
      }
    },
    "/users/{id}": {
      "get": {
        "operationId": "getUser",
        "tags": ["users"],
        "parameters": [
          {"name": "id", "in": "path", "required": true}
        ],
        "responses": {}
      }
    }
  }
}`

const openAPIWithQueryParams = `{
  "openapi": "3.0.0",
  "info": {"title": "Search API", "version": "1.0.0"},
  "servers": [{"url": "https://api.search.example.com"}],
  "paths": {
    "/search": {
      "get": {
        "operationId": "search",
        "tags": ["search"],
        "parameters": [
          {"name": "q", "in": "query", "required": true},
          {"name": "limit", "in": "query", "required": false}
        ],
        "responses": {}
      }
    }
  }
}`

const openAPIYAML = `openapi: "3.0.0"
info:
  title: YAML API
  version: "1.0.0"
servers:
  - url: https://api.yaml.example.com
paths:
  /items:
    get:
      operationId: listItems
      tags:
        - items
      responses: {}
`

func TestImportOpenAPI3JSON(t *testing.T) {
	spec := filepath.Join(t.TempDir(), "openapi.json")
	require.NoError(t, os.WriteFile(spec, []byte(simpleOpenAPI3JSON), 0644))

	out := t.TempDir()
	err := ImportOpenAPI(spec, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	// Should create pets.http
	content, err := os.ReadFile(filepath.Join(out, "pets.http"))
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "GET https://api.petstore.example.com/v1/pets")
	assert.Contains(t, s, "POST https://api.petstore.example.com/v1/pets")
	assert.Contains(t, s, "# @name listPets")
	assert.Contains(t, s, "# @name createPet")
	assert.Contains(t, s, "Content-Type: application/json")
}

func TestImportSwagger2JSON(t *testing.T) {
	spec := filepath.Join(t.TempDir(), "swagger.json")
	require.NoError(t, os.WriteFile(spec, []byte(swagger2JSON), 0644))

	out := t.TempDir()
	err := ImportOpenAPI(spec, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(out, "users.http"))
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "GET https://api.users.example.com/v1/users")
	assert.Contains(t, s, "# @name listUsers")
}

func TestImportOpenAPIPathParams(t *testing.T) {
	spec := filepath.Join(t.TempDir(), "swagger.json")
	require.NoError(t, os.WriteFile(spec, []byte(swagger2JSON), 0644))

	out := t.TempDir()
	require.NoError(t, ImportOpenAPI(spec, ImportOptions{OutputDir: out}))

	content, err := os.ReadFile(filepath.Join(out, "users.http"))
	require.NoError(t, err)
	// Path param {id} should become {{id}}
	assert.Contains(t, string(content), "{{id}}")
}

func TestImportOpenAPIQueryParams(t *testing.T) {
	spec := filepath.Join(t.TempDir(), "openapi.json")
	require.NoError(t, os.WriteFile(spec, []byte(openAPIWithQueryParams), 0644))

	out := t.TempDir()
	require.NoError(t, ImportOpenAPI(spec, ImportOptions{OutputDir: out}))

	content, err := os.ReadFile(filepath.Join(out, "search.http"))
	require.NoError(t, err)
	s := string(content)
	// Required query param should appear in URL
	assert.Contains(t, s, "q={{q}}")
	// Optional should not appear
	assert.NotContains(t, s, "limit=")
}

func TestImportOpenAPIYAML(t *testing.T) {
	spec := filepath.Join(t.TempDir(), "openapi.yaml")
	require.NoError(t, os.WriteFile(spec, []byte(openAPIYAML), 0644))

	out := t.TempDir()
	err := ImportOpenAPI(spec, ImportOptions{OutputDir: out})
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(out, "items.http"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "GET https://api.yaml.example.com/items")
}

func TestImportOpenAPIInvalidDoc(t *testing.T) {
	spec := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(spec, []byte(`{"info":{"title":"Bad"}}`), 0644))

	out := t.TempDir()
	err := ImportOpenAPI(spec, ImportOptions{OutputDir: out})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "openapi or swagger")
}
