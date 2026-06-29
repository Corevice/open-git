package presenter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

func TestToUserResponse(t *testing.T) {
	const (
		apiBase = "https://api.example.com/api/v3"
		webBase = "https://git.example.com"
	)

	userID := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	user := &entity.User{
		ID:        userID,
		Login:     "octocat",
		Email:     "octocat@example.com",
		Name:      "The Octocat",
		Bio:       "GitHub mascot",
		AvatarURL: "https://git.example.com/avatars/octocat",
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	tests := []struct {
		name         string
		includeEmail bool
		check        func(t *testing.T, resp UserResponse)
	}{
		{
			name:         "populates all fields with email",
			includeEmail: true,
			check: func(t *testing.T, resp UserResponse) {
				t.Helper()
				if resp.ID != 10 {
					t.Fatalf("ID = %d, want 10", resp.ID)
				}
				if resp.Login != "octocat" {
					t.Fatalf("Login = %q, want octocat", resp.Login)
				}
				if resp.NodeID == "" {
					t.Fatal("NodeID is empty")
				}
				if resp.URL != UserAPIURL(apiBase, "octocat") {
					t.Fatalf("URL = %q, want %q", resp.URL, UserAPIURL(apiBase, "octocat"))
				}
				if resp.HTMLURL != UserHTMLURL(webBase, "octocat") {
					t.Fatalf("HTMLURL = %q, want %q", resp.HTMLURL, UserHTMLURL(webBase, "octocat"))
				}
				if !strings.HasPrefix(resp.URL, apiBase) {
					t.Fatalf("URL %q does not start with apiBase", resp.URL)
				}
				if !strings.HasPrefix(resp.HTMLURL, webBase) {
					t.Fatalf("HTMLURL %q does not start with webBase", resp.HTMLURL)
				}
				if resp.AvatarURL != user.AvatarURL {
					t.Fatalf("AvatarURL = %q, want %q", resp.AvatarURL, user.AvatarURL)
				}
				if resp.Type != "User" {
					t.Fatalf("Type = %q, want User", resp.Type)
				}
				if resp.Name != "The Octocat" {
					t.Fatalf("Name = %q, want The Octocat", resp.Name)
				}
				if resp.Bio != "GitHub mascot" {
					t.Fatalf("Bio = %q, want GitHub mascot", resp.Bio)
				}
				if resp.Email != "octocat@example.com" {
					t.Fatalf("Email = %q, want octocat@example.com", resp.Email)
				}
				if !resp.CreatedAt.Equal(createdAt) {
					t.Fatalf("CreatedAt = %v, want %v", resp.CreatedAt, createdAt)
				}
				if !resp.UpdatedAt.Equal(updatedAt) {
					t.Fatalf("UpdatedAt = %v, want %v", resp.UpdatedAt, updatedAt)
				}
			},
		},
		{
			name:         "omits email when includeEmail is false",
			includeEmail: false,
			check: func(t *testing.T, resp UserResponse) {
				t.Helper()
				if resp.Email != "" {
					t.Fatalf("Email = %q, want empty", resp.Email)
				}
				raw, err := json.Marshal(resp)
				if err != nil {
					t.Fatalf("json.Marshal error = %v", err)
				}
				if strings.Contains(string(raw), "email") {
					t.Fatalf("json contains email field: %s", string(raw))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ToUserResponse(user, apiBase, webBase, tt.includeEmail)
			tt.check(t, resp)
		})
	}
}
