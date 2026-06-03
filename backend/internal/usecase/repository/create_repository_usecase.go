package repository

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/Corevice/open-git/backend/internal/domain"
	repo "github.com/Corevice/open-git/backend/internal/repository"
)

var (
	ErrDuplicateName = errors.New("repository name already exists")
	ErrInvalidName   = errors.New("invalid repository name")
)

var repoNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,100}$`)

type CreateRepositoryInput struct {
	OwnerID        int64
	OrganizationID int64
	Name           string
	Private        bool
	Description    string
}

type CreateRepositoryUsecase struct {
	repos repo.IRepositoryRepository
}

func NewCreateRepositoryUsecase(repos repo.IRepositoryRepository) *CreateRepositoryUsecase {
	return &CreateRepositoryUsecase{repos: repos}
}

func (u *CreateRepositoryUsecase) Execute(ctx context.Context, input CreateRepositoryInput) (*domain.Repository, error) {
	if !repoNameRegex.MatchString(input.Name) {
		return nil, ErrInvalidName
	}

	if _, err := u.repos.GetByOwnerAndName(ctx, input.OwnerID, input.Name); err == nil {
		return nil, ErrDuplicateName
	}

	if _, err := u.repos.NextNumber(ctx, input.OwnerID); err != nil {
		return nil, err
	}

	visibility := domain.VisibilityPublic
	if input.Private {
		visibility = domain.VisibilityPrivate
	}

	repository := &domain.Repository{
		OrganizationID: input.OrganizationID,
		OwnerID:        input.OwnerID,
		Name:           input.Name,
		Visibility:     visibility,
		DefaultBranch:  "main",
		Description:    input.Description,
		CreatedAt:      time.Now().UTC(),
	}

	if err := u.repos.Create(ctx, repository); err != nil {
		return nil, err
	}

	return repository, nil
}
