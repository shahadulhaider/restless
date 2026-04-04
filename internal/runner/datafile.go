package runner

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadDataFile reads a CSV or JSON data file and returns a slice of variable maps,
// one per iteration. Format is detected by file extension.
func LoadDataFile(path string) ([]map[string]string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return loadCSV(path)
	case ".json":
		return loadJSON(path)
	default:
		return nil, fmt.Errorf("unsupported data file format %q (use .csv or .json)", ext)
	}
}

func loadCSV(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV %s: %w", path, err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV %s has no data rows (only header or empty)", path)
	}

	headers := records[0]
	var rows []map[string]string
	for _, record := range records[1:] {
		row := make(map[string]string)
		for i, header := range headers {
			header = strings.TrimSpace(header)
			if header == "" {
				continue
			}
			val := ""
			if i < len(record) {
				val = record[i]
			}
			row[header] = val
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func loadJSON(path string) ([]map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	// Try array of objects first
	var arr []map[string]interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("JSON %s must be an array of objects: %w", path, err)
	}

	if len(arr) == 0 {
		return nil, fmt.Errorf("JSON %s has no data (empty array)", path)
	}

	var rows []map[string]string
	for _, obj := range arr {
		row := make(map[string]string)
		for k, v := range obj {
			switch val := v.(type) {
			case string:
				row[k] = val
			case float64:
				// Preserve integers without decimal
				if val == float64(int64(val)) {
					row[k] = fmt.Sprintf("%d", int64(val))
				} else {
					row[k] = fmt.Sprintf("%g", val)
				}
			case bool:
				row[k] = fmt.Sprintf("%t", val)
			case nil:
				row[k] = ""
			default:
				// Nested objects/arrays — serialize to JSON string
				b, _ := json.Marshal(val)
				row[k] = string(b)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}
