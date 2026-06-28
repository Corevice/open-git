package issue

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type ListCommentsInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	IssueNumber    int
	Page           int
	PerPage        int
}

type ListCommentsOutput struct {
	Comments []*entity.Comment
	Total    int
	Page     int
	PerPage  int
}

type ListCommentsUsecase struct {
	issueRepo   repository.IIssueRepository
	commentRepo repository.ICommentRepository
}

func NewListCommentsUsecase(
	issueRepo repository.IIssueRepository,
	commentRepo repository.ICommentRepository,
) *ListCommentsUsecase {
	return &ListCommentsUsecase{
		issueRepo:   issueRepo,
		commentRepo: commentRepo,
	}
}

func (uc *ListCommentsUsecase) Execute(ctx context.Context, input ListCommentsInput) (*ListCommentsOutput, error) {
	issue, err := uc.issueRepo.GetByNumber(ctx, input.RepositoryID, input.IssueNumber)
	if err != nil {
		return nil, err
	}
	if issue == nil || issue.State == "deleted" {
		return nil, apperror.ErrNotFound
	}
	if issue.OrganizationID != input.OrganizationID {
		return nil, apperror.ErrNotFound
	}

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

	comments, total, err := uc.commentRepo.ListByIssue(issue.ID, page, perPage)
	if err != nil {
		return nil, err
	}
	if comments == nil {
		comments = []*entity.Comment{}
	}

	return &ListCommentsOutput{
		Comments: comments,
		Total:    total,
		Page:     page,
		PerPage:  perPage,
	}, nil
}
