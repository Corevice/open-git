package worker

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain/entity"
	infrarepo "github.com/open-git/backend/internal/infrastructure/repository"
)

const artifactCleanupTestSchema = `
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

func newArtifactCleanupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(artifactCleanupTestSchema); err != nil {
		_ = db.Close()
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

type mockArtifactCleanupStorage struct {
	deletedKeys []string
}

func (m *mockArtifactCleanupStorage) PresignedPutURL(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}

func (m *mockArtifactCleanupStorage) PresignedGetURL(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}

func (m *mockArtifactCleanupStorage) DeleteObject(_ context.Context, _, key string) error {
	m.deletedKeys = append(m.deletedKeys, key)
	return nil
}

func TestArtifactCleanupWorker_HandleCleanup(t *testing.T) {
	db := newArtifactCleanupTestDB(t)
	ctx := context.Background()

	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	repoID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	runID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	mustExecArtifactCleanup(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID.String(), "acme", "Acme", "free")
	mustExecArtifactCleanup(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID.String(), orgID.String(), "widgets")
	mustExecArtifactCleanup(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID.String(), orgID.String(), repoID.String(), "ci.yml", "completed")

	now := time.Now().UTC()
	expiredKey := "org/acme/repo/widgets/runs/1/expired.zip"
	futureKey := "org/acme/repo/widgets/runs/1/future.zip"

	artifactRepo := infrarepo.NewArtifactRepository(db)
	expiredArtifact := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "expired-artifact",
		StorageKey:     expiredKey,
		ExpiresAt:      now.Add(-1 * time.Hour),
	}
	futureArtifact := &entity.Artifact{
		OrganizationID: orgID,
		RunID:          runID,
		Name:           "future-artifact",
		StorageKey:     futureKey,
		ExpiresAt:      now.Add(24 * time.Hour),
	}
	if err := artifactRepo.Create(ctx, expiredArtifact); err != nil {
		t.Fatalf("Create expired artifact: %v", err)
	}
	if err := artifactRepo.Create(ctx, futureArtifact); err != nil {
		t.Fatalf("Create future artifact: %v", err)
	}
	if err := artifactRepo.UpdateStatus(ctx, expiredArtifact.ID, entity.ArtifactStatusCompleted, 100); err != nil {
		t.Fatalf("UpdateStatus expired artifact: %v", err)
	}
	if err := artifactRepo.UpdateStatus(ctx, futureArtifact.ID, entity.ArtifactStatusCompleted, 200); err != nil {
		t.Fatalf("UpdateStatus future artifact: %v", err)
	}

	storage := &mockArtifactCleanupStorage{}
	worker := NewArtifactCleanupWorker(artifactRepo, storage, "artifacts")
	task := asynq.NewTask("artifact:cleanup", nil)
	if err := worker.HandleCleanup(ctx, task); err != nil {
		t.Fatalf("HandleCleanup returned error: %v", err)
	}

	var expiredDeletedAt sql.NullTime
	if err := db.QueryRowContext(ctx, `SELECT deleted_at FROM artifacts WHERE id = ?`, expiredArtifact.ID.String()).
		Scan(&expiredDeletedAt); err != nil {
		t.Fatalf("query expired artifact deleted_at: %v", err)
	}
	if !expiredDeletedAt.Valid {
		t.Fatal("expected expired artifact deleted_at to be set")
	}

	var futureDeletedAt sql.NullTime
	if err := db.QueryRowContext(ctx, `SELECT deleted_at FROM artifacts WHERE id = ?`, futureArtifact.ID.String()).
		Scan(&futureDeletedAt); err != nil {
		t.Fatalf("query future artifact deleted_at: %v", err)
	}
	if futureDeletedAt.Valid {
		t.Fatal("expected non-expired artifact deleted_at to remain NULL")
	}

	if len(storage.deletedKeys) != 1 || storage.deletedKeys[0] != expiredKey {
		t.Fatalf("DeleteObject calls = %v, want [%q]", storage.deletedKeys, expiredKey)
	}
}

func mustExecArtifactCleanup(t *testing.T, db *sqlx.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}
