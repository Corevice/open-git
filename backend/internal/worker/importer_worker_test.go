package worker

import (
	"context"
	"database/sql"
	"encoding/json"
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
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

const importTestSchema = `
CREATE TABLE organizations (
	id TEXT PRIMARY KEY,
	login TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	plan_tier TEXT NOT NULL DEFAULT 'free',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
	id TEXT PRIMARY KEY,
	login TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	name TEXT,
	bio TEXT,
	avatar_url TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP
);

CREATE TABLE repositories (
	id TEXT PRIMARY KEY,
	organization_id TEXT NOT NULL REFERENCES organizations(id),
	owner_id TEXT NOT NULL REFERENCES users(id),
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	git_path TEXT NOT NULL DEFAULT '',
	owner_login TEXT NOT NULL DEFAULT '',
	visibility TEXT NOT NULL DEFAULT 'private',
	default_branch TEXT NOT NULL DEFAULT 'main',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(owner_id, name)
);

CREATE TABLE issues (
	id TEXT PRIMARY KEY,
	organization_id TEXT NOT NULL REFERENCES organizations(id),
	repository_id TEXT NOT NULL REFERENCES repositories(id),
	number INTEGER NOT NULL,
	title TEXT NOT NULL,
	body TEXT NOT NULL DEFAULT '',
	state TEXT NOT NULL DEFAULT 'open',
	state_reason TEXT,
	author_id TEXT NOT NULL REFERENCES users(id),
	milestone_id TEXT,
	comments_count INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	closed_at TIMESTAMP,
	UNIQUE(repository_id, number)
);

CREATE TABLE labels (
	id TEXT PRIMARY KEY,
	organization_id TEXT NOT NULL REFERENCES organizations(id),
	repository_id TEXT NOT NULL REFERENCES repositories(id),
	name TEXT NOT NULL,
	color TEXT NOT NULL DEFAULT 'ededed',
	description TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(repository_id, name)
);

CREATE TABLE milestones (
	id TEXT PRIMARY KEY,
	organization_id TEXT NOT NULL REFERENCES organizations(id),
	repository_id TEXT NOT NULL REFERENCES repositories(id),
	number INTEGER NOT NULL DEFAULT 0,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	state TEXT NOT NULL DEFAULT 'open',
	due_on TIMESTAMP,
	open_issues INTEGER NOT NULL DEFAULT 0,
	closed_issues INTEGER NOT NULL DEFAULT 0,
	closed_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE comments (
	id TEXT PRIMARY KEY,
	organization_id TEXT NOT NULL REFERENCES organizations(id),
	issue_id TEXT NOT NULL REFERENCES issues(id),
	author_id TEXT NOT NULL REFERENCES users(id),
	body TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pull_requests (
	id TEXT PRIMARY KEY,
	organization_id TEXT NOT NULL REFERENCES organizations(id),
	repository_id TEXT NOT NULL REFERENCES repositories(id),
	number INTEGER NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	body TEXT NOT NULL DEFAULT '',
	draft INTEGER NOT NULL DEFAULT 0,
	head_ref TEXT NOT NULL,
	base_ref TEXT NOT NULL,
	head_sha TEXT NOT NULL DEFAULT '',
	base_sha TEXT NOT NULL DEFAULT '',
	state TEXT NOT NULL DEFAULT 'open',
	merged_at TIMESTAMP,
	merged_by TEXT REFERENCES users(id),
	merge_commit_sha TEXT,
	mergeable INTEGER,
	mergeable_state TEXT NOT NULL DEFAULT 'unknown',
	author_id TEXT NOT NULL REFERENCES users(id),
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(repository_id, number)
);

CREATE TABLE issue_labels (
	issue_id TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
	label_id TEXT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
	PRIMARY KEY (issue_id, label_id)
);

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
	if _, err := db.Exec(importTestSchema); err != nil {
		_ = db.Close()
		t.Fatalf("create import test schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func newTestImportRepos(db *sqlx.DB) (
	domainrepo.IImportJobRepository,
	domainrepo.IImportUserMappingRepository,
	domainrepo.IImportPhaseCheckpointRepository,
) {
	repo := repository.NewImportJobRepository(db)
	var (
		_ domainrepo.IImportJobRepository              = repo
		_ domainrepo.IImportUserMappingRepository    = repo
		_ domainrepo.IImportPhaseCheckpointRepository = repo
	)
	return repo, repo, repo
}

func newTestImporterWorker(t *testing.T, db *sqlx.DB) *ImporterWorker {
	t.Helper()

	jobs, mappings, checkpoints := newTestImportRepos(db)
	return NewImporterWorker(
		jobs,
		mappings,
		checkpoints,
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
				"name":            "repo",
				"description":     "Imported repo",
				"private":         false,
				"default_branch":  "main",
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
					"state":  "closed",
					"draft":  false,
					"merged": true,
					"merged_at": "2024-01-03T00:00:00Z",
					"merge_commit_sha": "deadbeef",
					"user": map[string]any{
						"login": "alice",
					},
					"merged_by": map[string]any{
						"login": "bob",
						"name":  "Bob",
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
		WithCloneFn(func(_ context.Context, _, _, _, _ string) error {
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

func TestHandleRateLimit_RetriesWhenRemainingZero(t *testing.T) {
	worker := NewImporterWorker(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set(rateLimitRemainingHeader, "0")
	resp.Header.Set(rateLimitResetHeader, "9999999999")

	retry, err := worker.handleRateLimit(context.Background(), resp, uuid.New())
	if err != nil {
		t.Fatalf("handleRateLimit returned error: %v", err)
	}
	if !retry {
		t.Fatal("expected retry when remaining is zero")
	}
}

func TestMapUser_CreatesGhostMapping(t *testing.T) {
	db := newTestImporterDB(t)
	_, userID, jobID := seedImportFixtures(t, db)
	worker := newTestImporterWorker(t, db)

	localID := worker.mapUser(context.Background(), jobID, "ghost-user", "Ghost User", userID)
	if localID != userID {
		t.Fatalf("expected fallback author for ghost user, got %s", localID)
	}

	userMappingRepo := repository.NewImportJobRepository(db)
	mapping, err := userMappingRepo.GetMappingByLogin(context.Background(), jobID, "ghost-user")
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
		WithCloneFn(func(_ context.Context, _, _, _, _ string) error {
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
	if strings.Contains(*got.Error, "boom") {
		t.Fatalf("expected sanitized error message, got %q", *got.Error)
	}
}

func TestRepoGitPath(t *testing.T) {
	worker := NewImporterWorker(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).
		WithGitStoragePath("/data/git")
	got, err := worker.repoGitPath("acme", "demo")
	if err != nil {
		t.Fatalf("repoGitPath: %v", err)
	}
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

	first := worker.mapUser(context.Background(), jobID, "alice", "Alice", userID)
	second := worker.mapUser(context.Background(), jobID, "alice", "Alice", userID)
	if first != userID || second != userID {
		t.Fatalf("expected mapped local user %s, got first=%s second=%s", userID, first, second)
	}
}

func TestResumePageFromCheckpoint(t *testing.T) {
	if got := resumePageFromCheckpoint(nil); got != 1 {
		t.Fatalf("nil checkpoint: got %d, want 1", got)
	}
	if got := resumePageFromCheckpoint(&entity.ImportPhaseCheckpoint{LastCursor: "2"}); got != 3 {
		t.Fatalf("completed page 2: got %d, want 3", got)
	}
}

func TestImportPullRequest_MapsMergedBy(t *testing.T) {
	db := newTestImporterDB(t)
	orgID, userID, jobID := seedImportFixtures(t, db)
	mustExecImporter(t, db, `INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`, uuid.New(), "bob", "bob@example.com", "hash")

	repoID := uuid.New()
	mustExecImporter(t, db, `INSERT INTO repositories (id, organization_id, owner_id, name, git_path, owner_login) VALUES (?, ?, ?, ?, ?, ?)`,
		repoID, orgID, userID, "repo", "/tmp/repo.git", "alice")

	worker := newTestImporterWorker(t, db)
	item := map[string]any{
		"number": 1,
		"title":  "Merged PR",
		"state":  "closed",
		"merged": true,
		"merged_at": "2024-01-03T00:00:00Z",
		"merge_commit_sha": "abc123",
		"user": map[string]any{"login": "alice"},
		"merged_by": map[string]any{"login": "bob", "name": "Bob"},
		"head": map[string]any{"ref": "feature", "sha": "abc"},
		"base": map[string]any{"ref": "main", "sha": "def"},
		"created_at": "2024-01-02T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z",
	}

	if err := worker.importPullRequest(context.Background(), jobID, orgID, repoID, "", userID, item); err != nil {
		t.Fatalf("importPullRequest: %v", err)
	}

	mapping, err := repository.NewImportJobRepository(db).GetMappingByLogin(context.Background(), jobID, "bob")
	if err != nil {
		t.Fatalf("GetMappingByLogin: %v", err)
	}
	if mapping == nil {
		t.Fatal("expected merged_by user mapping")
	}
}

func TestSanitizeLog_StripsControlCharacters(t *testing.T) {
	got := sanitizeLog("ok\ninjected\x1b[31mred\x1b[0m")
	if strings.Contains(got, "\n") || strings.Contains(got, "\x1b") {
		t.Fatalf("sanitizeLog did not strip control characters: %q", got)
	}
}
