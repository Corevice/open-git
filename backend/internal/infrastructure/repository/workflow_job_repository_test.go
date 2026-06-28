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
	"github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func newWorkflowJobTestDB(t *testing.T) (*sql.DB, *sqlx.DB) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.RunMigrations(db, "sqlite", "../../../migrations"); err != nil {
		_ = db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, sqlx.NewDb(db, "sqlite3")
}

func insertWorkflowRun(t *testing.T, db *sql.DB, runID, orgID, repoID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status, started_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, runID, orgID, repoID, "ci.yml", "in_progress", time.Now().UTC())
	if err != nil {
		t.Fatalf("insert workflow run: %v", err)
	}
}

func TestWorkflowJobRepository_CreateGetByID(t *testing.T) {
	sqlDB, xdb := newWorkflowJobTestDB(t)
	repo := repository.NewWorkflowJobRepository(sqlDB)

	orgID := createTestOrganization(t, xdb, "wf-job-org")
	userID := createTestUser(t, xdb, "wf-job-user")
	repoID := createTestRepositoryRecord(t, xdb, orgID, userID, "wf-job-repo")

	runID := uuid.NewString()
	insertWorkflowRun(t, sqlDB, runID, orgID.String(), repoID.String())

	createdAt := time.Now().UTC().Truncate(time.Second)
	job := &entity.WorkflowJob{
		ID:             uuid.NewString(),
		WorkflowRunID:  runID,
		OrganizationID: orgID.String(),
		RepositoryID:   repoID.String(),
		Name:           "build",
		Status:         entity.WorkflowJobStatusQueued,
		Conclusion:     "",
		CreatedAt:      createdAt,
	}

	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	got, err := repo.GetByID(context.Background(), orgID.String(), job.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.ID != job.ID {
		t.Fatalf("expected id %q, got %q", job.ID, got.ID)
	}
	if got.WorkflowRunID != runID {
		t.Fatalf("expected run id %q, got %q", runID, got.WorkflowRunID)
	}
	if got.Name != "build" {
		t.Fatalf("expected name build, got %q", got.Name)
	}
	if got.Status != entity.WorkflowJobStatusQueued {
		t.Fatalf("expected status queued, got %q", got.Status)
	}
}

func TestWorkflowJobRepository_UpdateStatus(t *testing.T) {
	sqlDB, xdb := newWorkflowJobTestDB(t)
	repo := repository.NewWorkflowJobRepository(sqlDB)

	orgID := createTestOrganization(t, xdb, "wf-update-org")
	userID := createTestUser(t, xdb, "wf-update-user")
	repoID := createTestRepositoryRecord(t, xdb, orgID, userID, "wf-update-repo")

	runID := uuid.NewString()
	insertWorkflowRun(t, sqlDB, runID, orgID.String(), repoID.String())

	job := &entity.WorkflowJob{
		ID:             uuid.NewString(),
		WorkflowRunID:  runID,
		OrganizationID: orgID.String(),
		RepositoryID:   repoID.String(),
		Name:           "test",
		Status:         entity.WorkflowJobStatusInProgress,
		CreatedAt:      time.Now().UTC(),
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	completedAt := time.Now().UTC().Truncate(time.Second)
	if err := repo.UpdateStatus(context.Background(), job.ID, entity.WorkflowJobStatusCompleted, "success", &completedAt); err != nil {
		t.Fatalf("update status: %v", err)
	}

	got, err := repo.GetByID(context.Background(), orgID.String(), job.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Status != entity.WorkflowJobStatusCompleted {
		t.Fatalf("expected status completed, got %q", got.Status)
	}
	if got.Conclusion != "success" {
		t.Fatalf("expected conclusion success, got %q", got.Conclusion)
	}
	if got.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

func TestWorkflowJobRepository_ListByRunID_TenantIsolation(t *testing.T) {
	sqlDB, xdb := newWorkflowJobTestDB(t)
	repo := repository.NewWorkflowJobRepository(sqlDB)

	orgA := createTestOrganization(t, xdb, "wf-list-org-a")
	orgB := createTestOrganization(t, xdb, "wf-list-org-b")
	userID := createTestUser(t, xdb, "wf-list-user")
	repoA := createTestRepositoryRecord(t, xdb, orgA, userID, "wf-list-repo-a")
	repoB := createTestRepositoryRecord(t, xdb, orgB, userID, "wf-list-repo-b")

	runA := uuid.NewString()
	runB := uuid.NewString()
	insertWorkflowRun(t, sqlDB, runA, orgA.String(), repoA.String())
	insertWorkflowRun(t, sqlDB, runB, orgB.String(), repoB.String())

	jobA := &entity.WorkflowJob{
		ID:             uuid.NewString(),
		WorkflowRunID:  runA,
		OrganizationID: orgA.String(),
		RepositoryID:   repoA.String(),
		Name:           "job-a",
		Status:         entity.WorkflowJobStatusQueued,
		CreatedAt:      time.Now().UTC(),
	}
	jobB := &entity.WorkflowJob{
		ID:             uuid.NewString(),
		WorkflowRunID:  runB,
		OrganizationID: orgB.String(),
		RepositoryID:   repoB.String(),
		Name:           "job-b",
		Status:         entity.WorkflowJobStatusQueued,
		CreatedAt:      time.Now().UTC(),
	}
	if err := repo.Create(context.Background(), jobA); err != nil {
		t.Fatalf("create job a: %v", err)
	}
	if err := repo.Create(context.Background(), jobB); err != nil {
		t.Fatalf("create job b: %v", err)
	}

	jobs, err := repo.ListByRunID(context.Background(), orgA.String(), runA)
	if err != nil {
		t.Fatalf("list by run id: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job for org A, got %d", len(jobs))
	}
	if jobs[0].ID != jobA.ID {
		t.Fatalf("expected job %q, got %q", jobA.ID, jobs[0].ID)
	}

	jobsOtherOrg, err := repo.ListByRunID(context.Background(), orgB.String(), runA)
	if err != nil {
		t.Fatalf("list by run id with other org: %v", err)
	}
	if len(jobsOtherOrg) != 0 {
		t.Fatalf("expected empty list for mismatched org, got %d jobs", len(jobsOtherOrg))
	}
}
