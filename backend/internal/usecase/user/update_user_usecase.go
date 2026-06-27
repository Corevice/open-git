package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/validator"
)

type UpdateUserInput struct {
	Name      string
	Bio       string
	AvatarURL string
	Email     string
}

type UpdateUserUsecase struct {
	users domainrepo.IUserRepository
}

func NewUpdateUserUsecase(users domainrepo.IUserRepository) *UpdateUserUsecase {
	return &UpdateUserUsecase{users: users}
}

func (u *UpdateUserUsecase) Execute(ctx context.Context, userID uuid.UUID, input UpdateUserInput) (*entity.User, error) {
	user, err := u.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}

	if input.Email != "" {
		if err := validator.ValidateEmail(input.Email); err != nil {
			return nil, err
		}
		user.Email = input.Email
	}

	user.Name = input.Name
	user.Bio = input.Bio
	user.AvatarURL = input.AvatarURL

	if err := u.users.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
