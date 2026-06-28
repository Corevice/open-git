package entity

import (
	"errors"
	"time"
)

const (
	ThemeLight  = "light"
	ThemeDark   = "dark"
	ThemeSystem = "system"
)

var validThemes = map[string]struct{}{
	ThemeLight:  {},
	ThemeDark:   {},
	ThemeSystem: {},
}

type UserPreferences struct {
	UserID    int64
	Theme     string
	UpdatedAt time.Time
}

func (p *UserPreferences) ValidateTheme() error {
	if _, ok := validThemes[p.Theme]; !ok {
		return errors.New("invalid theme")
	}
	return nil
}
