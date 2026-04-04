package assert

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/shahadulhaider/restless/internal/model"
	"github.com/tidwall/gjson"
)

// Evaluate runs a single assertion against a response and returns the result.
func Evaluate(a model.Assertion, resp *model.Response) model.AssertionResult {
	result := model.AssertionResult{Assertion: a}

	actual, err := extractValue(a.Target, resp)
	if err != nil {
		result.Error = err.Error()
		result.Actual = ""
		return result
	}
	result.Actual = actual

	result.Passed, result.Error = compare(actual, a.Operator, a.Expected)
	return result
}

// EvaluateAll runs all assertions on a request against a response.
func EvaluateAll(req *model.Request, resp *model.Response) []model.AssertionResult {
	results := make([]model.AssertionResult, 0, len(req.Assertions))
	for _, a := range req.Assertions {
		results = append(results, Evaluate(a, resp))
	}
	return results
}

// CountPassed returns the number of passed assertions.
func CountPassed(results []model.AssertionResult) int {
	n := 0
	for _, r := range results {
		if r.Passed {
			n++
		}
	}
	return n
}

// AllPassed returns true if all assertions passed.
func AllPassed(results []model.AssertionResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}

// extractValue gets the actual value from the response for the given target.
func extractValue(target string, resp *model.Response) (string, error) {
	switch {
	case target == "status":
		return strconv.Itoa(resp.StatusCode), nil

	case target == "body":
		return string(resp.Body), nil

	case target == "duration":
		return strconv.FormatInt(resp.Timing.Total.Milliseconds(), 10), nil

	case target == "size":
		return strconv.Itoa(len(resp.Body)), nil

	case strings.HasPrefix(target, "body.$."):
		path := strings.TrimPrefix(target, "body.$.")
		result := gjson.GetBytes(resp.Body, path)
		if !result.Exists() {
			return "", nil // empty string for non-existent paths
		}
		return result.String(), nil

	case strings.HasPrefix(target, "body.$"):
		// body.$ alone — the root
		path := strings.TrimPrefix(target, "body.$")
		if path == "" {
			return string(resp.Body), nil
		}
		return "", fmt.Errorf("invalid body path: %s", target)

	case strings.HasPrefix(target, "header."):
		headerName := strings.TrimPrefix(target, "header.")
		for _, h := range resp.Headers {
			if strings.EqualFold(h.Key, headerName) {
				return h.Value, nil
			}
		}
		return "", nil // empty for missing headers

	default:
		return "", fmt.Errorf("unknown assertion target: %s", target)
	}
}

// compare applies the operator between actual and expected values.
func compare(actual, operator, expected string) (bool, string) {
	switch operator {
	case "==":
		return actual == expected, ""

	case "!=":
		return actual != expected, ""

	case "<", ">", "<=", ">=":
		aNum, aErr := strconv.ParseFloat(actual, 64)
		eNum, eErr := strconv.ParseFloat(expected, 64)
		if aErr != nil || eErr != nil {
			return false, fmt.Sprintf("numeric comparison requires numbers: actual=%q expected=%q", actual, expected)
		}
		switch operator {
		case "<":
			return aNum < eNum, ""
		case ">":
			return aNum > eNum, ""
		case "<=":
			return aNum <= eNum, ""
		case ">=":
			return aNum >= eNum, ""
		}

	case "contains":
		return strings.Contains(actual, expected), ""

	case "matches":
		re, err := regexp.Compile(expected)
		if err != nil {
			return false, fmt.Sprintf("invalid regex: %s", err)
		}
		return re.MatchString(actual), ""

	case "exists":
		return actual != "" && actual != "null", ""

	case "!exists":
		return actual == "" || actual == "null", ""
	}

	return false, fmt.Sprintf("unknown operator: %s", operator)
}
