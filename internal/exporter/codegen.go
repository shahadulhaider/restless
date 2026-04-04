package exporter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shahadulhaider/restless/internal/model"
)

// Generators maps language keys to code generation functions.
var Generators = map[string]struct {
	Name     string
	Generate func(model.Request) string
}{
	"p": {"Python", GeneratePython},
	"j": {"JavaScript", GenerateJavaScript},
	"g": {"Go", GenerateGo},
	"v": {"Java", GenerateJava},
	"r": {"Ruby", GenerateRuby},
	"h": {"HTTPie", GenerateHTTPie},
	"c": {"curl", ToCurl},
	"w": {"PowerShell", GeneratePowerShell},
}

func hasJSONBody(req model.Request) bool {
	for _, h := range req.Headers {
		if strings.EqualFold(h.Key, "Content-Type") && strings.Contains(h.Value, "json") {
			return true
		}
	}
	return false
}

func escPython(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func escJS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func escGo(s string) string {
	return strings.ReplaceAll(s, "`", "` + \"`\" + `")
}

func escJava(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func escRuby(s string) string {
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

// --- Python (requests) ---

func GeneratePython(req model.Request) string {
	var sb strings.Builder
	sb.WriteString("import requests\n\n")

	method := strings.ToLower(req.Method)
	sb.WriteString(fmt.Sprintf("response = requests.%s(\n", method))
	sb.WriteString(fmt.Sprintf("    '%s',\n", escPython(req.URL)))

	if len(req.Headers) > 0 {
		sb.WriteString("    headers={\n")
		for _, h := range req.Headers {
			sb.WriteString(fmt.Sprintf("        '%s': '%s',\n", escPython(h.Key), escPython(h.Value)))
		}
		sb.WriteString("    },\n")
	}

	if req.Body != "" {
		if hasJSONBody(req) {
			// Try to parse as JSON for pretty output
			var obj interface{}
			if err := json.Unmarshal([]byte(req.Body), &obj); err == nil {
				formatted, _ := json.Marshal(obj)
				sb.WriteString(fmt.Sprintf("    json=%s,\n", string(formatted)))
			} else {
				sb.WriteString(fmt.Sprintf("    data='%s',\n", escPython(req.Body)))
			}
		} else {
			sb.WriteString(fmt.Sprintf("    data='%s',\n", escPython(req.Body)))
		}
	}

	sb.WriteString(")\n\n")
	sb.WriteString("print(response.status_code)\n")
	sb.WriteString("print(response.text)\n")
	return sb.String()
}

// --- JavaScript (fetch) ---

func GenerateJavaScript(req model.Request) string {
	var sb strings.Builder

	hasHeaders := len(req.Headers) > 0
	hasBody := req.Body != ""
	needsOptions := req.Method != "GET" || hasHeaders || hasBody

	if !needsOptions {
		sb.WriteString(fmt.Sprintf("const response = await fetch(\"%s\");\n", escJS(req.URL)))
	} else {
		sb.WriteString(fmt.Sprintf("const response = await fetch(\"%s\", {\n", escJS(req.URL)))
		if req.Method != "GET" {
			sb.WriteString(fmt.Sprintf("  method: \"%s\",\n", req.Method))
		}
		if hasHeaders {
			sb.WriteString("  headers: {\n")
			for _, h := range req.Headers {
				sb.WriteString(fmt.Sprintf("    \"%s\": \"%s\",\n", escJS(h.Key), escJS(h.Value)))
			}
			sb.WriteString("  },\n")
		}
		if hasBody {
			if hasJSONBody(req) {
				sb.WriteString(fmt.Sprintf("  body: JSON.stringify(%s),\n", req.Body))
			} else {
				sb.WriteString(fmt.Sprintf("  body: \"%s\",\n", escJS(req.Body)))
			}
		}
		sb.WriteString("});\n")
	}

	sb.WriteString("\n")
	sb.WriteString("const data = await response.text();\n")
	sb.WriteString("console.log(response.status, data);\n")
	return sb.String()
}

// --- Go (net/http) ---

func GenerateGo(req model.Request) string {
	var sb strings.Builder

	hasBody := req.Body != ""

	if hasBody {
		sb.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"log\"\n\t\"net/http\"\n\t\"strings\"\n)\n\n")
		sb.WriteString("func main() {\n")
		sb.WriteString(fmt.Sprintf("\treq, err := http.NewRequest(\"%s\", \"%s\", strings.NewReader(`%s`))\n",
			req.Method, req.URL, escGo(req.Body)))
	} else {
		sb.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"log\"\n\t\"net/http\"\n)\n\n")
		sb.WriteString("func main() {\n")
		sb.WriteString(fmt.Sprintf("\treq, err := http.NewRequest(\"%s\", \"%s\", nil)\n",
			req.Method, req.URL))
	}
	sb.WriteString("\tif err != nil {\n\t\tlog.Fatal(err)\n\t}\n")

	for _, h := range req.Headers {
		sb.WriteString(fmt.Sprintf("\treq.Header.Set(\"%s\", \"%s\")\n", h.Key, escJava(h.Value)))
	}

	sb.WriteString("\n\tresp, err := http.DefaultClient.Do(req)\n")
	sb.WriteString("\tif err != nil {\n\t\tlog.Fatal(err)\n\t}\n")
	sb.WriteString("\tdefer resp.Body.Close()\n\n")
	sb.WriteString("\tbody, _ := io.ReadAll(resp.Body)\n")
	sb.WriteString("\tfmt.Println(resp.StatusCode, string(body))\n")
	sb.WriteString("}\n")
	return sb.String()
}

// --- Java (HttpClient) ---

func GenerateJava(req model.Request) string {
	var sb strings.Builder

	sb.WriteString("import java.net.URI;\n")
	sb.WriteString("import java.net.http.HttpClient;\n")
	sb.WriteString("import java.net.http.HttpRequest;\n")
	sb.WriteString("import java.net.http.HttpResponse;\n\n")

	sb.WriteString("HttpClient client = HttpClient.newHttpClient();\n")
	sb.WriteString("HttpRequest request = HttpRequest.newBuilder()\n")
	sb.WriteString(fmt.Sprintf("    .uri(URI.create(\"%s\"))\n", escJava(req.URL)))

	if req.Body != "" {
		sb.WriteString(fmt.Sprintf("    .method(\"%s\", HttpRequest.BodyPublishers.ofString(\"%s\"))\n",
			req.Method, escJava(req.Body)))
	} else if req.Method != "GET" {
		sb.WriteString(fmt.Sprintf("    .method(\"%s\", HttpRequest.BodyPublishers.noBody())\n", req.Method))
	} else {
		sb.WriteString("    .GET()\n")
	}

	for _, h := range req.Headers {
		sb.WriteString(fmt.Sprintf("    .header(\"%s\", \"%s\")\n", escJava(h.Key), escJava(h.Value)))
	}

	sb.WriteString("    .build();\n\n")
	sb.WriteString("HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());\n")
	sb.WriteString("System.out.println(response.statusCode() + \" \" + response.body());\n")
	return sb.String()
}

// --- Ruby (net/http) ---

func GenerateRuby(req model.Request) string {
	var sb strings.Builder

	sb.WriteString("require 'net/http'\nrequire 'uri'\n\n")
	sb.WriteString(fmt.Sprintf("uri = URI('%s')\n", escRuby(req.URL)))
	sb.WriteString("http = Net::HTTP.new(uri.host, uri.port)\n")
	sb.WriteString("http.use_ssl = uri.scheme == 'https'\n\n")

	rubyMethod := strings.Title(strings.ToLower(req.Method))
	sb.WriteString(fmt.Sprintf("request = Net::HTTP::%s.new(uri)\n", rubyMethod))

	for _, h := range req.Headers {
		sb.WriteString(fmt.Sprintf("request['%s'] = '%s'\n", escRuby(h.Key), escRuby(h.Value)))
	}

	if req.Body != "" {
		sb.WriteString(fmt.Sprintf("request.body = '%s'\n", escRuby(req.Body)))
	}

	sb.WriteString("\nresponse = http.request(request)\n")
	sb.WriteString("puts \"#{response.code} #{response.body}\"\n")
	return sb.String()
}

// --- HTTPie ---

func GenerateHTTPie(req model.Request) string {
	var parts []string
	parts = append(parts, "http")

	if req.Method != "GET" {
		parts = append(parts, req.Method)
	}
	parts = append(parts, req.URL)

	for _, h := range req.Headers {
		parts = append(parts, fmt.Sprintf("%s:'%s'", h.Key, escRuby(h.Value)))
	}

	// For HTTPie, JSON fields can be passed inline
	if req.Body != "" {
		if hasJSONBody(req) {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(req.Body), &obj); err == nil {
				for k, v := range obj {
					switch val := v.(type) {
					case string:
						parts = append(parts, fmt.Sprintf("%s='%s'", k, val))
					default:
						b, _ := json.Marshal(val)
						parts = append(parts, fmt.Sprintf("%s:=%s", k, string(b)))
					}
				}
			} else {
				// Raw body — pipe it
				return fmt.Sprintf("echo '%s' | %s", escRuby(req.Body), strings.Join(parts, " \\\n  "))
			}
		} else {
			return fmt.Sprintf("echo '%s' | %s", escRuby(req.Body), strings.Join(parts, " \\\n  "))
		}
	}

	if len(parts) > 3 {
		return parts[0] + " " + strings.Join(parts[1:], " \\\n  ")
	}
	return strings.Join(parts, " ")
}

// --- PowerShell (Invoke-RestMethod) ---

func GeneratePowerShell(req model.Request) string {
	var sb strings.Builder

	if len(req.Headers) > 0 {
		sb.WriteString("$headers = @{\n")
		for _, h := range req.Headers {
			sb.WriteString(fmt.Sprintf("    \"%s\" = \"%s\"\n", h.Key, escJava(h.Value)))
		}
		sb.WriteString("}\n")
	}

	if req.Body != "" {
		sb.WriteString(fmt.Sprintf("$body = '%s'\n", escRuby(req.Body)))
	}

	sb.WriteString("\n$response = Invoke-RestMethod")
	sb.WriteString(fmt.Sprintf(" -Uri \"%s\"", req.URL))
	sb.WriteString(fmt.Sprintf(" -Method %s", req.Method))

	if len(req.Headers) > 0 {
		sb.WriteString(" -Headers $headers")
	}
	if req.Body != "" {
		sb.WriteString(" -Body $body")
	}

	sb.WriteString("\n$response\n")
	return sb.String()
}
