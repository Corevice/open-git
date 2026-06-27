package org

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	repo "github.com/open-git/backend/internal/repository"
)

type DeleteOrgInput struct {
	OrgID    uuid.UUID
	CallerID uuid.UUID
}

type DeleteOrgUsecase struct {
	orgs        domainrepo.IOrganizationRepository
	memberships domainrepo.IMembershipRepository
	auditLog    repo.IAuditLogRepository
}

func NewDeleteOrgUsecase(
	orgs domainrepo.IOrganizationRepository,
	memberships domainrepo.IMembershipRepository,
	auditLog repo.IAuditLogRepository,
) *DeleteOrgUsecase {
	return &DeleteOrgUsecase{
		orgs:        orgs,
		memberships: memberships,
		auditLog:    auditLog,
	}
}

func (u *DeleteOrgUsecase) Execute(ctx context.Context, input DeleteOrgInput) error {
	role, err := u.memberships.GetRole(ctx, input.OrgID, input.CallerID)
	if err != nil {
		return err
	}
	if role != entity.RoleOwner {
		return domain.ErrForbidden
	}

	if err := u.auditLog.Record(ctx, input.OrgID, input.CallerID, "org.delete", "Organization", input.OrgID, nil); err != nil {
		return err
	}

	return u.orgs.Delete(ctx, input.OrgID)
}
