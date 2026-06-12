package handler_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/handler"
)

func TestDomainErrorToHTTP(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "not found",
			err:        domain.ErrNotFound,
			wantStatus: 404,
			wantCode:   handler.CodeNotFound,
		},
		{
			name:       "conflict",
			err:        domain.ErrConflict,
			wantStatus: 409,
			wantCode:   handler.CodeConflict,
		},
		{
			name:       "validation",
			err:        domain.ErrValidation,
			wantStatus: 422,
			wantCode:   handler.CodeValidationFailed,
		},
		{
			name:       "unauthorized",
			err:        domain.ErrUnauthorized,
			wantStatus: 401,
			wantCode:   handler.CodeUnauthorized,
		},
		{
			name:       "forbidden",
			err:        domain.ErrForbidden,
			wantStatus: 403,
			wantCode:   handler.CodeForbidden,
		},
		{
			name:       "internal",
			err:        domain.ErrInternal,
			wantStatus: 500,
			wantCode:   handler.CodeInternal,
		},
		{
			name:       "wrapped not found",
			err:        fmt.Errorf("wrap: %w", domain.ErrNotFound),
			wantStatus: 404,
			wantCode:   handler.CodeNotFound,
		},
		{
			name:       "unknown error",
			err:        errors.New("something went wrong"),
			wantStatus: 500,
			wantCode:   handler.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotCode := handler.DomainErrorToHTTP(tt.err)
			if gotStatus != tt.wantStatus {
				t.Fatalf("status = %d, want %d", gotStatus, tt.wantStatus)
			}
			if gotCode != tt.wantCode {
				t.Fatalf("code = %q, want %q", gotCode, tt.wantCode)
			}
		})
	}
}
