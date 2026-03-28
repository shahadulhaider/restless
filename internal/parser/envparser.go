package parser

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	mrand "math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shahadulhaider/restless/internal/model"
)

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = crand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func LoadEnvironments(dir string) (*model.EnvironmentFile, error) {
	result := &model.EnvironmentFile{
		Shared:       make(map[string]string),
		Environments: make(map[string]model.Environment),
	}

	publicPath := filepath.Join(dir, "http-client.env.json")
	privatePath := filepath.Join(dir, "http-client.private.env.json")

	publicData, pubErr := parseEnvFile(publicPath)
	privateData, privErr := parseEnvFile(privatePath)

	if pubErr != nil && !os.IsNotExist(pubErr) {
		return nil, fmt.Errorf("reading %s: %w", publicPath, pubErr)
	}
	if privErr != nil && !os.IsNotExist(privErr) {
		return nil, fmt.Errorf("reading %s: %w", privatePath, privErr)
	}

	// Merge $shared
	if publicData != nil {
		if shared, ok := publicData["$shared"]; ok {
			for k, v := range shared {
				result.Shared[k] = v
			}
		}
	}

	// Collect all environment names from both files
	envNames := map[string]bool{}
	if publicData != nil {
		for k := range publicData {
			if k != "$shared" {
				envNames[k] = true
			}
		}
	}
	if privateData != nil {
		for k := range privateData {
			if k != "$shared" {
				envNames[k] = true
			}
		}
	}

	for name := range envNames {
		vars := make(map[string]string)

		// Start with public $shared
		if publicData != nil {
			if shared, ok := publicData["$shared"]; ok {
				for k, v := range shared {
					vars[k] = v
				}
			}
		}

		// Override with private $shared
		if privateData != nil {
			if shared, ok := privateData["$shared"]; ok {
				for k, v := range shared {
					vars[k] = v
				}
			}
		}

		// Override with public env-specific
		if publicData != nil {
			if envVars, ok := publicData[name]; ok {
				for k, v := range envVars {
					vars[k] = v
				}
			}
		}

		// Override with private env-specific (highest priority)
		if privateData != nil {
			if envVars, ok := privateData[name]; ok {
				for k, v := range envVars {
					vars[k] = v
				}
			}
		}

		result.Environments[name] = model.Environment{Name: name, Variables: vars}
	}

	return result, nil
}

func parseEnvFile(path string) (map[string]map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	result := make(map[string]map[string]string)
	for key, val := range raw {
		var vars map[string]string
		if err := json.Unmarshal(val, &vars); err != nil {
			continue
		}
		result[key] = vars
	}
	return result, nil
}

func ResolveEnvironment(envFile *model.EnvironmentFile, envName string) (map[string]string, error) {
	if envFile == nil {
		return make(map[string]string), nil
	}
	if envName == "" {
		result := make(map[string]string)
		for k, v := range envFile.Shared {
			result[k] = v
		}
		return result, nil
	}
	env, ok := envFile.Environments[envName]
	if !ok {
		return nil, fmt.Errorf("environment %q not found", envName)
	}
	return env.Variables, nil
}

var datetimeRe = regexp.MustCompile(`\{\{\$datetime\s+"([^"]+)"\}\}`)

func ResolveDynamicVars(s string) string {
	s = strings.ReplaceAll(s, "{{$uuid}}", generateUUID())
	s = strings.ReplaceAll(s, "{{$timestamp}}", strconv.FormatInt(time.Now().Unix(), 10))
	s = strings.ReplaceAll(s, "{{$randomInt}}", strconv.Itoa(mrand.IntN(1000)))
	s = datetimeRe.ReplaceAllStringFunc(s, func(match string) string {
		sub := datetimeRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		return time.Now().Format(sub[1])
	})
	return s
}
