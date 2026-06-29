package compat

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

type junitTestSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type jsonReportCoverage struct {
	TotalEndpoints int     `json:"total_endpoints"`
	Passing        int     `json:"passing"`
	Failing        int     `json:"failing"`
	Unimplemented  int     `json:"unimplemented"`
	Rate           float64 `json:"rate"`
}

type jsonReportEndpointCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Diff   string `json:"diff,omitempty"`
}

type jsonReportEndpoint struct {
	Method string                    `json:"method"`
	Path   string                    `json:"path"`
	Status string                    `json:"status"`
	Checks []jsonReportEndpointCheck `json:"checks"`
}

type jsonReport struct {
	GeneratedAt    string               `json:"generated_at"`
	CoverageRate   float64              `json:"coverage_rate"`
	TotalEndpoints int                  `json:"total_endpoints"`
	Passing        int                  `json:"passing"`
	Failing        int                  `json:"failing"`
	Unimplemented  int                  `json:"unimplemented"`
	Coverage       jsonReportCoverage   `json:"coverage"`
	Endpoints      []jsonReportEndpoint `json:"endpoints"`
}

// JUnitReport produces JUnit XML bytes from compatibility check results.
func JUnitReport(suiteName string, results map[string][]CheckResult) []byte {
	suite := junitTestSuite{Name: suiteName}
	for endpoint, checks := range results {
		for _, check := range checks {
			suite.Tests++
			tc := junitTestCase{
				Name:      fmt.Sprintf("%s/%s", endpoint, check.Name),
				Classname: suiteName,
			}
			if !check.Passed {
				suite.Failures++
				tc.Failure = &junitFailure{
					Message: check.Name,
					Body:    check.Diff,
				}
			}
			suite.Cases = append(suite.Cases, tc)
		}
	}

	doc := junitTestSuites{Suites: []junitTestSuite{suite}}
	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return []byte("<testsuites></testsuites>")
	}
	return append([]byte(xml.Header), out...)
}

// JSONReport produces JSON bytes matching the internal compat report structure.
func JSONReport(totalEndpoints, passing, failing, unimplemented int, results map[string][]CheckResult) []byte {
	rate := 0.0
	if totalEndpoints > 0 {
		rate = float64(passing) / float64(totalEndpoints)
	}

	endpoints := make([]jsonReportEndpoint, 0, len(results))
	for key, checks := range results {
		method, path := splitEndpointKey(key)
		reportChecks := make([]jsonReportEndpointCheck, 0, len(checks))
		for _, check := range checks {
			reportChecks = append(reportChecks, jsonReportEndpointCheck{
				Name:   check.Name,
				Passed: check.Passed,
				Diff:   check.Diff,
			})
		}
		endpoints = append(endpoints, jsonReportEndpoint{
			Method: method,
			Path:   path,
			Status: endpointStatus(checks),
			Checks: reportChecks,
		})
	}

	report := jsonReport{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		CoverageRate:   rate,
		TotalEndpoints: totalEndpoints,
		Passing:        passing,
		Failing:        failing,
		Unimplemented:  unimplemented,
		Coverage: jsonReportCoverage{
			TotalEndpoints: totalEndpoints,
			Passing:        passing,
			Failing:        failing,
			Unimplemented:  unimplemented,
			Rate:           rate,
		},
		Endpoints: endpoints,
	}

	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return []byte("{}")
	}
	return out
}

func splitEndpointKey(key string) (method, path string) {
	parts := strings.SplitN(key, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "GET", key
}

func endpointStatus(checks []CheckResult) string {
	if len(checks) == 0 {
		return "unimplemented"
	}
	for _, check := range checks {
		if !check.Passed {
			return "fail"
		}
	}
	return "pass"
}
