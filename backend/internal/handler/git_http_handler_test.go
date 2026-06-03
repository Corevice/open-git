package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/labstack/echo/v4"

	gitinfra "github.com/Corevice/open-git/backend/internal/infrastructure/git"
	"github.com/Corevice/open-git/backend/internal/handler"
)

// alwaysValidTokenStore is a test double that always grants write permission.
type alwaysValidTokenStore struct{}

func (a *alwaysValidTokenStore) ValidateWriteToken(_ context.Context, _ string) (string, error) {
	return "test-user", nil
}

// buildReceivePack creates a minimal git receive-pack pkt-line stream with one
// reference update for the given refname. oldSHA and newSHA must be 40-char hex strings.
func buildReceivePackPktLine(oldSHA, newSHA, refname string) string {
	// Format: "<old> <new> <refname>\0<capabilities>"
	// We omit capabilities for simplicity.
	line := fmt.Sprintf("%s %s %s\n", oldSHA, newSHA, refname)
	pktLen := len(line) + 4
	return fmt.Sprintf("%04x%s0000", pktLen, line)
}

// setupSQLite creates an in-memory SQLite database with the minimal schema needed
// for branch protection lookups and returns a populated db.
func setupSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
CREATE TABLE users (
	id TEXT PRIMARY KEY,
	login TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE organizations (
	id TEXT PRIMARY KEY,
	login TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	plan_tier TEXT NOT NULL DEFAULT 'free',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE repositories (
	id TEXT PRIMARY KEY,
	organization_id TEXT NOT NULL,
	owner_id TEXT NOT NULL REFERENCES users(id),
	name TEXT NOT NULL,
	visibility TEXT NOT NULL DEFAULT 'private',
	default_branch TEXT NOT NULL DEFAULT 'main',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(owner_id, name)
);
CREATE TABLE branch_protections (
	id TEXT PRIMARY KEY,
	organization_id TEXT NOT NULL,
	repository_id TEXT NOT NULL REFERENCES repositories(id),
	pattern TEXT NOT NULL,
	required_reviews INTEGER NOT NULL DEFAULT 0,
	required_checks TEXT NOT NULL DEFAULT '[]',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(repository_id, pattern)
);
`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	// Seed: user "alice", repo "alice/testrepo", protection on "main".
	seed := `
INSERT INTO users (id, login, email, password_hash) VALUES ('u1', 'alice', 'alice@example.com', 'hash');
INSERT INTO organizations (id, login, name) VALUES ('o1', 'alice', 'Alice Org');
INSERT INTO repositories (id, organization_id, owner_id, name) VALUES ('r1', 'o1', 'u1', 'testrepo');
INSERT INTO branch_protections (id, organization_id, repository_id, pattern) VALUES ('bp1', 'o1', 'r1', 'main');
`
	if _, err := db.Exec(seed); err != nil {
		t.Fatalf("seed data: %v", err)
	}
	return db
}

// TestInfoRefsContentType verifies that the info/refs endpoint returns the correct
// Content-Type for git-upload-pack service advertisements.
func TestInfoRefsContentType(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "alice", "myrepo.git")
	if err := gitinfra.InitBare(repoPath); err != nil {
		t.Fatalf("init bare: %v", err)
	}

	e := echo.New()
	handler.RegisterGitRoutes(e, nil, dir, &alwaysValidTokenStore{})

	req := httptest.NewRequest(http.MethodGet, "/alice/myrepo/info/refs?service=git-upload-pack", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	const want = "application/x-git-upload-pack-advertisement"
	if !strings.HasPrefix(ct, want) {
		t.Fatalf("Content-Type = %q; want prefix %q", ct, want)
	}
}

// TestForceRejectProtectedBranch verifies that a force-push to a protected branch
// returns HTTP 422 Unprocessable Entity.
func TestForceRejectProtectedBranch(t *testing.T) {
	db := setupSQLite(t)
	defer db.Close()

	dir := t.TempDir()

	e := echo.New()
	handler.RegisterGitRoutes(e, db, dir, &alwaysValidTokenStore{})

	// Construct a git receive-pack pkt-line stream that updates refs/heads/main
	// from a non-zero old SHA — this is a force-push.
	oldSHA := strings.Repeat("a", 40)
	newSHA := strings.Repeat("b", 40)
	body := buildReceivePackPktLine(oldSHA, newSHA, "refs/heads/main")

	req := httptest.NewRequest(
		http.MethodPost,
		"/alice/testrepo/git-receive-pack",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/x-git-receive-pack-request")
	req.Header.Set("Authorization", "Bearer valid-test-token")

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
