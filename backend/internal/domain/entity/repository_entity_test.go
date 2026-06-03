package entity_test

import (
	"strings"
	"testing"

	"github.com/Corevice/open-git/backend/internal/domain/entity"
)

func TestRepositoryValidateName(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{
			name:    "rejects invalid character",
			repo:    "a#b",
			wantErr: true,
		},
		{
			name:    "rejects too long",
			repo:    strings.Repeat("a", 101),
			wantErr: true,
		},
		{
			name:    "accepts valid name",
			repo:    "my-repo.v2",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &entity.Repository{Name: tt.repo}
			err := repository.ValidateName()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for name %q", tt.repo)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for name %q: %v", tt.repo, err)
			}
		})
	}
}
