package repository

import "github.com/google/uuid"

type ListIssuesFilter struct {
	OrganizationID  uuid.UUID
	RepositoryID    uuid.UUID
	State           string
	Labels          []string
	MilestoneNumber *int
	Assignee        string
	Sort            string
	Direction       string
	Page            int
	PerPage         int
}
