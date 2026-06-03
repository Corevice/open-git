package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"

	"github.com/Corevice/open-git/backend/internal/domain"
	"github.com/Corevice/open-git/backend/internal/repository"
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
	UserID int64 `json:"sub"`
	jwt.StandardClaims
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
		UserID: user.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: now.Add(24 * time.Hour).Unix(),
			IssuedAt:  now.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{Token: signed}, nil
}
