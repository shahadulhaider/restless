package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ImportBruno imports a Bruno collection directory and writes .http files to opts.OutputDir.
func ImportBruno(collectionDir string, opts ImportOptions) error {
	outDir := opts.OutputDir
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Walk the collection directory
	return walkBrunoDir(collectionDir, outDir, collectionDir)
}

func walkBrunoDir(srcDir, dstDir, rootDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	// Collect .bru files in this directory
	var bruFiles []string
	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()
			// Handle environments directory specially
			if name == "environments" {
				envDir := filepath.Join(srcDir, name)
				// Write env file to dstDir root
				dstRoot := strings.TrimSuffix(dstDir, filepath.Base(dstDir))
				if dstRoot == "" || dstRoot == "/" {
					dstRoot = dstDir
				}
				_ = writeBrunoEnvs(envDir, dstDir)
				continue
			}
			subSrc := filepath.Join(srcDir, name)
			subDst := filepath.Join(dstDir, sanitizeName(name))
			if err := os.MkdirAll(subDst, 0755); err != nil {
				return err
			}
			if err := walkBrunoDir(subSrc, subDst, rootDir); err != nil {
				return err
			}
		} else if strings.HasSuffix(e.Name(), ".bru") {
			bruFiles = append(bruFiles, filepath.Join(srcDir, e.Name()))
		}
	}

	if len(bruFiles) == 0 {
		return nil
	}

	sort.Strings(bruFiles)

	// Convert all .bru files in this directory to one .http file
	// Use the directory name as the .http file name
	dirName := filepath.Base(srcDir)
	if dirName == "." || dirName == "" {
		dirName = "collection"
	}
	outPath := filepath.Join(dstDir, sanitizeName(dirName)+".http")

	var sb strings.Builder
	first := true
	for _, bruPath := range bruFiles {
		req, err := parseBruFile(bruPath)
		if err != nil || req == nil {
			continue
		}
		if !first {
			sb.WriteString("\n###\n\n")
		}
		first = false
		sb.WriteString(convertBruRequest(req))
	}

	if sb.Len() == 0 {
		return nil
	}
	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}

type bruRequest struct {
	Name    string
	Method  string
	URL     string
	Headers map[string]string
	Body    string
	Auth    bruAuth
}

type bruAuth struct {
	Type     string
	Token    string
	Username string
	Password string
}

func parseBruFile(path string) (*bruRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	sections := parseBruSections(string(data))
	req := &bruRequest{
		Headers: make(map[string]string),
	}

	// Extract name from meta section
	for _, line := range sections["meta"] {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			req.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
	}

	// Extract method and URL — section names like "get", "post", etc.
	for _, method := range []string{"get", "post", "put", "delete", "patch", "head", "options"} {
		if lines, ok := sections[method]; ok {
			req.Method = strings.ToUpper(method)
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "url:") {
					req.URL = strings.TrimSpace(strings.TrimPrefix(line, "url:"))
				}
			}
			break
		}
	}

	if req.Method == "" || req.URL == "" {
		return nil, nil // not a valid request
	}

	// Headers
	for _, line := range sections["headers"] {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, ":"); idx > 0 {
			k := strings.TrimSpace(line[:idx])
			v := strings.TrimSpace(line[idx+1:])
			if k != "" {
				req.Headers[k] = v
			}
		}
	}

	// Body (body:json, body:text, body:xml, body:form-urlencoded)
	for _, bodyType := range []string{"body:json", "body:text", "body:xml", "body:form-urlencoded"} {
		if lines, ok := sections[bodyType]; ok {
			req.Body = strings.TrimSpace(strings.Join(lines, "\n"))
			break
		}
	}

	// Auth
	if lines, ok := sections["auth:bearer"]; ok {
		req.Auth.Type = "bearer"
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "token:") {
				req.Auth.Token = strings.TrimSpace(strings.TrimPrefix(line, "token:"))
			}
		}
	} else if lines, ok := sections["auth:basic"]; ok {
		req.Auth.Type = "basic"
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "username:") {
				req.Auth.Username = strings.TrimSpace(strings.TrimPrefix(line, "username:"))
			} else if strings.HasPrefix(line, "password:") {
				req.Auth.Password = strings.TrimSpace(strings.TrimPrefix(line, "password:"))
			}
		}
	}

	return req, nil
}

func parseBruSections(content string) map[string][]string {
	sections := make(map[string][]string)
	currentSection := ""
	depth := 0

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		// Detect section opening: word chars followed by optional space and {
		if depth == 0 && strings.HasSuffix(trimmed, "{") && !strings.Contains(trimmed, " ") || isBruSectionHeader(trimmed) {
			currentSection = strings.TrimSuffix(trimmed, " {")
			currentSection = strings.TrimSuffix(currentSection, "{")
			currentSection = strings.TrimSpace(currentSection)
			depth = 1
			continue
		}

		if trimmed == "}" && depth > 0 {
			depth--
			if depth == 0 {
				currentSection = ""
			}
			continue
		}

		if currentSection != "" && trimmed != "" {
			// Skip script and test blocks content
			if !strings.HasPrefix(currentSection, "script") && !strings.HasPrefix(currentSection, "tests") {
				sections[currentSection] = append(sections[currentSection], line)
			}
		}
	}

	return sections
}

func isBruSectionHeader(line string) bool {
	// Matches patterns like: "get {", "post {", "headers {", "body:json {", "auth:bearer {"
	if !strings.HasSuffix(line, "{") {
		return false
	}
	name := strings.TrimSuffix(strings.TrimSpace(line), "{")
	name = strings.TrimSpace(name)
	return len(name) > 0
}

func convertBruRequest(req *bruRequest) string {
	var sb strings.Builder

	if req.Name != "" {
		sb.WriteString(fmt.Sprintf("# @name %s\n", req.Name))
	}

	sb.WriteString(fmt.Sprintf("%s %s\n", req.Method, req.URL))

	// Headers in stable order
	headerKeys := make([]string, 0, len(req.Headers))
	for k := range req.Headers {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)
	for _, k := range headerKeys {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, req.Headers[k]))
	}

	// Auth header
	switch req.Auth.Type {
	case "bearer":
		if req.Auth.Token != "" {
			sb.WriteString(fmt.Sprintf("Authorization: Bearer %s\n", req.Auth.Token))
		}
	case "basic":
		if req.Auth.Username != "" {
			sb.WriteString(fmt.Sprintf("Authorization: Basic %s:%s\n", req.Auth.Username, req.Auth.Password))
		}
	}

	if req.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(req.Body)
		if !strings.HasSuffix(req.Body, "\n") {
			sb.WriteString("\n")
		}
	}

	result := sb.String()
	return strings.TrimRight(result, "\n")
}

func writeBrunoEnvs(envDir, outDir string) error {
	entries, err := os.ReadDir(envDir)
	if err != nil {
		return err
	}

	envFile := make(map[string]map[string]string)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".bru") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(envDir, e.Name()))
		if err != nil {
			continue
		}
		sections := parseBruSections(string(data))
		vars := make(map[string]string)
		for _, line := range sections["vars"] {
			line = strings.TrimSpace(line)
			if idx := strings.Index(line, ":"); idx > 0 {
				k := strings.TrimSpace(line[:idx])
				v := strings.TrimSpace(line[idx+1:])
				if k != "" {
					vars[k] = v
				}
			}
		}
		if len(vars) > 0 {
			envName := strings.TrimSuffix(e.Name(), ".bru")
			envFile[envName] = vars
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
