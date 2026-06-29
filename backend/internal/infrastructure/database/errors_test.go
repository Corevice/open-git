package database

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/open-git/backend/internal/domain"
)

func TestMapDBError(t *testing.T) {
	tests := []struct {
		name       string
		input      error
		wantNil    bool
		wantIs     error
		wantNotMsg string
	}{
		{
			name:    "nil input",
			input:   nil,
			wantNil: true,
		},
		{
			name:   "sql.ErrNoRows",
			input:  sql.ErrNoRows,
			wantIs: domain.ErrNotFound,
		},
		{
			name:   "pq unique violation",
			input:  &pq.Error{Code: "23505"},
			wantIs: domain.ErrConflict,
		},
		{
			name:   "pq foreign key violation",
			input:  &pq.Error{Code: "23503"},
			wantIs: domain.ErrValidation,
		},
		{
			name:   "sqlite unique constraint",
			input:  errors.New("UNIQUE constraint failed: t.id"),
			wantIs: domain.ErrConflict,
		},
		{
			name:   "sqlite database locked",
			input:  errors.New("database is locked"),
			wantIs: domain.ErrUnavailable,
		},
		{
			name:       "generic error",
			input:      errors.New("boom"),
			wantIs:     domain.ErrInternal,
			wantNotMsg: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapDBError(tt.input)

			if tt.wantNil {
				if got != nil {
					t.Fatalf("MapDBError() = %v, want nil", got)
				}
				return
			}

			if !errors.Is(got, tt.wantIs) {
				t.Fatalf("errors.Is(got, %v) = false, got = %v", tt.wantIs, got)
			}

			if tt.wantNotMsg != "" && strings.Contains(got.Error(), tt.wantNotMsg) {
				t.Fatalf("MapDBError() leaked raw message %q: %v", tt.wantNotMsg, got)
			}
		})
	}
}
