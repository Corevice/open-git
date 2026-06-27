package entity

import "github.com/google/uuid"

type Comment struct {
	ID       uuid.UUID
	IssueID  uuid.UUID
	AuthorID uuid.UUID
	Body     string
}

type BranchProtection struct {
	RequiredReviews int
	RequiredChecks  []string
}

type WorkflowRun struct {
	ID     uuid.UUID
	Status string
}
