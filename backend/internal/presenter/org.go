package presenter

import (
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

type OrgResponse struct {
	ID        int64     `json:"id"`
	NodeID    string    `json:"node_id"`
	Login     string    `json:"login"`
	URL       string    `json:"url"`
	HTMLURL   string    `json:"html_url"`
	AvatarURL string    `json:"avatar_url"`
	Type      string    `json:"type"`
	Name      string    `json:"name,omitempty"`
	Bio       string    `json:"bio,omitempty"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToOrgResponse(o *entity.Organization, apiBase, webBase string) OrgResponse {
	id := UUIDToInt64(o.ID)
	return OrgResponse{
		ID:          id,
		NodeID:      NodeID("Organization", id),
		Login:       o.Login,
		URL:         OrgAPIURL(apiBase, o.Login),
		HTMLURL:     OrgHTMLURL(webBase, o.Login),
		Type:        "Organization",
		Name:        o.Name,
		Bio:         o.Description,
		CreatedAt:   o.CreatedAt,
	}
}
