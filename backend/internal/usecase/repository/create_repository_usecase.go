package repository

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	infragit "github.com/open-git/backend/internal/infrastructure/git"
	repo "github.com/open-git/backend/internal/repository"
)

var (
	ErrDuplicateName = errors.New("repository name already exists")
	ErrInvalidName   = errors.New("invalid repository name")
)

var repoNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,100}$`)

type CreateRepositoryInput struct {
	OwnerID        uuid.UUID
	OwnerLogin     string
	OrganizationID uuid.UUID
	Name           string
	Private        bool
	Description    string
}

type CreateRepositoryUsecase struct {
	repos   repo.IRepositoryRepository
	gitRoot string
}

func NewCreateRepositoryUsecase(repos repo.IRepositoryRepository, gitRoot string) *CreateRepositoryUsecase {
	return &CreateRepositoryUsecase{repos: repos, gitRoot: gitRoot}
}

func (u *CreateRepositoryUsecase) Execute(ctx context.Context, input CreateRepositoryInput) (*entity.Repository, error) {
	if !repoNameRegex.MatchString(input.Name) {
		return nil, ErrInvalidName
	}

	if _, err := u.repos.GetByOwnerAndName(ctx, input.OwnerID, input.Name); err == nil {
		return nil, ErrDuplicateName
	}

	visibility := entity.VisibilityPublic
	if input.Private {
		visibility = entity.VisibilityPrivate
	}

	repository := &entity.Repository{
		OrganizationID: input.OrganizationID,
		OwnerID:        input.OwnerID,
		Name:           input.Name,
		Visibility:     visibility,
		DefaultBranch:  "main",
		CreatedAt:      time.Now().UTC(),
	}

	if err := u.repos.Create(ctx, repository); err != nil {
		return nil, err
	}

	diskPath := filepath.Join(u.gitRoot, input.OwnerLogin, repository.Name+".git")
	if err := infragit.InitBare(diskPath); err != nil {
		_ = u.repos.Delete(ctx, repository.ID)
		return nil, err
	}

	if err := u.repos.UpdateDiskPath(ctx, repository.ID, diskPath); err != nil {
		_ = u.repos.Delete(ctx, repository.ID)
		return nil, err
	}

	repository.DiskPath = diskPath
	return repository, nil
}
