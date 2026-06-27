package org

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type UpdateOrgInput struct {
	OrgID       uuid.UUID
	CallerID    uuid.UUID
	Name        string
	Description string
}

type UpdateOrgUsecase struct {
	orgs        domainrepo.IOrganizationRepository
	memberships domainrepo.IMembershipRepository
}

func NewUpdateOrgUsecase(
	orgs domainrepo.IOrganizationRepository,
	memberships domainrepo.IMembershipRepository,
) *UpdateOrgUsecase {
	return &UpdateOrgUsecase{
		orgs:        orgs,
		memberships: memberships,
	}
}

func (u *UpdateOrgUsecase) Execute(ctx context.Context, input UpdateOrgInput) (*entity.Organization, error) {
	role, err := u.memberships.GetRole(ctx, input.OrgID, input.CallerID)
	if err != nil {
		return nil, err
	}
	if role != entity.RoleOwner {
		return nil, domain.ErrForbidden
	}

	org, err := u.orgs.GetByID(ctx, input.OrgID)
	if err != nil {
		return nil, err
	}

	name := input.Name
	if name == "" {
		name = org.Name
	}
	org.Name = name
	org.Description = input.Description

	if err := u.orgs.Update(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}
