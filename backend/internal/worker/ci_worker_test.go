package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/hibiken/asynq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const ciTestSchema = `
CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    login TEXT NOT NULL,
    name TEXT NOT NULL,
    plan_tier TEXT NOT NULL DEFAULT 'free',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE repositories (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL
);
CREATE TABLE action_secrets (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT,
    name TEXT NOT NULL,
    encrypted_value BYTEA NOT NULL,
    key_id TEXT NOT NULL DEFAULT '',
    visibility TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE workflow_runs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    workflow TEXT NOT NULL,
    status TEXT NOT NULL,
    conclusion TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

func newCITestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(ciTestSchema); err != nil {
		_ = db.Close()
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSecretsAreMasked(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-1"
	repoID := "repo-1"
	runID := "run-1"
	const secretValue = "supersecret-token-xyz"

	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO action_secrets (id, organization_id, repository_id, name, encrypted_value) VALUES (?, ?, ?, ?, ?)`,
		"sec-1", orgID, repoID, "API_TOKEN", secretValue)
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	yamlSrc := []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - name: leak
        run: echo $API_TOKEN
`)

	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, env []string, _ string) ([]byte, error) {
		for _, kv := range env {
			if strings.HasPrefix(kv, "API_TOKEN=") {
				return []byte(strings.TrimPrefix(kv, "API_TOKEN=") + "\n"), nil
			}
		}
		return []byte("no token\n"), nil
	})

	payload, err := json.Marshal(CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   yamlSrc,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(TypeCIRun, payload)
	if err := worker.HandleCIRun(ctx, task); err != nil {
		t.Fatalf("HandleCIRun returned error: %v", err)
	}

	var status, conclusion sql.NullString
	err = db.QueryRowContext(ctx, `SELECT status, conclusion FROM workflow_runs WHERE id = ?`, runID).
		Scan(&status, &conclusion)
	if err != nil {
		t.Fatalf("query workflow_run: %v", err)
	}
	if status.String != ciStatusCompleted {
		t.Errorf("status: got %q, want %q", status.String, ciStatusCompleted)
	}
	if strings.Contains(conclusion.String, secretValue) {
		t.Errorf("conclusion contains plaintext secret %q: %q", secretValue, conclusion.String)
	}
	if !strings.Contains(conclusion.String, logMask) {
		t.Errorf("conclusion missing mask token %q: %q", logMask, conclusion.String)
	}
}

func TestFreeTierConcurrentLimit(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-free"
	repoID := "repo-free"
	firstRun := "run-first"
	secondRun := "run-second"

	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierFree)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		firstRun, orgID, repoID, "ci.yml", ciStatusInProgress)
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		secondRun, orgID, repoID, "ci.yml", ciStatusQueued)

	yamlSrc := []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - name: build
        run: echo hello
`)

	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, _ []string, _ string) ([]byte, error) {
		return []byte("hello\n"), nil
	})

	payload, err := json.Marshal(CIRunPayload{
		WorkflowRunID:  secondRun,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   yamlSrc,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(TypeCIRun, payload)
	err = worker.HandleCIRun(ctx, task)
	if err == nil {
		t.Fatal("expected error for free-tier 2nd concurrent run, got nil")
	}
	if !errors.Is(err, ErrConcurrentLimitExceeded) {
		t.Errorf("expected ErrConcurrentLimitExceeded, got: %v", err)
	}

	var status, conclusion sql.NullString
	err = db.QueryRowContext(ctx, `SELECT status, conclusion FROM workflow_runs WHERE id = ?`, secondRun).
		Scan(&status, &conclusion)
	if err != nil {
		t.Fatalf("query second workflow_run: %v", err)
	}
	if status.String != ciStatusFailed {
		t.Errorf("second run status: got %q, want %q", status.String, ciStatusFailed)
	}
	if !strings.Contains(conclusion.String, ciConclusionRateLimited) {
		t.Errorf("expected conclusion to include %q, got %q", ciConclusionRateLimited, conclusion.String)
	}
}

func TestProTierAllowsManyConcurrent(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-pro"
	repoID := "repo-pro"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")

	for i := 0; i < 5; i++ {
		mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
			"running-"+strconv.Itoa(i), orgID, repoID, "ci.yml", ciStatusInProgress)
	}
	runID := "candidate"
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	yamlSrc := []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - name: build
        run: echo ok
`)
	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, _ []string, _ string) ([]byte, error) {
		return []byte("ok\n"), nil
	})

	payload, _ := json.Marshal(CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   yamlSrc,
	})
	task := asynq.NewTask(TypeCIRun, payload)
	if err := worker.HandleCIRun(ctx, task); err != nil {
		t.Fatalf("HandleCIRun unexpected error: %v", err)
	}
}

func mustExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

type ciFakeJobLogRepo struct {
	lines []*entity.JobLogLine
}

func (f *ciFakeJobLogRepo) AppendLines(_ context.Context, lines []*entity.JobLogLine) error {
	f.lines = append(f.lines, lines...)
	return nil
}

func (f *ciFakeJobLogRepo) ListLines(_ context.Context, _, _ string, _ int64, _ int) ([]*entity.JobLogLine, error) {
	return f.lines, nil
}

func (f *ciFakeJobLogRepo) CountLines(_ context.Context, _, _ string) (int64, error) {
	return int64(len(f.lines)), nil
}

func (f *ciFakeJobLogRepo) SetMeta(_ context.Context, _ *domainrepo.JobLogMeta) error {
	return nil
}

func (f *ciFakeJobLogRepo) GetMeta(_ context.Context, _, _ string) (*domainrepo.JobLogMeta, error) {
	return nil, nil
}

func TestHandleCIRun_AppendsJobLogLinesWhenLogRepoInjected(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-log"
	repoID := "repo-log"
	runID := "run-log"

	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	yamlSrc := []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - name: greet
        run: echo hello
      - name: world
        run: echo world
`)

	logRepo := &ciFakeJobLogRepo{}
	worker := NewCIWorker(db).
		WithLogRepository(logRepo).
		WithStreamingCommandRunner(func(_ context.Context, _ string, _ []string, _ string, _ int, sink func(stream, line string)) error {
			sink(entity.LogStreamStdout, "hello")
			return nil
		})

	payload, err := json.Marshal(CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   yamlSrc,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(TypeCIRun, payload)
	if err := worker.HandleCIRun(ctx, task); err != nil {
		t.Fatalf("HandleCIRun returned error: %v", err)
	}

	if len(logRepo.lines) < 2 {
		t.Fatalf("expected at least one log line per step, got %d", len(logRepo.lines))
	}
	for _, line := range logRepo.lines {
		if line.RunID != runID || line.RepositoryID != repoID || line.OrganizationID != orgID {
			t.Fatalf("unexpected log line scope: %+v", line)
		}
	}
}

