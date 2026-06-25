package tui

import "testing"

func TestJSONLineToPath(t *testing.T) {
	// An inline empty {} must not leave its key on the path, else sibling keys
	// get a spurious "args." prefix (the D19 regression).
	emptyObjBody := "{\n" +
		"  \"args\": {},\n" +
		"  \"headers\": {\n" +
		"    \"Accept-Encoding\": \"gzip\"\n" +
		"  }\n" +
		"}"

	emptyArrBody := "{\n" +
		"  \"files\": [],\n" +
		"  \"method\": \"GET\"\n" +
		"}"

	nestedBody := "{\n" +
		"  \"data\": {\n" +
		"    \"id\": 42\n" +
		"  }\n" +
		"}"

	scalarBody := "{\n" +
		"  \"name\": \"x\",\n" +
		"  \"age\": 5\n" +
		"}"

	tests := []struct {
		name string
		body string
		line int
		want string
	}{
		{"empty-object line itself", emptyObjBody, 1, "$.args"},
		{"nested key after empty object", emptyObjBody, 3, "$.headers.Accept-Encoding"},
		{"empty-array line itself", emptyArrBody, 1, "$.files"},
		{"scalar after empty array", emptyArrBody, 2, "$.method"},
		{"normal nested key", nestedBody, 2, "$.data.id"},
		{"second scalar sibling", scalarBody, 2, "$.age"},
		{"root brace", nestedBody, 0, "$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonLineToPath(tt.body, tt.line)
			if got != tt.want {
				t.Errorf("jsonLineToPath(line %d) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}
