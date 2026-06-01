package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/usecase/auth"
)

type mockTokenRepo struct {
	created []*domain.AccessToken
}

func (m *mockTokenRepo) Create(_ context.Context, token *domain.AccessToken) error {
	m.created = append(m.created, token)
	token.ID = int64(len(m.created))
	return nil
}

func (m *mockTokenRepo) ListByUserID(_ context.Context, _ int64) ([]*domain.AccessToken, error) {
	return m.created, nil
}

func (m *mockTokenRepo) Revoke(_ context.Context, _, _ int64) error {
	return nil
}

func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestIssuePATRawNotStored(t *testing.T) {
	repo := &mockTokenRepo{}
	uc := auth.NewIssuePATUsecase(repo)

	out, err := uc.Execute(context.Background(), auth.IssuePATInput{
		UserID: 1,
		Scopes: []string{"read"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Token == "" {
		t.Fatal("expected raw token to be returned")
	}
	if len(repo.created) != 1 {
		t.Fatal("expected token record to be created")
	}

	stored := repo.created[0].TokenHash
	if stored == out.Token {
		t.Fatal("raw token must not be stored")
	}
	if stored != sha256Hex(out.Token) {
		t.Fatalf("stored hash %q does not match SHA-256 of raw token", stored)
	}
}
