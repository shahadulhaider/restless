package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ImportOpenAPI imports an OpenAPI 3.x or Swagger 2.0 file and writes .http files.
// Supports JSON and YAML input.
func ImportOpenAPI(specPath string, opts ImportOptions) error {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}

	doc, err := parseOpenAPIDoc(data, specPath)
	if err != nil {
		return fmt.Errorf("parse spec: %w", err)
	}

	outDir := opts.OutputDir
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	baseURL := doc.baseURL()

	// If base URL is empty or just "/" (common with FastAPI), use {{baseUrl}} variable
	// and generate an http-client.env.json with the original server URL
	if baseURL == "" || baseURL == "/" {
		baseURL = "{{baseUrl}}"
		writeOpenAPIEnvFile(outDir, doc)
	} else if !strings.HasPrefix(baseURL, "http") {
		// Relative path like "/api/v1" — prefix with {{baseUrl}}
		writeOpenAPIEnvFile(outDir, doc)
		baseURL = "{{baseUrl}}" + baseURL
	}

	colName := sanitizeName(doc.Info.Title)
	if colName == "" {
		colName = "api"
	}

	// Group operations by tag; untagged go into the collection file.
	type operation struct {
		method string
		path   string
		op     openAPIOperation
	}
	tagGroups := make(map[string][]operation)

	for path, methods := range doc.Paths {
		for method, op := range methods {
			method = strings.ToUpper(method)
			tag := colName // default tag = collection name
			if len(op.Tags) > 0 {
				tag = sanitizeName(op.Tags[0])
			}
			tagGroups[tag] = append(tagGroups[tag], operation{method, path, op})
		}
	}

	// Sort tags for deterministic output
	tags := make([]string, 0, len(tagGroups))
	for t := range tagGroups {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	for _, tag := range tags {
		ops := tagGroups[tag]
		// Sort by method+path for determinism
		sort.Slice(ops, func(i, j int) bool {
			if ops[i].path != ops[j].path {
				return ops[i].path < ops[j].path
			}
			return ops[i].method < ops[j].method
		})

		var sb strings.Builder
		first := true
		for _, o := range ops {
			if !first {
				sb.WriteString("\n###\n\n")
			}
			first = false
			sb.WriteString(renderOperation(o.method, o.path, o.op, baseURL))
		}

		outFile := filepath.Join(outDir, tag+".http")
		if err := os.WriteFile(outFile, []byte(sb.String()), 0644); err != nil {
			return err
		}
	}

	return nil
}

func renderOperation(method, path string, op openAPIOperation, baseURL string) string {
	var sb strings.Builder

	// Request name
	name := op.OperationID
	if name == "" {
		name = op.Summary
	}
	if name != "" {
		sb.WriteString(fmt.Sprintf("# @name %s\n", name))
	}

	// Build URL — substitute path params with template vars
	urlPath := path
	for _, p := range op.Parameters {
		if p.In == "path" {
			urlPath = strings.ReplaceAll(urlPath, "{"+p.Name+"}", "{{"+p.Name+"}}")
		}
	}

	// Build query string from required query params
	var queryParts []string
	for _, p := range op.Parameters {
		if p.In == "query" && p.Required {
			queryParts = append(queryParts, p.Name+"={{"+p.Name+"}}")
		}
	}
	fullURL := strings.TrimRight(baseURL, "/") + urlPath
	if len(queryParts) > 0 {
		sort.Strings(queryParts)
		fullURL += "?" + strings.Join(queryParts, "&")
	}

	sb.WriteString(fmt.Sprintf("%s %s\n", method, fullURL))

	// Header params
	for _, p := range op.Parameters {
		if p.In == "header" {
			sb.WriteString(fmt.Sprintf("%s: {{%s}}\n", p.Name, p.Name))
		}
	}

	// Request body (OpenAPI 3)
	if op.RequestBody != nil {
		contentType, body := extractRequestBody(op.RequestBody)
		if contentType != "" {
			sb.WriteString(fmt.Sprintf("Content-Type: %s\n", contentType))
		}
		if body != "" {
			sb.WriteString("\n")
			sb.WriteString(body)
			if !strings.HasSuffix(body, "\n") {
				sb.WriteString("\n")
			}
		}
	}

	// Swagger 2 body parameter
	for _, p := range op.Parameters {
		if p.In == "body" {
			body := schemaExample(p.Schema)
			if body != "" {
				sb.WriteString("Content-Type: application/json\n")
				sb.WriteString("\n")
				sb.WriteString(body)
				if !strings.HasSuffix(body, "\n") {
					sb.WriteString("\n")
				}
			}
			break
		}
	}

	result := sb.String()
	return strings.TrimRight(result, "\n")
}

func extractRequestBody(rb *openAPIRequestBody) (contentType, body string) {
	// Prefer JSON, then any content type
	preferred := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain",
	}
	for _, ct := range preferred {
		if content, ok := rb.Content[ct]; ok {
			return ct, schemaExample(content.Schema)
		}
	}
	// Fall back to first available
	for ct, content := range rb.Content {
		return ct, schemaExample(content.Schema)
	}
	return "", ""
}

func schemaExample(schema *openAPISchema) string {
	if schema == nil {
		return ""
	}
	// If an example is provided, use it
	if schema.Example != nil {
		b, err := json.MarshalIndent(schema.Example, "", "  ")
		if err == nil {
			return string(b)
		}
	}
	// Generate stub from properties
	if schema.Type == "object" && len(schema.Properties) > 0 {
		stub := make(map[string]interface{})
		keys := make([]string, 0, len(schema.Properties))
		for k := range schema.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			prop := schema.Properties[k]
			stub[k] = typeDefault(prop)
		}
		b, err := json.MarshalIndent(stub, "", "  ")
		if err == nil {
			return string(b)
		}
	}
	return ""
}

func typeDefault(s *openAPISchema) interface{} {
	if s == nil {
		return ""
	}
	switch s.Type {
	case "integer", "number":
		return 0
	case "boolean":
		return false
	case "array":
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		return ""
	}
}

// --- Data structures ---

type openAPIDoc struct {
	Swagger string `json:"swagger" yaml:"swagger"` // "2.0"
	OpenAPI string `json:"openapi" yaml:"openapi"` // "3.x.x"
	Info    struct {
		Title string `json:"title" yaml:"title"`
	} `json:"info" yaml:"info"`
	// OpenAPI 3
	Servers []struct {
		URL string `json:"url" yaml:"url"`
	} `json:"servers" yaml:"servers"`
	// Swagger 2
	Host     string   `json:"host" yaml:"host"`
	BasePath string   `json:"basePath" yaml:"basePath"`
	Schemes  []string `json:"schemes" yaml:"schemes"`
	// Both
	Paths map[string]map[string]openAPIOperation `json:"paths" yaml:"paths"`
}

func (d *openAPIDoc) baseURL() string {
	if d.OpenAPI != "" && len(d.Servers) > 0 {
		return d.Servers[0].URL
	}
	if d.Host != "" {
		scheme := "https"
		if len(d.Schemes) > 0 {
			scheme = d.Schemes[0]
		}
		base := scheme + "://" + d.Host
		if d.BasePath != "" && d.BasePath != "/" {
			base += d.BasePath
		}
		return base
	}
	return ""
}

type openAPIOperation struct {
	OperationID string               `json:"operationId" yaml:"operationId"`
	Summary     string               `json:"summary" yaml:"summary"`
	Tags        []string             `json:"tags" yaml:"tags"`
	Parameters  []openAPIParameter   `json:"parameters" yaml:"parameters"`
	RequestBody *openAPIRequestBody  `json:"requestBody" yaml:"requestBody"`
}

type openAPIParameter struct {
	Name     string         `json:"name" yaml:"name"`
	In       string         `json:"in" yaml:"in"`
	Required bool           `json:"required" yaml:"required"`
	Schema   *openAPISchema `json:"schema" yaml:"schema"`
}

type openAPIRequestBody struct {
	Required bool `json:"required" yaml:"required"`
	Content  map[string]struct {
		Schema *openAPISchema `json:"schema" yaml:"schema"`
	} `json:"content" yaml:"content"`
}

type openAPISchema struct {
	Type       string                    `json:"type" yaml:"type"`
	Properties map[string]*openAPISchema `json:"properties" yaml:"properties"`
	Example    interface{}               `json:"example" yaml:"example"`
}

// writeOpenAPIEnvFile generates http-client.env.json with baseUrl from the spec's servers.
func writeOpenAPIEnvFile(outDir string, doc *openAPIDoc) {
	serverURL := "http://localhost:8000"
	if len(doc.Servers) > 0 && doc.Servers[0].URL != "" && doc.Servers[0].URL != "/" {
		serverURL = doc.Servers[0].URL
	} else if doc.Host != "" {
		scheme := "https"
		if len(doc.Schemes) > 0 {
			scheme = doc.Schemes[0]
		}
		serverURL = scheme + "://" + doc.Host
		if doc.BasePath != "" && doc.BasePath != "/" {
			serverURL += doc.BasePath
		}
	}

	envData := map[string]map[string]string{
		"dev": {"baseUrl": serverURL},
	}
	out, err := json.MarshalIndent(envData, "", "  ")
	if err != nil {
		return
	}
	envPath := filepath.Join(outDir, "restless.env.json")
	// Don't overwrite existing env file
	if _, err := os.Stat(envPath); err == nil {
		return
	}
	_ = os.WriteFile(envPath, out, 0644)
}

func parseOpenAPIDoc(data []byte, path string) (*openAPIDoc, error) {
	var doc openAPIDoc
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
	} else {
		// Default to JSON; also try YAML if JSON fails
		if err := json.Unmarshal(data, &doc); err != nil {
			if err2 := yaml.Unmarshal(data, &doc); err2 != nil {
				return nil, err // return original JSON error
			}
		}
	}
	if doc.OpenAPI == "" && doc.Swagger == "" {
		return nil, fmt.Errorf("not a valid OpenAPI/Swagger document (missing openapi or swagger field)")
	}
	return &doc, nil
}
