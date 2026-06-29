package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func seedSystemSettingUser(t *testing.T, db *sqlx.DB, userID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`,
		userID.String(), "admin", "admin@example.com", "hash"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

func TestSystemSettingRepository_SetThenGet(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewSystemSettingRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	seedSystemSettingUser(t, db, userID)

	setting := &entity.SystemSetting{
		Key: "site.name",
		Value: map[string]any{
			"name": "Open Git",
		},
		UpdatedBy: userID,
		UpdatedAt: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	if err := repo.Set(ctx, setting); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := repo.Get(ctx, "site.name")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get: expected setting, got nil")
	}
	if got.Key != setting.Key {
		t.Fatalf("Key: got %q, want %q", got.Key, setting.Key)
	}
	if got.Value["name"] != "Open Git" {
		t.Fatalf("Value[name]: got %v, want %q", got.Value["name"], "Open Git")
	}
	if got.UpdatedBy != userID {
		t.Fatalf("UpdatedBy: got %v, want %v", got.UpdatedBy, userID)
	}
	if !got.UpdatedAt.Equal(setting.UpdatedAt) {
		t.Fatalf("UpdatedAt: got %v, want %v", got.UpdatedAt, setting.UpdatedAt)
	}

	setting.Value = map[string]any{"name": "Open Git Forge"}
	setting.UpdatedAt = time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC)
	if err := repo.Set(ctx, setting); err != nil {
		t.Fatalf("Set update: %v", err)
	}

	got, err = repo.Get(ctx, "site.name")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Value["name"] != "Open Git Forge" {
		t.Fatalf("Value[name] after update: got %v, want %q", got.Value["name"], "Open Git Forge")
	}
	if !got.UpdatedAt.Equal(setting.UpdatedAt) {
		t.Fatalf("UpdatedAt after update: got %v, want %v", got.UpdatedAt, setting.UpdatedAt)
	}
}

func TestSystemSettingRepository_GetMissingKey(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewSystemSettingRepository(db)
	ctx := context.Background()

	got, err := repo.Get(ctx, "missing.key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Fatalf("Get: expected nil, got %+v", got)
	}
}

func TestSystemSettingRepository_SetDoesNotMutateUpdatedAt(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewSystemSettingRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	seedSystemSettingUser(t, db, userID)

	originalUpdatedAt := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	setting := &entity.SystemSetting{
		Key:       "site.mutation",
		Value:     map[string]any{"enabled": true},
		UpdatedBy: userID,
		UpdatedAt: originalUpdatedAt,
	}

	if err := repo.Set(ctx, setting); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !setting.UpdatedAt.Equal(originalUpdatedAt) {
		t.Fatalf("Set mutated UpdatedAt: got %v, want %v", setting.UpdatedAt, originalUpdatedAt)
	}
}

func TestSystemSettingRepository_GetInvalidUpdatedBy(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewSystemSettingRepository(db)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO system_settings (key, value, updated_by, updated_at)
		VALUES (?, ?, ?, ?)`,
		"legacy.key", `{}`, "", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("insert setting: %v", err)
	}

	got, err := repo.Get(ctx, "legacy.key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get: expected setting, got nil")
	}
	if got.UpdatedBy != uuid.Nil {
		t.Fatalf("UpdatedBy: got %v, want uuid.Nil for empty updated_by", got.UpdatedBy)
	}
	if got.Value == nil {
		t.Fatal("Value: expected non-nil empty map")
	}
}
