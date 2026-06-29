package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/repository"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type LoginInput struct {
	LoginOrEmail string
	Password     string
}

type LoginOutput struct {
	Token string
}

type LoginUsecase struct {
	users     repository.IUserRepository
	jwtSecret []byte
}

func NewLoginUsecase(users repository.IUserRepository, jwtSecret string) *LoginUsecase {
	return &LoginUsecase{users: users, jwtSecret: []byte(jwtSecret)}
}

type jwtClaims struct {
	jwt.RegisteredClaims
}

func (u *LoginUsecase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	var (
		user *domain.User
		err  error
	)

	if strings.Contains(input.LoginOrEmail, "@") {
		user, err = u.users.GetByEmail(ctx, input.LoginOrEmail)
	} else {
		user, err = u.users.GetByLogin(ctx, input.LoginOrEmail)
	}
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{Token: signed}, nil
}
