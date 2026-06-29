package handler

import (
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
)

func TestUserNodeID(t *testing.T) {
	got := UserNodeID(1)
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(decoded) != "User:1" {
		t.Fatalf("decoded = %q, want User:1", string(decoded))
	}
}

func TestRepoNodeID(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	got := RepoNodeID(id)
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := "Repository:550e8400-e29b-41d4-a716-446655440000"
	if string(decoded) != want {
		t.Fatalf("decoded = %q, want %q", string(decoded), want)
	}
}
