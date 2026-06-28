package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

const migration011WorkflowJobsSchema = `
CREATE TABLE workflow_jobs (
    id TEXT PRIMARY KEY,
    workflow_run_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','in_progress','completed','failed','cancelled')),
    conclusion TEXT NOT NULL DEFAULT '',
    assigned_runner_id TEXT,
    runs_on TEXT NOT NULL DEFAULT '[]',
    acquire_lock_version INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    timeout_minutes INTEGER NOT NULL DEFAULT 360,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

func newWorkflowJobTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(migration011WorkflowJobsSchema); err != nil {
		_ = db.Close()
		t.Fatalf("apply workflow_jobs schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func TestWorkflowJobRepository_Create(t *testing.T) {
	db := newWorkflowJobTestDB(t)
	repo := repository.NewWorkflowJobRepository(db)

	job := &entity.WorkflowJob{
		RunID:              uuid.New(),
		OrganizationID:     uuid.New(),
		RepositoryID:       uuid.New(),
		Name:               "build",
		Status:             entity.WorkflowJobStatusQueued,
		RunsOn:             []string{"ubuntu-latest"},
		AcquireLockVersion: 0,
		TimeoutMinutes:     360,
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "build" {
		t.Fatalf("name = %q, want build", got.Name)
	}
	if len(got.RunsOn) != 1 || got.RunsOn[0] != "ubuntu-latest" {
		t.Fatalf("runs_on = %v, want [ubuntu-latest]", got.RunsOn)
	}
}

func TestWorkflowJobRepository_AcquireForRunnerSuccess(t *testing.T) {
	db := newWorkflowJobTestDB(t)
	repo := repository.NewWorkflowJobRepository(db)

	jobID := uuid.New()
	runnerID := uuid.New()
	job := &entity.WorkflowJob{
		ID:                 jobID,
		RunID:              uuid.New(),
		OrganizationID:     uuid.New(),
		RepositoryID:       uuid.New(),
		Name:               "test",
		Status:             entity.WorkflowJobStatusQueued,
		AcquireLockVersion: 0,
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	ok, err := repo.AcquireForRunner(context.Background(), jobID, runnerID, 0)
	if err != nil {
		t.Fatalf("AcquireForRunner: %v", err)
	}
	if !ok {
		t.Fatal("expected AcquireForRunner to return true")
	}

	got, err := repo.GetByID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != entity.WorkflowJobStatusInProgress {
		t.Fatalf("status = %q, want in_progress", got.Status)
	}
	if got.AssignedRunnerID == nil || *got.AssignedRunnerID != runnerID {
		t.Fatalf("assigned_runner_id = %v, want %s", got.AssignedRunnerID, runnerID)
	}
	if got.AcquireLockVersion != 1 {
		t.Fatalf("acquire_lock_version = %d, want 1", got.AcquireLockVersion)
	}
	if got.StartedAt == nil {
		t.Fatal("expected started_at to be set")
	}
}

func TestWorkflowJobRepository_AcquireForRunnerLockVersionConflict(t *testing.T) {
	db := newWorkflowJobTestDB(t)
	repo := repository.NewWorkflowJobRepository(db)

	jobID := uuid.New()
	runnerID := uuid.New()
	job := &entity.WorkflowJob{
		ID:                 jobID,
		RunID:              uuid.New(),
		OrganizationID:     uuid.New(),
		RepositoryID:       uuid.New(),
		Name:               "test",
		Status:             entity.WorkflowJobStatusQueued,
		AcquireLockVersion: 0,
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	ok, err := repo.AcquireForRunner(context.Background(), jobID, runnerID, 0)
	if err != nil {
		t.Fatalf("first AcquireForRunner: %v", err)
	}
	if !ok {
		t.Fatal("expected first AcquireForRunner to return true")
	}

	ok, err = repo.AcquireForRunner(context.Background(), jobID, uuid.New(), 0)
	if err != nil {
		t.Fatalf("second AcquireForRunner: %v", err)
	}
	if ok {
		t.Fatal("expected second AcquireForRunner with same lockVersion to return false")
	}
}

func TestWorkflowJobRepository_CancelSetsStatusCancelled(t *testing.T) {
	db := newWorkflowJobTestDB(t)
	repo := repository.NewWorkflowJobRepository(db)

	jobID := uuid.New()
	job := &entity.WorkflowJob{
		ID:             jobID,
		RunID:          uuid.New(),
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Name:           "cancel-me",
		Status:         entity.WorkflowJobStatusQueued,
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Cancel(context.Background(), jobID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	got, err := repo.GetByID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "cancelled" {
		t.Fatalf("status = %q, want cancelled", got.Status)
	}
}

func TestWorkflowJobRepository_CompleteSetsStatusConclusionFinishedAt(t *testing.T) {
	db := newWorkflowJobTestDB(t)
	repo := repository.NewWorkflowJobRepository(db)

	jobID := uuid.New()
	job := &entity.WorkflowJob{
		ID:             jobID,
		RunID:          uuid.New(),
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Name:           "complete-me",
		Status:         entity.WorkflowJobStatusInProgress,
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	finishedAt := time.Date(2026, 6, 28, 15, 30, 0, 0, time.UTC)
	if err := repo.Complete(context.Background(), jobID, "success", finishedAt); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	got, err := repo.GetByID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != entity.WorkflowJobStatusCompleted {
		t.Fatalf("status = %q, want completed", got.Status)
	}
	if got.Conclusion != "success" {
		t.Fatalf("conclusion = %q, want success", got.Conclusion)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finishedAt) {
		t.Fatalf("finished_at = %v, want %v", got.FinishedAt, finishedAt)
	}
}

func TestWorkflowJobRepository_ListQueuedReturnsOnlyQueuedJobsForOrg(t *testing.T) {
	db := newWorkflowJobTestDB(t)
	repo := repository.NewWorkflowJobRepository(db)

	orgA := uuid.New()
	orgB := uuid.New()
	runID := uuid.New()
	repoID := uuid.New()

	jobs := []*entity.WorkflowJob{
		{
			RunID:          runID,
			OrganizationID: orgA,
			RepositoryID:   repoID,
			Name:           "queued-a1",
			Status:         entity.WorkflowJobStatusQueued,
		},
		{
			RunID:          runID,
			OrganizationID: orgA,
			RepositoryID:   repoID,
			Name:           "queued-a2",
			Status:         entity.WorkflowJobStatusQueued,
		},
		{
			RunID:          runID,
			OrganizationID: orgA,
			RepositoryID:   repoID,
			Name:           "in-progress-a",
			Status:         entity.WorkflowJobStatusInProgress,
		},
		{
			RunID:          runID,
			OrganizationID: orgB,
			RepositoryID:   repoID,
			Name:           "queued-b",
			Status:         entity.WorkflowJobStatusQueued,
		},
	}
	for _, job := range jobs {
		if err := repo.Create(context.Background(), job); err != nil {
			t.Fatalf("Create %s: %v", job.Name, err)
		}
	}

	got, err := repo.ListQueued(context.Background(), orgA)
	if err != nil {
		t.Fatalf("ListQueued: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 queued jobs for orgA, got %d", len(got))
	}
	for _, job := range got {
		if job.OrganizationID != orgA {
			t.Fatalf("job %s belongs to org %s, want %s", job.Name, job.OrganizationID, orgA)
		}
		if job.Status != entity.WorkflowJobStatusQueued {
			t.Fatalf("job %s status = %q, want queued", job.Name, job.Status)
		}
	}
}
