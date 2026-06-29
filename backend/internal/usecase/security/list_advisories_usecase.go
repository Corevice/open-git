package security

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type ListAdvisoriesInput struct {
	OrganizationID uuid.UUID
	State          string
	Severity       string
	Page           int
	PerPage        int
}

type ListAdvisoriesOutput struct {
	Advisories []*entity.SecurityAdvisory
	Total      int
}

type ListAdvisoriesUsecase struct {
	advisoryRepo repository.ISecurityAdvisoryRepository
}

func NewListAdvisoriesUsecase(advisoryRepo repository.ISecurityAdvisoryRepository) *ListAdvisoriesUsecase {
	return &ListAdvisoriesUsecase{advisoryRepo: advisoryRepo}
}

func (uc *ListAdvisoriesUsecase) Execute(ctx context.Context, input ListAdvisoriesInput) (*ListAdvisoriesOutput, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	advisories, total, err := uc.advisoryRepo.ListByOrg(ctx, input.OrganizationID, input.State, input.Severity, page, perPage)
	if err != nil {
		return nil, err
	}

	return &ListAdvisoriesOutput{
		Advisories: advisories,
		Total:      total,
	}, nil
}
