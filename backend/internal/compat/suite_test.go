package compat_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/compat"
)

func TestCompatSuiteSmoke(t *testing.T) {
	e := echo.New()

	e.GET("/user", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"login":      "octocat",
			"id":         1,
			"type":       "User",
			"site_admin": false,
			"avatar_url": "https://github.com/images/error/octocat_happy.gif",
			"html_url":   "https://github.com/octocat",
			"created_at": "2024-01-01T00:00:00Z",
		})
	})

	e.GET("/users/:username", func(c echo.Context) error {
		username := c.Param("username")
		return c.JSON(http.StatusOK, map[string]any{
			"login":      username,
			"id":         2,
			"type":       "User",
			"site_admin": false,
			"avatar_url": "https://github.com/images/error/octocat_happy.gif",
			"html_url":   "https://github.com/" + username,
			"created_at": "2024-01-01T00:00:00Z",
		})
	})

	e.GET("/rate_limit", func(c echo.Context) error {
		c.Response().Header().Set("X-RateLimit-Limit", "5000")
		c.Response().Header().Set("X-RateLimit-Remaining", "4999")
		c.Response().Header().Set("X-RateLimit-Reset", "1372700873")
		c.Response().Header().Set("X-RateLimit-Used", "1")
		return c.JSON(http.StatusOK, map[string]any{
			"resources": map[string]any{
				"core": map[string]any{
					"limit":     5000,
					"remaining": 4999,
					"reset":     1372700873,
					"used":      1,
				},
			},
			"rate": map[string]any{
				"limit":     5000,
				"remaining": 4999,
				"reset":     1372700873,
				"used":      1,
			},
		})
	})

	server := httptest.NewServer(e)
	defer server.Close()

	cases := []compat.EndpointTestCase{
		{
			Method: "GET",
			Path:   "/user",
			Checks: []compat.CheckFunc{
				compat.CheckStatusCode(http.StatusOK),
				compat.CheckSnakeCaseFields(),
				compat.CheckDatetimeFields(),
			},
		},
		{
			Method: "GET",
			Path:   "/users/octocat",
			Checks: []compat.CheckFunc{
				compat.CheckStatusCode(http.StatusOK),
				compat.CheckSnakeCaseFields(),
				compat.CheckDatetimeFields(),
			},
		},
		{
			Method: "GET",
			Path:   "/rate_limit",
			Checks: []compat.CheckFunc{
				compat.CheckStatusCode(http.StatusOK),
				compat.CheckRateLimitHeaders(),
				compat.CheckSnakeCaseFields(),
			},
		},
	}

	runner := &compat.Runner{}
	client := server.Client()
	results := make(map[string][]compat.CheckResult)
	passing := 0
	failing := 0

	for _, tc := range cases {
		key := tc.Method + " " + tc.Path
		checkResults := runner.Run(tc, client, server.URL)
		results[key] = checkResults

		allPassed := true
		for _, cr := range checkResults {
			if !cr.Passed {
				allPassed = false
			}
		}
		if allPassed {
			passing++
		} else {
			failing++
		}
	}

	for key, checkResults := range results {
		for _, cr := range checkResults {
			if !cr.Passed {
				t.Fatalf("%s: check %s failed: %s", key, cr.Name, cr.Diff)
			}
		}
	}

	report := compat.JSONReport(len(cases), passing, failing, 0, results)

	var parsed map[string]any
	if err := json.Unmarshal(report, &parsed); err != nil {
		t.Fatalf("JSONReport unmarshal: %v", err)
	}
	if _, ok := parsed["coverage_rate"]; !ok {
		t.Fatal("JSONReport missing coverage_rate field")
	}
}

func TestCompatSuiteExtended(t *testing.T) {
	e := echo.New()

	e.GET("/user", func(c echo.Context) error {
		if c.Request().Header.Get("Authorization") == "" {
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"message":            "Bad credentials",
				"documentation_url":  "https://docs.github.com/rest",
			})
		}
		return c.JSON(http.StatusOK, map[string]any{
			"login":      "octocat",
			"id":         1,
			"type":       "User",
			"site_admin": false,
		})
	})

	e.GET("/repos/:owner/:repo/issues", func(c echo.Context) error {
		return c.JSON(http.StatusOK, []map[string]any{
			{
				"id":         1,
				"number":     1,
				"title":      "Found a bug",
				"state":      "open",
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z",
			},
		})
	})

	e.GET("/repos/:owner/:repo/pulls", func(c echo.Context) error {
		return c.JSON(http.StatusOK, []map[string]any{
			{
				"id":         1,
				"number":     1,
				"title":      "Amazing new feature",
				"state":      "open",
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z",
			},
		})
	})

	e.GET("/repos/:owner/:repo/hooks", func(c echo.Context) error {
		return c.JSON(http.StatusOK, []map[string]any{
			{
				"id":         1,
				"name":       "web",
				"active":     true,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z",
				"config": map[string]any{
					"url":          "https://example.com/webhook",
					"content_type": "json",
				},
			},
		})
	})

	e.GET("/repos/:owner/:repo", func(c echo.Context) error {
		owner := c.Param("owner")
		repo := c.Param("repo")
		if owner == "nonexistent-user" && repo == "no-repo" {
			return c.JSON(http.StatusNotFound, map[string]any{
				"message":           "Not Found",
				"documentation_url": "https://docs.github.com/rest",
			})
		}
		return c.JSON(http.StatusOK, map[string]any{
			"id":         1296269,
			"name":       repo,
			"full_name":  owner + "/" + repo,
			"clone_url":  "https://github.com/" + owner + "/" + repo + ".git",
			"created_at": "2024-01-01T00:00:00Z",
		})
	})

	server := httptest.NewServer(e)
	defer server.Close()

	cases := []compat.EndpointTestCase{
		{
			Method: "GET",
			Path:   "/repos/octocat/hello-world",
			Checks: []compat.CheckFunc{
				compat.CheckStatusCode(http.StatusOK),
				compat.CheckIntegerID(),
				compat.CheckRequiredFields("name", "full_name", "clone_url", "created_at"),
				compat.CheckSnakeCaseFields(),
			},
		},
		{
			Method: "GET",
			Path:   "/repos/octocat/hello-world/issues",
			Checks: []compat.CheckFunc{
				compat.CheckStatusCode(http.StatusOK),
				compat.CheckSnakeCaseFields(),
			},
		},
		{
			Method: "GET",
			Path:   "/repos/octocat/hello-world/pulls",
			Checks: []compat.CheckFunc{
				compat.CheckStatusCode(http.StatusOK),
				compat.CheckSnakeCaseFields(),
			},
		},
		{
			Method: "GET",
			Path:   "/repos/octocat/hello-world/hooks",
			Checks: []compat.CheckFunc{
				compat.CheckStatusCode(http.StatusOK),
				compat.CheckSnakeCaseFields(),
			},
		},
		{
			Method: "GET",
			Path:   "/repos/nonexistent-user/no-repo",
			Checks: []compat.CheckFunc{
				compat.CheckStatusCode(http.StatusNotFound),
				compat.CheckDocumentationURL(),
			},
		},
		{
			Method: "GET",
			Path:   "/user",
			Checks: []compat.CheckFunc{
				compat.CheckStatusCode(http.StatusUnauthorized),
				compat.CheckDocumentationURL(),
			},
		},
	}

	runner := &compat.Runner{}
	client := server.Client()
	results := make(map[string][]compat.CheckResult)
	passing := 0
	failing := 0

	for _, tc := range cases {
		key := tc.Method + " " + tc.Path
		checkResults := runner.Run(tc, client, server.URL)
		results[key] = checkResults

		allPassed := true
		for _, cr := range checkResults {
			if !cr.Passed {
				allPassed = false
			}
		}
		if allPassed {
			passing++
		} else {
			failing++
		}
	}

	for key, checkResults := range results {
		for _, cr := range checkResults {
			if !cr.Passed {
				t.Fatalf("%s: check %s failed: %s", key, cr.Name, cr.Diff)
			}
		}
	}

	report := compat.JSONReport(len(cases), passing, failing, 0, results)

	var parsed map[string]any
	if err := json.Unmarshal(report, &parsed); err != nil {
		t.Fatalf("JSONReport unmarshal: %v", err)
	}
	if _, ok := parsed["coverage_rate"]; !ok {
		t.Fatal("JSONReport missing coverage_rate field")
	}
}
