package repository_test

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func seedAccessTokenUser(t *testing.T, db *sqlx.DB) *entity.User {
	t.Helper()

	userRepo := repository.NewUserRepository(db)
	user := &entity.User{
		Login:        "token-user",
		Email:        "token-user@example.com",
		PasswordHash: "hashed",
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func TestAccessTokenRepository_CreateAndFindByTokenHash(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	repo := repository.NewAccessTokenRepository(db)

	token := &domain.AccessToken{
		UserID:    userIDFromUUID(user.ID),
		TokenHash: "hash-valid",
		Scopes:    []string{"repo"},
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token.ID == 0 {
		t.Fatal("expected generated token ID")
	}

	got, err := repo.FindByTokenHash(context.Background(), "hash-valid")
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if got == nil {
		t.Fatal("expected token, got nil")
	}
	if got.TokenHash != "hash-valid" || got.UserID != token.UserID {
		t.Fatalf("unexpected token: %+v", got)
	}
}

func TestAccessTokenRepository_FindByTokenHash_RevokedExcluded(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	repo := repository.NewAccessTokenRepository(db)

	now := time.Now().UTC()
	token := &domain.AccessToken{
		UserID:    userIDFromUUID(user.ID),
		TokenHash: "hash-revoked",
		Scopes:    []string{"repo"},
		RevokedAt: &now,
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByTokenHash(context.Background(), "hash-revoked")
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for revoked token, got %+v", got)
	}
}

func TestAccessTokenRepository_FindByTokenHash_ExpiredExcluded(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	repo := repository.NewAccessTokenRepository(db)

	expired := time.Now().UTC().Add(-time.Hour)
	token := &domain.AccessToken{
		UserID:    userIDFromUUID(user.ID),
		TokenHash: "hash-expired",
		Scopes:    []string{"repo"},
		ExpiresAt: &expired,
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByTokenHash(context.Background(), "hash-expired")
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for expired token, got %+v", got)
	}
}

func TestAccessTokenRepository_ListByUserIDAndRevoke(t *testing.T) {
	db := newUserTestDB(t)
	user := seedAccessTokenUser(t, db)
	repo := repository.NewAccessTokenRepository(db)
	userID := userIDFromUUID(user.ID)

	active := &domain.AccessToken{
		UserID:    userID,
		TokenHash: "hash-active",
		Scopes:    []string{"repo"},
	}
	if err := repo.Create(context.Background(), active); err != nil {
		t.Fatalf("Create active: %v", err)
	}

	revokedAt := time.Now().UTC()
	revoked := &domain.AccessToken{
		UserID:    userID,
		TokenHash: "hash-revoked-list",
		Scopes:    []string{"repo"},
		RevokedAt: &revokedAt,
	}
	if err := repo.Create(context.Background(), revoked); err != nil {
		t.Fatalf("Create revoked: %v", err)
	}

	tokens, err := repo.ListByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserID: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("token count = %d, want 1", len(tokens))
	}
	if tokens[0].TokenHash != "hash-active" {
		t.Fatalf("unexpected listed token: %+v", tokens[0])
	}

	if err := repo.Revoke(context.Background(), active.ID, userID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	tokens, err = repo.ListByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("ListByUserID after revoke: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("token count after revoke = %d, want 0", len(tokens))
	}
}

func userIDFromUUID(id uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(id[8:]))
}
