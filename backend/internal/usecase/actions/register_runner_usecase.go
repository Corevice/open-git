package actions

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

var labelPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

type RegisterRunnerRequest struct {
	RegistrationToken string
	Name              string
	Labels            []string
	OS                string
	Arch              string
	RunnerType        string
}

type RegisterRunnerUsecase struct {
	runnerRepo  domainrepo.IRunnerRepository
	tokenRepo   domainrepo.IRunnerRegistrationTokenRepository
	auditLogRepo domainrepo.IAuditLogRepository
}

func NewRegisterRunnerUsecase(
	runnerRepo domainrepo.IRunnerRepository,
	tokenRepo domainrepo.IRunnerRegistrationTokenRepository,
	auditLogRepo domainrepo.IAuditLogRepository,
) *RegisterRunnerUsecase {
	return &RegisterRunnerUsecase{
		runnerRepo:   runnerRepo,
		tokenRepo:    tokenRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (uc *RegisterRunnerUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	req RegisterRunnerRequest,
) (*entity.Runner, error) {
	if err := validateRunnerLabels(req.Labels); err != nil {
		return nil, err
	}

	tokenHash := hashRegistrationToken(req.RegistrationToken)
	token, err := uc.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	if token.OrganizationID != orgID {
		return nil, domain.ErrUnauthorized
	}
	if token.UsedAt != nil {
		return nil, domain.ErrUnauthorized
	}
	if !time.Now().UTC().Before(token.ExpiresAt) {
		return nil, domain.ErrUnauthorized
	}

	now := time.Now().UTC()
	if err := uc.tokenRepo.MarkUsed(ctx, token.ID, now); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	runner := &entity.Runner{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           req.Name,
		Labels:         req.Labels,
		OS:             req.OS,
		Arch:           req.Arch,
		RunnerType:     req.RunnerType,
		Status:         entity.RunnerStatusOnline,
		LastSeenAt:     &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.runnerRepo.Create(ctx, runner); err != nil {
		return nil, err
	}

	if err := uc.auditLogRepo.Create(ctx, &entity.AuditLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Action:         "runner_registered",
		TargetType:     "runner",
		TargetID:       runner.ID.String(),
		CreatedAt:      now,
	}); err != nil {
		return nil, err
	}

	return runner, nil
}

func validateRunnerLabels(labels []string) error {
	if len(labels) == 0 {
		return fmt.Errorf("%w: labels must not be empty", apperror.ErrValidation)
	}
	for _, label := range labels {
		if !labelPattern.MatchString(label) {
			return fmt.Errorf("%w: invalid label %q", apperror.ErrValidation, label)
		}
	}
	return nil
}
