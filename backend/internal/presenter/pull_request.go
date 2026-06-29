package presenter

import "time"

type BranchRefResponse struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

type PullRequestResponse struct {
	ID        int64             `json:"id"`
	NodeID    string            `json:"node_id"`
	URL       string            `json:"url"`
	HTMLURL   string            `json:"html_url"`
	Number    int               `json:"number"`
	State     string            `json:"state"`
	Title     string            `json:"title"`
	Body      *string           `json:"body"`
	User      UserResponse      `json:"user"`
	Head      BranchRefResponse `json:"head"`
	Base      BranchRefResponse `json:"base"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	MergedAt  *time.Time        `json:"merged_at"`
}
