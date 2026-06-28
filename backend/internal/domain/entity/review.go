package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	ReviewStateApproved         = "APPROVED"
	ReviewStateChangesRequested = "CHANGES_REQUESTED"
	ReviewStateCommented        = "COMMENTED"
	ReviewStatePending          = "PENDING"
	ReviewStateDismissed        = "DISMISSED"
)

type Review struct {
	ID             uuid.UUID
	PullRequestID  uuid.UUID
	ReviewerID     uuid.UUID
	State          string
	Body           string
	CommitSHA      string
	SubmittedAt    *time.Time
	CreatedAt      time.Time
}
