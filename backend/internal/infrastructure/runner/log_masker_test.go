package runner_test

import (
	"testing"

	"github.com/open-git/backend/internal/infrastructure/runner"
)

func TestMaskSecrets_SingleSecret(t *testing.T) {
	got := runner.MaskSecrets("token=abc123 done", []string{"abc123"})
	want := "token=*** done"
	if got != want {
		t.Fatalf("MaskSecrets() = %q, want %q", got, want)
	}
}

func TestMaskSecrets_MultipleSecrets(t *testing.T) {
	got := runner.MaskSecrets("user=alice pass=secret1 key=secret2", []string{"secret1", "secret2"})
	want := "user=alice pass=*** key=***"
	if got != want {
		t.Fatalf("MaskSecrets() = %q, want %q", got, want)
	}
}

func TestMaskSecrets_EmptySecretList(t *testing.T) {
	input := "nothing to mask"
	got := runner.MaskSecrets(input, nil)
	if got != input {
		t.Fatalf("MaskSecrets() = %q, want %q", got, input)
	}
}

func TestMaskSecrets_EmptySecretValueSkipped(t *testing.T) {
	input := "value=keep"
	got := runner.MaskSecrets(input, []string{"", "keep"})
	want := "value=***"
	if got != want {
		t.Fatalf("MaskSecrets() = %q, want %q", got, want)
	}
}

func TestMaskSecrets_SecretAppearsMultipleTimes(t *testing.T) {
	got := runner.MaskSecrets("abc abc abc", []string{"abc"})
	want := "*** *** ***"
	if got != want {
		t.Fatalf("MaskSecrets() = %q, want %q", got, want)
	}
}
