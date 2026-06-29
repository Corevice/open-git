package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	SideLeft  = "LEFT"
	SideRight = "RIGHT"
)

type ReviewComment struct {
	ID             uuid.UUID
	PullRequestID  uuid.UUID
	AuthorID       uuid.UUID
	ReviewID       *uuid.UUID
	Path           string
	DiffHunk       string
	Body           string
	Line           int
	Side           string
	InReplyToID    *uuid.UUID
	Resolved       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
