package compat_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-git/backend/internal/compat"
)

func runCheck(t *testing.T, handler http.HandlerFunc, check compat.CheckFunc, requestURL string) compat.CheckResult {
	t.Helper()

	server := httptest.NewServer(handler)
	defer server.Close()

	url := server.URL
	if requestURL != "" {
		url = server.URL + requestURL
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	resp.Request = req
	return check(resp, body)
}

func TestCheckStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected int
		passed   bool
	}{
		{name: "200 matches 200", status: http.StatusOK, expected: 200, passed: true},
		{name: "201 does not match 200", status: http.StatusCreated, expected: 200, passed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCheck(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}, compat.CheckStatusCode(tt.expected), "")

			if result.Passed != tt.passed {
				t.Fatalf("Passed=%v, want %v (diff=%q)", result.Passed, tt.passed, result.Diff)
			}
		})
	}
}

func TestCheckRateLimitHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		passed  bool
	}{
		{
			name: "all headers present",
			headers: map[string]string{
				"X-RateLimit-Limit":     "5000",
				"X-RateLimit-Remaining": "4999",
				"X-RateLimit-Reset":     "1372700873",
				"X-RateLimit-Used":      "1",
			},
			passed: true,
		},
		{
			name: "missing X-RateLimit-Used",
			headers: map[string]string{
				"X-RateLimit-Limit":     "5000",
				"X-RateLimit-Remaining": "4999",
				"X-RateLimit-Reset":     "1372700873",
			},
			passed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCheck(t, func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(http.StatusOK)
			}, compat.CheckRateLimitHeaders(), "")

			if result.Passed != tt.passed {
				t.Fatalf("Passed=%v, want %v (diff=%q)", result.Passed, tt.passed, result.Diff)
			}
		})
	}
}

func TestCheckLinkPagination(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		link    string
		passed  bool
	}{
		{name: "object body passes", body: `{"login":"octocat"}`, passed: true},
		{name: "array with link passes", body: `[{"id":1}]`, link: `<https://api.github.com/user/repos?page=2>; rel="next"`, passed: true},
		{name: "array without link fails", body: `[{"id":1}]`, passed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCheck(t, func(w http.ResponseWriter, r *http.Request) {
				if tt.link != "" {
					w.Header().Set("Link", tt.link)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.body))
			}, compat.CheckLinkPagination(), "")

			if result.Passed != tt.passed {
				t.Fatalf("Passed=%v, want %v (diff=%q)", result.Passed, tt.passed, result.Diff)
			}
		})
	}
}

func TestCheckPerPageCap(t *testing.T) {
	items := make([]byte, 0, 1024)
	items = append(items, '[')
	for i := 0; i < 101; i++ {
		if i > 0 {
			items = append(items, ',')
		}
		items = append(items, `{"id":1}`...)
	}
	items = append(items, ']')

	tests := []struct {
		name       string
		requestURL string
		body       string
		passed     bool
	}{
		{name: "per_page over 100 with 101 items fails", requestURL: "/?per_page=150", body: string(items), passed: false},
		{name: "per_page over 100 with 100 items passes", requestURL: "/?per_page=150", body: `[{"id":1}]`, passed: true},
		{name: "per_page under cap passes", requestURL: "/?per_page=30", body: string(items), passed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCheck(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.body))
			}, compat.CheckPerPageCap(), tt.requestURL)

			if result.Passed != tt.passed {
				t.Fatalf("Passed=%v, want %v (diff=%q)", result.Passed, tt.passed, result.Diff)
			}
		})
	}
}

func TestCheckErrorFormat(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		passed bool
	}{
		{name: "200 passes vacuously", status: http.StatusOK, body: `{}`, passed: true},
		{
			name:   "404 with valid error body passes",
			status: http.StatusNotFound,
			body:   `{"message":"Not Found","documentation_url":"https://docs.github.com/rest"}`,
			passed: true,
		},
		{
			name:   "404 missing message fails",
			status: http.StatusNotFound,
			body:   `{"documentation_url":"https://docs.github.com/rest"}`,
			passed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCheck(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}, compat.CheckErrorFormat(), "")

			if result.Passed != tt.passed {
				t.Fatalf("Passed=%v, want %v (diff=%q)", result.Passed, tt.passed, result.Diff)
			}
		})
	}
}

func TestCheckValidationErrorFormat(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		passed bool
	}{
		{name: "200 passes vacuously", status: http.StatusOK, body: `{}`, passed: true},
		{
			name:   "422 with valid errors passes",
			status: http.StatusUnprocessableEntity,
			body:   `{"message":"Validation Failed","errors":[{"resource":"Issue","field":"title","code":"missing_field"}]}`,
			passed: true,
		},
		{
			name:   "422 missing errors array fails",
			status: http.StatusUnprocessableEntity,
			body:   `{"message":"Validation Failed"}`,
			passed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCheck(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}, compat.CheckValidationErrorFormat(), "")

			if result.Passed != tt.passed {
				t.Fatalf("Passed=%v, want %v (diff=%q)", result.Passed, tt.passed, result.Diff)
			}
		})
	}
}

func TestCheckSnakeCaseFields(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		passed bool
	}{
		{name: "snake_case passes", body: `{"login":"octocat","site_admin":false}`, passed: true},
		{name: "camelCase fails", body: `{"login":"octocat","siteAdmin":false}`, passed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCheck(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.body))
			}, compat.CheckSnakeCaseFields(), "")

			if result.Passed != tt.passed {
				t.Fatalf("Passed=%v, want %v (diff=%q)", result.Passed, tt.passed, result.Diff)
			}
		})
	}
}

func TestCheckDatetimeFields(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		passed bool
	}{
		{name: "RFC3339 created_at passes", body: `{"created_at":"2024-01-01T00:00:00Z"}`, passed: true},
		{name: "date-only created_at fails", body: `{"created_at":"2024-01-01"}`, passed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runCheck(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.body))
			}, compat.CheckDatetimeFields(), "")

			if result.Passed != tt.passed {
				t.Fatalf("Passed=%v, want %v (diff=%q)", result.Passed, tt.passed, result.Diff)
			}
		})
	}
}
