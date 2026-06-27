package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/user"
)

var testUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type mockUserRepo struct {
	byID    map[uuid.UUID]*entity.User
	updated *entity.User
}

func (m *mockUserRepo) Create(_ context.Context, _ *entity.User) error {
	return nil
}

func (m *mockUserRepo) Update(_ context.Context, u *entity.User) error {
	m.updated = u
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetByLogin(_ context.Context, _ string) (*entity.User, error) {
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*entity.User, error) {
	return nil, domain.ErrNotFound
}

func TestUpdateUserHappyPath(t *testing.T) {
	repo := &mockUserRepo{
		byID: map[uuid.UUID]*entity.User{
			testUserID: {ID: testUserID, Login: "alice", Email: "alice@example.com"},
		},
	}
	uc := user.NewUpdateUserUsecase(repo)

	out, err := uc.Execute(context.Background(), testUserID, user.UpdateUserInput{
		Name: "Alice",
		Bio:  "Hello world",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "Alice" || out.Bio != "Hello world" {
		t.Fatalf("expected updated name/bio, got name=%q bio=%q", out.Name, out.Bio)
	}
	if repo.updated == nil {
		t.Fatal("expected Update to be called")
	}
	if repo.updated.ID != testUserID {
		t.Fatalf("expected Update with UUID %v, got %v", testUserID, repo.updated.ID)
	}
}

func TestUpdateUserInvalidEmail(t *testing.T) {
	repo := &mockUserRepo{
		byID: map[uuid.UUID]*entity.User{
			testUserID: {ID: testUserID, Login: "alice"},
		},
	}
	uc := user.NewUpdateUserUsecase(repo)

	_, err := uc.Execute(context.Background(), testUserID, user.UpdateUserInput{
		Email: "not-an-email",
	})
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	if err.Error() != "invalid email" {
		t.Fatalf("expected invalid email error, got %v", err)
	}
	if repo.updated != nil {
		t.Fatal("Update should not be called for invalid email")
	}
}

func TestUpdateUserCallsUpdateWithCorrectUUID(t *testing.T) {
	repo := &mockUserRepo{
		byID: map[uuid.UUID]*entity.User{
			testUserID: {ID: testUserID, Login: "bob"},
		},
	}
	uc := user.NewUpdateUserUsecase(repo)

	_, err := uc.Execute(context.Background(), testUserID, user.UpdateUserInput{Name: "Bob"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated == nil {
		t.Fatal("expected Update to be called")
	}
	if repo.updated.ID != testUserID {
		t.Fatalf("Update called with wrong UUID: got %v want %v", repo.updated.ID, testUserID)
	}
}
