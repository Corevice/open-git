package auth

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/repository"
	"github.com/open-git/backend/internal/validator"
)

var ErrDuplicateLogin = errors.New("duplicate login")

type RegisterUserInput struct {
	Login    string
	Email    string
	Password string
}

type RegisterUserUsecase struct {
	users repository.IUserRepository
}

func NewRegisterUserUsecase(users repository.IUserRepository) *RegisterUserUsecase {
	return &RegisterUserUsecase{users: users}
}

func (u *RegisterUserUsecase) Execute(ctx context.Context, input RegisterUserInput) (*domain.User, error) {
	if err := validator.ValidateLogin(input.Login); err != nil {
		return nil, err
	}
	if err := validator.ValidateEmail(input.Email); err != nil {
		return nil, err
	}

	if _, err := u.users.GetByLogin(ctx, input.Login); err == nil {
		return nil, ErrDuplicateLogin
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Login:        input.Login,
		Email:        input.Email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UTC(),
	}

	if err := u.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
