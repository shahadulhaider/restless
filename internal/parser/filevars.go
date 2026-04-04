package parser

import (
	"os"
	"regexp"
	"strings"
)

// fileVarRe matches lines like: @varName = value
var fileVarRe = regexp.MustCompile(`^@(\w+)\s*=\s*(.+)$`)

// ExtractFileVariables scans the raw content of a .http file and extracts
// inline variable definitions (@varName = value). These are defined at the
// top of files (or between requests) and are local to that file.
func ExtractFileVariables(content []byte) map[string]string {
	vars := make(map[string]string)
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if m := fileVarRe.FindStringSubmatch(trimmed); m != nil {
			vars[m[1]] = strings.TrimSpace(m[2])
		}
	}
	return vars
}

// ExtractFileVariablesFromFile reads a file and extracts inline variables.
func ExtractFileVariablesFromFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractFileVariables(data), nil
}
