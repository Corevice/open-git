package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

const importExtraSchema = `
CREATE TABLE IF NOT EXISTS import_jobs (
	id TEXT PRIMARY KEY,
	organization_id TEXT NOT NULL REFERENCES organizations(id),
	created_by TEXT NOT NULL REFERENCES users(id),
	source_url TEXT NOT NULL,
	target_repository_id TEXT,
	target_name TEXT NOT NULL,
	include TEXT NOT NULL DEFAULT '[]',
	status TEXT NOT NULL DEFAULT 'queued',
	phase TEXT NOT NULL DEFAULT 'clone',
	progress TEXT NOT NULL DEFAULT '{}',
	token_secret_ref TEXT,
	error TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS import_user_mappings (
	id TEXT PRIMARY KEY,
	import_job_id TEXT NOT NULL REFERENCES import_jobs(id),
	github_login TEXT NOT NULL,
	github_display_name TEXT NOT NULL DEFAULT '',
	local_user_id TEXT REFERENCES users(id),
	UNIQUE(import_job_id, github_login)
);

CREATE TABLE IF NOT EXISTS import_phase_checkpoints (
	import_job_id TEXT NOT NULL REFERENCES import_jobs(id),
	phase TEXT NOT NULL,
	last_cursor TEXT NOT NULL DEFAULT '',
	completed INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (import_job_id, phase)
);
`

func newTestImporterDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.RunMigrations(db, "sqlite", "../../migrations"); err != nil {
		_ = db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	if _, err := db.Exec(importExtraSchema); err != nil {
		_ = db.Close()
		t.Fatalf("create import schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func newTestImporterWorker(t *testing.T, db *sqlx.DB) *ImporterWorker {
	t.Helper()

	return NewImporterWorker(
		repository.NewImportJobRepository(db),
		repository.NewImportJobRepository(db),
		repository.NewImportJobRepository(db),
		repository.NewIssueRepository(db),
		repository.NewLabelRepository(db),
		repository.NewMilestoneRepository(db),
		repository.NewCommentRepository(db),
		repository.NewPullRequestRepository(db),
		repository.NewRepositoryRepository(db),
		repository.NewUserRepository(db),
	)
}

func seedImportFixtures(t *testing.T, db *sqlx.DB) (orgID, userID, jobID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	orgID = uuid.New()
	userID = uuid.New()
	jobID = uuid.New()

	mustExecImporter(t, db, `INSERT INTO organizations (id, login, name) VALUES (?, ?, ?)`, orgID, "acme", "Acme")
	mustExecImporter(t, db, `INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`, userID, "alice", "alice@example.com", "hash")

	job := &entity.ImportJob{
		ID:             jobID,
		OrganizationID: orgID,
		CreatedBy:      userID,
		SourceURL:      "https://github.com/owner/repo",
		TargetName:     "repo",
		Status:         entity.ImportJobStatusQueued,
		Phase:          entity.ImportJobPhaseClone,
	}
	if err := repository.NewImportJobRepository(db).Create(ctx, job); err != nil {
		t.Fatalf("create import job: %v", err)
	}
	return orgID, userID, jobID
}

func mustExecImporter(t *testing.T, db *sqlx.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func newMockGitHubServer(t *testing.T, failIssues bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(rateLimitRemainingHeader, "100")
		w.Header().Set(rateLimitResetHeader, "9999999999")

		switch {
		case r.URL.Path == "/repos/owner/repo" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":        "repo",
				"description": "Imported repo",
				"private":     false,
			})
		case r.URL.Path == "/repos/owner/repo/issues":
			if failIssues {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"number": 1,
					"title":  "First issue",
					"body":   "issue body",
					"state":  "open",
					"user": map[string]any{
						"login": "alice",
						"name":  "Alice",
					},
					"labels": []map[string]any{
						{"name": "bug", "color": "ff0000"},
					},
					"created_at": "2024-01-01T00:00:00Z",
					"updated_at": "2024-01-01T00:00:00Z",
				},
			})
		case strings.HasPrefix(r.URL.Path, "/repos/owner/repo/issues/") && strings.HasSuffix(r.URL.Path, "/comments"):
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.URL.Path == "/repos/owner/repo/pulls":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"number": 2,
					"title":  "First PR",
					"body":   "pr body",
					"state":  "open",
					"draft":  false,
					"merged": false,
					"user": map[string]any{
						"login": "alice",
					},
					"head": map[string]any{
						"ref": "feature",
						"sha": "abc",
					},
					"base": map[string]any{
						"ref": "main",
						"sha": "def",
					},
					"created_at": "2024-01-02T00:00:00Z",
					"updated_at": "2024-01-02T00:00:00Z",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestHandleGitHubImport_CompletesJob(t *testing.T) {
	db := newTestImporterDB(t)
	orgID, _, jobID := seedImportFixtures(t, db)
	server := newMockGitHubServer(t, false)
	t.Cleanup(server.Close)

	gitRoot := t.TempDir()
	worker := newTestImporterWorker(t, db).
		WithAPIBase(server.URL).
		WithGitStoragePath(gitRoot).
		WithCloneFn(func(_ context.Context, _, _, destPath string) error {
			return nil
		})

	payload, err := json.Marshal(GitHubImportPayload{
		ImportJobID:    jobID.String(),
		OrganizationID: orgID.String(),
		SourceOwner:    "owner",
		SourceRepo:     "repo",
		Token:          "test-token",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(TypeGitHubImport, payload)
	if err := worker.HandleGitHubImport(context.Background(), task); err != nil {
		t.Fatalf("HandleGitHubImport returned error: %v", err)
	}

	jobRepo := repository.NewImportJobRepository(db)
	got, err := jobRepo.GetByID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != entity.ImportJobStatusCompleted {
		t.Fatalf("status: got %q, want %q", got.Status, entity.ImportJobStatusCompleted)
	}
	if got.Phase != entity.ImportJobPhaseDone {
		t.Fatalf("phase: got %q, want %q", got.Phase, entity.ImportJobPhaseDone)
	}
}

func TestCheckRateLimitHeaders_ReturnsErrRateLimitExceeded(t *testing.T) {
	worker := NewImporterWorker(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	resp := &http.Response{
		Header: http.Header{
			rateLimitRemainingHeader: []string{"0"},
			rateLimitResetHeader:     []string{"9999999999"},
		},
	}

	err := worker.checkRateLimitHeaders(context.Background(), resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Fatalf("expected ErrRateLimitExceeded, got %v", err)
	}
}

func TestMapUser_CreatesGhostMapping(t *testing.T) {
	db := newTestImporterDB(t)
	_, _, jobID := seedImportFixtures(t, db)
	worker := newTestImporterWorker(t, db)

	localID := worker.mapUser(context.Background(), jobID, "ghost-user", "Ghost User")
	if localID != uuid.Nil {
		t.Fatalf("expected nil UUID for ghost user, got %s", localID)
	}

	mappingRepo := repository.NewImportJobRepository(db)
	mapping, err := mappingRepo.GetMappingByLogin(context.Background(), jobID, "ghost-user")
	if err != nil {
		t.Fatalf("GetMappingByLogin: %v", err)
	}
	if mapping == nil {
		t.Fatal("expected mapping to be created")
	}
	if mapping.LocalUserID != nil {
		t.Fatalf("expected local_user_id NULL, got %v", mapping.LocalUserID)
	}
}

func TestHandleGitHubImport_FailedPhaseSetsFailedStatus(t *testing.T) {
	db := newTestImporterDB(t)
	orgID, _, jobID := seedImportFixtures(t, db)
	server := newMockGitHubServer(t, true)
	t.Cleanup(server.Close)

	gitRoot := t.TempDir()
	worker := newTestImporterWorker(t, db).
		WithAPIBase(server.URL).
		WithGitStoragePath(gitRoot).
		WithCloneFn(func(_ context.Context, _, _, _ string) error {
			return nil
		})

	payload, err := json.Marshal(GitHubImportPayload{
		ImportJobID:    jobID.String(),
		OrganizationID: orgID.String(),
		SourceOwner:    "owner",
		SourceRepo:     "repo",
		Token:          "test-token",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(TypeGitHubImport, payload)
	if err := worker.HandleGitHubImport(context.Background(), task); err == nil {
		t.Fatal("expected error from failed issues phase")
	}

	jobRepo := repository.NewImportJobRepository(db)
	got, err := jobRepo.GetByID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != entity.ImportJobStatusFailed {
		t.Fatalf("status: got %q, want %q", got.Status, entity.ImportJobStatusFailed)
	}
	if got.Error == nil || *got.Error == "" {
		t.Fatal("expected error message on failed job")
	}
}

func TestRepoGitPath(t *testing.T) {
	worker := NewImporterWorker(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).
		WithGitStoragePath("/data/git")
	got := worker.repoGitPath("acme", "demo")
	want := filepath.Join("/data/git", "acme", "demo.git")
	if got != want {
		t.Fatalf("repoGitPath: got %q, want %q", got, want)
	}
}

func TestHandleGitHubImport_InvalidPayload(t *testing.T) {
	worker := NewImporterWorker(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	task := asynq.NewTask(TypeGitHubImport, []byte(`{"source_owner":"owner","source_repo":"repo"}`))
	err := worker.HandleGitHubImport(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for missing import job id")
	}
	if !strings.Contains(err.Error(), "import job id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMapUser_ReusesExistingMapping(t *testing.T) {
	db := newTestImporterDB(t)
	_, userID, jobID := seedImportFixtures(t, db)
	worker := newTestImporterWorker(t, db)

	first := worker.mapUser(context.Background(), jobID, "alice", "Alice")
	second := worker.mapUser(context.Background(), jobID, "alice", "Alice")
	if first != userID || second != userID {
		t.Fatalf("expected mapped local user %s, got first=%s second=%s", userID, first, second)
	}
}

func TestMockGitHubServerRoutes(t *testing.T) {
	server := newMockGitHubServer(t, false)
	t.Cleanup(server.Close)

	for _, path := range []string{
		"/repos/owner/repo",
		"/repos/owner/repo/issues",
		"/repos/owner/repo/issues/1/comments",
		"/repos/owner/repo/pulls",
	} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, resp.StatusCode)
		}
	}
}
