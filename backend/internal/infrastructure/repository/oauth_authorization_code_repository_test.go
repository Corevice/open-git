package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func TestOAuthAuthorizationCodeRepository_Create(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "code-create-app")
	repo := repository.NewOAuthAuthorizationCodeRepository(db)

	code := &domain.OAuthAuthorizationCode{
		CodeHash:    "code-hash-create",
		OAuthAppID:  app.ID,
		UserID:      userID,
		RedirectURI: "https://example.com/callback",
		Scopes:      []string{"repo"},
		ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
	}
	if err := repo.Create(context.Background(), code); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if code.ID == "" {
		t.Fatal("expected generated code ID")
	}

	got, err := repo.ConsumeByCodeHash(context.Background(), "code-hash-create")
	if err != nil {
		t.Fatalf("ConsumeByCodeHash: %v", err)
	}
	if got == nil || got.CodeHash != "code-hash-create" {
		t.Fatalf("unexpected code: %+v", got)
	}
}

func TestOAuthAuthorizationCodeRepository_ConsumeByCodeHash_OneTimeUse(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "code-once-app")
	repo := repository.NewOAuthAuthorizationCodeRepository(db)

	code := &domain.OAuthAuthorizationCode{
		CodeHash:    "code-hash-once",
		OAuthAppID:  app.ID,
		UserID:      userID,
		RedirectURI: "https://example.com/callback",
		Scopes:      []string{"repo"},
		ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
	}
	if err := repo.Create(context.Background(), code); err != nil {
		t.Fatalf("Create: %v", err)
	}

	first, err := repo.ConsumeByCodeHash(context.Background(), "code-hash-once")
	if err != nil {
		t.Fatalf("first ConsumeByCodeHash: %v", err)
	}
	if first == nil {
		t.Fatal("expected code on first consume")
	}

	second, err := repo.ConsumeByCodeHash(context.Background(), "code-hash-once")
	if err != nil {
		t.Fatalf("second ConsumeByCodeHash: %v", err)
	}
	if second != nil {
		t.Fatalf("expected nil on second consume, got %+v", second)
	}
}

func TestOAuthAuthorizationCodeRepository_ConsumeByCodeHash_Expired(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	userID := userIDFromUUID(user.ID)
	app := seedOAuthApp(t, db, userID, "code-expired-app")
	repo := repository.NewOAuthAuthorizationCodeRepository(db)

	code := &domain.OAuthAuthorizationCode{
		CodeHash:    "code-hash-expired",
		OAuthAppID:  app.ID,
		UserID:      userID,
		RedirectURI: "https://example.com/callback",
		Scopes:      []string{"repo"},
		ExpiresAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.Create(context.Background(), code); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.ConsumeByCodeHash(context.Background(), "code-hash-expired")
	if err != nil {
		t.Fatalf("ConsumeByCodeHash: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for expired code, got %+v", got)
	}
}
