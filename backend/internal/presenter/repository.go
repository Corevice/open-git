package presenter

import (
	"strings"
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

type RepositoryResponse struct {
	ID            int64        `json:"id"`
	NodeID        string       `json:"node_id"`
	Name          string       `json:"name"`
	FullName      string       `json:"full_name"`
	Private       bool         `json:"private"`
	Owner         UserResponse `json:"owner"`
	Description   string       `json:"description"`
	URL           string       `json:"url"`
	HTMLURL       string       `json:"html_url"`
	CloneURL      string       `json:"clone_url"`
	SSHURL        string       `json:"ssh_url"`
	DefaultBranch string       `json:"default_branch"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

func ToRepositoryResponse(r *entity.Repository, owner *entity.User, apiBase, webBase string) RepositoryResponse {
	id := UUIDToInt64(r.ID)
	return RepositoryResponse{
		ID:            id,
		NodeID:        NodeID("Repository", id),
		Name:          r.Name,
		FullName:      r.OwnerLogin + "/" + r.Name,
		Private:       r.Visibility == entity.VisibilityPrivate,
		Owner:         ToUserResponse(owner, apiBase, webBase, false),
		Description:   r.Description,
		URL:           RepoAPIURL(apiBase, r.OwnerLogin, r.Name),
		HTMLURL:       RepoHTMLURL(webBase, r.OwnerLogin, r.Name),
		CloneURL:      webBase + "/" + r.OwnerLogin + "/" + r.Name + ".git",
		SSHURL:        "git@" + extractHost(webBase) + ":" + r.OwnerLogin + "/" + r.Name + ".git",
		DefaultBranch: r.DefaultBranch,
		CreatedAt:     r.CreatedAt,
	}
}

func extractHost(webBase string) string {
	host := webBase
	if strings.HasPrefix(host, "https://") {
		host = strings.TrimPrefix(host, "https://")
	} else if strings.HasPrefix(host, "http://") {
		host = strings.TrimPrefix(host, "http://")
	}
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	return host
}
