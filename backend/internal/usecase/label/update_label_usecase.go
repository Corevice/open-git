package label

import (
	"context"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type UpdateLabelInput struct {
	RepositoryID uuid.UUID
	CurrentName  string
	NewName      *string
	Color        *string
	Description  *string
}

type UpdateLabelUsecase struct {
	labelRepo repository.ILabelRepository
}

func NewUpdateLabelUsecase(labelRepo repository.ILabelRepository) *UpdateLabelUsecase {
	return &UpdateLabelUsecase{labelRepo: labelRepo}
}

func (uc *UpdateLabelUsecase) Execute(ctx context.Context, input UpdateLabelInput) (*entity.Label, error) {
	label, err := uc.labelRepo.GetByName(ctx, input.RepositoryID, input.CurrentName)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, apperror.ErrNotFound
	}

	if input.NewName != nil {
		if utf8.RuneCountInString(*input.NewName) > 50 {
			return nil, apperror.ErrValidation
		}
		label.Name = *input.NewName
	}
	if input.Color != nil {
		label.Color = *input.Color
		if err := label.ValidateColor(); err != nil {
			return nil, apperror.ErrValidation
		}
	}
	if input.Description != nil {
		label.Description = *input.Description
	}

	if err := uc.labelRepo.Update(ctx, label); err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.ErrConflict
		}
		return nil, err
	}

	return label, nil
}
