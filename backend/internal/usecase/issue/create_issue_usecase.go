package issue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/Corevice/open-git/backend/internal/apperror"
	"github.com/Corevice/open-git/backend/internal/domain/entity"
	"github.com/Corevice/open-git/backend/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const maxNumberRetries = 5

type CreateIssueInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Title          string
	Body           string
}

type CreateIssueUsecase struct {
	issueRepo    repository.IIssueRepository
	auditLogRepo repository.IAuditLogRepository
	txManager    repository.TransactionManager
}

func NewCreateIssueUsecase(
	issueRepo repository.IIssueRepository,
	auditLogRepo repository.IAuditLogRepository,
	txManager repository.TransactionManager,
) *CreateIssueUsecase {
	return &CreateIssueUsecase{
		issueRepo:    issueRepo,
		auditLogRepo: auditLogRepo,
		txManager:    txManager,
	}
}

func (uc *CreateIssueUsecase) Execute(ctx context.Context, input CreateIssueInput) (*entity.Issue, error) {
	if err := validateTitle(input.Title); err != nil {
		return nil, err
	}

	var created *entity.Issue
	for attempt := 0; attempt < maxNumberRetries; attempt++ {
		issue := &entity.Issue{
			ID:             uuid.New(),
			OrganizationID: input.OrganizationID,
			RepositoryID:   input.RepositoryID,
			Title:          input.Title,
			Body:           input.Body,
			State:          "open",
			AuthorID:       input.ActorID,
		}

		err := uc.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
			number, err := uc.issueRepo.NextNumber(txCtx, input.RepositoryID)
			if err != nil {
				return err
			}
			issue.Number = number

			if err := uc.issueRepo.Create(txCtx, issue); err != nil {
				return err
			}

			return uc.auditLogRepo.InsertAuditLog(
				txCtx,
				input.OrganizationID,
				input.ActorID,
				"issue.create",
				"issue",
				issue.ID,
				json.RawMessage(`{}`),
			)
		})
		if err == nil {
			created = issue
			break
		}
		if !isUniqueViolation(err) {
			return nil, err
		}
	}

	if created == nil {
		return nil, errors.New("failed to allocate issue number")
	}

	return created, nil
}

func validateTitle(title string) error {
	length := utf8.RuneCountInString(title)
	if length < 1 || length > 256 {
		return apperror.ErrValidation
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
