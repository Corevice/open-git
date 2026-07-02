package auth_test

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/usecase/auth"
)

type mockUserRepo struct {
	byLogin map[string]*domain.User
	created []*domain.User
}

func (m *mockUserRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	for _, u := range m.byLogin {
		if u.ID == id {
			return u, nil
		}
	}
	for _, u := range m.created {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) Create(_ context.Context, user *domain.User) error {
	m.created = append(m.created, user)
	user.ID = int64(len(m.created))
	return nil
}

func (m *mockUserRepo) GetByLogin(_ context.Context, login string) (*domain.User, error) {
	if u, ok := m.byLogin[login]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, errors.New("not found")
}

func TestRegisterDuplicateLogin(t *testing.T) {
	repo := &mockUserRepo{
		byLogin: map[string]*domain.User{
			"existing": {Login: "existing"},
		},
	}
	uc := auth.NewRegisterUserUsecase(repo, nil)

	_, err := uc.Execute(context.Background(), auth.RegisterUserInput{
		Login:    "existing",
		Email:    "user@example.com",
		Password: "password123",
	})
	if !errors.Is(err, auth.ErrDuplicateLogin) {
		t.Fatalf("expected ErrDuplicateLogin, got %v", err)
	}
}

func TestRegisterHashesPassword(t *testing.T) {
	repo := &mockUserRepo{byLogin: map[string]*domain.User{}}
	uc := auth.NewRegisterUserUsecase(repo, nil)

	password := "password123"
	_, err := uc.Execute(context.Background(), auth.RegisterUserInput{
		Login:    "newuser",
		Email:    "new@example.com",
		Password: password,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatal("expected user to be created")
	}

	hash := repo.created[0].PasswordHash
	if hash == password {
		t.Fatal("password must not be stored in plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Fatal("password hash should verify with bcrypt")
	}
}
