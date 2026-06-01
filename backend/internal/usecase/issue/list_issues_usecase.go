package issue

import (
	"context"

	"github.com/Corevice/open-git/backend/internal/domain/entity"
	"github.com/Corevice/open-git/backend/internal/domain/repository"
	"github.com/google/uuid"
)

type ListIssuesInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	State          string
	Labels         []string
	Page           int
	PerPage        int
}

type ListIssuesOutput struct {
	Issues []*entity.Issue
	Total  int
	Page   int
	PerPage int
}

type ListIssuesUsecase struct {
	issueRepo repository.IIssueRepository
}

func NewListIssuesUsecase(issueRepo repository.IIssueRepository) *ListIssuesUsecase {
	return &ListIssuesUsecase{issueRepo: issueRepo}
}

func (uc *ListIssuesUsecase) Execute(ctx context.Context, input ListIssuesInput) (*ListIssuesOutput, error) {
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

	filter := repository.ListIssuesFilter{
		OrganizationID: input.OrganizationID,
		RepositoryID:   input.RepositoryID,
		State:          input.State,
		Labels:         input.Labels,
		Page:           page,
		PerPage:        perPage,
	}

	issues, total, err := uc.issueRepo.ListByRepo(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListIssuesOutput{
		Issues:  issues,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}
