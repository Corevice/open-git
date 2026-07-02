package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/config"
	infraDB "github.com/open-git/backend/internal/infrastructure/database"

	_ "github.com/mattn/go-sqlite3"
)

// TestIntegration_CoreFlows boots the real router against a freshly-migrated
// SQLite database and drives the primary API flows end to end. Its purpose is
// to catch the class of regression that unit tests (which build their own
// schema from mocks) miss: a migration that doesn't create a table/column the
// handlers query. Any 5xx here means an endpoint is broken against the real
// schema.
func TestIntegration_CoreFlows(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("WEBHOOK_SECRET_KEY", "0000000000000000000000000000000000000000000000000000000000000000")
	// The login usecase signs JWTs with cfg.JWTSecret while the auth middleware
	// verifies with os.Getenv("JWT_SECRET"); in production both come from the
	// same env var, so keep them equal here.
	const jwtSecret = "integration-test-secret-key-1234567890"
	t.Setenv("JWT_SECRET", jwtSecret)

	db, err := sql.Open("sqlite3", filepath.Join(tmp, "og.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		t.Fatalf("enable fk: %v", err)
	}

	// Migrations live at backend/migrations, two levels up from cmd/server.
	if err := infraDB.RunMigrations(db, "sqlite", "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	cfg := config.Config{
		DBType:         "sqlite",
		JWTSecret:      jwtSecret,
		GitDataRoot:    filepath.Join(tmp, "git"),
		TLSMode:        "selfsigned",
		CISandboxMode:  "none",
		CISandboxImage: "alpine:3",
		AppName:        "opengit-test",
		LicenseName:    "MIT",
		WebBaseURL:     "http://localhost:8080",
		APIBaseURL:     "http://localhost:8080/api/v3",
	}

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = newHTTPErrorHandler()
	if _, err := registerHandlers(e, cfg, db); err != nil {
		t.Fatalf("register handlers: %v", err)
	}

	srv := &intServer{t: t, e: e}

	// Register + login.
	srv.do("POST", "/register", "", map[string]any{
		"login": "alice", "email": "alice@example.com", "password": "password12345",
	}, http.StatusCreated)

	var login struct {
		Token string `json:"token"`
	}
	srv.doJSON("POST", "/login", "", map[string]any{
		"login": "alice", "password": "password12345",
	}, http.StatusOK, &login)
	if login.Token == "" {
		t.Fatal("login returned empty token")
	}
	jwt := login.Token

	// Personal access token (used for the rest of the flows).
	var tok struct {
		Token string `json:"token"`
	}
	srv.doJSON("POST", "/user/tokens", jwt, map[string]any{
		"note":   "it",
		"scopes": []string{"repo", "admin:org", "read", "write", "admin", "user"},
	}, http.StatusCreated, &tok)
	if tok.Token == "" {
		t.Fatal("token create returned empty token")
	}
	pat := tok.Token

	// Each of these exercises a distinct table that a missing migration would
	// break. Status assertions are the important part.
	srv.do("GET", "/api/v3/user/preferences", pat, nil, http.StatusOK)
	srv.do("PUT", "/api/v3/user/preferences", pat, map[string]any{"theme": "dark"}, http.StatusOK)
	srv.do("POST", "/user/repos", pat, map[string]any{"name": "proj", "private": false}, http.StatusCreated)
	srv.do("GET", "/api/v3/repos/alice/proj", pat, nil, http.StatusOK)
	srv.do("POST", "/api/v3/repos/alice/proj/issues", pat, map[string]any{"title": "bug", "body": "x"}, http.StatusCreated)
	srv.do("GET", "/api/v3/repos/alice/proj/issues", pat, nil, http.StatusOK)
	srv.do("POST", "/api/v3/repos/alice/proj/labels", pat, map[string]any{"name": "bug", "color": "ff0000"}, http.StatusCreated)
	srv.do("POST", "/api/v3/repos/alice/proj/milestones", pat, map[string]any{"title": "v1"}, http.StatusCreated)
	srv.do("POST", "/api/v3/repos/alice/proj/hooks", pat, map[string]any{
		"config": map[string]any{"url": "http://example.com/h", "content_type": "json"},
		"events": []string{"push"}, "active": true,
	}, http.StatusCreated)
	srv.do("GET", "/api/v3/repos/alice/proj/hooks", pat, nil, http.StatusOK)
	srv.do("GET", "/api/v3/repos/alice/proj/actions/secrets", pat, nil, http.StatusOK)
	srv.do("GET", "/api/v3/repos/alice/proj/audit-log", pat, nil, http.StatusOK)
	srv.do("GET", "/api/v3/repos/alice/proj/actions/runs", pat, nil, http.StatusOK)
	srv.do("GET", "/api/v3/users/alice/repos", pat, nil, http.StatusOK)
	srv.do("POST", "/api/v3/orgs", pat, map[string]any{"login": "acme", "name": "Acme"}, http.StatusCreated)
	srv.do("POST", "/api/v3/orgs/acme/repos", pat, map[string]any{"name": "orgrepo"}, http.StatusCreated)
	srv.do("POST", "/api/v3/oauth-apps", pat, map[string]any{
		"name": "T", "homepage_url": "https://e.com", "callback_urls": []string{"http://localhost:9/cb"}, "owner_type": "user",
	}, http.StatusCreated)
	srv.do("POST", "/api/v1/alice/actions/runners/registration-token", pat, nil, http.StatusCreated)
}

type intServer struct {
	t *testing.T
	e *echo.Echo
}

func (s *intServer) request(method, path, token string, body any) *httptest.ResponseRecorder {
	s.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			s.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

func (s *intServer) do(method, path, token string, body any, want int) {
	s.t.Helper()
	rec := s.request(method, path, token, body)
	if rec.Code >= 500 {
		s.t.Fatalf("%s %s: server error %d: %s", method, path, rec.Code, rec.Body.String())
	}
	if rec.Code != want {
		s.t.Fatalf("%s %s: status = %d, want %d: %s", method, path, rec.Code, want, rec.Body.String())
	}
}

func (s *intServer) doJSON(method, path, token string, body any, want int, out any) {
	s.t.Helper()
	rec := s.request(method, path, token, body)
	if rec.Code != want {
		s.t.Fatalf("%s %s: status = %d, want %d: %s", method, path, rec.Code, want, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		s.t.Fatalf("%s %s: decode response: %v (%s)", method, path, err, rec.Body.String())
	}
}
