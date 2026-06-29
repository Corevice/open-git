package entity

import (
	"errors"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	PullRequestStateOpen   = "open"
	PullRequestStateClosed = "closed"
	PullRequestStateMerged = "merged"

	MergeableStateClean   = "clean"
	MergeableStateDirty   = "dirty"
	MergeableStateBlocked = "blocked"
	MergeableStateBehind  = "behind"
	MergeableStateUnknown = "unknown"
)

type PullRequest struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	RepositoryID     uuid.UUID
	Number           int
	Title            string
	Body             string
	Draft            bool
	HeadRef          string
	BaseRef          string
	HeadSHA          string
	BaseSHA          string
	State            string
	MergedAt         *time.Time
	MergedBy         *uuid.UUID
	MergeCommitSHA   string
	Mergeable        *bool
	MergeableState   string
	AuthorID         uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (pr *PullRequest) ValidateTitle() error {
	length := utf8.RuneCountInString(pr.Title)
	if length < 1 || length > 256 {
		return errors.New("invalid title")
	}
	return nil
}

func ValidateBaseHeadRefs(baseRef, headRef string) error {
	if baseRef == headRef {
		return errors.New("base and head must be different")
	}
	return nil
}
