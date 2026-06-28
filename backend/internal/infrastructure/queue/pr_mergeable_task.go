package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type PRMergeableWorker struct {
	prRepo domainrepo.IPullRequestRepository
	gitSvc domainservice.GitService
}

func NewPRMergeableWorker(prRepo domainrepo.IPullRequestRepository, gitSvc domainservice.GitService) *PRMergeableWorker {
	return &PRMergeableWorker{
		prRepo: prRepo,
		gitSvc: gitSvc,
	}
}

func (w *PRMergeableWorker) HandlePRMergeableCheck(ctx context.Context, task *asynq.Task) error {
	var payload PRMergeableCheckPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal pr mergeable check payload: %v: %w", err, asynq.SkipRetry)
	}
	if payload.GitPath == "" || payload.HeadRef == "" || payload.BaseRef == "" || payload.PRID == uuid.Nil {
		return fmt.Errorf("pr mergeable check payload missing fields: %w", asynq.SkipRetry)
	}

	pr, err := w.prRepo.GetByID(ctx, payload.PRID)
	if err != nil {
		return fmt.Errorf("load pull request: %w", err)
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
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	branch = strings.TrimPrefix(branch, "refs/heads/")
	refName := plumbing.NewBranchReferenceName(branch)
	ref := plumbing.NewHashReference(refName, plumbing.NewHash(sha))
	return repo.Storer.SetReference(ref)
}
