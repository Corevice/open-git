package compat

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Runner executes endpoint compatibility test cases.
type Runner struct{}

// Run sends the request for tc to baseURL+tc.Path, reads the response body,
// and runs each check, returning all results.
func (r *Runner) Run(tc EndpointTestCase, client *http.Client, baseURL string) []CheckResult {
	if client == nil {
		client = http.DefaultClient
	}

	var req *http.Request
	if tc.BuildRequest != nil {
		req = tc.BuildRequest()
	}
	targetURL := strings.TrimRight(baseURL, "/") + tc.Path
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return []CheckResult{{
			Name:   "request",
			Passed: false,
			Diff:   err.Error(),
		}}
	}

	if req == nil {
		req, err = http.NewRequestWithContext(context.Background(), tc.Method, targetURL, nil)
		if err != nil {
			return []CheckResult{{
				Name:   "request",
				Passed: false,
				Diff:   err.Error(),
			}}
		}
	} else {
		clone := req.Clone(context.Background())
		clone.URL = parsedURL
		if clone.Method == "" {
			clone.Method = tc.Method
		}
		req = clone
	}

	resp, err := client.Do(req)
	if err != nil {
		return []CheckResult{{
			Name:   "request",
			Passed: false,
			Diff:   err.Error(),
		}}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []CheckResult{{
			Name:   "read_body",
			Passed: false,
			Diff:   err.Error(),
		}}
	}

	resp.Request = req

	results := make([]CheckResult, 0, len(tc.Checks))
	for _, check := range tc.Checks {
		results = append(results, check(resp, body))
	}
	return results
}
