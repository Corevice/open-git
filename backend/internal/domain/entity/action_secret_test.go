package entity_test

import (
	"errors"
	"testing"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
)

func TestActionSecretValidate(t *testing.T) {
	tests := []struct {
		name    string
		secret  *entity.ActionSecret
		wantErr bool
	}{
		{
			name: "valid name MY_SECRET passes",
			secret: &entity.ActionSecret{
				Name:       "MY_SECRET",
				Visibility: entity.VisibilityAll,
			},
			wantErr: false,
		},
		{
			name: "lowercase name fails",
			secret: &entity.ActionSecret{
				Name:       "my_secret",
				Visibility: entity.VisibilityAll,
			},
			wantErr: true,
		},
		{
			name: "GITHUB_TOKEN fails",
			secret: &entity.ActionSecret{
				Name:       "GITHUB_TOKEN",
				Visibility: entity.VisibilityAll,
			},
			wantErr: true,
		},
		{
			name: "empty name fails",
			secret: &entity.ActionSecret{
				Name:       "",
				Visibility: entity.VisibilityAll,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.secret.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				if !errors.Is(err, apperror.ErrValidation) {
					t.Fatalf("expected ErrValidation, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSecretVisibilityConstants(t *testing.T) {
	if entity.VisibilityAll != "all" {
		t.Fatalf("VisibilityAll = %q, want %q", entity.VisibilityAll, "all")
	}
	if entity.VisibilityPrivate != "private" {
		t.Fatalf("VisibilityPrivate = %q, want %q", entity.VisibilityPrivate, "private")
	}
	if entity.VisibilitySelected != "selected" {
		t.Fatalf("VisibilitySelected = %q, want %q", entity.VisibilitySelected, "selected")
	}
}
