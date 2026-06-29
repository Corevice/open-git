package presenter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

func TestToIssueResponse(t *testing.T) {
	const (
		apiBase = "https://api.example.com/api/v3"
		webBase = "https://git.example.com"
		owner   = "octocat"
		repo    = "Hello-World"
	)

	author := &entity.User{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Login:     "octocat",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name  string
		issue *entity.Issue
		check func(t *testing.T, resp IssueResponse)
	}{
		{
			name: "open issue with empty body",
			issue: &entity.Issue{
				ID:            uuid.MustParse("00000000-0000-0000-0000-00000000000c"),
				Number:        42,
				Title:         "Bug report",
				Body:          "",
				State:         "open",
				CommentsCount: 0,
				CreatedAt:     time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:     time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
			},
			check: func(t *testing.T, resp IssueResponse) {
				t.Helper()
				if resp.Body != nil {
					t.Fatalf("Body = %v, want nil for empty body", resp.Body)
				}
				if resp.ClosedAt != nil {
					t.Fatalf("ClosedAt = %v, want nil for open issue", resp.ClosedAt)
				}
				if resp.Labels == nil {
					t.Fatal("Labels is nil, want non-nil empty slice")
				}
				if len(resp.Labels) != 0 {
					t.Fatalf("Labels len = %d, want 0", len(resp.Labels))
				}
			},
		},
		{
			name: "issue with body and labels",
			issue: &entity.Issue{
				ID:            uuid.MustParse("00000000-0000-0000-0000-00000000000d"),
				Number:        7,
				Title:         "Feature",
				Body:          "Details here",
				State:         "open",
				CommentsCount: 3,
				Labels: []entity.Label{
					{
						ID:    uuid.MustParse("00000000-0000-0000-0000-00000000000e"),
						Name:  "bug",
						Color: "ff0000",
					},
				},
				CreatedAt: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2024, 4, 2, 0, 0, 0, 0, time.UTC),
			},
			check: func(t *testing.T, resp IssueResponse) {
				t.Helper()
				if resp.Body == nil || *resp.Body != "Details here" {
					t.Fatalf("Body = %v, want Details here", resp.Body)
				}
				if len(resp.Labels) != 1 {
					t.Fatalf("Labels len = %d, want 1", len(resp.Labels))
				}
				if resp.Labels[0].Name != "bug" {
					t.Fatalf("Labels[0].Name = %q, want bug", resp.Labels[0].Name)
				}
				if resp.Comments != 3 {
					t.Fatalf("Comments = %d, want 3", resp.Comments)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ToIssueResponse(tt.issue, author, owner, repo, apiBase, webBase)

			if resp.NodeID == "" {
				t.Fatal("NodeID is empty")
			}
			if resp.URL != IssueAPIURL(apiBase, owner, repo, tt.issue.Number) {
				t.Fatalf("URL = %q, want %q", resp.URL, IssueAPIURL(apiBase, owner, repo, tt.issue.Number))
			}
			if resp.HTMLURL != IssueHTMLURL(webBase, owner, repo, tt.issue.Number) {
				t.Fatalf("HTMLURL = %q, want %q", resp.HTMLURL, IssueHTMLURL(webBase, owner, repo, tt.issue.Number))
			}
			if !strings.HasPrefix(resp.URL, apiBase) {
				t.Fatalf("URL %q does not start with apiBase", resp.URL)
			}
			if !strings.HasPrefix(resp.HTMLURL, webBase) {
				t.Fatalf("HTMLURL %q does not start with webBase", resp.HTMLURL)
			}
			if resp.RepositoryURL != RepoAPIURL(apiBase, owner, repo) {
				t.Fatalf("RepositoryURL = %q, want %q", resp.RepositoryURL, RepoAPIURL(apiBase, owner, repo))
			}
			if resp.Milestone != nil {
				t.Fatalf("Milestone = %v, want nil", resp.Milestone)
			}

			raw, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("json.Marshal error = %v", err)
			}
			if strings.Contains(string(raw), `"labels":null`) {
				t.Fatalf("labels serialized as null: %s", string(raw))
			}

			tt.check(t, resp)
		})
	}
}
