package validator_test

import (
	"testing"

	"github.com/open-git/backend/internal/validator"
)

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		wantErr bool
	}{
		{name: "valid login", login: "alice-dev", wantErr: false},
		{name: "too short", login: "ab", wantErr: true},
		{name: "contains space", login: "hello world", wantErr: true},
		{name: "empty", login: "", wantErr: true},
		{name: "too long", login: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrst", wantErr: true},
		{name: "invalid characters", login: "alice_dev", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLogin(tt.login)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{name: "valid email", email: "a@b.com", wantErr: false},
		{name: "invalid email", email: "notanemail", wantErr: true},
		{name: "missing domain", email: "a@", wantErr: true},
		{name: "missing local part", email: "@b.com", wantErr: true},
		{name: "contains space", email: "a @b.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEmail(tt.email)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
