package main

import (
	"strings"
	"testing"
)

func TestValidateRequiredEnv_AllSet(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("DB_DSN", "postgres://localhost/db")

	err := validateRequiredEnv([]string{"JWT_SECRET", "DB_DSN"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidateRequiredEnv_OneEmpty(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DB_DSN", "postgres://localhost/db")

	err := validateRequiredEnv([]string{"JWT_SECRET", "DB_DSN"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected error to contain JWT_SECRET, got %v", err)
	}
	if err.Error() != "missing required environment variable: JWT_SECRET" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}
