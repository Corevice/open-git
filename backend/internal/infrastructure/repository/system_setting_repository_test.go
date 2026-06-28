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
