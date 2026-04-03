package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Insomnia v4 export structures
type insomniaExport struct {
	ExportType string              `json:"__export_type"`
	Resources  []insomniaResource  `json:"resources"`
}

type insomniaResource struct {
	Type     string             `json:"_type"`
	ID       string             `json:"_id"`
	ParentID string             `json:"parentId"`
	Name     string             `json:"name"`
	Method   string             `json:"method"`
	URL      string             `json:"url"`
	Headers  []insomniaHeader   `json:"headers"`
	Body     insomniaBody       `json:"body"`
	Auth     insomniaAuth       `json:"authentication"`
}

type insomniaHeader struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type insomniaBody struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type insomniaAuth struct {
	Type     string `json:"type"`
	Token    string `json:"token"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var insomniaVarRe = regexp.MustCompile(`\{\{\s*_\.(\w+)\s*\}\}`)

func convertInsomniaVars(s string) string {
	return insomniaVarRe.ReplaceAllString(s, "{{$1}}")
}

// ImportInsomnia imports an Insomnia v4 JSON export and writes .http files to opts.OutputDir.
func ImportInsomnia(collectionPath string, opts ImportOptions) error {
	data, err := os.ReadFile(collectionPath)
	if err != nil {
		return fmt.Errorf("reading collection: %w", err)
	}

	var export insomniaExport
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("parsing insomnia export: %w", err)
	}

	outDir := opts.OutputDir
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Build index maps
	byID := make(map[string]*insomniaResource)
	for i := range export.Resources {
		r := &export.Resources[i]
		byID[r.ID] = r
	}

	// Find workspace root
	var workspaceID string
	var workspaceName string
	for _, r := range export.Resources {
		if r.Type == "workspace" {
			workspaceID = r.ID
			workspaceName = r.Name
			break
		}
	}
	if workspaceName == "" {
		workspaceName = "collection"
	}

	// Find top-level groups and requests (children of workspace)
	var rootRequests []*insomniaResource
	groups := make(map[string][]*insomniaResource) // groupID → its children requests
	groupOrder := []string{}

	for i := range export.Resources {
		r := &export.Resources[i]
		switch r.Type {
		case "request_group":
			groupOrder = append(groupOrder, r.ID)
			// Requests inside this group are added below
		case "request":
			if r.ParentID == workspaceID {
				rootRequests = append(rootRequests, r)
			} else {
				groups[r.ParentID] = append(groups[r.ParentID], r)
			}
		}
	}

	// Write root-level requests
	if len(rootRequests) > 0 {
		path := filepath.Join(outDir, sanitizeName(workspaceName)+".http")
		if err := writeInsomniaGroup(rootRequests, path); err != nil {
			return err
		}
	}

	// Write each group to its own subdir
	for _, groupID := range groupOrder {
		group, ok := byID[groupID]
		if !ok {
			continue
		}
		reqs := groups[groupID]
		if len(reqs) == 0 {
			continue
		}
		subDir := filepath.Join(outDir, sanitizeName(group.Name))
		if err := os.MkdirAll(subDir, 0755); err != nil {
			return err
		}
		path := filepath.Join(subDir, sanitizeName(group.Name)+".http")
		if err := writeInsomniaGroup(reqs, path); err != nil {
			return err
		}
	}

	// Write environments if any
	var envVars []map[string]string
	var envNames []string
	for _, r := range export.Resources {
		if r.Type == "environment" && r.ParentID == workspaceID {
			vars := make(map[string]string)
			// Environments store data in a special field; use Data if present via raw decode
			// For now we handle the common case where variables are passed as name/data pairs
			// Note: Insomnia environment variables are in the "data" field (not in insomniaResource)
			// We need a raw decode for that
			envVars = append(envVars, vars)
			envNames = append(envNames, r.Name)
		}
	}

	// Write environments using raw decode for the data field
	if err := writeInsomniaEnvs(data, outDir, workspaceID); err != nil {
		// Non-fatal — environments are optional
		_ = err
	}
	_ = envVars
	_ = envNames

	return nil
}

func writeInsomniaEnvs(rawData []byte, outDir, workspaceID string) error {
	// Decode resources as raw messages to extract environment data field
	var raw struct {
		Resources []json.RawMessage `json:"resources"`
	}
	if err := json.Unmarshal(rawData, &raw); err != nil {
		return err
	}

	envFile := make(map[string]map[string]string)
	for _, rawRes := range raw.Resources {
		var res struct {
			Type     string                 `json:"_type"`
			ParentID string                 `json:"parentId"`
			Name     string                 `json:"name"`
			Data     map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(rawRes, &res); err != nil {
			continue
		}
		if res.Type != "environment" || res.ParentID != workspaceID {
			continue
		}
		vars := make(map[string]string)
		for k, v := range res.Data {
			if s, ok := v.(string); ok {
				vars[k] = convertInsomniaVars(s)
			}
		}
		if len(vars) > 0 {
			envFile[res.Name] = vars
		}
	}

	if len(envFile) == 0 {
		return nil
	}

	out, err := json.MarshalIndent(envFile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "http-client.env.json"), out, 0644)
}

func writeInsomniaGroup(reqs []*insomniaResource, path string) error {
	var sb strings.Builder
	first := true
	for _, r := range reqs {
		if !first {
			sb.WriteString("\n###\n\n")
		}
		first = false
		sb.WriteString(convertInsomniaRequest(r))
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func convertInsomniaRequest(r *insomniaResource) string {
	var sb strings.Builder

	if r.Name != "" {
		sb.WriteString(fmt.Sprintf("# @name %s\n", r.Name))
	}

	method := r.Method
	if method == "" {
		method = "GET"
	}
	url := convertInsomniaVars(r.URL)
	sb.WriteString(fmt.Sprintf("%s %s\n", method, url))

	for _, h := range r.Headers {
		if h.Disabled {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", h.Name, convertInsomniaVars(h.Value)))
	}

	// Auth header
	authHeader := convertInsomniaAuth(r.Auth)
	if authHeader != "" {
		sb.WriteString(authHeader + "\n")
	}

	if r.Body.Text != "" {
		sb.WriteString("\n")
		sb.WriteString(r.Body.Text)
		if !strings.HasSuffix(r.Body.Text, "\n") {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func convertInsomniaAuth(auth insomniaAuth) string {
	switch auth.Type {
	case "bearer":
		token := convertInsomniaVars(auth.Token)
		if token != "" {
			return fmt.Sprintf("Authorization: Bearer %s", token)
		}
	case "basic":
		user := convertInsomniaVars(auth.Username)
		if user != "" {
			return fmt.Sprintf("Authorization: Basic %s:%s", user, convertInsomniaVars(auth.Password))
		}
	}
	return ""
}
