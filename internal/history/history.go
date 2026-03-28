package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shahadulhaider/restless/internal/model"
)

const maxEntries = 100

type HistoryEntry struct {
	Request     *model.Request  `json:"request"`
	Response    *model.Response `json:"response"`
	Environment string          `json:"environment"`
	Timestamp   time.Time       `json:"timestamp"`
	FilePath    string          `json:"-"`
}

func historyDir(rootDir string) string {
	return filepath.Join(rootDir, ".restless", "history")
}

func Save(rootDir string, req *model.Request, resp *model.Response, envName string) error {
	dir := historyDir(rootDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	entry := HistoryEntry{
		Request:     req,
		Response:    resp,
		Environment: envName,
		Timestamp:   time.Now().UTC(),
	}

	slug := slugify(req.Method + "_" + req.URL)
	name := fmt.Sprintf("%s_%s.json", entry.Timestamp.Format("20060102T150405.000000000Z"), slug)
	path := filepath.Join(dir, name)

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func List(rootDir string, req *model.Request) ([]HistoryEntry, error) {
	dir := historyDir(rootDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	key := strings.ToUpper(req.Method) + " " + req.URL
	var results []HistoryEntry

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		entry, err := Load(path)
		if err != nil {
			continue
		}
		if entry.Request == nil {
			continue
		}
		entryKey := strings.ToUpper(entry.Request.Method) + " " + entry.Request.URL
		if entryKey == key {
			entry.FilePath = path
			results = append(results, *entry)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	if len(results) > maxEntries {
		results = results[:maxEntries]
	}
	return results, nil
}

func Load(path string) (*HistoryEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entry HistoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	entry.FilePath = path
	return &entry, nil
}

func Diff(a, b *HistoryEntry) string {
	var sb strings.Builder

	if a.Response.StatusCode != b.Response.StatusCode {
		sb.WriteString(fmt.Sprintf("- Status: %d %s\n", a.Response.StatusCode, a.Response.Status))
		sb.WriteString(fmt.Sprintf("+ Status: %d %s\n", b.Response.StatusCode, b.Response.Status))
		sb.WriteString("\n")
	}

	aHeaders := headersMap(a.Response.Headers)
	bHeaders := headersMap(b.Response.Headers)
	for k, av := range aHeaders {
		if bv, ok := bHeaders[k]; !ok {
			sb.WriteString(fmt.Sprintf("- Header %s: %s\n", k, av))
		} else if av != bv {
			sb.WriteString(fmt.Sprintf("- Header %s: %s\n", k, av))
			sb.WriteString(fmt.Sprintf("+ Header %s: %s\n", k, bv))
		}
	}
	for k, bv := range bHeaders {
		if _, ok := aHeaders[k]; !ok {
			sb.WriteString(fmt.Sprintf("+ Header %s: %s\n", k, bv))
		}
	}

	aLines := strings.Split(strings.TrimSpace(string(a.Response.Body)), "\n")
	bLines := strings.Split(strings.TrimSpace(string(b.Response.Body)), "\n")
	sb.WriteString("\n--- body\n+++ body\n")
	sb.WriteString(lineDiff(aLines, bLines))

	return sb.String()
}

func headersMap(headers []model.Header) map[string]string {
	m := make(map[string]string, len(headers))
	for _, h := range headers {
		m[h.Key] = h.Value
	}
	return m
}

func lineDiff(a, b []string) string {
	aSet := make(map[string]bool, len(a))
	bSet := make(map[string]bool, len(b))
	for _, l := range a {
		aSet[l] = true
	}
	for _, l := range b {
		bSet[l] = true
	}

	var sb strings.Builder
	for _, l := range a {
		if !bSet[l] {
			sb.WriteString("- " + l + "\n")
		}
	}
	for _, l := range b {
		if !aSet[l] {
			sb.WriteString("+ " + l + "\n")
		}
	}
	return sb.String()
}

func slugify(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	result := sb.String()
	if len(result) > 60 {
		result = result[:60]
	}
	return result
}
