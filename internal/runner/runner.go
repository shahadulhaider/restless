package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shahadulhaider/restless/internal/assert"
	"github.com/shahadulhaider/restless/internal/engine"
	"github.com/shahadulhaider/restless/internal/parser"
)

// RunConfig holds all configuration for a headless run.
type RunConfig struct {
	FilePath   string
	EnvName    string
	FailFast   bool
	Insecure   bool
	ProxyURL   string
	DataFile   string        // CSV or JSON path (empty = single run)
	Iterations int           // 0 = all rows in data file
	Delay      time.Duration // delay between iterations
	Output     io.Writer     // output writer (default: os.Stdout)
	ErrOutput  io.Writer     // error writer (default: os.Stderr)
}

// RunResult summarizes the outcome of a run.
type RunResult struct {
	TotalIterations  int
	PassedIterations int
	TotalRequests    int
	PassedRequests   int
	TotalAssertions  int
	PassedAssertions int
	AnyFailed        bool
}

// Run executes all requests in a .http file, optionally iterating with data.
func Run(cfg RunConfig) (RunResult, error) {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	if cfg.ErrOutput == nil {
		cfg.ErrOutput = os.Stderr
	}

	rootDir := filepath.Dir(cfg.FilePath)
	reqs, err := parser.ParseFile(cfg.FilePath)
	if err != nil {
		return RunResult{}, fmt.Errorf("parsing %s: %w", cfg.FilePath, err)
	}

	// Base variables: env + file vars
	baseVars := make(map[string]string)
	if cfg.EnvName != "" {
		envFile, loadErr := parser.LoadEnvironments(rootDir)
		if loadErr == nil && envFile != nil {
			vars, _ := parser.ResolveEnvironment(envFile, cfg.EnvName)
			for k, v := range vars {
				baseVars[k] = v
			}
		}
	}
	fileVars, _ := parser.ExtractFileVariablesFromFile(cfg.FilePath)
	for k, v := range fileVars {
		if _, exists := baseVars[k]; !exists {
			baseVars[k] = v
		}
	}

	// Apply CLI overrides to requests
	for i := range reqs {
		if cfg.Insecure {
			reqs[i].Metadata.Insecure = true
		}
		if cfg.ProxyURL != "" {
			reqs[i].Metadata.Proxy = cfg.ProxyURL
		}
	}

	// Load data file for iterations
	var dataRows []map[string]string
	if cfg.DataFile != "" {
		dataRows, err = LoadDataFile(cfg.DataFile)
		if err != nil {
			return RunResult{}, err
		}
		if cfg.Iterations > 0 && cfg.Iterations < len(dataRows) {
			dataRows = dataRows[:cfg.Iterations]
		}
	}

	// If no data file, single iteration with no extra vars
	if len(dataRows) == 0 {
		dataRows = []map[string]string{nil}
	}

	result := RunResult{TotalIterations: len(dataRows)}
	multiIteration := len(dataRows) > 1

	for iterIdx, dataRow := range dataRows {
		if multiIteration {
			// Print iteration header
			preview := iterationPreview(dataRow, 60)
			fmt.Fprintf(cfg.Output, "\n── Iteration %d/%d%s ──\n", iterIdx+1, len(dataRows), preview)
		}

		// Merge variables: base + data row
		iterVars := make(map[string]string)
		for k, v := range fileVars {
			iterVars[k] = v
		}
		if dataRow != nil {
			for k, v := range dataRow {
				iterVars[k] = v
			}
		}
		// Env vars override everything
		for k, v := range baseVars {
			iterVars[k] = v
		}

		iterPassed := true
		chainCtx := parser.NewChainContext()
		cookies := engine.NewCookieManager()

		for _, req := range reqs {
			result.TotalRequests++
			resolved, _ := parser.ResolveRequest(&req, iterVars, chainCtx)
			loaded, loadErr := parser.LoadFileBody(resolved, rootDir)
			if loadErr != nil {
				loaded = resolved
			}

			jar := cookies.JarForEnv(cfg.EnvName)
			resp, execErr := engine.ExecuteWithJar(loaded, jar)
			if execErr != nil {
				fmt.Fprintf(cfg.ErrOutput, "\033[31m✗\033[0m %s %s: %v\n", loaded.Method, loaded.URL, execErr)
				result.AnyFailed = true
				iterPassed = false
				if cfg.FailFast {
					return result, nil
				}
				continue
			}

			if loaded.Name != "" {
				chainCtx.StoreResponse(loaded.Name, resp)
			}

			// Run assertions
			results := assert.EvaluateAll(loaded, resp)
			resp.AssertionResults = results
			passed := assert.CountPassed(results)
			allOk := assert.AllPassed(results)
			result.TotalAssertions += len(results)
			result.PassedAssertions += passed

			name := loaded.Name
			if name == "" {
				name = loaded.Method + " " + loaded.URL
			}

			if len(results) == 0 {
				fmt.Fprintf(cfg.Output, "\033[32m✓\033[0m %s  %s  %dms\n",
					name, resp.Status, resp.Timing.Total.Milliseconds())
				result.PassedRequests++
			} else if allOk {
				fmt.Fprintf(cfg.Output, "\033[32m✓\033[0m %s  %s  %dms  (%d/%d passed)\n",
					name, resp.Status, resp.Timing.Total.Milliseconds(), passed, len(results))
				result.PassedRequests++
			} else {
				fmt.Fprintf(cfg.Output, "\033[31m✗\033[0m %s  %s  %dms\n",
					name, resp.Status, resp.Timing.Total.Milliseconds())
				for _, r := range results {
					if r.Passed {
						fmt.Fprintf(cfg.Output, "    \033[32m✓\033[0m %s\n", r.Assertion.Raw)
					} else {
						msg := fmt.Sprintf("    \033[31m✗\033[0m %s", r.Assertion.Raw)
						if r.Error != "" {
							msg += fmt.Sprintf(" (%s)", r.Error)
						} else {
							msg += fmt.Sprintf(" (got %s)", r.Actual)
						}
						fmt.Fprintln(cfg.Output, msg)
					}
				}
				result.AnyFailed = true
				iterPassed = false
			}

			if !allOk && cfg.FailFast {
				return result, nil
			}
		}

		if iterPassed {
			result.PassedIterations++
		}

		// Delay between iterations
		if cfg.Delay > 0 && iterIdx < len(dataRows)-1 {
			time.Sleep(cfg.Delay)
		}
	}

	// Print summary
	if multiIteration {
		fmt.Fprintf(cfg.Output, "\n%s\n", strings.Repeat("═", 50))
		fmt.Fprintf(cfg.Output, "%d iterations | %d passed | %d failed\n",
			result.TotalIterations, result.PassedIterations, result.TotalIterations-result.PassedIterations)
	}
	if result.TotalAssertions > 0 {
		fmt.Fprintf(cfg.Output, "%d requests | %d passed | %d failed | %d/%d assertions passed\n",
			result.TotalRequests, result.PassedRequests, result.TotalRequests-result.PassedRequests,
			result.PassedAssertions, result.TotalAssertions)
	}

	return result, nil
}

func iterationPreview(row map[string]string, maxLen int) string {
	if row == nil || len(row) == 0 {
		return ""
	}
	var parts []string
	for k, v := range row {
		if len(v) > 20 {
			v = v[:17] + "..."
		}
		parts = append(parts, k+"="+v)
	}
	preview := " (" + strings.Join(parts, ", ") + ")"
	if len(preview) > maxLen {
		preview = preview[:maxLen-3] + "...)"
	}
	return preview
}
