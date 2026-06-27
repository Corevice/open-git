package org

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	repo "github.com/open-git/backend/internal/repository"
)

var ErrLastOwner = errors.New("last owner")

type RemoveMemberInput struct {
	OrgID        uuid.UUID
	CallerID     uuid.UUID
	TargetUserID uuid.UUID
}

type RemoveMemberUsecase struct {
	memberships domainrepo.IMembershipRepository
	auditLog    repo.IAuditLogRepository
}

func NewRemoveMemberUsecase(
	memberships domainrepo.IMembershipRepository,
	auditLog repo.IAuditLogRepository,
) *RemoveMemberUsecase {
	return &RemoveMemberUsecase{
		memberships: memberships,
		auditLog:    auditLog,
	}
}

func (u *RemoveMemberUsecase) Execute(ctx context.Context, input RemoveMemberInput) error {
	callerRole, err := u.memberships.GetRole(ctx, input.OrgID, input.CallerID)
	if err != nil {
		return err
	}
	if callerRole != entity.RoleOwner {
		return domain.ErrForbidden
	}

	targetRole, err := u.memberships.GetRole(ctx, input.OrgID, input.TargetUserID)
	if err != nil {
		return err
	}
	if targetRole == entity.RoleOwner {
		members, err := u.memberships.ListByOrg(ctx, input.OrgID, 1, 1000)
		if err != nil {
			return err
		}
		ownerCount := 0
		for _, m := range members {
			if m.Role == entity.RoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return ErrLastOwner
		}
	}

	if err := u.auditLog.Record(ctx, input.OrgID, input.CallerID, "org.member.remove", "User", input.TargetUserID, nil); err != nil {
		return err
	}

	return u.memberships.Remove(ctx, input.OrgID, input.TargetUserID)
}
