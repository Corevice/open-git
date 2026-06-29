package label

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/repository"
)

type RemoveIssueLabelInput struct {
	RepositoryID uuid.UUID
	IssueNumber  int
	Name         string
}

type RemoveIssueLabelUsecase struct {
	labelRepo repository.ILabelRepository
}

func NewRemoveIssueLabelUsecase(labelRepo repository.ILabelRepository) *RemoveIssueLabelUsecase {
	return &RemoveIssueLabelUsecase{labelRepo: labelRepo}
}

func (uc *RemoveIssueLabelUsecase) Execute(ctx context.Context, input RemoveIssueLabelInput) error {
	lbl, err := uc.labelRepo.GetByName(ctx, input.RepositoryID, input.Name)
	if err != nil {
		return err
	}
	if lbl == nil {
		return apperror.ErrNotFound
	}

	return uc.labelRepo.RemoveFromIssue(ctx, input.RepositoryID, input.IssueNumber, lbl.ID)
}
