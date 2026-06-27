package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
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

type CreateRepositoryResult struct {
	Repository *entity.Repository
	OwnerLogin string
}

type CreateRepositoryUsecase struct {
	repos   repo.IRepositoryRepository
	users   repo.IUserRepository
	gitRoot string
}

func NewCreateRepositoryUsecase(
	repos repo.IRepositoryRepository,
	users repo.IUserRepository,
	gitRoot string,
) *CreateRepositoryUsecase {
	return &CreateRepositoryUsecase{repos: repos, users: users, gitRoot: gitRoot}
}

func (u *CreateRepositoryUsecase) Execute(ctx context.Context, input CreateRepositoryInput) (*CreateRepositoryResult, error) {
	if !repoNameRegex.MatchString(input.Name) {
		return nil, ErrInvalidName
	}

	ownerLogin, err := u.resolveOwnerLogin(ctx, input)
	if err != nil {
		return nil, err
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

	diskPath := filepath.Join(u.gitRoot, ownerLogin, repository.Name+".git")
	if err := infragit.InitBare(diskPath); err != nil {
		return nil, errors.Join(err, u.rollbackCreate(ctx, repository.ID, diskPath))
	}

	if err := u.repos.UpdateDiskPath(ctx, repository.ID, diskPath); err != nil {
		return nil, errors.Join(err, u.rollbackCreate(ctx, repository.ID, diskPath))
	}

	repository.DiskPath = diskPath
	return &CreateRepositoryResult{
		Repository: repository,
		OwnerLogin: ownerLogin,
	}, nil
}

func (u *CreateRepositoryUsecase) resolveOwnerLogin(ctx context.Context, input CreateRepositoryInput) (string, error) {
	if input.OwnerLogin != "" {
		return input.OwnerLogin, nil
	}

	user, err := u.users.GetByID(ctx, int64FromUUID(input.OwnerID))
	if err != nil {
		return "", fmt.Errorf("resolve owner login: %w", err)
	}
	if user == nil || user.Login == "" {
		return "", errors.New("resolve owner login: user not found")
	}
	return user.Login, nil
}

func (u *CreateRepositoryUsecase) rollbackCreate(ctx context.Context, repositoryID uuid.UUID, diskPath string) error {
	var rollbackErr error
	if diskPath != "" {
		if err := os.RemoveAll(diskPath); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove bare repository: %w", err))
		}
	}
	if err := u.repos.Delete(ctx, repositoryID); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("delete repository record: %w", err))
	}
	return rollbackErr
}
