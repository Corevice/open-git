package auth_test

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/usecase/auth"
)

type loginMockUserRepo struct {
	users map[string]*domain.User
}

func (m *loginMockUserRepo) Create(_ context.Context, _ *domain.User) error {
	return nil
}

func (m *loginMockUserRepo) GetByLogin(_ context.Context, login string) (*domain.User, error) {
	if u, ok := m.users[login]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func (m *loginMockUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func TestLoginWrongPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), 12)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repo := &loginMockUserRepo{
		users: map[string]*domain.User{
			"testuser": {
				ID:           1,
				Login:        "testuser",
				PasswordHash: string(hash),
			},
		},
	}
	uc := auth.NewLoginUsecase(repo, "test-secret")

	_, err = uc.Execute(context.Background(), auth.LoginInput{
		LoginOrEmail: "testuser",
		Password:     "wrong-password",
	})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginValidCredentials(t *testing.T) {
	password := "correct-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repo := &loginMockUserRepo{
		users: map[string]*domain.User{
			"testuser": {
				ID:           1,
				Login:        "testuser",
				PasswordHash: string(hash),
			},
		},
	}
	uc := auth.NewLoginUsecase(repo, "test-secret")

	out, err := uc.Execute(context.Background(), auth.LoginInput{
		LoginOrEmail: "testuser",
		Password:     password,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Token == "" {
		t.Fatal("expected non-empty JWT token")
	}
}
