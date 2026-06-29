package middleware_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/middleware"
)

type mockAccessTokenRepo struct {
	byHash map[string]*domain.AccessToken
}

func (m *mockAccessTokenRepo) Create(_ context.Context, _ *domain.AccessToken) error {
	return nil
}

func (m *mockAccessTokenRepo) ListByUserID(_ context.Context, _ int64) ([]*domain.AccessToken, error) {
	return nil, nil
}

func (m *mockAccessTokenRepo) Revoke(_ context.Context, _, _ int64) error {
	return nil
}

func (m *mockAccessTokenRepo) FindByTokenHash(_ context.Context, tokenHash string) (*domain.AccessToken, error) {
	if m.byHash == nil {
		return nil, nil
	}
	return m.byHash[tokenHash], nil
}

func newAuthTestEcho(repo interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.AccessToken, error)
}) *echo.Echo {
	e := echo.New()
	e.Use(middleware.AuthMiddleware(repo))
	e.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	return e
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func assertUnauthorizedWithDocURL(t *testing.T, rec *httptest.ResponseRecorder, wantMessage string) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["message"] != wantMessage {
		t.Fatalf("message = %q, want %q", body["message"], wantMessage)
	}
	if body["documentation_url"] != "https://docs.github.com/rest" {
		t.Fatalf("documentation_url = %q", body["documentation_url"])
	}
}

func TestMissingToken(t *testing.T) {
	e := newAuthTestEcho(&mockAccessTokenRepo{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assertUnauthorizedWithDocURL(t, rec, "missing authorization token")
}

func TestRevokedToken(t *testing.T) {
	raw := "revoked-token"
	now := time.Now().UTC()
	repo := &mockAccessTokenRepo{
		byHash: map[string]*domain.AccessToken{
			tokenHash(raw): {
				UserID:    1,
				Scopes:    []string{"read"},
				RevokedAt: &now,
			},
		},
	}
	e := newAuthTestEcho(repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assertUnauthorizedWithDocURL(t, rec, "token has been revoked")
}

func TestValidToken(t *testing.T) {
	raw := "valid-token"
	repo := &mockAccessTokenRepo{
		byHash: map[string]*domain.AccessToken{
			tokenHash(raw): {
				UserID: 42,
				Scopes: []string{"read", "write"},
			},
		},
	}
	e := newAuthTestEcho(repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestBearerToken(t *testing.T) {
	raw := "ghp_xxx"
	repo := &mockAccessTokenRepo{
		byHash: map[string]*domain.AccessToken{
			tokenHash(raw): {
				UserID: 42,
				Scopes: []string{"read"},
			},
		},
	}
	e := newAuthTestEcho(repo)

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{name: "token prefix lowercase", header: "token " + raw, wantStatus: http.StatusOK},
		{name: "token prefix uppercase", header: "TOKEN " + raw, wantStatus: http.StatusOK},
		{name: "bearer prefix", header: "Bearer " + raw, wantStatus: http.StatusOK},
		{name: "empty header", header: "", wantStatus: http.StatusUnauthorized},
		{name: "basic auth", header: "Basic xyz", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestExpiredToken(t *testing.T) {
	raw := "expired-token"
	expired := time.Now().UTC().Add(-time.Hour)
	repo := &mockAccessTokenRepo{
		byHash: map[string]*domain.AccessToken{
			tokenHash(raw): {
				UserID:    1,
				Scopes:    []string{"read"},
				ExpiresAt: &expired,
			},
		},
	}
	e := newAuthTestEcho(repo)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assertUnauthorizedWithDocURL(t, rec, "token has expired")
}

func TestInvalidToken(t *testing.T) {
	raw := "unknown-token"
	e := newAuthTestEcho(&mockAccessTokenRepo{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assertUnauthorizedWithDocURL(t, rec, "invalid authorization token")
}

type errAccessTokenRepo struct {
	mockAccessTokenRepo
}

func (m *errAccessTokenRepo) FindByTokenHash(_ context.Context, _ string) (*domain.AccessToken, error) {
	return nil, errors.New("lookup failed")
}

func TestInvalidTokenLookupError(t *testing.T) {
	raw := "lookup-error-token"
	e := newAuthTestEcho(&errAccessTokenRepo{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assertUnauthorizedWithDocURL(t, rec, "invalid authorization token")
}
