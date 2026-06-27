package entity

import (
	"errors"

	"github.com/google/uuid"
)

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

type Membership struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           string
}

func (m *Membership) ValidateRole() error {
	switch m.Role {
	case RoleOwner, RoleAdmin, RoleMember:
		return nil
	default:
		return errors.New("invalid role")
	}
}
