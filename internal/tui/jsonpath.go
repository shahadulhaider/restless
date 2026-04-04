package tui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// jsonLineToPath maps a line number in pretty-printed JSON to its JSON path.
// Returns something like "$.data[0].name" for the line at lineNum.
func jsonLineToPath(prettyJSON string, lineNum int) string {
	lines := strings.Split(prettyJSON, "\n")
	if lineNum < 0 || lineNum >= len(lines) {
		return "$"
	}

	// Parse the raw JSON to build a path map
	var raw interface{}
	// Strip ANSI codes before parsing
	cleaned := stripANSI(prettyJSON)
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return "$"
	}

	// Walk pretty-printed lines to track path
	var path []string
	arrayIdx := map[int]int{} // depth → current array index
	depth := 0

	for i := 0; i <= lineNum && i < len(lines); i++ {
		line := strings.TrimSpace(stripANSI(lines[i]))

		if line == "" {
			continue
		}

		// Track depth from braces/brackets
		if strings.HasSuffix(line, "{") || strings.HasSuffix(line, "[") {
			if strings.Contains(line, ":") {
				// "key": { or "key": [
				key := extractJSONKey(line)
				if key != "" {
					for len(path) > depth {
						path = path[:len(path)-1]
					}
					path = append(path, key)
				}
			} else if depth > 0 {
				// Array element start
			}
			depth++
			if strings.HasSuffix(line, "[") {
				arrayIdx[depth] = 0
			}
		} else if line == "}" || line == "}," || line == "]" || line == "]," {
			depth--
			if depth < 0 {
				depth = 0
			}
			if len(path) > depth {
				path = path[:depth]
			}
		} else if strings.Contains(line, ":") {
			// "key": value
			key := extractJSONKey(line)
			if key != "" {
				for len(path) > depth {
					path = path[:len(path)-1]
				}
				path = append(path, key)
			}
		} else {
			// Array element (plain value)
			if _, ok := arrayIdx[depth]; ok {
				for len(path) > depth-1 {
					path = path[:len(path)-1]
				}
				path = append(path, fmt.Sprintf("[%d]", arrayIdx[depth]))
				arrayIdx[depth]++
			}
		}
	}

	if len(path) == 0 {
		return "$"
	}

	result := "$"
	for _, p := range path {
		if strings.HasPrefix(p, "[") {
			result += p
		} else {
			result += "." + p
		}
	}
	return result
}

func extractJSONKey(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "\"") {
		return ""
	}
	end := strings.Index(line[1:], "\"")
	if end < 0 {
		return ""
	}
	return line[1 : end+1]
}
