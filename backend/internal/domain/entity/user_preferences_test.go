package entity_test

import (
	"testing"
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestUserPreferencesStructFields(t *testing.T) {
	now := time.Now()
	prefs := entity.UserPreferences{
		UserID:    42,
		Theme:     entity.ThemeLight,
		UpdatedAt: now,
	}

	if prefs.UserID != 42 {
		t.Fatalf("UserID = %d, want 42", prefs.UserID)
	}
	if prefs.Theme != entity.ThemeLight {
		t.Fatalf("Theme = %q, want %q", prefs.Theme, entity.ThemeLight)
	}
	if !prefs.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt = %v, want %v", prefs.UpdatedAt, now)
	}
}

func TestUserPreferencesValidateTheme(t *testing.T) {
	tests := []struct {
		name    string
		theme   string
		wantErr bool
	}{
		{name: "light", theme: entity.ThemeLight, wantErr: false},
		{name: "dark", theme: entity.ThemeDark, wantErr: false},
		{name: "system", theme: entity.ThemeSystem, wantErr: false},
		{name: "invalid", theme: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefs := &entity.UserPreferences{Theme: tt.theme}
			err := prefs.ValidateTheme()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for theme %q", tt.theme)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for theme %q: %v", tt.theme, err)
			}
		})
	}
}
