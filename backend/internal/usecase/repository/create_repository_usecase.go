package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	gitinfra "github.com/open-git/backend/internal/infrastructure/git"
	repo "github.com/open-git/backend/internal/repository"
)

var (
	ErrDuplicateName        = errors.New("repository name already exists")
	ErrInvalidName          = errors.New("invalid repository name")
	ErrOwnerLoginRequired   = errors.New("owner login is required for auto init")
	ErrGitDataRootNotConfig = errors.New("git data root is not configured")
)

var repoNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,100}$`)

type GitInitService interface {
	AutoInitRepository(bareRepoPath string, opts gitinfra.AutoInitOpts) error
	// InitBare creates an empty bare repository so a freshly-created
	// (non-auto-init) repo can accept its first push.
	InitBare(path string) error
}

type defaultGitInitService struct{}

func (defaultGitInitService) AutoInitRepository(bareRepoPath string, opts gitinfra.AutoInitOpts) error {
	return gitinfra.AutoInitRepository(bareRepoPath, opts)
}

func (defaultGitInitService) InitBare(path string) error {
	return gitinfra.InitBare(path)
}

type CreateRepositoryInput struct {
	OwnerID           uuid.UUID
	OrganizationID    uuid.UUID
	Name              string
	Private           bool
	Description       string
	AutoInit          bool
	GitIgnoreTemplate string
	LicenseTemplate   string
	GitDataRoot       string
	OwnerLogin        string
}

type ownerLoginResolver interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
}

type createRepositoryUsecaseOptions struct {
	gitDataRoot string
	gitInit     GitInitService
	users       ownerLoginResolver
}

type CreateRepositoryUsecaseOption func(*createRepositoryUsecaseOptions)

func WithGitDataRoot(gitDataRoot string) CreateRepositoryUsecaseOption {
	return func(o *createRepositoryUsecaseOptions) {
		o.gitDataRoot = gitDataRoot
	}
}

func WithGitInitService(gitInit GitInitService) CreateRepositoryUsecaseOption {
	return func(o *createRepositoryUsecaseOptions) {
		o.gitInit = gitInit
	}
}

func WithOwnerLoginResolver(users ownerLoginResolver) CreateRepositoryUsecaseOption {
	return func(o *createRepositoryUsecaseOptions) {
		o.users = users
	}
}

type CreateRepositoryUsecase struct {
	repos       repo.IRepositoryRepository
	gitDataRoot string
	gitInit     GitInitService
	users       ownerLoginResolver
}

func NewCreateRepositoryUsecase(repos repo.IRepositoryRepository, opts ...CreateRepositoryUsecaseOption) *CreateRepositoryUsecase {
	cfg := &createRepositoryUsecaseOptions{}
	for _, opt := range opts {
		opt(cfg)
	}
	gitInit := cfg.gitInit
	if gitInit == nil {
		gitInit = defaultGitInitService{}
	}
	return &CreateRepositoryUsecase{
		repos:       repos,
		gitDataRoot: cfg.gitDataRoot,
		gitInit:     gitInit,
		users:       cfg.users,
	}
}

func isSafePathSegment(s string) bool {
	return s != "" &&
		!strings.Contains(s, "/") &&
		!strings.Contains(s, "\\") &&
		!strings.Contains(s, "..")
}

func resolveGitPath(gitDataRoot, ownerLogin, name string) (string, error) {
	if !isSafePathSegment(ownerLogin) || !isSafePathSegment(name) {
		return "", ErrInvalidName
	}

	gitPath := filepath.Join(gitDataRoot, ownerLogin, name+".git")
	absPath, err := filepath.Abs(gitPath)
	if err != nil {
		return "", fmt.Errorf("resolve git path: %w", err)
	}
	absRoot, err := filepath.Abs(gitDataRoot)
	if err != nil {
		return "", fmt.Errorf("resolve git data root: %w", err)
	}

	rootPrefix := absRoot + string(os.PathSeparator)
	if absPath != absRoot && !strings.HasPrefix(absPath, rootPrefix) {
		return "", ErrInvalidName
	}

	return gitPath, nil
}

func (u *CreateRepositoryUsecase) Execute(ctx context.Context, input CreateRepositoryInput) (*entity.Repository, error) {
	if !repoNameRegex.MatchString(input.Name) || strings.Contains(input.Name, "..") {
		return nil, ErrInvalidName
	}

	// GetByOwnerAndName returns (nil, nil) when no repository exists (the
	// contract the sqlx implementation and other callers rely on). Only treat
	// it as a duplicate when an existing repository is actually returned;
	// checking err == nil alone made every create fail with ErrDuplicateName.
	if existing, err := u.repos.GetByOwnerAndName(ctx, input.OwnerID, input.Name); err == nil && existing != nil {
		return nil, ErrDuplicateName
	}

	gitDataRoot := u.gitDataRoot
	if gitDataRoot == "" {
		return nil, ErrGitDataRootNotConfig
	}

	// Resolve the owner login for EVERY repository, not just auto-init ones.
	// It is needed to compute the on-disk git path, to render full_name and
	// clone URLs, and because git push/clone resolve repositories by owner
	// login. Previously this only happened in the AutoInit branch, so plain
	// repos were stored with an empty owner_login/git_path and were unreachable
	// over git (every push 404'd).
	ownerLogin := input.OwnerLogin
	if ownerLogin == "" && u.users != nil {
		user, err := u.users.GetByID(ctx, input.OwnerID)
		if err != nil {
			return nil, err
		}
		if user != nil {
			ownerLogin = user.Login
		}
	}
	if ownerLogin == "" {
		return nil, ErrOwnerLoginRequired
	}

	gitPath, err := resolveGitPath(gitDataRoot, ownerLogin, input.Name)
	if err != nil {
		return nil, err
	}

	visibility := entity.VisibilityPublic
	if input.Private {
		visibility = entity.VisibilityPrivate
	}

	repository := &entity.Repository{
		OrganizationID: input.OrganizationID,
		OwnerID:        input.OwnerID,
		Name:           input.Name,
		Description:    input.Description,
		OwnerLogin:     ownerLogin,
		GitPath:        gitPath,
		Visibility:     visibility,
		DefaultBranch:  "main",
		CreatedAt:      time.Now().UTC(),
	}

	if err := u.repos.Create(ctx, repository); err != nil {
		return nil, err
	}

	if input.AutoInit {
		if err := u.gitInit.AutoInitRepository(gitPath, gitinfra.AutoInitOpts{
			Readme:            input.Name,
			GitIgnoreTemplate: input.GitIgnoreTemplate,
			LicenseTemplate:   input.LicenseTemplate,
		}); err != nil {
			_ = u.repos.Delete(ctx, repository.ID)
			_ = os.RemoveAll(gitPath)
			return nil, fmt.Errorf("auto init repository: %w", err)
		}
	} else {
		// Create an empty bare repository so the new repo can accept its first
		// push immediately (matching GitHub's "empty repository" behaviour).
		if err := u.gitInit.InitBare(gitPath); err != nil {
			_ = u.repos.Delete(ctx, repository.ID)
			_ = os.RemoveAll(gitPath)
			return nil, fmt.Errorf("init bare repository: %w", err)
		}
	}

	return repository, nil
}
