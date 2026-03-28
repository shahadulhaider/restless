package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	postman "github.com/rbretecher/go-postman-collection"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/shahadulhaider/restless/internal/parser"
)

type ImportOptions struct {
	OutputDir   string
	EnvFilePath string
}

func ImportPostman(collectionPath string, opts ImportOptions) error {
	f, err := os.Open(collectionPath)
	if err != nil {
		return fmt.Errorf("open collection: %w", err)
	}
	defer f.Close()

	col, err := postman.ParseCollection(f)
	if err != nil {
		return fmt.Errorf("parse collection: %w", err)
	}

	outDir := opts.OutputDir
	if outDir == "" {
		outDir = sanitizeName(col.Info.Name)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	return writeItems(col.Items, outDir, sanitizeName(col.Info.Name))
}

func writeItems(items []*postman.Items, dir string, defaultFileName string) error {
	var rootRequests []*postman.Items
	for _, item := range items {
		if item.IsGroup() {
			subDir := filepath.Join(dir, sanitizeName(item.Name))
			if err := os.MkdirAll(subDir, 0755); err != nil {
				return err
			}
			fileName := sanitizeName(item.Name) + ".http"
			if err := writeGroupToFile(item.Items, filepath.Join(subDir, fileName)); err != nil {
				return err
			}
		} else {
			rootRequests = append(rootRequests, item)
		}
	}

	if len(rootRequests) > 0 {
		fileName := defaultFileName + ".http"
		if err := writeGroupToFile(rootRequests, filepath.Join(dir, fileName)); err != nil {
			return err
		}
	}
	return nil
}

func writeGroupToFile(items []*postman.Items, path string) error {
	var sb strings.Builder
	for i, item := range items {
		if item.Request == nil {
			continue
		}
		if i > 0 {
			sb.WriteString("\n###\n\n")
		}
		sb.WriteString(convertItem(item))
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func convertItem(item *postman.Items) string {
	req := item.Request
	var sb strings.Builder

	if item.Name != "" {
		sb.WriteString(fmt.Sprintf("# @name %s\n", item.Name))
	}

	url := ""
	if req.URL != nil {
		url = req.URL.Raw
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", string(req.Method), url))

	for _, h := range req.Header {
		sb.WriteString(fmt.Sprintf("%s: %s\n", h.Key, h.Value))
	}

	if req.Auth != nil {
		authHeader := convertAuth(req.Auth)
		if authHeader != "" {
			sb.WriteString(authHeader + "\n")
		}
	}

	if req.Body != nil && req.Body.Raw != "" {
		sb.WriteString("\n")
		sb.WriteString(req.Body.Raw)
		sb.WriteString("\n")
	}

	return sb.String()
}

func convertAuth(auth *postman.Auth) string {
	if auth == nil {
		return ""
	}
	switch auth.Type {
	case postman.Bearer:
		for _, p := range auth.Bearer {
			if p.Key == "token" {
				return fmt.Sprintf("Authorization: Bearer %v", p.Value)
			}
		}
	case postman.Basic:
		var username, password string
		for _, p := range auth.Basic {
			switch p.Key {
			case "username":
				username = fmt.Sprintf("%v", p.Value)
			case "password":
				password = fmt.Sprintf("%v", p.Value)
			}
		}
		if username != "" {
			return fmt.Sprintf("Authorization: Basic {{%s}}:{{%s}}", username, password)
		}
	case postman.APIKey:
		for _, p := range auth.APIKey {
			if p.Key == "key" {
				return fmt.Sprintf("Authorization: ApiKey %v", p.Value)
			}
		}
	}
	return ""
}

type PostmanEnvironment struct {
	Name   string `json:"name"`
	Values []struct {
		Key     string `json:"key"`
		Value   string `json:"value"`
		Enabled bool   `json:"enabled"`
	} `json:"values"`
}

func ImportPostmanEnv(envPath string, outputDir string) error {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return err
	}
	var env PostmanEnvironment
	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}

	vars := make(map[string]string)
	for _, v := range env.Values {
		if v.Enabled {
			vars[v.Key] = v.Value
		}
	}

	envFile := map[string]interface{}{
		env.Name: vars,
	}
	out, err := json.MarshalIndent(envFile, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "http-client.env.json"), out, 0644)
}

func ValidateHTTPFile(path string) error {
	_, err := parser.ParseFile(path)
	return err
}

func GetRequests(collectionPath string) ([]*model.Request, error) {
	f, err := os.Open(collectionPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	col, err := postman.ParseCollection(f)
	if err != nil {
		return nil, err
	}

	var reqs []*model.Request
	collectRequests(col.Items, &reqs)
	return reqs, nil
}

func collectRequests(items []*postman.Items, reqs *[]*model.Request) {
	for _, item := range items {
		if item.IsGroup() {
			collectRequests(item.Items, reqs)
		} else if item.Request != nil {
			req := itemToModelRequest(item)
			*reqs = append(*reqs, req)
		}
	}
}

func itemToModelRequest(item *postman.Items) *model.Request {
	r := item.Request
	req := &model.Request{
		Name:   item.Name,
		Method: string(r.Method),
	}
	if r.URL != nil {
		req.URL = r.URL.Raw
	}
	for _, h := range r.Header {
		req.Headers = append(req.Headers, model.Header{Key: h.Key, Value: h.Value})
	}
	if r.Body != nil {
		req.Body = r.Body.Raw
	}
	return req
}

var nonAlpha = regexp.MustCompile(`[^a-z0-9_-]+`)

func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "_")
	s = nonAlpha.ReplaceAllString(s, "")
	if s == "" {
		s = "collection"
	}
	return s
}
