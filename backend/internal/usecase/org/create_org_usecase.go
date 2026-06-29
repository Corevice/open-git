package org

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/validator"
)

var (
	ErrDuplicateLogin = errors.New("duplicate login")
	ErrReservedLogin  = errors.New("reserved login")
)

type CreateOrgInput struct {
	CreatorID   uuid.UUID
	Login       string
	Name        string
	Description string
}

type CreateOrgUsecase struct {
	orgs        domainrepo.IOrganizationRepository
	memberships domainrepo.IMembershipRepository
}

func NewCreateOrgUsecase(
	orgs domainrepo.IOrganizationRepository,
	memberships domainrepo.IMembershipRepository,
) *CreateOrgUsecase {
	return &CreateOrgUsecase{
		orgs:        orgs,
		memberships: memberships,
	}
}

func (u *CreateOrgUsecase) Execute(ctx context.Context, input CreateOrgInput) (*entity.Organization, error) {
	if err := validator.ValidateLogin(input.Login); err != nil {
		if errors.Is(err, validator.ErrReservedLogin) {
			return nil, ErrReservedLogin
		}
		return nil, err
	}

	existing, err := u.orgs.GetByLogin(ctx, input.Login)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateLogin
	}

	name := input.Name
	if name == "" {
		name = input.Login
	}

	org := &entity.Organization{
		Login:       input.Login,
		Name:        name,
		Description: input.Description,
	}

	if err := u.orgs.Create(ctx, org); err != nil {
		return nil, err
	}

	if err := u.memberships.Add(ctx, &entity.Membership{
		OrganizationID: org.ID,
		UserID:         input.CreatorID,
		Role:           entity.RoleOwner,
	}); err != nil {
		return nil, err
	}

	return org, nil
}
