package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	userpreferencesUC "github.com/open-git/backend/internal/usecase/user_preferences"
)

type mockUserPreferencesRepo struct {
	byUserID map[int64]*entity.UserPreferences
}

func (m *mockUserPreferencesRepo) GetByUserID(_ context.Context, userID int64) (*entity.UserPreferences, error) {
	if m.byUserID == nil {
		return nil, nil
	}
	return m.byUserID[userID], nil
}

func (m *mockUserPreferencesRepo) Upsert(_ context.Context, prefs *entity.UserPreferences) error {
	if m.byUserID == nil {
		m.byUserID = map[int64]*entity.UserPreferences{}
	}
	copyPrefs := *prefs
	m.byUserID[prefs.UserID] = &copyPrefs
	return nil
}

func newUserPreferencesHandlerEcho(t *testing.T, repo *mockUserPreferencesRepo, auth echo.MiddlewareFunc) *echo.Echo {
	t.Helper()

	e := echo.New()
	h := handler.NewUserPreferencesHandler(
		userpreferencesUC.NewGetUserPreferencesUsecase(repo),
		userpreferencesUC.NewUpdateUserPreferencesUsecase(repo),
	)

	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func TestUserPreferencesGetDefault(t *testing.T) {
	repo := &mockUserPreferencesRepo{}
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", int64(42))
			return next(c)
		}
	}
	e := newUserPreferencesHandlerEcho(t, repo, auth)

	req := httptest.NewRequest(http.MethodGet, "/user/preferences", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["theme"] != "system" {
		t.Fatalf("theme = %q, want system", resp["theme"])
	}
}

func TestUserPreferencesPutValid(t *testing.T) {
	repo := &mockUserPreferencesRepo{}
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", int64(42))
			return next(c)
		}
	}
	e := newUserPreferencesHandlerEcho(t, repo, auth)

	body := bytes.NewBufferString(`{"theme":"dark"}`)
	req := httptest.NewRequest(http.MethodPut, "/user/preferences", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["theme"] != "dark" {
		t.Fatalf("theme = %q, want dark", resp["theme"])
	}
}

func TestUserPreferencesPutInvalid(t *testing.T) {
	repo := &mockUserPreferencesRepo{}
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", int64(42))
			return next(c)
		}
	}
	e := newUserPreferencesHandlerEcho(t, repo, auth)

	body := bytes.NewBufferString(`{"theme":"rainbow"}`)
	req := httptest.NewRequest(http.MethodPut, "/user/preferences", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestUserPreferencesGetUnauthorized(t *testing.T) {
	repo := &mockUserPreferencesRepo{}
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "Bad credentials"})
		}
	}
	e := newUserPreferencesHandlerEcho(t, repo, auth)

	req := httptest.NewRequest(http.MethodGet, "/user/preferences", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
