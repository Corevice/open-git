package presenter

import (
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

type UserResponse struct {
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

func ToUserResponse(u *entity.User, apiBase, webBase string, includeEmail bool) UserResponse {
	id := UUIDToInt64(u.ID)
	resp := UserResponse{
		ID:        id,
		NodeID:    NodeID("User", id),
		Login:     u.Login,
		URL:       UserAPIURL(apiBase, u.Login),
		HTMLURL:   UserHTMLURL(webBase, u.Login),
		AvatarURL: u.AvatarURL,
		Type:      "User",
		Name:      u.Name,
		Bio:       u.Bio,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	if includeEmail {
		resp.Email = u.Email
	}
	return resp
}
