package validator_test

import (
	"errors"
	"testing"

	"github.com/open-git/backend/internal/validator"
)

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		wantErr error
	}{
		{name: "valid alice-dev", login: "alice-dev", wantErr: nil},
		{name: "valid alice123", login: "alice123", wantErr: nil},
		{name: "valid single char", login: "a", wantErr: nil},
		{name: "40 chars", login: "abcdefghijklmnopqrstuvwxyzabcdefghijklmn", wantErr: validator.ErrInvalidLogin},
		{name: "leading hyphen", login: "-alice", wantErr: validator.ErrInvalidLogin},
		{name: "trailing hyphen", login: "alice-", wantErr: validator.ErrInvalidLogin},
		{name: "double hyphen", login: "alice--dev", wantErr: validator.ErrInvalidLogin},
		{name: "admin reserved", login: "admin", wantErr: validator.ErrReservedLogin},
		{name: "contains space", login: "hello world", wantErr: validator.ErrInvalidLogin},
		{name: "empty", login: "", wantErr: validator.ErrInvalidLogin},
		{name: "invalid characters", login: "alice_dev", wantErr: validator.ErrInvalidLogin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLogin(tt.login)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
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
