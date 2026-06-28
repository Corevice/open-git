package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

const migration011RunnersSchema = `
CREATE TABLE runners (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    labels TEXT NOT NULL DEFAULT '[]',
    os TEXT NOT NULL DEFAULT '',
    arch TEXT NOT NULL DEFAULT '',
    runner_type TEXT NOT NULL DEFAULT 'official',
    status TEXT NOT NULL DEFAULT 'offline',
    last_seen_at TIMESTAMP,
    ephemeral INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

func newRunnerTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(migration011RunnersSchema); err != nil {
		_ = db.Close()
		t.Fatalf("apply runners schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func TestRunnerRepository_CreateAndGetByID(t *testing.T) {
	db := newRunnerTestDB(t)
	repo := repository.NewRunnerRepository(db)

	orgID := uuid.New()
	runner := &entity.Runner{
		OrganizationID: orgID,
		Name:           "linux-runner",
		Labels:         []string{"self-hosted", "linux"},
		OS:             "linux",
		Arch:           "amd64",
		RunnerType:     "official",
		Status:         "online",
		LastSeenAt:     time.Now().UTC(),
		Ephemeral:      false,
	}
	if err := repo.Create(context.Background(), runner); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), runner.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != runner.Name {
		t.Fatalf("name = %q, want %q", got.Name, runner.Name)
	}
	if len(got.Labels) != 2 || got.Labels[0] != "self-hosted" || got.Labels[1] != "linux" {
		t.Fatalf("labels = %v, want [self-hosted linux]", got.Labels)
	}
}

func TestRunnerRepository_ListByOrgReturnsOnlyOwnOrgRunners(t *testing.T) {
	db := newRunnerTestDB(t)
	repo := repository.NewRunnerRepository(db)

	orgA := uuid.New()
	orgB := uuid.New()

	for _, name := range []string{"runner-a1", "runner-a2"} {
		if err := repo.Create(context.Background(), &entity.Runner{
			OrganizationID: orgA,
			Name:           name,
			Labels:         []string{"linux"},
			Status:         "online",
		}); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}
	if err := repo.Create(context.Background(), &entity.Runner{
		OrganizationID: orgB,
		Name:           "runner-b1",
		Labels:         []string{"linux"},
		Status:         "online",
	}); err != nil {
		t.Fatalf("Create runner-b1: %v", err)
	}

	got, err := repo.ListByOrg(context.Background(), orgA)
	if err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 runners for orgA, got %d", len(got))
	}
	for _, r := range got {
		if r.OrganizationID != orgA {
			t.Fatalf("runner %s belongs to org %s, want %s", r.Name, r.OrganizationID, orgA)
		}
	}
}

func TestRunnerRepository_UpdateStatusChangesStatusAndLastSeenAt(t *testing.T) {
	db := newRunnerTestDB(t)
	repo := repository.NewRunnerRepository(db)

	runner := &entity.Runner{
		OrganizationID: uuid.New(),
		Name:           "heartbeat-runner",
		Labels:         []string{"linux"},
		Status:         "offline",
	}
	if err := repo.Create(context.Background(), runner); err != nil {
		t.Fatalf("Create: %v", err)
	}

	lastSeen := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	if err := repo.UpdateStatus(context.Background(), runner.ID, "online", lastSeen); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := repo.GetByID(context.Background(), runner.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "online" {
		t.Fatalf("status = %q, want online", got.Status)
	}
	if !got.LastSeenAt.Equal(lastSeen) {
		t.Fatalf("last_seen_at = %v, want %v", got.LastSeenAt, lastSeen)
	}
}

func TestRunnerRepository_DeleteRemovesRunner(t *testing.T) {
	db := newRunnerTestDB(t)
	repo := repository.NewRunnerRepository(db)

	runner := &entity.Runner{
		OrganizationID: uuid.New(),
		Name:           "delete-me",
		Labels:         []string{"linux"},
		Status:         "offline",
	}
	if err := repo.Create(context.Background(), runner); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(context.Background(), runner.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.GetByID(context.Background(), runner.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
	if err != domain.ErrNotFound {
		t.Fatalf("GetByID after delete: %v, want ErrNotFound", err)
	}
}

func TestRunnerRepository_FindAvailableReturnsNilWhenNoLabelsMatch(t *testing.T) {
	db := newRunnerTestDB(t)
	repo := repository.NewRunnerRepository(db)

	orgID := uuid.New()
	if err := repo.Create(context.Background(), &entity.Runner{
		OrganizationID: orgID,
		Name:           "linux-only",
		Labels:         []string{"linux"},
		Status:         "online",
		Ephemeral:      false,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindAvailable(context.Background(), orgID, []string{"self-hosted", "linux"})
	if err != nil {
		t.Fatalf("FindAvailable: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil runner, got %+v", got)
	}
}

func TestRunnerRepository_FindAvailableReturnsRunnerWhenLabelsAreSuperset(t *testing.T) {
	db := newRunnerTestDB(t)
	repo := repository.NewRunnerRepository(db)

	orgID := uuid.New()
	if err := repo.Create(context.Background(), &entity.Runner{
		OrganizationID: orgID,
		Name:           "self-hosted-linux",
		Labels:         []string{"self-hosted", "linux"},
		Status:         "online",
		Ephemeral:      false,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindAvailable(context.Background(), orgID, []string{"linux"})
	if err != nil {
		t.Fatalf("FindAvailable: %v", err)
	}
	if got == nil {
		t.Fatal("expected runner, got nil")
	}
	if got.Name != "self-hosted-linux" {
		t.Fatalf("name = %q, want self-hosted-linux", got.Name)
	}
}
