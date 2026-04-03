package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shahadulhaider/restless/internal/exporter"
	"github.com/shahadulhaider/restless/internal/importer"
	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
	"github.com/shahadulhaider/restless/internal/writer"
)

// --- Serializer round-trip ---

func TestSerializerRoundTripMultiRequest(t *testing.T) {
	tmp := t.TempDir()
	content := "# @name GetHealth\nGET https://example.com/health\n\n###\n\n# @name CreateUser\nPOST https://example.com/users\nContent-Type: application/json\n\n{\"name\":\"Alice\"}\n"
	httpFile := filepath.Join(tmp, "col.http")
	require.NoError(t, os.WriteFile(httpFile, []byte(content), 0644))

	reqs, err := parser.ParseFile(httpFile)
	require.NoError(t, err)
	require.Len(t, reqs, 2)

	// Serialize and re-parse
	serialized := writer.SerializeRequests(reqs)
	reqs2, err := parser.ParseBytes([]byte(serialized), "round-trip")
	require.NoError(t, err)
	require.Len(t, reqs2, 2)
	assert.Equal(t, reqs[0].Method, reqs2[0].Method)
	assert.Equal(t, reqs[0].URL, reqs2[0].URL)
	assert.Equal(t, reqs[0].Name, reqs2[0].Name)
	assert.Equal(t, reqs[1].Method, reqs2[1].Method)
	assert.Equal(t, reqs[1].URL, reqs2[1].URL)
	assert.NotEmpty(t, reqs2[1].Body)
}

func TestSerializerRoundTripMetadata(t *testing.T) {
	req := model.Request{
		Name:   "MyReq",
		Method: "GET",
		URL:    "https://example.com/data",
		Metadata: model.RequestMetadata{
			NoRedirect: true,
		},
	}
	serialized := writer.SerializeRequest(req)
	reqs, err := parser.ParseBytes([]byte(serialized), "meta-test")
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "MyReq", reqs[0].Name)
	assert.True(t, reqs[0].Metadata.NoRedirect)
}

// --- File CRUD integration ---

func TestFileCRUDInsertUpdateDelete(t *testing.T) {
	tmp := t.TempDir()
	httpFile := filepath.Join(tmp, "requests.http")

	// Insert
	req1 := model.Request{Method: "GET", URL: "https://example.com/a", Name: "A"}
	require.NoError(t, writer.InsertRequest(httpFile, req1))

	req2 := model.Request{Method: "POST", URL: "https://example.com/b", Name: "B"}
	require.NoError(t, writer.InsertRequest(httpFile, req2))

	reqs, err := parser.ParseFile(httpFile)
	require.NoError(t, err)
	require.Len(t, reqs, 2)
	assert.Equal(t, "A", reqs[0].Name)
	assert.Equal(t, "B", reqs[1].Name)

	// Update
	updated := reqs[0]
	updated.Name = "A-updated"
	updated.URL = "https://example.com/a-updated"
	require.NoError(t, writer.UpdateRequest(httpFile, reqs[0], updated))

	reqs, err = parser.ParseFile(httpFile)
	require.NoError(t, err)
	require.Len(t, reqs, 2)
	assert.Equal(t, "A-updated", reqs[0].Name)
	assert.Equal(t, "B", reqs[1].Name)

	// Delete
	require.NoError(t, writer.DeleteRequest(httpFile, reqs[0]))

	reqs, err = parser.ParseFile(httpFile)
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "B", reqs[0].Name)
}

func TestFileCRUDDuplicate(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.http")
	dst := filepath.Join(tmp, "dst.http")

	require.NoError(t, writer.InsertRequest(src, model.Request{
		Method: "DELETE",
		URL:    "https://example.com/items/1",
		Name:   "DeleteItem",
	}))

	reqs, err := parser.ParseFile(src)
	require.NoError(t, err)
	require.Len(t, reqs, 1)

	require.NoError(t, writer.DuplicateRequest(reqs[0], dst))

	dstReqs, err := parser.ParseFile(dst)
	require.NoError(t, err)
	require.Len(t, dstReqs, 1)
	assert.Equal(t, "DELETE", dstReqs[0].Method)
	assert.Equal(t, "DeleteItem", dstReqs[0].Name)
}

// --- Importer integration ---

func TestInsomniaImportPipeline(t *testing.T) {
	tmp := t.TempDir()
	insomniaJSON := `{
		"__export_type": "insomnia",
		"resources": [
			{"_type":"workspace","_id":"wrk_1","name":"Test API","parentId":""},
			{
				"_type":"request","_id":"req_1","parentId":"wrk_1",
				"name":"Health","method":"GET","url":"https://api.test.com/health",
				"headers":[],"body":{},"authentication":{}
			},
			{
				"_type":"request","_id":"req_2","parentId":"wrk_1",
				"name":"Create","method":"POST","url":"https://api.test.com/users",
				"headers":[{"name":"Content-Type","value":"application/json","disabled":false}],
				"body":{"mimeType":"application/json","text":"{\"name\":\"Bob\"}"},
				"authentication":{}
			}
		]
	}`
	specFile := filepath.Join(tmp, "export.json")
	require.NoError(t, os.WriteFile(specFile, []byte(insomniaJSON), 0644))

	out := t.TempDir()
	require.NoError(t, importer.ImportInsomnia(specFile, importer.ImportOptions{OutputDir: out}))

	files, err := filepath.Glob(filepath.Join(out, "*.http"))
	require.NoError(t, err)
	require.Len(t, files, 1)

	reqs, err := parser.ParseFile(files[0])
	require.NoError(t, err)
	require.Len(t, reqs, 2)
	assert.Equal(t, "GET", reqs[0].Method)
	assert.Equal(t, "POST", reqs[1].Method)
}

func TestOpenAPIImportPipeline(t *testing.T) {
	specJSON := `{
		"openapi": "3.0.0",
		"info": {"title": "Integration API", "version": "1.0.0"},
		"servers": [{"url": "https://api.integration.example.com"}],
		"paths": {
			"/items": {
				"get": {
					"operationId": "listItems",
					"tags": ["items"],
					"responses": {}
				},
				"post": {
					"operationId": "createItem",
					"tags": ["items"],
					"requestBody": {
						"required": true,
						"content": {
							"application/json": {
								"schema": {"type": "object", "properties": {"name": {"type": "string"}}}
							}
						}
					},
					"responses": {}
				}
			}
		}
	}`
	tmp := t.TempDir()
	specFile := filepath.Join(tmp, "spec.json")
	require.NoError(t, os.WriteFile(specFile, []byte(specJSON), 0644))

	out := t.TempDir()
	require.NoError(t, importer.ImportOpenAPI(specFile, importer.ImportOptions{OutputDir: out}))

	httpFile := filepath.Join(out, "items.http")
	require.FileExists(t, httpFile)
	content, err := os.ReadFile(httpFile)
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "GET https://api.integration.example.com/items")
	assert.Contains(t, s, "POST https://api.integration.example.com/items")
	assert.Contains(t, s, "Content-Type: application/json")
}

// --- curl round-trip ---

func TestCurlRoundTrip(t *testing.T) {
	original := model.Request{
		Method: "PUT",
		URL:    "https://api.example.com/items/42",
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
			{Key: "Authorization", Value: "Bearer secret"},
		},
		Body: `{"status":"active"}`,
	}

	curlCmd := exporter.ToCurl(original)
	assert.Contains(t, curlCmd, "PUT")
	assert.Contains(t, curlCmd, "https://api.example.com/items/42")

	parsed, err := importer.ParseCurlCommand(curlCmd)
	require.NoError(t, err)
	assert.Equal(t, original.Method, parsed.Method)
	assert.Equal(t, original.URL, parsed.URL)
	assert.Equal(t, original.Body, parsed.Body)
	require.Len(t, parsed.Headers, 2)
}

func TestCurlImportPipeline(t *testing.T) {
	out := t.TempDir()
	cmd := `curl -X POST https://api.example.com/data -H "Content-Type: application/json" -d '{"key":"value"}'`
	require.NoError(t, importer.ImportCurl(cmd, importer.ImportOptions{OutputDir: out}))

	files, err := filepath.Glob(filepath.Join(out, "*.http"))
	require.NoError(t, err)
	require.Len(t, files, 1)

	reqs, err := parser.ParseFile(files[0])
	require.NoError(t, err)
	require.Len(t, reqs, 1)
	assert.Equal(t, "POST", reqs[0].Method)
	assert.Equal(t, "https://api.example.com/data", reqs[0].URL)
}

// --- Directory operations integration ---

func TestDirOpsCreateAndRename(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, writer.CreateDirectory(root, "my-folder"))
	require.DirExists(t, filepath.Join(root, "my-folder"))

	require.NoError(t, writer.CreateHTTPFile(root, "my-folder/api.http"))
	require.FileExists(t, filepath.Join(root, "my-folder", "api.http"))

	require.NoError(t, writer.RenameEntry(root, "my-folder/api.http", "my-folder/requests.http"))
	require.FileExists(t, filepath.Join(root, "my-folder", "requests.http"))
	assert.NoFileExists(t, filepath.Join(root, "my-folder", "api.http"))
}
