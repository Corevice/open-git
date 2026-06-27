package middleware_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
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

func newAuthTestEcho(repo *mockAccessTokenRepo) *echo.Echo {
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

func TestMissingToken(t *testing.T) {
	e := newAuthTestEcho(&mockAccessTokenRepo{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
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

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
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

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGetUserUUIDAbsent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := middleware.GetUserUUID(c)
	if err == nil {
		t.Fatal("expected error when user uuid is absent")
	}

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected echo.HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", httpErr.Code)
	}
}

func TestGetUserUUIDPresent(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	want := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	middleware.SetUserUUID(c, want)

	got, err := middleware.GetUserUUID(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
