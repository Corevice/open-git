package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func TestOAuthAccessTokenRepository_CreateAndFindByTokenHash(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "token-find-app")
	repo := repository.NewOAuthAccessTokenRepository(db)

	token := &domain.OAuthAccessToken{
		TokenHash:  "oauth-token-hash",
		OAuthAppID: app.ID,
		UserID:     userID,
		Scopes:     []string{"repo"},
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token.ID == "" {
		t.Fatal("expected generated token ID")
	}

	got, err := repo.FindByTokenHash(context.Background(), "oauth-token-hash")
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if got == nil || got.TokenHash != "oauth-token-hash" {
		t.Fatalf("unexpected token: %+v", got)
	}
}

func TestOAuthAccessTokenRepository_FindByTokenHash_Revoked(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "token-revoked-app")
	repo := repository.NewOAuthAccessTokenRepository(db)

	now := time.Now().UTC()
	token := &domain.OAuthAccessToken{
		TokenHash:  "oauth-token-revoked",
		OAuthAppID: app.ID,
		UserID:     userID,
		Scopes:     []string{"repo"},
		RevokedAt:  &now,
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByTokenHash(context.Background(), "oauth-token-revoked")
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for revoked token, got %+v", got)
	}
}

func TestOAuthAccessTokenRepository_RevokeByUserAndApp(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "token-revoke-app")
	repo := repository.NewOAuthAccessTokenRepository(db)

	token := &domain.OAuthAccessToken{
		TokenHash:  "oauth-token-revoke",
		OAuthAppID: app.ID,
		UserID:     userID,
		Scopes:     []string{"repo"},
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.RevokeByUserAndApp(context.Background(), userID, app.ID); err != nil {
		t.Fatalf("RevokeByUserAndApp: %v", err)
	}

	got, err := repo.FindByTokenHash(context.Background(), "oauth-token-revoke")
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil after revoke, got %+v", got)
	}
}

func TestOAuthAccessTokenRepository_RevokeAllByAppID(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "token-revoke-all-app")
	repo := repository.NewOAuthAccessTokenRepository(db)

	token := &domain.OAuthAccessToken{
		TokenHash:  "oauth-token-revoke-all",
		OAuthAppID: app.ID,
		UserID:     userID,
		Scopes:     []string{"repo"},
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.RevokeAllByAppID(context.Background(), app.ID, userID); err != nil {
		t.Fatalf("RevokeAllByAppID: %v", err)
	}

	got, err := repo.FindByTokenHash(context.Background(), "oauth-token-revoke-all")
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil after revoke all, got %+v", got)
	}
}
