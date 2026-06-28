package repository

import (
	"context"

	"github.com/google/uuid"
)

type ListIssuesFilter struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	State          string
	Labels         []string
	Page           int
	PerPage        int
}

type ListPullRequestsFilter struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	State          string
	Page           int
	PerPage        int
}

type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(context.Context) error) error
}
