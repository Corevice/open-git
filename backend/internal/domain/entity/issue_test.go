package entity_test

import (
	"strings"
	"testing"

	"github.com/Corevice/open-git/backend/internal/domain/entity"
)

func TestIssueValidateTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{
			name:    "rejects empty string",
			title:   "",
			wantErr: true,
		},
		{
			name:    "rejects too long",
			title:   strings.Repeat("a", 257),
			wantErr: true,
		},
		{
			name:    "accepts valid title",
			title:   "Fix login bug",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := &entity.Issue{Title: tt.title}
			err := issue.ValidateTitle()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for title %q", tt.title)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for title %q: %v", tt.title, err)
			}
		})
	}
}
