package entity

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var loginRegex = regexp.MustCompile(`^[a-zA-Z0-9-]{3,39}$`)

type User struct {
	ID           uuid.UUID
	Login        string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

func (u *User) ValidateLogin() error {
	if !loginRegex.MatchString(u.Login) {
		return errors.New("invalid login")
	}
	return nil
}
