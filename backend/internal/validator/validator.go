package validator

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrReservedLogin = errors.New("reserved login")
	ErrInvalidLogin  = errors.New("invalid login")
)

var loginRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,37}[a-zA-Z0-9]|[a-zA-Z0-9]{0,38})?$`)

var reservedLogins = map[string]bool{
	"admin":         true,
	"api":           true,
	"settings":      true,
	"new":           true,
	"login":         true,
	"root":          true,
	"support":       true,
	"help":          true,
	"git":           true,
	"ssh":           true,
	"www":           true,
	"organizations": true,
}

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func ValidateLogin(login string) error {
	if strings.Contains(login, "--") {
		return ErrInvalidLogin
	}
	if !loginRegex.MatchString(login) {
		return ErrInvalidLogin
	}
	if reservedLogins[strings.ToLower(login)] {
		return ErrReservedLogin
	}
	return nil
}

func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email")
	}
	return nil
}
