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
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	infragit "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
	"github.com/open-git/backend/internal/validator"
)

var (
	ErrDuplicateName      = errors.New("repository name already exists")
	ErrInvalidName        = errors.New("invalid repository name")
	ErrOwnerLoginMismatch = errors.New("owner login does not match owner id")
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

	if _, err := u.repos.GetByOwnerAndName(ctx, input.OwnerID, input.Name); err == nil {
		return nil, ErrDuplicateName
	}

	ownerLogin, err := u.resolveOwnerLogin(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := validator.ValidateLogin(ownerLogin); err != nil {
		return nil, fmt.Errorf("resolve owner login: %w", err)
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
		return nil, u.joinWithRollback(err, "init bare repository", u.rollbackCreate(repository.ID, ownerLogin, repository.Name, diskPath))
	}

	if err := u.repos.UpdateDiskPath(ctx, repository.ID, diskPath); err != nil {
		return nil, u.joinWithRollback(err, "update disk path", u.rollbackCreate(repository.ID, ownerLogin, repository.Name, diskPath))
	}

	repository.DiskPath = diskPath
	return &CreateRepositoryResult{
		Repository: repository,
		OwnerLogin: ownerLogin,
	}, nil
}

func (u *CreateRepositoryUsecase) resolveOwnerLogin(ctx context.Context, input CreateRepositoryInput) (string, error) {
	if input.OwnerLogin == "" {
		return "", errors.New("resolve owner login: owner login required")
	}

	user, err := u.users.GetByLogin(ctx, input.OwnerLogin)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", errors.New("resolve owner login: user not found")
		}
		return "", fmt.Errorf("resolve owner login: %w", err)
	}
	if user == nil || user.Login == "" {
		return "", errors.New("resolve owner login: user not found")
	}
	if middleware.Int64ToUUID(user.ID) != input.OwnerID {
		return "", ErrOwnerLoginMismatch
	}
	return user.Login, nil
}

func (u *CreateRepositoryUsecase) rollbackCreate(repositoryID uuid.UUID, ownerLogin, repoName, diskPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var rollbackErr error
	expectedPath := ""
	if validator.ValidateLogin(ownerLogin) == nil && repoNameRegex.MatchString(repoName) {
		expectedPath = filepath.Join(u.gitRoot, ownerLogin, repoName+".git")
	}
	if diskPath != "" && expectedPath != "" && filepath.Clean(diskPath) == filepath.Clean(expectedPath) {
		if err := os.RemoveAll(diskPath); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove bare repository: %w", err))
		}
	}
	if err := u.repos.Delete(ctx, repositoryID); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("delete repository record: %w", err))
	}
	return rollbackErr
}

func (u *CreateRepositoryUsecase) joinWithRollback(err error, step string, rollbackErr error) error {
	if rollbackErr != nil {
		return fmt.Errorf("%s: %w; rollback failed: %v", step, err, rollbackErr)
	}
	return fmt.Errorf("%s: %w", step, err)
}
