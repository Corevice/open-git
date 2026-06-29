package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestRequireScope(t *testing.T) {
	tests := []struct {
		name           string
		scopes         []string
		jwtAuth        bool
		required       string
		wantStatus     int
		wantAccepted   string
		wantOAuthScope string
	}{
		{
			name:       "PAT missing required scope returns 403",
			scopes:     []string{"read"},
			required:   "repo",
			wantStatus: http.StatusForbidden,
			wantAccepted: "repo",
			wantOAuthScope: "read",
		},
		{
			name:       "PAT with required scope passes",
			scopes:     []string{"read", "repo"},
			required:   "repo",
			wantStatus: http.StatusOK,
		},
		{
			name:       "JWT auth bypasses scope gate",
			jwtAuth:    true,
			required:   "repo",
			wantStatus: http.StatusOK,
		},
		{
			name:       "PAT with empty scopes returns 403",
			scopes:     []string{},
			required:   "repo",
			wantStatus: http.StatusForbidden,
			wantAccepted: "repo",
			wantOAuthScope: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					if tt.jwtAuth {
						middleware.SetAuthContext(c, 1, nil)
					} else {
						middleware.SetAuthContext(c, 1, tt.scopes)
					}
					return next(c)
				}
			})
			e.Use(middleware.RequireScope(tt.required))
			e.GET("/", func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusForbidden {
				return
			}
			if got := rec.Header().Get("X-Accepted-OAuth-Scopes"); got != tt.wantAccepted {
				t.Fatalf("X-Accepted-OAuth-Scopes = %q, want %q", got, tt.wantAccepted)
			}
			if got := rec.Header().Get("X-OAuth-Scopes"); got != tt.wantOAuthScope {
				t.Fatalf("X-OAuth-Scopes = %q, want %q", got, tt.wantOAuthScope)
			}

			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["message"] != "Resource not accessible by personal access token" {
				t.Fatalf("unexpected message: %q", body["message"])
			}
		})
	}
}

func TestRequireAnyScope(t *testing.T) {
	tests := []struct {
		name           string
		scopes         []string
		required       []string
		wantStatus     int
		wantAccepted   string
		wantOAuthScope string
	}{
		{
			name:       "PAT with one accepted scope passes",
			scopes:     []string{"read", "admin:org"},
			required:   []string{"repo", "admin:org"},
			wantStatus: http.StatusOK,
		},
		{
			name:           "PAT missing all accepted scopes returns 403",
			scopes:         []string{"read"},
			required:       []string{"repo", "admin:org"},
			wantStatus:     http.StatusForbidden,
			wantAccepted:   "repo, admin:org",
			wantOAuthScope: "read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					middleware.SetAuthContext(c, 1, tt.scopes)
					return next(c)
				}
			})
			e.Use(middleware.RequireAnyScope(tt.required...))
			e.GET("/", func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusForbidden {
				return
			}
			if got := rec.Header().Get("X-Accepted-OAuth-Scopes"); got != tt.wantAccepted {
				t.Fatalf("X-Accepted-OAuth-Scopes = %q, want %q", got, tt.wantAccepted)
			}
			if got := rec.Header().Get("X-OAuth-Scopes"); got != tt.wantOAuthScope {
				t.Fatalf("X-OAuth-Scopes = %q, want %q", got, tt.wantOAuthScope)
			}
		})
	}
}
