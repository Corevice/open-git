package validator

import (
	"errors"
	"regexp"
)

var (
	loginRegex = regexp.MustCompile(`^[a-zA-Z0-9-]{3,39}$`)
	emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

func ValidateLogin(login string) error {
	if !loginRegex.MatchString(login) {
		return errors.New("invalid login: must be 3-39 alphanumeric chars or dashes")
	}
	return nil
}

func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email address")
	}
	return nil
}
