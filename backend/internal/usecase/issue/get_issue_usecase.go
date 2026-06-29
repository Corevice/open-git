package issue

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type GetIssueInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	IssueNumber    int
}

type GetIssueUsecase struct {
	issueRepo repository.IIssueRepository
}

func NewGetIssueUsecase(issueRepo repository.IIssueRepository) *GetIssueUsecase {
	return &GetIssueUsecase{issueRepo: issueRepo}
}

func (uc *GetIssueUsecase) Execute(ctx context.Context, input GetIssueInput) (*entity.Issue, error) {
	issue, err := uc.issueRepo.GetByNumber(ctx, input.RepositoryID, input.IssueNumber)
	if err != nil {
		return nil, err
	}
	if issue == nil || issue.State == "deleted" {
		return nil, apperror.ErrNotFound
	}
	return issue, nil
}
