package runner

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

// OutputFormat determines the output format for the runner.
type OutputFormat string

const (
	FormatText  OutputFormat = "text"  // default colored output
	FormatJSON  OutputFormat = "json"  // machine-readable JSON
	FormatJUnit OutputFormat = "junit" // JUnit XML for CI
)

// --- JSON output ---

type jsonReport struct {
	Summary    jsonSummary     `json:"summary"`
	Iterations []jsonIteration `json:"iterations,omitempty"`
	Requests   []jsonRequest   `json:"requests"`
}

type jsonSummary struct {
	TotalIterations  int  `json:"total_iterations"`
	PassedIterations int  `json:"passed_iterations"`
	TotalRequests    int  `json:"total_requests"`
	PassedRequests   int  `json:"passed_requests"`
	TotalAssertions  int  `json:"total_assertions"`
	PassedAssertions int  `json:"passed_assertions"`
	AnyFailed        bool `json:"any_failed"`
}

type jsonIteration struct {
	Index    int            `json:"index"`
	Passed   bool           `json:"passed"`
	DataVars map[string]string `json:"data_vars,omitempty"`
}

type jsonRequest struct {
	Name       string          `json:"name"`
	Method     string          `json:"method"`
	URL        string          `json:"url"`
	Status     int             `json:"status"`
	StatusText string          `json:"status_text"`
	Duration   int64           `json:"duration_ms"`
	Passed     bool            `json:"passed"`
	Assertions []jsonAssertion `json:"assertions,omitempty"`
	Error      string          `json:"error,omitempty"`
	Iteration  int             `json:"iteration,omitempty"`
}

type jsonAssertion struct {
	Expression string `json:"expression"`
	Passed     bool   `json:"passed"`
	Actual     string `json:"actual,omitempty"`
	Error      string `json:"error,omitempty"`
}

func writeJSON(w io.Writer, report *jsonReport) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(report)
}

// --- JUnit XML output ---

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Time     string          `xml:"time,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

func writeJUnit(w io.Writer, report *jsonReport, filePath string) {
	var cases []junitTestCase
	failures := 0
	totalTime := 0.0

	for _, req := range report.Requests {
		dur := float64(req.Duration) / 1000.0
		totalTime += dur
		tc := junitTestCase{
			Name:      req.Name,
			ClassName: filePath,
			Time:      fmt.Sprintf("%.3f", dur),
		}
		if !req.Passed {
			failures++
			msg := fmt.Sprintf("HTTP %d", req.Status)
			if req.Error != "" {
				msg = req.Error
			}
			var details string
			for _, a := range req.Assertions {
				if !a.Passed {
					details += fmt.Sprintf("%s (got %s)\n", a.Expression, a.Actual)
				}
			}
			tc.Failure = &junitFailure{
				Message: msg,
				Type:    "AssertionFailure",
				Body:    details,
			}
		}
		cases = append(cases, tc)
	}

	suites := junitTestSuites{
		Suites: []junitTestSuite{{
			Name:     filePath,
			Tests:    len(cases),
			Failures: failures,
			Time:     fmt.Sprintf("%.3f", totalTime),
			Cases:    cases,
		}},
	}

	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(suites)
	w.Write([]byte("\n"))
}

// newJSONReport creates a report from RunResult for JSON/JUnit output.
func newJSONReport(result RunResult) *jsonReport {
	return &jsonReport{
		Summary: jsonSummary{
			TotalIterations:  result.TotalIterations,
			PassedIterations: result.PassedIterations,
			TotalRequests:    result.TotalRequests,
			PassedRequests:   result.PassedRequests,
			TotalAssertions:  result.TotalAssertions,
			PassedAssertions: result.PassedAssertions,
			AnyFailed:        result.AnyFailed,
		},
	}
}

// RequestRecord is collected during a run for structured output.
type RequestRecord struct {
	Name       string
	Method     string
	URL        string
	Status     int
	StatusText string
	Duration   time.Duration
	Passed     bool
	Assertions []AssertionRecord
	Error      string
	Iteration  int
}

type AssertionRecord struct {
	Expression string
	Passed     bool
	Actual     string
	Error      string
}
