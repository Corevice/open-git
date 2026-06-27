package user

import (
	"context"

	"github.com/open-git/backend/internal/domain"
	repo "github.com/open-git/backend/internal/repository"
)

type GetCurrentUserUsecase struct {
	users repo.IUserRepository
}

func NewGetCurrentUserUsecase(users repo.IUserRepository) *GetCurrentUserUsecase {
	return &GetCurrentUserUsecase{users: users}
}

func (u *GetCurrentUserUsecase) Execute(ctx context.Context, userID int64) (*domain.User, error) {
	return u.users.GetByID(ctx, userID)
}

type GetUserByLoginUsecase struct {
	users repo.IUserRepository
}

func NewGetUserByLoginUsecase(users repo.IUserRepository) *GetUserByLoginUsecase {
	return &GetUserByLoginUsecase{users: users}
}

func (u *GetUserByLoginUsecase) Execute(ctx context.Context, login string) (*domain.User, error) {
	user, err := u.users.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}
	return user, nil
}
