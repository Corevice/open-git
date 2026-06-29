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

	gogit "github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
	"github.com/open-git/backend/internal/validator"
)

var (
	ErrDuplicateName     = errors.New("repository name already exists")
	ErrInvalidName       = errors.New("invalid repository name")
	ErrInvalidUserUUID   = errors.New("invalid user uuid")
)

var repoNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,100}$`)

type CreateRepositoryInput struct {
	OwnerID        uuid.UUID
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

	ownerID, err := userUUIDToInt64(input.OwnerID)
	if err != nil {
		return nil, errors.New("resolve owner: user not found")
	}
	owner, err := u.users.GetByID(ctx, ownerID)
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
	orgID := input.OrganizationID
	if orgID == uuid.Nil {
		orgID = int64ToUserUUID(owner.ID)
	}

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
		return nil, u.joinWithRollback(err, "build disk path", u.rollbackCreate(repository.ID, u.gitRoot, "", repository.Name))
	}

	if err := initBareRepository(diskPath); err != nil {
		return nil, u.joinWithRollback(err, "init bare repository", u.rollbackCreate(repository.ID, u.gitRoot, diskPath, repository.Name))
	}

	if err := u.repos.UpdateDiskPath(ctx, repository.ID, diskPath); err != nil {
		return nil, u.joinWithRollback(err, "update disk path", u.rollbackCreate(repository.ID, u.gitRoot, diskPath, repository.Name))
	}

	repository.DiskPath = diskPath
	return &CreateRepositoryResult{
		Repository: repository,
		OwnerLogin: ownerLogin,
	}, nil
}

func initBareRepository(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_, err := gogit.PlainInit(path, true)
	return err
}

func isUserUUID(id uuid.UUID) bool {
	for i := 0; i < 8; i++ {
		if id[i] != 0 {
			return false
		}
	}
	return true
}

func userUUIDToInt64(id uuid.UUID) (int64, error) {
	if id == uuid.Nil {
		return 0, ErrInvalidUserUUID
	}
	if !isUserUUID(id) {
		return 0, ErrInvalidUserUUID
	}
	return int64(binary.BigEndian.Uint64(id[8:])), nil
}

func int64ToUserUUID(id int64) uuid.UUID {
	if id == 0 {
		return uuid.Nil
	}
	var u uuid.UUID
	binary.BigEndian.PutUint64(u[8:], uint64(id))
	return u
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

func (u *CreateRepositoryUsecase) rollbackCreate(repositoryID uuid.UUID, gitRoot, diskPath, repoName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var rollbackErr error
	if err := removeRepositoryDiskDir(gitRoot, diskPath, repoName); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove bare repository: %w", err))
	}
	if err := u.repos.Delete(ctx, repositoryID); err != nil {
		rollbackErr = errors.Join(rollbackErr, fmt.Errorf("delete repository record: %w", err))
	}
	return rollbackErr
}

func removeRepositoryDiskDir(gitRoot, diskPath, repoName string) error {
	safePath, ok := validateRepositoryDiskPath(gitRoot, diskPath, repoName)
	if !ok {
		return nil
	}

	fi, err := os.Lstat(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe repository disk path")
	}

	resolvedRoot, err := filepath.EvalSymlinks(filepath.Clean(gitRoot))
	if err != nil {
		return err
	}
	resolvedPath, err := filepath.EvalSymlinks(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	rel, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return errors.New("unsafe repository disk path")
	}
	if filepath.Base(resolvedPath) != repoName+".git" {
		return errors.New("unsafe repository disk path")
	}

	return os.RemoveAll(resolvedPath)
}

func validateRepositoryDiskPath(gitRoot, diskPath, repoName string) (string, bool) {
	if diskPath == "" || repoName == "" || gitRoot == "" {
		return "", false
	}

	cleanRoot := filepath.Clean(gitRoot)
	cleanPath := filepath.Clean(diskPath)
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", false
	}
	if filepath.Base(cleanPath) != repoName+".git" {
		return "", false
	}
	return cleanPath, true
}

func (u *CreateRepositoryUsecase) joinWithRollback(err error, step string, rollbackErr error) error {
	if rollbackErr != nil {
		return fmt.Errorf("%s: %w; rollback failed: %v", step, err, rollbackErr)
	}
	return fmt.Errorf("%s: %w", step, err)
}
