package org

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type InviteMemberInput struct {
	OrgID        uuid.UUID
	CallerID     uuid.UUID
	TargetUserID uuid.UUID
	Role         string
}

type InviteMemberUsecase struct {
	memberships domainrepo.IMembershipRepository
}

func NewInviteMemberUsecase(memberships domainrepo.IMembershipRepository) *InviteMemberUsecase {
	return &InviteMemberUsecase{memberships: memberships}
}

func (u *InviteMemberUsecase) Execute(ctx context.Context, input InviteMemberInput) error {
	callerRole, err := u.memberships.GetRole(ctx, input.OrgID, input.CallerID)
	if err != nil {
		return err
	}
	if callerRole != entity.RoleOwner {
		return domain.ErrForbidden
	}

	if input.Role != entity.RoleOwner && input.Role != entity.RoleMember {
		return domain.ErrValidation
	}

	currentRole, err := u.memberships.GetRole(ctx, input.OrgID, input.TargetUserID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if currentRole == entity.RoleOwner && input.Role == entity.RoleMember {
		if err := u.ensureNotLastOwner(ctx, input.OrgID); err != nil {
			return err
		}
	}

	err = u.memberships.Add(ctx, &entity.Membership{
		OrganizationID: input.OrgID,
		UserID:         input.TargetUserID,
		Role:           input.Role,
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrConflict) {
		return err
	}

	return u.memberships.UpdateRole(ctx, input.OrgID, input.TargetUserID, input.Role)
}

func (u *InviteMemberUsecase) ensureNotLastOwner(ctx context.Context, orgID uuid.UUID) error {
	members, err := u.memberships.ListByOrg(ctx, orgID, 1, 1000)
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
	return nil
}
