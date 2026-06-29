package presenter

import (
	"time"

	"github.com/open-git/backend/internal/domain/entity"
)

type LabelResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type IssueResponse struct {
	ID            int64           `json:"id"`
	NodeID        string          `json:"node_id"`
	URL           string          `json:"url"`
	RepositoryURL string          `json:"repository_url"`
	HTMLURL       string          `json:"html_url"`
	Number        int             `json:"number"`
	State         string          `json:"state"`
	Title         string          `json:"title"`
	Body          *string         `json:"body"`
	User          UserResponse    `json:"user"`
	Labels        []LabelResponse `json:"labels"`
	Milestone     *interface{}    `json:"milestone"`
	Comments      int             `json:"comments"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	ClosedAt      *time.Time      `json:"closed_at"`
}

func ToIssueResponse(issue *entity.Issue, author *entity.User, owner, repo, apiBase, webBase string) IssueResponse {
	id := UUIDToInt64(issue.ID)

	var body *string
	if issue.Body != "" {
		body = &issue.Body
	}

	labels := make([]LabelResponse, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labelID := UUIDToInt64(label.ID)
		labels = append(labels, LabelResponse{
			ID:    labelID,
			Name:  label.Name,
			Color: label.Color,
		})
	}

	createdAt := issue.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := issue.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	var milestone *interface{}
	var closedAt *time.Time
	if issue.ClosedAt != nil {
		closedAt = issue.ClosedAt
	}

	return IssueResponse{
		ID:            id,
		NodeID:        NodeID("Issue", id),
		URL:           IssueAPIURL(apiBase, owner, repo, issue.Number),
		RepositoryURL: RepoAPIURL(apiBase, owner, repo),
		HTMLURL:       IssueHTMLURL(webBase, owner, repo, issue.Number),
		Number:        issue.Number,
		State:         issue.State,
		Title:         issue.Title,
		Body:          body,
		User:          ToUserResponse(author, apiBase, webBase, false),
		Labels:        labels,
		Milestone:     milestone,
		Comments:      issue.CommentsCount,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		ClosedAt:      closedAt,
	}
}
