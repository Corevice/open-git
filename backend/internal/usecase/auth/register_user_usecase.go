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

// PersonalOrgCreator provisions the personal organization that backs a user's
// own namespace. Personal repositories use the owner's id as their
// organization id, and many tables carry a foreign key to organizations(id),
// so every user needs a matching organizations row (id == user id) for those
// references to resolve.
type PersonalOrgCreator interface {
	EnsurePersonalOrg(ctx context.Context, userID int64, login string) error
}

type RegisterUserUsecase struct {
	users       repository.IUserRepository
	personalOrg PersonalOrgCreator
}

func NewRegisterUserUsecase(users repository.IUserRepository, personalOrg PersonalOrgCreator) *RegisterUserUsecase {
	return &RegisterUserUsecase{users: users, personalOrg: personalOrg}
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

	// Provision the user's personal organization so org-scoped foreign keys
	// resolve for their personal repositories. Idempotent in the creator.
	if u.personalOrg != nil {
		if err := u.personalOrg.EnsurePersonalOrg(ctx, user.ID, user.Login); err != nil {
			return nil, err
		}
	}

	return user, nil
}
