package repository_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func TestOAuthAppRepository_Create(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	repo := repository.NewOAuthAppRepository(db)

	app := &domain.OAuthApp{
		ClientID:         "client-create",
		ClientSecretHash: "secret-hash",
		RedirectURIs:     []string{"https://example.com/callback"},
		Name:             "Test App",
		HomepageURL:      "https://example.com",
		OwnerType:        "user",
		OwnerUserID:      userID,
	}
	if err := repo.Create(context.Background(), app); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if app.ID == "" {
		t.Fatal("expected generated app ID")
	}

	got, err := repo.GetByID(context.Background(), app.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected app, got nil")
	}
	if got.ClientID != "client-create" || got.Name != "Test App" {
		t.Fatalf("unexpected app: %+v", got)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be populated")
	}
}

func TestOAuthAppRepository_GetByClientID(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	repo := repository.NewOAuthAppRepository(db)

	app := &domain.OAuthApp{
		ClientID:         "client-lookup",
		ClientSecretHash: "secret-hash",
		RedirectURIs:     []string{"https://example.com/callback"},
		Name:             "Lookup App",
		OwnerType:        "user",
		OwnerUserID:      userID,
	}
	if err := repo.Create(context.Background(), app); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByClientID(context.Background(), "client-lookup")
	if err != nil {
		t.Fatalf("GetByClientID: %v", err)
	}
	if got == nil || got.ClientID != "client-lookup" {
		t.Fatalf("unexpected app: %+v", got)
	}
}

func TestOAuthAppRepository_GetByClientID_NotFound(t *testing.T) {
	db := newUserTestDB(t)
	repo := repository.NewOAuthAppRepository(db)

	got, err := repo.GetByClientID(context.Background(), "unknown-client")
	if err != nil {
		t.Fatalf("GetByClientID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestOAuthAppRepository_ListByOwnerUser(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	repo := repository.NewOAuthAppRepository(db)

	for _, clientID := range []string{"app-one", "app-two"} {
		app := &domain.OAuthApp{
			ClientID:         clientID,
			ClientSecretHash: "secret-hash",
			RedirectURIs:     []string{"https://example.com/callback"},
			Name:             clientID,
			OwnerType:        "user",
			OwnerUserID:      userID,
		}
		if err := repo.Create(context.Background(), app); err != nil {
			t.Fatalf("Create %s: %v", clientID, err)
		}
	}

	apps, err := repo.ListByOwnerUser(context.Background(), userID, 1, 10)
	if err != nil {
		t.Fatalf("ListByOwnerUser: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("app count = %d, want 2", len(apps))
	}
}

func TestOAuthAppRepository_Update(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	repo := repository.NewOAuthAppRepository(db)

	app := &domain.OAuthApp{
		ClientID:         "client-update",
		ClientSecretHash: "secret-hash",
		RedirectURIs:     []string{"https://example.com/callback"},
		Name:             "Old Name",
		HomepageURL:      "https://old.example.com",
		OwnerType:        "user",
		OwnerUserID:      userID,
	}
	if err := repo.Create(context.Background(), app); err != nil {
		t.Fatalf("Create: %v", err)
	}

	app.Name = "New Name"
	app.HomepageURL = "https://new.example.com"
	if err := repo.Update(context.Background(), app); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(context.Background(), app.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "New Name" || got.HomepageURL != "https://new.example.com" {
		t.Fatalf("unexpected app after update: %+v", got)
	}
}

func TestOAuthAppRepository_Delete(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	repo := repository.NewOAuthAppRepository(db)

	app := &domain.OAuthApp{
		ClientID:         "client-delete",
		ClientSecretHash: "secret-hash",
		RedirectURIs:     []string{"https://example.com/callback"},
		Name:             "Delete App",
		OwnerType:        "user",
		OwnerUserID:      userID,
	}
	if err := repo.Create(context.Background(), app); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(context.Background(), app.ID, userID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByID(context.Background(), app.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil after delete, got %+v", got)
	}
}

func TestOAuthAppRepository_UpdateSecretHash(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	repo := repository.NewOAuthAppRepository(db)

	app := &domain.OAuthApp{
		ClientID:         "client-secret",
		ClientSecretHash: "old-hash",
		RedirectURIs:     []string{"https://example.com/callback"},
		Name:             "Secret App",
		OwnerType:        "user",
		OwnerUserID:      userID,
	}
	if err := repo.Create(context.Background(), app); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateSecretHash(context.Background(), app.ID, "new-hash"); err != nil {
		t.Fatalf("UpdateSecretHash: %v", err)
	}

	got, err := repo.GetByID(context.Background(), app.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ClientSecretHash != "new-hash" {
		t.Fatalf("unexpected secret hash: %q", got.ClientSecretHash)
	}
}

func seedOAuthApp(t *testing.T, db *sqlx.DB, userID int64, clientID string) *domain.OAuthApp {
	t.Helper()

	repo := repository.NewOAuthAppRepository(db)
	app := &domain.OAuthApp{
		ClientID:         clientID,
		ClientSecretHash: "secret-hash",
		RedirectURIs:     []string{"https://example.com/callback"},
		Name:             clientID,
		OwnerType:        "user",
		OwnerUserID:      userID,
	}
	if err := repo.Create(context.Background(), app); err != nil {
		t.Fatalf("create oauth app: %v", err)
	}
	return app
}
