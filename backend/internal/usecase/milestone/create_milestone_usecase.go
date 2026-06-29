package milestone

import (
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type CreateMilestoneInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	Title          string
	Description    string
	DueOn          *time.Time
}

type CreateMilestoneUsecase struct {
	milestoneRepo repository.IMilestoneRepository
	auditLogRepo  repository.IAuditLogRepository
}

func NewCreateMilestoneUsecase(
	milestoneRepo repository.IMilestoneRepository,
	auditLogRepo repository.IAuditLogRepository,
) *CreateMilestoneUsecase {
	return &CreateMilestoneUsecase{
		milestoneRepo: milestoneRepo,
		auditLogRepo:  auditLogRepo,
	}
}

func (uc *CreateMilestoneUsecase) Execute(ctx context.Context, input CreateMilestoneInput) (*entity.Milestone, error) {
	if utf8.RuneCountInString(input.Title) < 1 {
		return nil, apperror.ErrValidation
	}

	number, err := uc.milestoneRepo.NextNumber(ctx, input.RepositoryID)
	if err != nil {
		return nil, err
	}

	milestone := &entity.Milestone{
		ID:             uuid.New(),
		OrganizationID: input.OrganizationID,
		RepositoryID:   input.RepositoryID,
		Number:         number,
		Title:          input.Title,
		Description:    input.Description,
		State:          "open",
		DueOn:          input.DueOn,
	}

	if err := uc.milestoneRepo.Create(ctx, milestone); err != nil {
		return nil, err
	}

	if err := uc.auditLogRepo.InsertAuditLog(
		ctx,
		input.OrganizationID,
		input.ActorID,
		"milestone.create",
		"milestone",
		milestone.ID,
		json.RawMessage(`{}`),
	); err != nil {
		return nil, err
	}

	return milestone, nil
}
