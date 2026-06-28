package repository_test

import (
	"context"
	"testing"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func TestOAuthAuthorizationRepository_Upsert_Create(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "auth-upsert-app")
	repo := repository.NewOAuthAuthorizationRepository(db)

	auth := &domain.OAuthAuthorization{
		OAuthAppID:    app.ID,
		UserID:        userID,
		GrantedScopes: []string{"repo"},
	}
	if err := repo.Upsert(context.Background(), auth); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if auth.ID == "" {
		t.Fatal("expected generated authorization ID")
	}

	got, err := repo.GetByUserAndApp(context.Background(), userID, app.ID)
	if err != nil {
		t.Fatalf("GetByUserAndApp: %v", err)
	}
	if got == nil || len(got.GrantedScopes) != 1 || got.GrantedScopes[0] != "repo" {
		t.Fatalf("unexpected authorization: %+v", got)
	}
}

func TestOAuthAuthorizationRepository_Upsert_Update(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "auth-update-app")
	repo := repository.NewOAuthAuthorizationRepository(db)

	auth := &domain.OAuthAuthorization{
		OAuthAppID:    app.ID,
		UserID:        userID,
		GrantedScopes: []string{"repo"},
	}
	if err := repo.Upsert(context.Background(), auth); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	auth.GrantedScopes = []string{"repo", "read:user"}
	if err := repo.Upsert(context.Background(), auth); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	got, err := repo.GetByUserAndApp(context.Background(), userID, app.ID)
	if err != nil {
		t.Fatalf("GetByUserAndApp: %v", err)
	}
	if got == nil || len(got.GrantedScopes) != 2 {
		t.Fatalf("unexpected scopes after update: %+v", got)
	}
}

func TestOAuthAuthorizationRepository_GetByUserAndApp(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "auth-get-app")
	repo := repository.NewOAuthAuthorizationRepository(db)

	auth := &domain.OAuthAuthorization{
		OAuthAppID:    app.ID,
		UserID:        userID,
		GrantedScopes: []string{"repo"},
	}
	if err := repo.Upsert(context.Background(), auth); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetByUserAndApp(context.Background(), userID, app.ID)
	if err != nil {
		t.Fatalf("GetByUserAndApp: %v", err)
	}
	if got == nil || got.OAuthAppID != app.ID {
		t.Fatalf("unexpected authorization: %+v", got)
	}
}

func TestOAuthAuthorizationRepository_ListByUser(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	repo := repository.NewOAuthAuthorizationRepository(db)

	for _, clientID := range []string{"auth-list-one", "auth-list-two"} {
		app := seedOAuthApp(t, db, userID, clientID)
		auth := &domain.OAuthAuthorization{
			OAuthAppID:    app.ID,
			UserID:        userID,
			GrantedScopes: []string{"repo"},
		}
		if err := repo.Upsert(context.Background(), auth); err != nil {
			t.Fatalf("Upsert %s: %v", clientID, err)
		}
	}

	auths, err := repo.ListByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(auths) != 2 {
		t.Fatalf("authorization count = %d, want 2", len(auths))
	}
}

func TestOAuthAuthorizationRepository_Delete(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "auth-delete-app")
	repo := repository.NewOAuthAuthorizationRepository(db)

	auth := &domain.OAuthAuthorization{
		OAuthAppID:    app.ID,
		UserID:        userID,
		GrantedScopes: []string{"repo"},
	}
	if err := repo.Upsert(context.Background(), auth); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := repo.Delete(context.Background(), userID, app.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.GetByUserAndApp(context.Background(), userID, app.ID)
	if err != nil {
		t.Fatalf("GetByUserAndApp: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil after delete, got %+v", got)
	}
}
