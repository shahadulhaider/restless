package gui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shahadulhaider/restless/internal/exporter"
	"github.com/shahadulhaider/restless/internal/model"
)

// ExporterService wraps the exporter package for Wails GUI consumption.
type ExporterService struct{}

// GenerateCode generates code for the given request in the specified language.
// Language is matched case-insensitively against available generator names
// (e.g. "Python", "JavaScript", "Go", "Java", "Ruby", "HTTPie", "curl", "PowerShell").
func (s *ExporterService) GenerateCode(req model.Request, language string) (string, error) {
	lower := strings.ToLower(language)
	for _, gen := range exporter.Generators {
		if strings.ToLower(gen.Name) == lower {
			return gen.Generate(req), nil
		}
	}
	return "", fmt.Errorf("unsupported language: %s", language)
}

// ToCurl generates a curl command for the given request.
func (s *ExporterService) ToCurl(req model.Request) string {
	return exporter.ToCurl(req)
}

// AvailableLanguages returns the sorted list of supported code generation language names.
func (s *ExporterService) AvailableLanguages() []string {
	seen := make(map[string]bool)
	var langs []string
	for _, gen := range exporter.Generators {
		if !seen[gen.Name] {
			seen[gen.Name] = true
			langs = append(langs, gen.Name)
		}
	}
	sort.Strings(langs)
	return langs
}
