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

const migration011RunnerRegistrationTokensSchema = `
CREATE TABLE runner_registration_tokens (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP
);
`

func newRunnerRegistrationTokenTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(migration011RunnerRegistrationTokensSchema); err != nil {
		_ = db.Close()
		t.Fatalf("apply runner_registration_tokens schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func TestRunnerRegistrationTokenRepository_CreateAndGetByTokenHash(t *testing.T) {
	db := newRunnerRegistrationTokenTestDB(t)
	repo := repository.NewRunnerRegistrationTokenRepository(db)

	token := &entity.RunnerRegistrationToken{
		OrganizationID: uuid.New(),
		TokenHash:      "hash-create-get",
		ExpiresAt:      time.Now().UTC().Add(time.Hour),
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByTokenHash(context.Background(), token.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if got.ID != token.ID {
		t.Fatalf("id = %s, want %s", got.ID, token.ID)
	}
	if got.UsedAt != nil {
		t.Fatalf("used_at = %v, want nil", got.UsedAt)
	}
}

func TestRunnerRegistrationTokenRepository_GetByTokenHashRejectsExpiredToken(t *testing.T) {
	db := newRunnerRegistrationTokenTestDB(t)
	repo := repository.NewRunnerRegistrationTokenRepository(db)

	token := &entity.RunnerRegistrationToken{
		OrganizationID: uuid.New(),
		TokenHash:      "hash-expired",
		ExpiresAt:      time.Now().UTC().Add(-time.Hour),
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err := repo.GetByTokenHash(context.Background(), token.TokenHash)
	if err != domain.ErrNotFound {
		t.Fatalf("GetByTokenHash expired token: %v, want ErrNotFound", err)
	}
}

func TestRunnerRegistrationTokenRepository_MarkUsedSetsUsedAt(t *testing.T) {
	db := newRunnerRegistrationTokenTestDB(t)
	repo := repository.NewRunnerRegistrationTokenRepository(db)

	token := &entity.RunnerRegistrationToken{
		OrganizationID: uuid.New(),
		TokenHash:      "hash-mark-used",
		ExpiresAt:      time.Now().UTC().Add(time.Hour),
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	usedAt := time.Now().UTC()
	if err := repo.MarkUsed(context.Background(), token.ID, usedAt); err != nil {
		t.Fatalf("MarkUsed: %v", err)
	}

	_, err := repo.GetByTokenHash(context.Background(), token.TokenHash)
	if err != domain.ErrNotFound {
		t.Fatalf("GetByTokenHash after MarkUsed: %v, want ErrNotFound", err)
	}
}

func TestRunnerRegistrationTokenRepository_MarkUsedReturnsNotFoundForMissingToken(t *testing.T) {
	db := newRunnerRegistrationTokenTestDB(t)
	repo := repository.NewRunnerRegistrationTokenRepository(db)

	err := repo.MarkUsed(context.Background(), uuid.New(), time.Now().UTC())
	if err != domain.ErrNotFound {
		t.Fatalf("MarkUsed missing token: %v, want ErrNotFound", err)
	}
}

func TestRunnerRegistrationTokenRepository_MarkUsedReturnsNotFoundWhenAlreadyUsed(t *testing.T) {
	db := newRunnerRegistrationTokenTestDB(t)
	repo := repository.NewRunnerRegistrationTokenRepository(db)

	token := &entity.RunnerRegistrationToken{
		OrganizationID: uuid.New(),
		TokenHash:      "hash-already-used",
		ExpiresAt:      time.Now().UTC().Add(time.Hour),
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.MarkUsed(context.Background(), token.ID, time.Now().UTC()); err != nil {
		t.Fatalf("first MarkUsed: %v", err)
	}

	err := repo.MarkUsed(context.Background(), token.ID, time.Now().UTC())
	if err != domain.ErrNotFound {
		t.Fatalf("second MarkUsed: %v, want ErrNotFound", err)
	}
}
