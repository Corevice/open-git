package validator

import (
	"errors"
	"regexp"
)

var loginRegex = regexp.MustCompile(`^[a-zA-Z0-9-]{3,39}$`)

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func ValidateLogin(login string) error {
	if !loginRegex.MatchString(login) {
		return errors.New("invalid login")
	}
	return nil
}

func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email")
	}
	return nil
}
