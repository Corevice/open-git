package issue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type UpdateIssueInput struct {
	OrganizationID  uuid.UUID
	RepositoryID    uuid.UUID
	IssueNumber     int
	ActorID         uuid.UUID
	Title           *string
	Body            *string
	State           *string
	StateReason     *string
	LabelNames      []string
	MilestoneNumber *int
}

type UpdateIssueUsecase struct {
	issueRepo     repository.IIssueRepository
	labelRepo     repository.ILabelRepository
	milestoneRepo repository.IMilestoneRepository
	auditLogRepo  repository.IAuditLogRepository
}

func NewUpdateIssueUsecase(
	issueRepo repository.IIssueRepository,
	labelRepo repository.ILabelRepository,
	milestoneRepo repository.IMilestoneRepository,
	auditLogRepo repository.IAuditLogRepository,
) *UpdateIssueUsecase {
	return &UpdateIssueUsecase{
		issueRepo:     issueRepo,
		labelRepo:     labelRepo,
		milestoneRepo: milestoneRepo,
		auditLogRepo:  auditLogRepo,
	}
}

func (uc *UpdateIssueUsecase) Execute(ctx context.Context, input UpdateIssueInput) (*entity.Issue, error) {
	issue, err := uc.issueRepo.GetByNumber(ctx, input.RepositoryID, input.IssueNumber)
	if err != nil {
		return nil, err
	}
	if issue == nil || issue.State == "deleted" {
		return nil, apperror.ErrNotFound
	}

	if input.Title != nil {
		if err := validateTitle(*input.Title); err != nil {
			return nil, err
		}
		issue.Title = *input.Title
	}

	if input.Body != nil {
		issue.Body = *input.Body
	}

	if input.LabelNames != nil {
		labelIDs := make([]uuid.UUID, 0, len(input.LabelNames))
		for _, name := range input.LabelNames {
			label, err := uc.labelRepo.GetByName(ctx, input.RepositoryID, name)
			if err != nil {
				return nil, err
			}
			if label == nil {
				return nil, apperror.ErrValidation
			}
			labelIDs = append(labelIDs, label.ID)
		}
		issue.LabelIDs = labelIDs
	}

	if input.MilestoneNumber != nil {
		if *input.MilestoneNumber == 0 {
			issue.MilestoneID = nil
		} else {
			milestone, err := uc.milestoneRepo.GetByNumber(ctx, input.RepositoryID, *input.MilestoneNumber)
			if err != nil {
				return nil, err
			}
			if milestone == nil {
				return nil, apperror.ErrValidation
			}
			milestoneID := milestone.ID
			issue.MilestoneID = &milestoneID
		}
	}

	prevState := issue.State

	if input.State != nil {
		issue.State = *input.State
	}

	if input.StateReason != nil {
		issue.StateReason = *input.StateReason
	}

	if input.State != nil && *input.State == "closed" && prevState == "open" {
		now := time.Now().UTC()
		issue.ClosedAt = &now
	}

	if err := uc.issueRepo.Update(ctx, issue); err != nil {
		return nil, err
	}

	if err := uc.auditLogRepo.InsertAuditLog(
		ctx,
		input.OrganizationID,
		input.ActorID,
		"issue.update",
		"issue",
		issue.ID,
		json.RawMessage(`{}`),
	); err != nil {
		return nil, err
	}

	return issue, nil
}
