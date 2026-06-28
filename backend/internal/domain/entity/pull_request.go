package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	PullRequestStateOpen   = "open"
	PullRequestStateClosed = "closed"
	PullRequestStateMerged = "merged"
)

type PullRequest struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Number         int
	HeadRef        string
	BaseRef        string
	State          string
	MergedAt       *time.Time
	Title          string
	Body           string
	AuthorID       uuid.UUID
}
