package pr

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/service"
)

const maxNumberRetries = 5

type CreatePRInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	GitPath        string
	ActorID        uuid.UUID
	Title          string
	Body           string
	HeadRef        string
	BaseRef        string
}

type CreatePRUsecase struct {
	prRepo       repository.IPullRequestRepository
	auditLogRepo repository.IAuditLogRepository
	gitService   service.GitService
	txManager    repository.ITransactionManager
}

func NewCreatePRUsecase(
	prRepo repository.IPullRequestRepository,
	auditLogRepo repository.IAuditLogRepository,
	gitService service.GitService,
	txManager repository.ITransactionManager,
) *CreatePRUsecase {
	return &CreatePRUsecase{
		prRepo:       prRepo,
		auditLogRepo: auditLogRepo,
		gitService:   gitService,
		txManager:    txManager,
	}
}

func (uc *CreatePRUsecase) Execute(ctx context.Context, input CreatePRInput) (*entity.PullRequest, error) {
	if err := validatePRTitle(input.Title); err != nil {
		return nil, err
	}
	if input.HeadRef == input.BaseRef {
		return nil, apperror.ErrValidation
	}

	headExists, err := uc.gitService.BranchExists(input.GitPath, input.HeadRef)
	if err != nil {
		return nil, err
	}
	baseExists, err := uc.gitService.BranchExists(input.GitPath, input.BaseRef)
	if err != nil {
		return nil, err
	}
	if !headExists || !baseExists {
		return nil, apperror.ErrValidation
	}

	var created *entity.PullRequest
	for attempt := 0; attempt < maxNumberRetries; attempt++ {
		pr := &entity.PullRequest{
			ID:             uuid.New(),
			OrganizationID: input.OrganizationID,
			RepositoryID:   input.RepositoryID,
			HeadRef:        input.HeadRef,
			BaseRef:        input.BaseRef,
			Title:          input.Title,
			Body:           input.Body,
			State:          "open",
			AuthorID:       input.ActorID,
		}

		err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
			number, err := uc.prRepo.NextNumber(txCtx, input.RepositoryID)
			if err != nil {
				return err
			}
			pr.Number = number

			if err := uc.prRepo.Create(txCtx, pr); err != nil {
				return err
			}

			return uc.auditLogRepo.Create(txCtx, &entity.AuditLog{
				ID:             uuid.New(),
				OrganizationID: input.OrganizationID,
				ActorID:        input.ActorID,
				Action:         "pr.open",
				TargetType:     "pull_request",
				TargetID:       pr.ID.String(),
			})
		})
		if err == nil {
			created = pr
			break
		}
		if !isUniqueViolation(err) {
			return nil, err
		}
	}

	if created == nil {
		return nil, errors.New("failed to allocate pull request number")
	}

	return created, nil
}

func validatePRTitle(title string) error {
	length := utf8.RuneCountInString(title)
	if length < 1 || length > 256 {
		return apperror.ErrValidation
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if errors.Is(err, apperror.ErrConflict) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "23505")
}
