package middleware_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/middleware"
)

type mockTokenLookup struct {
	byHash map[string]*domain.AccessToken
}

func (m *mockTokenLookup) FindByTokenHash(_ context.Context, tokenHash string) (*domain.AccessToken, error) {
	if m.byHash == nil {
		return nil, nil
	}
	return m.byHash[tokenHash], nil
}

func gitBasicAuthHeader(username, password string) string {
	creds := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return "Basic " + creds
}

func gitTokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestGitBasicAuthMiddleware(t *testing.T) {
	validToken := "valid-pat"
	expired := time.Now().UTC().Add(-time.Hour)
	revoked := time.Now().UTC()

	tests := []struct {
		name           string
		useOptional    bool
		authHeader     string
		lookup         *mockTokenLookup
		wantStatus     int
		wantWWWAuth    bool
		wantUserID     int64
		wantNextCalled bool
	}{
		{
			name:       "valid basic auth sets context and calls next",
			authHeader: gitBasicAuthHeader("user", validToken),
			lookup: &mockTokenLookup{
				byHash: map[string]*domain.AccessToken{
					gitTokenHash(validToken): {
						UserID: 42,
						Scopes: []string{"read", "write"},
					},
				},
			},
			wantStatus:     http.StatusOK,
			wantUserID:     42,
			wantNextCalled: true,
		},
		{
			name:           "missing authorization header returns 401",
			wantStatus:     http.StatusUnauthorized,
			wantWWWAuth:    true,
			wantNextCalled: false,
		},
		{
			name:           "bearer scheme returns 401",
			authHeader:     "Bearer xyz",
			wantStatus:     http.StatusUnauthorized,
			wantWWWAuth:    true,
			wantNextCalled: false,
		},
		{
			name:       "token not found returns 401",
			authHeader: gitBasicAuthHeader("user", "missing-token"),
			lookup:     &mockTokenLookup{byHash: map[string]*domain.AccessToken{}},
			wantStatus: http.StatusUnauthorized,
			wantWWWAuth: true,
			wantNextCalled: false,
		},
		{
			name:       "expired token returns 401",
			authHeader: gitBasicAuthHeader("user", "expired-token"),
			lookup: &mockTokenLookup{
				byHash: map[string]*domain.AccessToken{
					gitTokenHash("expired-token"): {
						UserID:    1,
						Scopes:    []string{"read"},
						ExpiresAt: &expired,
					},
				},
			},
			wantStatus:     http.StatusUnauthorized,
			wantWWWAuth:    true,
			wantNextCalled: false,
		},
		{
			name:       "revoked token returns 401",
			authHeader: gitBasicAuthHeader("user", "revoked-token"),
			lookup: &mockTokenLookup{
				byHash: map[string]*domain.AccessToken{
					gitTokenHash("revoked-token"): {
						UserID:    1,
						Scopes:    []string{"read"},
						RevokedAt: &revoked,
					},
				},
			},
			wantStatus:     http.StatusUnauthorized,
			wantWWWAuth:    true,
			wantNextCalled: false,
		},
		{
			name:           "optional auth with missing header calls next",
			useOptional:    true,
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:       "optional auth with valid credentials sets context",
			useOptional: true,
			authHeader: gitBasicAuthHeader("git", validToken),
			lookup: &mockTokenLookup{
				byHash: map[string]*domain.AccessToken{
					gitTokenHash(validToken): {
						UserID: 99,
						Scopes: []string{"read:repo"},
					},
				},
			},
			wantStatus:     http.StatusOK,
			wantUserID:     99,
			wantNextCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := tt.lookup
			if lookup == nil {
				lookup = &mockTokenLookup{}
			}

			nextCalled := false
			e := echo.New()
			if tt.useOptional {
				e.Use(middleware.OptionalGitAuth(lookup))
			} else {
				e.Use(middleware.GitBasicAuthMiddleware(lookup))
			}
			e.GET("/", func(c echo.Context) error {
				nextCalled = true
				if tt.wantUserID != 0 {
					userID, err := middleware.GetUserID(c)
					if err != nil {
						return err
					}
					if userID != tt.wantUserID {
						t.Fatalf("expected userID %d, got %d", tt.wantUserID, userID)
					}
				}
				return c.NoContent(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantWWWAuth {
				if got := rec.Header().Get("WWW-Authenticate"); got != `Basic realm="OpenGit"` {
					t.Fatalf("expected WWW-Authenticate header, got %q", got)
				}
			}
			if nextCalled != tt.wantNextCalled {
				t.Fatalf("expected nextCalled=%v, got %v", tt.wantNextCalled, nextCalled)
			}
		})
	}
}
