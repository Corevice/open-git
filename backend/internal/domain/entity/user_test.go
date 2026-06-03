package entity_test

import (
	"testing"

	"github.com/Corevice/open-git/backend/internal/domain/entity"
)

func TestUserValidateLogin(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		wantErr bool
	}{
		{
			name:    "rejects too short",
			login:   "ab",
			wantErr: true,
		},
		{
			name:    "rejects space",
			login:   "hello world",
			wantErr: true,
		},
		{
			name:    "accepts valid login",
			login:   "alice-dev",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &entity.User{Login: tt.login}
			err := user.ValidateLogin()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for login %q", tt.login)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for login %q: %v", tt.login, err)
			}
		})
	}
}
