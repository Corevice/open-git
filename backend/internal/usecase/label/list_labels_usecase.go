package label

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type ListLabelsInput struct {
	RepositoryID uuid.UUID
	Page         int
	PerPage      int
}

type ListLabelsOutput struct {
	Labels  []*entity.Label
	Total   int
	Page    int
	PerPage int
}

type ListLabelsUsecase struct {
	labelRepo repository.ILabelRepository
}

func NewListLabelsUsecase(labelRepo repository.ILabelRepository) *ListLabelsUsecase {
	return &ListLabelsUsecase{labelRepo: labelRepo}
}

func (uc *ListLabelsUsecase) Execute(ctx context.Context, input ListLabelsInput) (*ListLabelsOutput, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	labels, total, err := uc.labelRepo.ListByRepo(ctx, input.RepositoryID, page, perPage)
	if err != nil {
		return nil, err
	}

	return &ListLabelsOutput{
		Labels:  labels,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}
