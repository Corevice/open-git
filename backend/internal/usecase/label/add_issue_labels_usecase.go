package label

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/repository"
)

type AddIssueLabelsInput struct {
	RepositoryID uuid.UUID
	IssueNumber  int
	Names        []string
}

type AddIssueLabelsUsecase struct {
	labelRepo repository.ILabelRepository
}

func NewAddIssueLabelsUsecase(labelRepo repository.ILabelRepository) *AddIssueLabelsUsecase {
	return &AddIssueLabelsUsecase{labelRepo: labelRepo}
}

func (uc *AddIssueLabelsUsecase) Execute(ctx context.Context, input AddIssueLabelsInput) error {
	for _, name := range input.Names {
		lbl, err := uc.labelRepo.GetByName(ctx, input.RepositoryID, name)
		if err != nil {
			return err
		}
		if lbl == nil {
			return apperror.ErrValidation
		}

		if err := uc.labelRepo.AddToIssue(ctx, input.RepositoryID, input.IssueNumber, lbl.ID); err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return err
		}
	}

	return nil
}
