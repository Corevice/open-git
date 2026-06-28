package entity_test

import (
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestComputeOverallStatus(t *testing.T) {
	t.Run("AllPass", func(t *testing.T) {
		checks := []*entity.MCPVerificationCheck{
			{Status: entity.CheckStatusPass},
			{Status: entity.CheckStatusPass},
		}
		got := entity.ComputeOverallStatus(checks)
		if got != entity.OverallStatusCompatible {
			t.Fatalf("ComputeOverallStatus() = %q, want %q", got, entity.OverallStatusCompatible)
		}
	})

	t.Run("AnyFail", func(t *testing.T) {
		checks := []*entity.MCPVerificationCheck{
			{Status: entity.CheckStatusPass},
			{Status: entity.CheckStatusFail},
			{Status: entity.CheckStatusSkip},
		}
		got := entity.ComputeOverallStatus(checks)
		if got != entity.OverallStatusIncompatible {
			t.Fatalf("ComputeOverallStatus() = %q, want %q", got, entity.OverallStatusIncompatible)
		}
	})

	t.Run("NoFailWithSkip", func(t *testing.T) {
		checks := []*entity.MCPVerificationCheck{
			{Status: entity.CheckStatusPass},
			{Status: entity.CheckStatusSkip},
		}
		got := entity.ComputeOverallStatus(checks)
		if got != entity.OverallStatusPartial {
			t.Fatalf("ComputeOverallStatus() = %q, want %q", got, entity.OverallStatusPartial)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		got := entity.ComputeOverallStatus(nil)
		if got != entity.OverallStatusCompatible {
			t.Fatalf("ComputeOverallStatus() = %q, want %q", got, entity.OverallStatusCompatible)
		}
	})
}

func TestMCPVerificationRunValidate(t *testing.T) {
	t.Run("empty RepositoryFullName returns non-nil error", func(t *testing.T) {
		run := &entity.MCPVerificationRun{}
		if err := run.Validate(); err == nil {
			t.Fatal("expected error for empty RepositoryFullName")
		}
	})
}
