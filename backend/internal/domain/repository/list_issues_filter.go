package repository

import "github.com/google/uuid"

// ListIssuesFilter holds the criteria for listing issues within a repository.
type ListIssuesFilter struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	State          string
	Labels         []string
	Page           int
	PerPage        int
}
