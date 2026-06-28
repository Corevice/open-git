package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func newJobLogTestDB(t *testing.T) (*sql.DB, *sqlx.DB) {
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

func insertJobLogWorkflowRun(t *testing.T, db *sql.DB, runID, orgID, repoID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status, started_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, runID, orgID, repoID, "ci.yml", "in_progress", time.Now().UTC())
	if err != nil {
		t.Fatalf("insert workflow run: %v", err)
	}
}

func insertWorkflowJob(t *testing.T, db *sql.DB, job *entity.WorkflowJob) {
	t.Helper()
	repo := repository.NewWorkflowJobRepository(db)
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("insert workflow job: %v", err)
	}
}

func TestJobLogRepository_AppendLines(t *testing.T) {
	sqlDB, xdb := newJobLogTestDB(t)
	repo := repository.NewJobLogRepository(sqlDB)

	orgID := createTestOrganization(t, xdb, "log-append-org")
	userID := createTestUser(t, xdb, "log-append-user")
	repoID := createTestRepositoryRecord(t, xdb, orgID, userID, "log-append-repo")

	runID := uuid.NewString()
	insertJobLogWorkflowRun(t, sqlDB, runID, orgID.String(), repoID.String())

	jobID := uuid.NewString()
	insertWorkflowJob(t, sqlDB, &entity.WorkflowJob{
		ID:             jobID,
		WorkflowRunID:  runID,
		OrganizationID: orgID.String(),
		RepositoryID:   repoID.String(),
		Name:           "build",
		Status:         entity.WorkflowJobStatusInProgress,
		CreatedAt:      time.Now().UTC(),
	})

	now := time.Now().UTC()
	lines := []*entity.JobLogLine{
		{
			OrganizationID: orgID.String(),
			RepositoryID:   repoID.String(),
			RunID:          runID,
			JobID:          jobID,
			StepIndex:      0,
			LineNumber:     1,
			Stream:         entity.LogStreamStdout,
			Text:           "line one",
			CreatedAt:      now,
		},
		{
			OrganizationID: orgID.String(),
			RepositoryID:   repoID.String(),
			RunID:          runID,
			JobID:          jobID,
			StepIndex:      0,
			LineNumber:     2,
			Stream:         entity.LogStreamStderr,
			Text:           "line two",
			CreatedAt:      now,
		},
	}
	if err := repo.AppendLines(context.Background(), lines); err != nil {
		t.Fatalf("append lines: %v", err)
	}

	count, err := repo.CountLines(context.Background(), orgID.String(), jobID)
	if err != nil {
		t.Fatalf("count lines: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 lines, got %d", count)
	}
}

func TestJobLogRepository_AppendLines_Idempotent(t *testing.T) {
	sqlDB, xdb := newJobLogTestDB(t)
	repo := repository.NewJobLogRepository(sqlDB)

	orgID := createTestOrganization(t, xdb, "log-idem-org")
	userID := createTestUser(t, xdb, "log-idem-user")
	repoID := createTestRepositoryRecord(t, xdb, orgID, userID, "log-idem-repo")

	runID := uuid.NewString()
	insertJobLogWorkflowRun(t, sqlDB, runID, orgID.String(), repoID.String())

	jobID := uuid.NewString()
	insertWorkflowJob(t, sqlDB, &entity.WorkflowJob{
		ID:             jobID,
		WorkflowRunID:  runID,
		OrganizationID: orgID.String(),
		RepositoryID:   repoID.String(),
		Name:           "build",
		Status:         entity.WorkflowJobStatusInProgress,
		CreatedAt:      time.Now().UTC(),
	})

	line := &entity.JobLogLine{
		OrganizationID: orgID.String(),
		RepositoryID:   repoID.String(),
		RunID:          runID,
		JobID:          jobID,
		StepIndex:      0,
		LineNumber:     1,
		Stream:         entity.LogStreamStdout,
		Text:           "first",
		CreatedAt:      time.Now().UTC(),
	}
	if err := repo.AppendLines(context.Background(), []*entity.JobLogLine{line}); err != nil {
		t.Fatalf("first append: %v", err)
	}

	duplicate := *line
	duplicate.Text = "should be ignored"
	if err := repo.AppendLines(context.Background(), []*entity.JobLogLine{&duplicate}); err != nil {
		t.Fatalf("duplicate append: %v", err)
	}

	got, err := repo.ListLines(context.Background(), orgID.String(), jobID, 1, 10)
	if err != nil {
		t.Fatalf("list lines: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 line, got %d", len(got))
	}
	if got[0].Text != "first" {
		t.Fatalf("expected original text preserved, got %q", got[0].Text)
	}
}

func TestJobLogRepository_ListLines_FromLine(t *testing.T) {
	sqlDB, xdb := newJobLogTestDB(t)
	repo := repository.NewJobLogRepository(sqlDB)

	orgID := createTestOrganization(t, xdb, "log-fromline-org")
	userID := createTestUser(t, xdb, "log-fromline-user")
	repoID := createTestRepositoryRecord(t, xdb, orgID, userID, "log-fromline-repo")

	runID := uuid.NewString()
	insertJobLogWorkflowRun(t, sqlDB, runID, orgID.String(), repoID.String())

	jobID := uuid.NewString()
	insertWorkflowJob(t, sqlDB, &entity.WorkflowJob{
		ID:             jobID,
		WorkflowRunID:  runID,
		OrganizationID: orgID.String(),
		RepositoryID:   repoID.String(),
		Name:           "build",
		Status:         entity.WorkflowJobStatusInProgress,
		CreatedAt:      time.Now().UTC(),
	})

	now := time.Now().UTC()
	lines := make([]*entity.JobLogLine, 0, 10)
	for i := int64(1); i <= 10; i++ {
		lines = append(lines, &entity.JobLogLine{
			OrganizationID: orgID.String(),
			RepositoryID:   repoID.String(),
			RunID:          runID,
			JobID:          jobID,
			StepIndex:      0,
			LineNumber:     i,
			Stream:         entity.LogStreamStdout,
			Text:           "line",
			CreatedAt:      now,
		})
	}
	if err := repo.AppendLines(context.Background(), lines); err != nil {
		t.Fatalf("append lines: %v", err)
	}

	got, err := repo.ListLines(context.Background(), orgID.String(), jobID, 5, 100)
	if err != nil {
		t.Fatalf("list lines: %v", err)
	}
	if len(got) != 6 {
		t.Fatalf("expected 6 lines from line 5, got %d", len(got))
	}
	if got[0].LineNumber != 5 {
		t.Fatalf("expected first line number 5, got %d", got[0].LineNumber)
	}
}

func TestJobLogRepository_SetMetaGetMeta(t *testing.T) {
	sqlDB, xdb := newJobLogTestDB(t)
	repo := repository.NewJobLogRepository(sqlDB)

	orgID := createTestOrganization(t, xdb, "log-meta-org")
	userID := createTestUser(t, xdb, "log-meta-user")
	repoID := createTestRepositoryRecord(t, xdb, orgID, userID, "log-meta-repo")

	runID := uuid.NewString()
	insertJobLogWorkflowRun(t, sqlDB, runID, orgID.String(), repoID.String())

	jobID := uuid.NewString()
	insertWorkflowJob(t, sqlDB, &entity.WorkflowJob{
		ID:             jobID,
		WorkflowRunID:  runID,
		OrganizationID: orgID.String(),
		RepositoryID:   repoID.String(),
		Name:           "build",
		Status:         entity.WorkflowJobStatusCompleted,
		CreatedAt:      time.Now().UTC(),
	})

	meta := &domainrepo.JobLogMeta{
		JobID:          jobID,
		OrganizationID: orgID.String(),
		Status:         "success",
		TotalLines:     42,
	}
	if err := repo.SetMeta(context.Background(), meta); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	got, err := repo.GetMeta(context.Background(), orgID.String(), jobID)
	if err != nil {
		t.Fatalf("get meta: %v", err)
	}
	if got.Status != "success" {
		t.Fatalf("expected status success, got %q", got.Status)
	}
	if got.TotalLines != 42 {
		t.Fatalf("expected total_lines 42, got %d", got.TotalLines)
	}
}

func TestJobLogRepository_GetMeta_WrongOrg(t *testing.T) {
	sqlDB, xdb := newJobLogTestDB(t)
	repo := repository.NewJobLogRepository(sqlDB)

	orgA := createTestOrganization(t, xdb, "log-meta-org-a")
	orgB := createTestOrganization(t, xdb, "log-meta-org-b")
	userID := createTestUser(t, xdb, "log-meta-user")
	repoID := createTestRepositoryRecord(t, xdb, orgA, userID, "log-meta-repo")

	runID := uuid.NewString()
	insertJobLogWorkflowRun(t, sqlDB, runID, orgA.String(), repoID.String())

	jobID := uuid.NewString()
	insertWorkflowJob(t, sqlDB, &entity.WorkflowJob{
		ID:             jobID,
		WorkflowRunID:  runID,
		OrganizationID: orgA.String(),
		RepositoryID:   repoID.String(),
		Name:           "build",
		Status:         entity.WorkflowJobStatusCompleted,
		CreatedAt:      time.Now().UTC(),
	})

	if err := repo.SetMeta(context.Background(), &domainrepo.JobLogMeta{
		JobID:          jobID,
		OrganizationID: orgA.String(),
		Status:         "success",
		TotalLines:     10,
	}); err != nil {
		t.Fatalf("set meta: %v", err)
	}

	got, err := repo.GetMeta(context.Background(), orgB.String(), jobID)
	if got != nil {
		t.Fatalf("expected nil meta for wrong org, got %+v", got)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
