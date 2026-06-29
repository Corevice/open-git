package security

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type ListDependabotAlertsInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	State          string
	Page           int
	PerPage        int
}

type ListDependabotAlertsOutput struct {
	Alerts []*entity.DependabotAlert
	Total  int
}

type ListDependabotAlertsUsecase struct {
	alertRepo repository.IDependabotAlertRepository
}

func NewListDependabotAlertsUsecase(alertRepo repository.IDependabotAlertRepository) *ListDependabotAlertsUsecase {
	return &ListDependabotAlertsUsecase{alertRepo: alertRepo}
}

func (uc *ListDependabotAlertsUsecase) Execute(ctx context.Context, input ListDependabotAlertsInput) (*ListDependabotAlertsOutput, error) {
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

	alerts, total, err := uc.alertRepo.ListByRepo(ctx, input.OrganizationID, input.RepositoryID, input.State, page, perPage)
	if err != nil {
		return nil, err
	}

	return &ListDependabotAlertsOutput{
		Alerts: alerts,
		Total:  total,
	}, nil
}
