package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	domainservice "github.com/open-git/backend/internal/domain/service"
)

const TypePRMergeableCheck = "pr:mergeable_check"

type PRMergeableCheckPayload struct {
	GitPath string    `json:"git_path"`
	HeadRef string    `json:"head_ref"`
	BaseRef string    `json:"base_ref"`
	PRID    uuid.UUID `json:"pr_id"`
}

func EnqueuePRMergeableCheck(ctx context.Context, client *asynq.Client, payload PRMergeableCheckPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal pr mergeable check payload: %w", err)
	}
	task := asynq.NewTask(TypePRMergeableCheck, data)
	return client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
}

type repositoryLookup interface {
	GetByID(ctx context.Context, repositoryID, organizationID uuid.UUID) (*entity.Repository, error)
}

type PRMergeableWorker struct {
	prRepo   domainrepo.IPullRequestRepository
	repoRepo repositoryLookup
	gitSvc   domainservice.GitService
}

func NewPRMergeableWorker(
	prRepo domainrepo.IPullRequestRepository,
	repoRepo repositoryLookup,
	gitSvc domainservice.GitService,
) *PRMergeableWorker {
	return &PRMergeableWorker{
		prRepo:   prRepo,
		repoRepo: repoRepo,
		gitSvc:   gitSvc,
	}
}

func (w *PRMergeableWorker) HandlePRMergeableCheck(ctx context.Context, task *asynq.Task) error {
	var payload PRMergeableCheckPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal pr mergeable check payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.GitPath == "" || payload.HeadRef == "" || payload.BaseRef == "" || payload.PRID == uuid.Nil {
		return fmt.Errorf("pr mergeable check payload missing fields: %w", asynq.SkipRetry)
	}
	if err := validateGitPath(payload.GitPath); err != nil {
		return fmt.Errorf("validate git path: %w: %w", err, asynq.SkipRetry)
	}
	if err := validateBranchRef(payload.HeadRef); err != nil {
		return fmt.Errorf("validate head ref: %w: %w", err, asynq.SkipRetry)
	}
	if err := validateBranchRef(payload.BaseRef); err != nil {
		return fmt.Errorf("validate base ref: %w: %w", err, asynq.SkipRetry)
	}

	pr, err := w.prRepo.GetByID(ctx, payload.PRID)
	if err != nil {
		return fmt.Errorf("load pull request: %w", err)
	}
	if payload.HeadRef != pr.HeadRef || payload.BaseRef != pr.BaseRef {
		return fmt.Errorf("payload refs do not match pull request: %w", asynq.SkipRetry)
	}

	repo, err := w.repoRepo.GetByID(ctx, pr.RepositoryID, pr.OrganizationID)
	if err != nil {
		return fmt.Errorf("load repository: %w", err)
	}
	if repo == nil || repo.GitPath != payload.GitPath {
		return fmt.Errorf("git path does not match pull request repository: %w", asynq.SkipRetry)
	}

	state, mergeable, err := determineMergeableState(ctx, w.gitSvc, payload.GitPath, payload.BaseRef, payload.HeadRef)
	if err != nil {
		return fmt.Errorf("determine mergeable state: %w", err)
	}

	pr.MergeableState = state
	pr.Mergeable = mergeable
	if err := w.prRepo.Update(ctx, pr); err != nil {
		return fmt.Errorf("update pull request mergeable state: %w", err)
	}

	return nil
}

func determineMergeableState(ctx context.Context, gitSvc domainservice.GitService, gitPath, baseRef, headRef string) (string, *bool, error) {
	baseSHA, err := gitSvc.ResolveRef(ctx, gitPath, baseRef)
	if err != nil {
		return "", nil, fmt.Errorf("resolve base ref: %w", err)
	}

	_, err = gitSvc.Merge(ctx, gitPath, baseRef, headRef, "merge")
	if errors.Is(err, domainservice.ErrMergeConflict) {
		if restoreErr := restoreBranchRef(gitPath, baseRef, baseSHA); restoreErr != nil {
			return "", nil, fmt.Errorf("restore base ref after merge conflict simulation: %w", restoreErr)
		}
		v := false
		return entity.MergeableStateDirty, &v, nil
	}
	if err != nil {
		return "", nil, err
	}

	if err := restoreBranchRef(gitPath, baseRef, baseSHA); err != nil {
		return "", nil, fmt.Errorf("restore base ref after merge simulation: %w", err)
	}

	v := true
	return entity.MergeableStateClean, &v, nil
}

func restoreBranchRef(repoPath, branch, sha string) error {
	if err := validateGitPath(repoPath); err != nil {
		return err
	}
	if err := validateBranchRef(branch); err != nil {
		return err
	}
	if err := validateCommitSHA(sha); err != nil {
		return err
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return err
	}
	defer closeRepository(repo)

	branch = strings.TrimPrefix(branch, "refs/heads/")
	refName := plumbing.NewBranchReferenceName(branch)
	ref := plumbing.NewHashReference(refName, plumbing.NewHash(sha))
	return repo.Storer.SetReference(ref)
}

func validateGitPath(gitPath string) error {
	if gitPath == "" {
		return fmt.Errorf("empty git path")
	}
	if strings.Contains(gitPath, "..") || strings.Contains(gitPath, "\x00") {
		return fmt.Errorf("invalid git path")
	}
	cleaned := filepath.Clean(gitPath)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("invalid git path")
	}
	return nil
}

func validateBranchRef(ref string) error {
	branch := strings.TrimPrefix(ref, "refs/heads/")
	if branch == "" {
		return fmt.Errorf("empty branch ref")
	}
	if strings.Contains(branch, "..") || strings.Contains(branch, "/") || strings.Contains(branch, "\\") || strings.Contains(branch, "\x00") {
		return fmt.Errorf("invalid branch ref")
	}
	return nil
}

func validateCommitSHA(sha string) error {
	if sha == "" {
		return fmt.Errorf("empty commit sha")
	}
	hash := plumbing.NewHash(sha)
	if hash == plumbing.ZeroHash {
		return fmt.Errorf("invalid commit sha")
	}
	return nil
}

func closeRepository(repo *gogit.Repository) {
	if closer, ok := repo.Storer.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}
