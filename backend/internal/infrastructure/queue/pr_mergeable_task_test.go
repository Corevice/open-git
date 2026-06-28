package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/git"
)

type mockPRRepo struct {
	pr      *entity.PullRequest
	updated *entity.PullRequest
}

func (m *mockPRRepo) Create(_ context.Context, _ *entity.PullRequest) error { return nil }

func (m *mockPRRepo) GetByNumber(_ context.Context, _ uuid.UUID, _ int) (*entity.PullRequest, error) {
	return nil, nil
}

func (m *mockPRRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.PullRequest, error) {
	if m.pr == nil || m.pr.ID != id {
		return nil, fmt.Errorf("pull request not found")
	}
	return m.pr, nil
}

func (m *mockPRRepo) ListByRepo(_ context.Context, _ uuid.UUID, _ domainrepo.ListPullRequestsFilter) ([]*entity.PullRequest, int, error) {
	return nil, 0, nil
}

func (m *mockPRRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }

func (m *mockPRRepo) Update(_ context.Context, pr *entity.PullRequest) error {
	m.updated = pr
	return nil
}

func (m *mockPRRepo) SetMerged(_ context.Context, _ uuid.UUID, _ time.Time, _ uuid.UUID, _ string) error {
	return nil
}

type mockRepositoryLookup struct {
	repo *entity.Repository
}

func (m *mockRepositoryLookup) GetByID(_ context.Context, repositoryID, organizationID uuid.UUID) (*entity.Repository, error) {
	if m.repo == nil || m.repo.ID != repositoryID || m.repo.OrganizationID != organizationID {
		return nil, fmt.Errorf("repository not found")
	}
	return m.repo, nil
}

func TestEnqueuePRMergeableCheck(t *testing.T) {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"})
	defer func() { _ = client.Close() }()

	prID := uuid.New()
	payload := PRMergeableCheckPayload{
		GitPath: "/tmp/repo.git",
		HeadRef: "feature",
		BaseRef: "main",
		PRID:    prID,
	}

	info, err := EnqueuePRMergeableCheck(context.Background(), client, payload)
	if err != nil {
		t.Skipf("redis unavailable: %v", err)
	}
	if info == nil {
		t.Fatal("expected task info")
	}
	if info.Type != TypePRMergeableCheck {
		t.Fatalf("expected task type %q, got %q", TypePRMergeableCheck, info.Type)
	}

	var decoded PRMergeableCheckPayload
	if err := json.Unmarshal(info.Payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if decoded.GitPath != payload.GitPath {
		t.Fatalf("expected git path %q, got %q", payload.GitPath, decoded.GitPath)
	}
	if decoded.HeadRef != payload.HeadRef {
		t.Fatalf("expected head ref %q, got %q", payload.HeadRef, decoded.HeadRef)
	}
	if decoded.BaseRef != payload.BaseRef {
		t.Fatalf("expected base ref %q, got %q", payload.BaseRef, decoded.BaseRef)
	}
	if decoded.PRID != payload.PRID {
		t.Fatalf("expected pr id %s, got %s", payload.PRID, decoded.PRID)
	}
}

func TestHandlePRMergeableCheckClean(t *testing.T) {
	repoPath := initBareRepoWithDivergentBranches(t)
	orgID := uuid.New()
	repoID := uuid.New()
	prID := uuid.New()
	pr := &entity.PullRequest{
		ID:             prID,
		OrganizationID: orgID,
		RepositoryID:   repoID,
		HeadRef:        "feature",
		BaseRef:        "main",
		MergeableState: entity.MergeableStateUnknown,
	}

	prRepo := &mockPRRepo{pr: pr}
	repoLookup := &mockRepositoryLookup{repo: &entity.Repository{
		ID:             repoID,
		OrganizationID: orgID,
		GitPath:        repoPath,
	}}
	worker := NewPRMergeableWorker(prRepo, repoLookup, git.NewGitServiceAdapter())

	task, err := newPRMergeableTask(PRMergeableCheckPayload{
		GitPath: repoPath,
		HeadRef: "feature",
		BaseRef: "main",
		PRID:    prID,
	})
	if err != nil {
		t.Fatalf("newPRMergeableTask: %v", err)
	}

	if err := worker.HandlePRMergeableCheck(context.Background(), task); err != nil {
		t.Fatalf("HandlePRMergeableCheck: %v", err)
	}

	if prRepo.updated == nil {
		t.Fatal("expected pull request update")
	}
	if prRepo.updated.MergeableState != entity.MergeableStateClean {
		t.Fatalf("expected mergeable state %q, got %q", entity.MergeableStateClean, prRepo.updated.MergeableState)
	}
	if prRepo.updated.Mergeable == nil || !*prRepo.updated.Mergeable {
		t.Fatalf("expected mergeable true, got %v", prRepo.updated.Mergeable)
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	mainRef, err := repo.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		t.Fatalf("main ref: %v", err)
	}
	featureRef, err := repo.Reference(plumbing.NewBranchReferenceName("feature"), true)
	if err != nil {
		t.Fatalf("feature ref: %v", err)
	}
	if mainRef.Hash() == featureRef.Hash() {
		t.Fatal("expected main branch to remain unchanged after mergeable simulation")
	}
}

func TestHandlePRMergeableCheckDirty(t *testing.T) {
	repoPath := initBareRepoWithConflictingBranches(t)
	orgID := uuid.New()
	repoID := uuid.New()
	prID := uuid.New()
	pr := &entity.PullRequest{
		ID:             prID,
		OrganizationID: orgID,
		RepositoryID:   repoID,
		HeadRef:        "feature",
		BaseRef:        "main",
		MergeableState: entity.MergeableStateUnknown,
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	mainRefBefore, err := repo.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		t.Fatalf("main ref before: %v", err)
	}
	mainSHA := mainRefBefore.Hash()

	prRepo := &mockPRRepo{pr: pr}
	repoLookup := &mockRepositoryLookup{repo: &entity.Repository{
		ID:             repoID,
		OrganizationID: orgID,
		GitPath:        repoPath,
	}}
	worker := NewPRMergeableWorker(prRepo, repoLookup, git.NewGitServiceAdapter())

	task, err := newPRMergeableTask(PRMergeableCheckPayload{
		GitPath: repoPath,
		HeadRef: "feature",
		BaseRef: "main",
		PRID:    prID,
	})
	if err != nil {
		t.Fatalf("newPRMergeableTask: %v", err)
	}

	if err := worker.HandlePRMergeableCheck(context.Background(), task); err != nil {
		t.Fatalf("HandlePRMergeableCheck: %v", err)
	}

	if prRepo.updated == nil {
		t.Fatal("expected pull request update")
	}
	if prRepo.updated.MergeableState != entity.MergeableStateDirty {
		t.Fatalf("expected mergeable state %q, got %q", entity.MergeableStateDirty, prRepo.updated.MergeableState)
	}
	if prRepo.updated.Mergeable == nil || *prRepo.updated.Mergeable {
		t.Fatalf("expected mergeable false, got %v", prRepo.updated.Mergeable)
	}

	mainRefAfter, err := repo.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		t.Fatalf("main ref after: %v", err)
	}
	if mainRefAfter.Hash() != mainSHA {
		t.Fatalf("expected main ref %s to be restored, got %s", mainSHA, mainRefAfter.Hash())
	}
}

func newPRMergeableTask(payload PRMergeableCheckPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypePRMergeableCheck, data), nil
}

func initBareRepoWithDivergentBranches(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	repoPath := filepath.Join(dir, "test.git")
	if err := git.InitBare(repoPath); err != nil {
		t.Fatalf("InitBare: %v", err)
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}

	initHash, err := storeBlobCommit(repo, "init", map[string]string{"README": "base"}, nil)
	if err != nil {
		t.Fatalf("store init commit: %v", err)
	}

	mainRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), initHash)
	if err := repo.Storer.SetReference(mainRef); err != nil {
		t.Fatalf("set main ref: %v", err)
	}
	featureRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName("feature"), initHash)
	if err := repo.Storer.SetReference(featureRef); err != nil {
		t.Fatalf("set feature ref: %v", err)
	}

	mainHash, err := storeBlobCommit(repo, "main change", map[string]string{"README": "base", "main.txt": "main"}, []plumbing.Hash{initHash})
	if err != nil {
		t.Fatalf("store main commit: %v", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), mainHash)); err != nil {
		t.Fatalf("update main ref: %v", err)
	}

	featureHash, err := storeBlobCommit(repo, "feature change", map[string]string{"README": "base", "feature.txt": "feature"}, []plumbing.Hash{initHash})
	if err != nil {
		t.Fatalf("store feature commit: %v", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("feature"), featureHash)); err != nil {
		t.Fatalf("update feature ref: %v", err)
	}

	return repoPath
}

func initBareRepoWithConflictingBranches(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	repoPath := filepath.Join(dir, "test.git")
	if err := git.InitBare(repoPath); err != nil {
		t.Fatalf("InitBare: %v", err)
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}

	initHash, err := storeBlobCommit(repo, "init", map[string]string{"README": "base"}, nil)
	if err != nil {
		t.Fatalf("store init commit: %v", err)
	}

	mainRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), initHash)
	if err := repo.Storer.SetReference(mainRef); err != nil {
		t.Fatalf("set main ref: %v", err)
	}
	featureRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName("feature"), initHash)
	if err := repo.Storer.SetReference(featureRef); err != nil {
		t.Fatalf("set feature ref: %v", err)
	}

	mainHash, err := storeBlobCommit(repo, "main change", map[string]string{"README": "main version"}, []plumbing.Hash{initHash})
	if err != nil {
		t.Fatalf("store main commit: %v", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), mainHash)); err != nil {
		t.Fatalf("update main ref: %v", err)
	}

	featureHash, err := storeBlobCommit(repo, "feature change", map[string]string{"README": "feature version"}, []plumbing.Hash{initHash})
	if err != nil {
		t.Fatalf("store feature commit: %v", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("feature"), featureHash)); err != nil {
		t.Fatalf("update feature ref: %v", err)
	}

	return repoPath
}

func storeBlobCommit(repo *gogit.Repository, message string, files map[string]string, parents []plumbing.Hash) (plumbing.Hash, error) {
	entries := make([]object.TreeEntry, 0, len(files))
	for name, content := range files {
		obj := repo.Storer.NewEncodedObject()
		obj.SetType(plumbing.BlobObject)
		obj.SetSize(int64(len(content)))
		w, err := obj.Writer()
		if err != nil {
			return plumbing.ZeroHash, err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return plumbing.ZeroHash, err
		}
		if err := w.Close(); err != nil {
			return plumbing.ZeroHash, err
		}
		blobHash, err := repo.Storer.SetEncodedObject(obj)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		entries = append(entries, object.TreeEntry{
			Name: name,
			Mode: filemode.Regular,
			Hash: blobHash,
		})
	}
	sortTreeEntries(entries)

	tree := &object.Tree{Entries: entries}
	treeObj := repo.Storer.NewEncodedObject()
	if err := tree.Encode(treeObj); err != nil {
		return plumbing.ZeroHash, err
	}
	treeHash, err := repo.Storer.SetEncodedObject(treeObj)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	commit := &object.Commit{
		Message:      message,
		TreeHash:     treeHash,
		ParentHashes: parents,
		Author: object.Signature{
			Name:  "test",
			Email: "test@test.com",
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  "test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	}
	commitObj := repo.Storer.NewEncodedObject()
	if err := commit.Encode(commitObj); err != nil {
		return plumbing.ZeroHash, err
	}
	return repo.Storer.SetEncodedObject(commitObj)
}

func sortTreeEntries(entries []object.TreeEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
}
