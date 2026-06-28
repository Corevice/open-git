package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&mode=memory")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE action_verifications (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL,
			trigger TEXT NOT NULL,
			status TEXT NOT NULL,
			requested_by TEXT,
			started_at TIMESTAMP,
			finished_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		_ = db.Close()
		t.Fatalf("create action_verifications table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE action_compatibility_results (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL,
			repository_id TEXT,
			action_name TEXT NOT NULL,
			action_version TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'untested',
			note TEXT,
			golden_diff TEXT,
			verified_at TIMESTAMP,
			verification_id TEXT NOT NULL REFERENCES action_verifications(id),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(organization_id, action_name, action_version)
		)
	`)
	if err != nil {
		_ = db.Close()
		t.Fatalf("create action_compatibility_results table: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX idx_action_verifications_organization_id ON action_verifications(organization_id)`)
	if err != nil {
		_ = db.Close()
		t.Fatalf("create action_verifications index: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX idx_action_compatibility_results_organization_id ON action_compatibility_results(organization_id)`)
	if err != nil {
		_ = db.Close()
		t.Fatalf("create action_compatibility_results index: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func insertVerification(t *testing.T, db *sqlx.DB, orgID uuid.UUID) uuid.UUID {
	t.Helper()

	verifRepo := repository.NewActionVerificationRepository(db)
	verification := &entity.ActionVerification{
		OrganizationID: orgID,
		Trigger:        entity.TriggerManual,
		Status:         entity.VerifStatusCompleted,
	}
	if err := verifRepo.Create(context.Background(), verification); err != nil {
		t.Fatalf("create verification: %v", err)
	}
	return verification.ID
}

func TestUpsertResult_CreatesThenUpdates(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewActionCompatibilityRepository(db)

	orgID := uuid.New()
	verificationID := insertVerification(t, db, orgID)

	first := &entity.ActionCompatibilityResult{
		OrganizationID: orgID,
		ActionName:     "actions/checkout",
		ActionVersion:  "v4",
		Status:         entity.StatusUntested,
		VerificationID: verificationID,
	}
	if err := repo.UpsertResult(context.Background(), first); err != nil {
		t.Fatalf("first UpsertResult: %v", err)
	}

	got, err := repo.GetResult(context.Background(), orgID, "actions/checkout", "v4")
	if err != nil {
		t.Fatalf("GetResult after create: %v", err)
	}
	if got == nil {
		t.Fatal("expected result after create, got nil")
	}
	if got.Status != entity.StatusUntested {
		t.Fatalf("status after create: got %q, want %q", got.Status, entity.StatusUntested)
	}

	second := &entity.ActionCompatibilityResult{
		OrganizationID: orgID,
		ActionName:     "actions/checkout",
		ActionVersion:  "v4",
		Status:         entity.StatusPass,
		VerificationID: verificationID,
	}
	if err := repo.UpsertResult(context.Background(), second); err != nil {
		t.Fatalf("second UpsertResult: %v", err)
	}

	got, err = repo.GetResult(context.Background(), orgID, "actions/checkout", "v4")
	if err != nil {
		t.Fatalf("GetResult after update: %v", err)
	}
	if got == nil {
		t.Fatal("expected result after update, got nil")
	}
	if got.Status != entity.StatusPass {
		t.Fatalf("status after update: got %q, want %q", got.Status, entity.StatusPass)
	}
}

func TestListResults_IsolatesByOrg(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewActionCompatibilityRepository(db)

	orgA := uuid.New()
	orgB := uuid.New()
	verifA := insertVerification(t, db, orgA)
	verifB := insertVerification(t, db, orgB)

	if err := repo.UpsertResult(context.Background(), &entity.ActionCompatibilityResult{
		OrganizationID: orgA,
		ActionName:     "actions/checkout",
		ActionVersion:  "v4",
		Status:         entity.StatusPass,
		VerificationID: verifA,
	}); err != nil {
		t.Fatalf("upsert org A result: %v", err)
	}

	if err := repo.UpsertResult(context.Background(), &entity.ActionCompatibilityResult{
		OrganizationID: orgB,
		ActionName:     "actions/setup-node",
		ActionVersion:  "v4",
		Status:         entity.StatusUntested,
		VerificationID: verifB,
	}); err != nil {
		t.Fatalf("upsert org B result: %v", err)
	}

	results, err := repo.ListResults(context.Background(), orgB, nil)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for org B, got %d", len(results))
	}
	if results[0].OrganizationID != orgB {
		t.Fatalf("organization_id: got %v, want %v", results[0].OrganizationID, orgB)
	}
	if results[0].ActionName == "actions/checkout" {
		t.Fatal("org A result leaked into org B listing")
	}
}

func TestGetResult_NilWhenNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewActionCompatibilityRepository(db)

	got, err := repo.GetResult(context.Background(), uuid.New(), "actions/checkout", "v4")
	if err != nil {
		t.Fatalf("GetResult: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result, got %+v", got)
	}
}
