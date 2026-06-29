package secret_test

import (
	"testing"

	secretusecase "github.com/open-git/backend/internal/usecase/secret"
)

func TestMaskSecretValuesSingleValue(t *testing.T) {
	line := "token=abc123 done"
	masked := secretusecase.MaskSecretValues(line, []string{"abc123"})
	if masked != "token=*** done" {
		t.Fatalf("masked = %q, want token=*** done", masked)
	}
}

func TestMaskSecretValuesMultipleSecrets(t *testing.T) {
	line := "first=alpha second=beta third=alpha"
	masked := secretusecase.MaskSecretValues(line, []string{"alpha", "beta"})
	if masked != "first=*** second=*** third=***" {
		t.Fatalf("masked = %q, want all values replaced", masked)
	}
}

func TestMaskSecretValuesEmptySecretsSlice(t *testing.T) {
	line := "unchanged line"
	masked := secretusecase.MaskSecretValues(line, nil)
	if masked != line {
		t.Fatalf("masked = %q, want unchanged line", masked)
	}
}

func TestMaskSecretValuesSkipsEmptyValue(t *testing.T) {
	line := "keep=visible"
	masked := secretusecase.MaskSecretValues(line, []string{"", "visible"})
	if masked != "keep=***" {
		t.Fatalf("masked = %q, want keep=***", masked)
	}
}

func TestMaskSecretValuesPartialMatchWithinWord(t *testing.T) {
	line := "prefix-secret-suffix"
	masked := secretusecase.MaskSecretValues(line, []string{"secret"})
	if masked != "prefix-***-suffix" {
		t.Fatalf("masked = %q, want prefix-***-suffix", masked)
	}
}

func TestMaskSecretValuesReplacesAllOccurrences(t *testing.T) {
	line := "secret secret secret"
	masked := secretusecase.MaskSecretValues(line, []string{"secret"})
	if masked != "*** *** ***" {
		t.Fatalf("masked = %q, want all occurrences replaced", masked)
	}
}
