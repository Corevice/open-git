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
	Title          string
	Body           string
	HeadRef        string
	BaseRef        string
	State          string
	AuthorID       uuid.UUID
	MergedAt       *time.Time
}
