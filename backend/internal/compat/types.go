package compat

import "net/http"

// CheckFunc validates an HTTP response and body, returning a CheckResult.
type CheckFunc func(resp *http.Response, body []byte) CheckResult

// CheckResult holds the outcome of a single compatibility check.
type CheckResult struct {
	Name   string
	Passed bool
	Diff   string
}

// EndpointTestCase describes one endpoint compatibility test.
type EndpointTestCase struct {
	Method       string
	Path         string
	BuildRequest func() *http.Request
	Checks       []CheckFunc
}
