package gui

import (
	"sort"
	"strings"
	"testing"

	"github.com/shahadulhaider/restless/internal/model"
)

func TestGenerateCode_Python(t *testing.T) {
	svc := &ExporterService{}
	req := model.Request{
		Method: "GET",
		URL:    "https://example.com/api",
	}
	code, err := svc.GenerateCode(req, "Python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "requests.get") {
		t.Errorf("expected python requests code, got:\n%s", code)
	}
}

func TestGenerateCode_CaseInsensitive(t *testing.T) {
	svc := &ExporterService{}
	req := model.Request{Method: "GET", URL: "https://example.com"}
	code, err := svc.GenerateCode(req, "python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(code, "requests") {
		t.Errorf("expected python code, got:\n%s", code)
	}
}

func TestGenerateCode_UnsupportedLanguage(t *testing.T) {
	svc := &ExporterService{}
	req := model.Request{Method: "GET", URL: "https://example.com"}
	_, err := svc.GenerateCode(req, "cobol")
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
	if !strings.Contains(err.Error(), "unsupported language") {
		t.Errorf("expected 'unsupported language' in error, got: %v", err)
	}
}

func TestToCurl(t *testing.T) {
	svc := &ExporterService{}
	req := model.Request{
		Method: "POST",
		URL:    "https://example.com/api",
		Headers: []model.Header{
			{Key: "Content-Type", Value: "application/json"},
		},
		Body: `{"key":"value"}`,
	}
	result := svc.ToCurl(req)
	if !strings.Contains(result, "curl") {
		t.Errorf("expected curl command, got:\n%s", result)
	}
	if !strings.Contains(result, "https://example.com/api") {
		t.Errorf("expected URL in curl output, got:\n%s", result)
	}
}

func TestAvailableLanguages(t *testing.T) {
	svc := &ExporterService{}
	langs := svc.AvailableLanguages()
	if len(langs) == 0 {
		t.Fatal("expected at least one language")
	}
	if !sort.StringsAreSorted(langs) {
		t.Errorf("expected sorted languages, got: %v", langs)
	}
	found := false
	for _, l := range langs {
		if strings.ToLower(l) == "python" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Python in languages, got: %v", langs)
	}
}
