package org

import (
	"context"

	"github.com/open-git/backend/internal/domain"
	repo "github.com/open-git/backend/internal/repository"
)

type GetOrgUsecase struct {
	orgs repo.IOrganizationRepository
}

func NewGetOrgUsecase(orgs repo.IOrganizationRepository) *GetOrgUsecase {
	return &GetOrgUsecase{orgs: orgs}
}

func (u *GetOrgUsecase) Execute(ctx context.Context, login string) (*domain.Organization, error) {
	org, err := u.orgs.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, domain.ErrNotFound
	}
	return org, nil
}

type ListUserOrgsUsecase struct {
	orgs repo.IOrganizationRepository
}

func NewListUserOrgsUsecase(orgs repo.IOrganizationRepository) *ListUserOrgsUsecase {
	return &ListUserOrgsUsecase{orgs: orgs}
}

func (u *ListUserOrgsUsecase) Execute(ctx context.Context, userID int64) ([]*domain.Organization, error) {
	return u.orgs.ListByUserID(ctx, userID)
}
