package presenter

import (
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
)

func TestNodeID(t *testing.T) {
	got := NodeID("Issue", 1)
	if got == "" {
		t.Fatal("NodeID returned empty string")
	}

	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		// No-padding encoding may need padding for decode.
		decoded, err = base64.RawStdEncoding.DecodeString(got)
		if err != nil {
			t.Fatalf("NodeID decode error = %v", err)
		}
	}
	if string(decoded) != "Issue:1" {
		t.Fatalf("decoded NodeID = %q, want Issue:1", string(decoded))
	}

	got2 := NodeID("Issue", 1)
	if got != got2 {
		t.Fatalf("NodeID not consistent: %q vs %q", got, got2)
	}
}

func TestURLHelpers(t *testing.T) {
	apiBase := "https://api.example.com/api/v3"
	webBase := "https://git.example.com"

	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "UserAPIURL",
			got:  UserAPIURL(apiBase, "octocat"),
			want: "https://api.example.com/api/v3/users/octocat",
		},
		{
			name: "UserHTMLURL",
			got:  UserHTMLURL(webBase, "octocat"),
			want: "https://git.example.com/octocat",
		},
		{
			name: "RepoAPIURL",
			got:  RepoAPIURL(apiBase, "octocat", "Hello-World"),
			want: "https://api.example.com/api/v3/repos/octocat/Hello-World",
		},
		{
			name: "RepoHTMLURL",
			got:  RepoHTMLURL(webBase, "octocat", "Hello-World"),
			want: "https://git.example.com/octocat/Hello-World",
		},
		{
			name: "IssueAPIURL",
			got:  IssueAPIURL(apiBase, "octocat", "Hello-World", 42),
			want: "https://api.example.com/api/v3/repos/octocat/Hello-World/issues/42",
		},
		{
			name: "IssueHTMLURL",
			got:  IssueHTMLURL(webBase, "octocat", "Hello-World", 42),
			want: "https://git.example.com/octocat/Hello-World/issues/42",
		},
		{
			name: "CommentAPIURL",
			got:  CommentAPIURL(apiBase, "octocat", "Hello-World", 99),
			want: "https://api.example.com/api/v3/repos/octocat/Hello-World/issues/comments/99",
		},
		{
			name: "OrgAPIURL",
			got:  OrgAPIURL(apiBase, "github"),
			want: "https://api.example.com/api/v3/orgs/github",
		},
		{
			name: "OrgHTMLURL",
			got:  OrgHTMLURL(webBase, "github"),
			want: "https://git.example.com/github",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestUUIDToInt64(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	got := UUIDToInt64(id)
	got2 := UUIDToInt64(id)
	if got != got2 {
		t.Fatalf("UUIDToInt64 not deterministic: %d vs %d", got, got2)
	}
	if got != 10 {
		t.Fatalf("UUIDToInt64 = %d, want 10", got)
	}
}
