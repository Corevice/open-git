package label

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type CreateLabelInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Name           string
	Color          string
	Description    string
}

type CreateLabelUsecase struct {
	labelRepo repository.ILabelRepository
}

func NewCreateLabelUsecase(labelRepo repository.ILabelRepository) *CreateLabelUsecase {
	return &CreateLabelUsecase{labelRepo: labelRepo}
}

func (uc *CreateLabelUsecase) Execute(ctx context.Context, input CreateLabelInput) (*entity.Label, error) {
	if utf8.RuneCountInString(input.Name) > 50 {
		return nil, apperror.ErrValidation
	}

	label := &entity.Label{
		ID:             uuid.New(),
		OrganizationID: input.OrganizationID,
		RepositoryID:   input.RepositoryID,
		Name:           input.Name,
		Color:          input.Color,
		Description:    input.Description,
	}
	if err := label.ValidateColor(); err != nil {
		return nil, apperror.ErrValidation
	}

	if err := uc.labelRepo.Create(ctx, label); err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.ErrConflict
		}
		return nil, err
	}

	return label, nil
}

func isUniqueViolation(err error) bool {
	if errors.Is(err, apperror.ErrConflict) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "23505") || strings.Contains(msg, "unique")
}
