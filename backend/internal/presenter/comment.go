package presenter

import (
	"strconv"
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

type CommentResponse struct {
	ID        int64        `json:"id"`
	NodeID    string       `json:"node_id"`
	URL       string       `json:"url"`
	HTMLURL   string       `json:"html_url"`
	Body      string       `json:"body"`
	User      UserResponse `json:"user"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func ToCommentResponse(c *entity.Comment, author *entity.User, owner, repo, apiBase, webBase string) CommentResponse {
	id := UUIDToInt64(c.ID)
	return CommentResponse{
		ID:        id,
		NodeID:    NodeID("IssueComment", id),
		URL:       CommentAPIURL(apiBase, owner, repo, id),
		HTMLURL:   webBase + "/" + owner + "/" + repo + "/issues/comments/" + strconv.FormatInt(id, 10),
		Body:      c.Body,
		User:      ToUserResponse(author, apiBase, webBase, false),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
