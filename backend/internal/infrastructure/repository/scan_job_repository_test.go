package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func seedScanJobFixtures(t *testing.T, db *sqlx.DB) (orgID, repoID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	orgID = uuid.New()
	ownerID := uuid.New()
	repoID = uuid.New()

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec %q: %v", query, err)
		}
	}

	exec(`INSERT INTO organizations (id, login, name) VALUES (?, ?, ?)`, orgID, "scan-org", "Scan Org")
	exec(`INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`, ownerID, "scanner", "scanner@example.com", "hash")
	exec(`INSERT INTO repositories (id, organization_id, owner_id, name) VALUES (?, ?, ?, ?)`, repoID, orgID, ownerID, "scan-target")

	return orgID, repoID
}

func TestScanJobRepository_CreateGetByID(t *testing.T) {
	db := openTestDB(t)
	orgID, repoID := seedScanJobFixtures(t, db)
	repo := repository.NewScanJobRepository(db)
	ctx := context.Background()

	job := &entity.ScanJob{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Type:           entity.ScanJobTypeDependency,
		Status:         entity.ScanJobStatusQueued,
	}

	if err := repo.Create(ctx, job); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if job.ID == uuid.Nil {
		t.Fatal("expected job ID to be assigned")
	}

	got, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected job, got nil")
	}
	if got.Type != entity.ScanJobTypeDependency || got.Status != entity.ScanJobStatusQueued {
		t.Fatalf("unexpected job: %+v", got)
	}
	if got.OrganizationID != orgID || got.RepositoryID != repoID {
		t.Fatalf("unexpected org/repo IDs: %+v", got)
	}
}

func TestScanJobRepository_UpdateStatusTransitions(t *testing.T) {
	db := openTestDB(t)
	orgID, repoID := seedScanJobFixtures(t, db)
	repo := repository.NewScanJobRepository(db)
	ctx := context.Background()

	job := &entity.ScanJob{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Type:           entity.ScanJobTypeDependency,
		Status:         entity.ScanJobStatusQueued,
	}
	if err := repo.Create(ctx, job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateStatus(ctx, job.ID, entity.ScanJobStatusRunning, ""); err != nil {
		t.Fatalf("UpdateStatus running: %v", err)
	}

	running, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID after running: %v", err)
	}
	if running.Status != entity.ScanJobStatusRunning {
		t.Fatalf("expected running status, got %s", running.Status)
	}
	if running.StartedAt == nil {
		t.Fatal("expected started_at to be set when running")
	}
	if running.FinishedAt != nil {
		t.Fatal("expected finished_at to remain unset while running")
	}

	if err := repo.UpdateStatus(ctx, job.ID, entity.ScanJobStatusCompleted, ""); err != nil {
		t.Fatalf("UpdateStatus completed: %v", err)
	}

	completed, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID after completed: %v", err)
	}
	if completed.Status != entity.ScanJobStatusCompleted {
		t.Fatalf("expected completed status, got %s", completed.Status)
	}
	if completed.StartedAt == nil {
		t.Fatal("expected started_at to remain set after completion")
	}
	if completed.FinishedAt == nil {
		t.Fatal("expected finished_at to be set when completed")
	}
}
