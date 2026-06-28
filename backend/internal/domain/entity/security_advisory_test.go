package entity_test

import (
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestSecurityAdvisoryCanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		next     string
		expected bool
	}{
		{
			name:     "open to acknowledged is valid",
			current:  entity.StateOpen,
			next:     entity.StateAcknowledged,
			expected: true,
		},
		{
			name:     "open to resolved is valid",
			current:  entity.StateOpen,
			next:     entity.StateResolved,
			expected: true,
		},
		{
			name:     "acknowledged to dismissed is valid",
			current:  entity.StateAcknowledged,
			next:     entity.StateDismissed,
			expected: true,
		},
		{
			name:     "resolved to open is invalid",
			current:  entity.StateResolved,
			next:     entity.StateOpen,
			expected: false,
		},
		{
			name:     "dismissed to acknowledged is invalid",
			current:  entity.StateDismissed,
			next:     entity.StateAcknowledged,
			expected: false,
		},
		{
			name:     "resolved to dismissed is invalid",
			current:  entity.StateResolved,
			next:     entity.StateDismissed,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advisory := &entity.SecurityAdvisory{State: tt.current}
			if got := advisory.CanTransitionTo(tt.next); got != tt.expected {
				t.Fatalf("CanTransitionTo(%q) = %v, want %v", tt.next, got, tt.expected)
			}
		})
	}
}
