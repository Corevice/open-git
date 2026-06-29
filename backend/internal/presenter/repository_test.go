package presenter

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

func TestToRepositoryResponse(t *testing.T) {
	const (
		apiBase = "https://api.example.com/api/v3"
		webBase = "https://git.example.com"
	)

	owner := &entity.User{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Login:     "octocat",
		Name:      "The Octocat",
		AvatarURL: "https://git.example.com/avatars/octocat",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name       string
		visibility string
		check      func(t *testing.T, resp RepositoryResponse)
	}{
		{
			name:       "public repository",
			visibility: entity.VisibilityPublic,
			check: func(t *testing.T, resp RepositoryResponse) {
				t.Helper()
				if resp.Private {
					t.Fatal("Private = true, want false for public repo")
				}
				if resp.FullName != "octocat/Hello-World" {
					t.Fatalf("FullName = %q, want octocat/Hello-World", resp.FullName)
				}
				if resp.CloneURL != "https://git.example.com/octocat/Hello-World.git" {
					t.Fatalf("CloneURL = %q, want https://git.example.com/octocat/Hello-World.git", resp.CloneURL)
				}
				if resp.SSHURL != "git@git.example.com:octocat/Hello-World.git" {
					t.Fatalf("SSHURL = %q, want git@git.example.com:octocat/Hello-World.git", resp.SSHURL)
				}
			},
		},
		{
			name:       "private repository",
			visibility: entity.VisibilityPrivate,
			check: func(t *testing.T, resp RepositoryResponse) {
				t.Helper()
				if !resp.Private {
					t.Fatal("Private = false, want true for private repo")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &entity.Repository{
				ID:            uuid.MustParse("00000000-0000-0000-0000-00000000000b"),
				Name:          "Hello-World",
				OwnerLogin:    "octocat",
				Description:   "My first repo",
				Visibility:    tt.visibility,
				DefaultBranch: "main",
				CreatedAt:     time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			}

			resp := ToRepositoryResponse(repo, owner, apiBase, webBase)

			if resp.NodeID == "" {
				t.Fatal("NodeID is empty")
			}
			if resp.URL != RepoAPIURL(apiBase, "octocat", "Hello-World") {
				t.Fatalf("URL = %q, want %q", resp.URL, RepoAPIURL(apiBase, "octocat", "Hello-World"))
			}
			if resp.HTMLURL != RepoHTMLURL(webBase, "octocat", "Hello-World") {
				t.Fatalf("HTMLURL = %q, want %q", resp.HTMLURL, RepoHTMLURL(webBase, "octocat", "Hello-World"))
			}
			if !strings.HasPrefix(resp.URL, apiBase) {
				t.Fatalf("URL %q does not start with apiBase", resp.URL)
			}
			if !strings.HasPrefix(resp.HTMLURL, webBase) {
				t.Fatalf("HTMLURL %q does not start with webBase", resp.HTMLURL)
			}
			if resp.Owner.Login != "octocat" {
				t.Fatalf("Owner.Login = %q, want octocat", resp.Owner.Login)
			}
			if resp.Owner.NodeID == "" {
				t.Fatal("Owner.NodeID is empty")
			}

			tt.check(t, resp)
		})
	}
}
