package entity

import "github.com/google/uuid"

const (
	CollaboratorPermRead  = "read"
	CollaboratorPermWrite = "write"
	CollaboratorPermAdmin = "admin"
)

type RepositoryCollaborator struct {
	RepositoryID uuid.UUID
	UserID       uuid.UUID
	Permission   string
}
