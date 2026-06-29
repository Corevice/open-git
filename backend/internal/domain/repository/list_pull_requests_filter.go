package repository

import "github.com/google/uuid"

// ListPullRequestsFilter holds the criteria for listing pull requests within a repository.
type ListPullRequestsFilter struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	State          string
	Page           int
	PerPage        int
}
