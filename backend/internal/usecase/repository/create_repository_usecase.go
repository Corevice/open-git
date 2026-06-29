package repository

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	ErrDuplicateName = errors.New("repository name already exists")
	ErrInvalidName   = errors.New("invalid repository name")
)

var repoNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,100}$`)

type CreateRepositoryInput struct {
	OwnerID     uuid.UUID
	Name        string
	Private     bool
	Description string
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

	owner, err := u.users.GetByID(ctx, uuidToInt64(input.OwnerID))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, errors.New("resolve owner: user not found")
		}
		return nil, fmt.Errorf("resolve owner: %w", err)
	}
	if owner == nil || owner.Login == "" {
		return nil, errors.New("resolve owner: user not found")
	}
	if err := validator.ValidateLogin(owner.Login); err != nil {
		return nil, fmt.Errorf("resolve owner login: %w", err)
	}

	ownerLogin := owner.Login
	orgID := middleware.Int64ToUUID(owner.ID)

	visibility := entity.VisibilityPublic
	if input.Private {
		visibility = entity.VisibilityPrivate
	}

	repository := &entity.Repository{
		OrganizationID: orgID,
		OwnerID:        input.OwnerID,
		Name:           input.Name,
		Visibility:     visibility,
		DefaultBranch:  "main",
		CreatedAt:      time.Now().UTC(),
	}

	if err := u.repos.Create(ctx, repository); err != nil {
		return nil, err
	}

	diskPath, err := buildRepositoryDiskPath(u.gitRoot, ownerLogin, repository.Name)
	if err != nil {
		return nil, u.joinWithRollback(err, "build disk path", u.rollbackCreate(ctx, repository.ID, "", repository.Name))
	}

	if err := infragit.InitBare(diskPath); err != nil {
		return nil, u.joinWithRollback(err, "init bare repository", u.rollbackCreate(ctx, repository.ID, diskPath, repository.Name))
	}

	if err := u.repos.UpdateDiskPath(ctx, repository.ID, diskPath); err != nil {
		return nil, u.joinWithRollback(err, "update disk path", u.rollbackCreate(ctx, repository.ID, diskPath, repository.Name))
	}

	repository.DiskPath = diskPath
	return &CreateRepositoryResult{
		Repository: repository,
		OwnerLogin: ownerLogin,
	}, nil
}

func buildRepositoryDiskPath(gitRoot, ownerLogin, repoName string) (string, error) {
	if err := validator.ValidateLogin(ownerLogin); err != nil {
		return "", fmt.Errorf("invalid owner login: %w", err)
	}
	if !repoNameRegex.MatchString(repoName) {
		return "", ErrInvalidName
	}

	diskPath := filepath.Join(gitRoot, ownerLogin, repoName+".git")
	cleanRoot := filepath.Clean(gitRoot)
	cleanPath := filepath.Clean(diskPath)
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("invalid repository disk path")
	}
	if filepath.Base(cleanPath) != repoName+".git" {
		return "", errors.New("invalid repository disk path")
	}
	return cleanPath, nil
}

func (u *CreateRepositoryUsecase) rollbackCreate(ctx context.Context, repositoryID uuid.UUID, diskPath, repoName string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var rollbackErr error
	if err := removeRepositoryDiskDir(diskPath, repoName); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove bare repository: %w", err))
	}
	if err := u.repos.Delete(ctx, repositoryID); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("delete repository record: %w", err))
	}
	return rollbackErr
}

func removeRepositoryDiskDir(diskPath, repoName string) error {
	if diskPath == "" || repoName == "" {
		return nil
	}

	cleanPath := filepath.Clean(diskPath)
	if filepath.Base(cleanPath) != repoName+".git" {
		return nil
	}

	fi, err := os.Lstat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe repository disk path")
	}

	return os.RemoveAll(cleanPath)
}

func (u *CreateRepositoryUsecase) joinWithRollback(err error, step string, rollbackErr error) error {
	if rollbackErr != nil {
		return fmt.Errorf("%s: %w; rollback failed", step, err)
	}
	return fmt.Errorf("%s: %w", step, err)
}
