package issue

import (
	"context"
	"encoding/json"

	"github.com/Corevice/open-git/backend/internal/apperror"
	"github.com/Corevice/open-git/backend/internal/domain/entity"
	"github.com/Corevice/open-git/backend/internal/domain/repository"
	"github.com/google/uuid"
)

type CreateCommentInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	IssueNumber    int
	ActorID        uuid.UUID
	Body           string
}

type CreateCommentUsecase struct {
	issueRepo    repository.IIssueRepository
	commentRepo  repository.ICommentRepository
	auditLogRepo repository.IAuditLogRepository
}

func NewCreateCommentUsecase(
	issueRepo repository.IIssueRepository,
	commentRepo repository.ICommentRepository,
	auditLogRepo repository.IAuditLogRepository,
) *CreateCommentUsecase {
	return &CreateCommentUsecase{
		issueRepo:    issueRepo,
		commentRepo:  commentRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (uc *CreateCommentUsecase) Execute(ctx context.Context, input CreateCommentInput) (*entity.Comment, error) {
	issue, err := uc.issueRepo.GetByNumber(ctx, input.RepositoryID, input.IssueNumber)
	if err != nil {
		return nil, err
	}
	if issue.State == "deleted" {
		return nil, apperror.ErrGone
	}

	comment := &entity.Comment{
		ID:       uuid.New(),
		IssueID:  issue.ID,
		AuthorID: input.ActorID,
		Body:     input.Body,
	}

	if err := uc.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	if err := uc.auditLogRepo.InsertAuditLog(
		ctx,
		input.OrganizationID,
		input.ActorID,
		"issue.comment",
		"comment",
		comment.ID,
		json.RawMessage(`{}`),
	); err != nil {
		return nil, err
	}

	return comment, nil
}
