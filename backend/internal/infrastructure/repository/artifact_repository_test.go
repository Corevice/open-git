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
	"github.com/open-git/backend/internal/infrastructure/repository"
)

const (
	testArtifactOrgID      = "11111111-1111-1111-1111-111111111111"
	testArtifactOtherOrgID = "22222222-2222-2222-2222-222222222222"
	testArtifactRepoID     = "33333333-3333-3333-3333-333333333333"
	testArtifactRunID      = "44444444-4444-4444-4444-444444444444"
)

const migration011ArtifactStorageSchema = `
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
CREATE TABLE workflow_runs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    workflow TEXT NOT NULL,
    status TEXT NOT NULL,
    conclusion TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);
CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    workflow_run_id TEXT NOT NULL,
    name TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    size_in_bytes INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK(status IN ('pending','uploading','completed','failed','expired')),
    retention_days INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP
);
CREATE UNIQUE INDEX idx_artifacts_run_name ON artifacts(workflow_run_id, name) WHERE deleted_at IS NULL;
`

func newArtifactTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(migration011ArtifactStorageSchema); err != nil {
		_ = db.Close()
		t.Fatalf("apply artifact schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func seedArtifactTestFixtures(t *testing.T, db *sqlx.DB) {
	t.Helper()

	mustExecArtifactTest(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		testArtifactOrgID, "acme", "Acme", "free")
	mustExecArtifactTest(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		testArtifactOtherOrgID, "other", "Other", "free")
	mustExecArtifactTest(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		testArtifactRepoID, testArtifactOrgID, "widgets")
	mustExecArtifactTest(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		testArtifactRunID, testArtifactOrgID, testArtifactRepoID, "ci.yml", "completed")
}

func mustExecArtifactTest(t *testing.T, db *sqlx.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func parseArtifactTestUUID(t *testing.T, raw string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(raw)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", raw, err)
	}
	return id
}

func TestArtifactRepository_CreateAndGetByID(t *testing.T) {
	db := newArtifactTestDB(t)
	seedArtifactTestFixtures(t, db)
	repo := repository.NewArtifactRepository(db)
	ctx := context.Background()

	orgID := parseArtifactTestUUID(t, testArtifactOrgID)
	runID := parseArtifactTestUUID(t, testArtifactRunID)
	artifact := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "build-output",
		StorageKey:     "org/acme/repo/widgets/runs/1/build-output.zip",
		SizeBytes:      1024,
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}

	if err := repo.Create(ctx, artifact); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, artifact.ID, orgID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "build-output" {
		t.Fatalf("name = %q, want build-output", got.Name)
	}
	if got.SizeBytes != 1024 {
		t.Fatalf("size_in_bytes = %d, want 1024", got.SizeBytes)
	}
	if got.StorageKey != artifact.StorageKey {
		t.Fatalf("storage_key = %q, want %q", got.StorageKey, artifact.StorageKey)
	}
}

func TestArtifactRepository_GetByIDWrongOrgReturnsNotFound(t *testing.T) {
	db := newArtifactTestDB(t)
	seedArtifactTestFixtures(t, db)
	repo := repository.NewArtifactRepository(db)
	ctx := context.Background()

	orgID := parseArtifactTestUUID(t, testArtifactOrgID)
	otherOrgID := parseArtifactTestUUID(t, testArtifactOtherOrgID)
	runID := parseArtifactTestUUID(t, testArtifactRunID)
	artifact := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "build-output",
		StorageKey:     "org/acme/repo/widgets/runs/1/build-output.zip",
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}
	if err := repo.Create(ctx, artifact); err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err := repo.GetByID(ctx, artifact.ID, otherOrgID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByID with wrong org: got %v, want %v", err, domain.ErrNotFound)
	}
}

func TestArtifactRepository_CreateDuplicateReturnsConflict(t *testing.T) {
	db := newArtifactTestDB(t)
	seedArtifactTestFixtures(t, db)
	repo := repository.NewArtifactRepository(db)
	ctx := context.Background()

	orgID := parseArtifactTestUUID(t, testArtifactOrgID)
	runID := parseArtifactTestUUID(t, testArtifactRunID)
	first := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "build-output",
		StorageKey:     "org/acme/repo/widgets/runs/1/build-output.zip",
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}
	second := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "build-output",
		StorageKey:     "org/acme/repo/widgets/runs/1/build-output-2.zip",
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("Create first: %v", err)
	}
	if err := repo.Create(ctx, second); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("Create duplicate: got %v, want %v", err, domain.ErrConflict)
	}
}

func TestArtifactRepository_SoftDeleteHidesArtifact(t *testing.T) {
	db := newArtifactTestDB(t)
	seedArtifactTestFixtures(t, db)
	repo := repository.NewArtifactRepository(db)
	ctx := context.Background()

	orgID := parseArtifactTestUUID(t, testArtifactOrgID)
	runID := parseArtifactTestUUID(t, testArtifactRunID)
	artifact := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "build-output",
		StorageKey:     "org/acme/repo/widgets/runs/1/build-output.zip",
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}
	if err := repo.Create(ctx, artifact); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.SoftDelete(ctx, artifact.ID, orgID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err := repo.GetByID(ctx, artifact.ID, orgID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByID after SoftDelete: got %v, want %v", err, domain.ErrNotFound)
	}
}

func TestArtifactRepository_ListExpired(t *testing.T) {
	db := newArtifactTestDB(t)
	seedArtifactTestFixtures(t, db)
	repo := repository.NewArtifactRepository(db)
	ctx := context.Background()

	orgID := parseArtifactTestUUID(t, testArtifactOrgID)
	runID := parseArtifactTestUUID(t, testArtifactRunID)
	now := time.Now().UTC()

	expiredCompleted := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "expired-completed",
		StorageKey:     "org/acme/repo/widgets/runs/1/expired-completed.zip",
		ExpiresAt:      now.Add(-1 * time.Hour),
	}
	futureCompleted := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "future-completed",
		StorageKey:     "org/acme/repo/widgets/runs/1/future-completed.zip",
		ExpiresAt:      now.Add(24 * time.Hour),
	}
	expiredPending := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "expired-pending",
		StorageKey:     "org/acme/repo/widgets/runs/1/expired-pending.zip",
		ExpiresAt:      now.Add(-2 * time.Hour),
	}
	expiredDeleted := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "expired-deleted",
		StorageKey:     "org/acme/repo/widgets/runs/1/expired-deleted.zip",
		ExpiresAt:      now.Add(-3 * time.Hour),
	}

	for _, artifact := range []*entity.Artifact{
		expiredCompleted,
		futureCompleted,
		expiredPending,
		expiredDeleted,
	} {
		if err := repo.Create(ctx, artifact); err != nil {
			t.Fatalf("Create %q: %v", artifact.Name, err)
		}
	}

	if err := repo.UpdateStatus(ctx, expiredCompleted.ID, entity.ArtifactStatusCompleted, 100); err != nil {
		t.Fatalf("UpdateStatus expiredCompleted: %v", err)
	}
	if err := repo.UpdateStatus(ctx, futureCompleted.ID, entity.ArtifactStatusCompleted, 200); err != nil {
		t.Fatalf("UpdateStatus futureCompleted: %v", err)
	}
	if err := repo.UpdateStatus(ctx, expiredDeleted.ID, entity.ArtifactStatusCompleted, 300); err != nil {
		t.Fatalf("UpdateStatus expiredDeleted: %v", err)
	}
	if err := repo.SoftDelete(ctx, expiredDeleted.ID, orgID); err != nil {
		t.Fatalf("SoftDelete expiredDeleted: %v", err)
	}

	expired, err := repo.ListExpired(ctx, 100)
	if err != nil {
		t.Fatalf("ListExpired: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("ListExpired returned %d artifacts, want 1", len(expired))
	}
	if expired[0].ID != expiredCompleted.ID {
		t.Fatalf("ListExpired artifact ID = %v, want %v", expired[0].ID, expiredCompleted.ID)
	}
}
