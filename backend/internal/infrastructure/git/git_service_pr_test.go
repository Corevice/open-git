package git

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	domainservice "github.com/open-git/backend/internal/domain/service"
)

func initBareRepoWithDivergentBranches(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	repoPath := filepath.Join(dir, "test.git")
	if err := InitBare(repoPath); err != nil {
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
	if err := InitBare(repoPath); err != nil {
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
	treeHash, err := storeTree(repo, tree)
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
	return storeCommit(repo, commit)
}

func TestBranchExists(t *testing.T) {
	repoPath := initBareRepoWithDivergentBranches(t)

	exists, err := BranchExists(repoPath, "main")
	if err != nil {
		t.Fatalf("BranchExists main: %v", err)
	}
	if !exists {
		t.Fatal("expected main branch to exist")
	}

	exists, err = BranchExists(repoPath, "missing")
	if err != nil {
		t.Fatalf("BranchExists missing: %v", err)
	}
	if exists {
		t.Fatal("expected missing branch to not exist")
	}
}

func TestResolveRef(t *testing.T) {
	repoPath := initBareRepoWithDivergentBranches(t)

	sha, err := ResolveRef(repoPath, "feature")
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if sha == "" {
		t.Fatal("expected non-empty SHA")
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	ref, err := repo.Reference(plumbing.NewBranchReferenceName("feature"), true)
	if err != nil {
		t.Fatalf("Reference: %v", err)
	}
	if ref.Hash().String() != sha {
		t.Fatalf("expected SHA %s, got %s", ref.Hash().String(), sha)
	}
}

func TestMergeCreatesMergeCommit(t *testing.T) {
	repoPath := initBareRepoWithDivergentBranches(t)

	mergeSHA, err := Merge(repoPath, "main", "feature", "merge")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if mergeSHA == "" {
		t.Fatal("expected merge SHA")
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}

	mergeCommit, err := repo.CommitObject(plumbing.NewHash(mergeSHA))
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	if len(mergeCommit.ParentHashes) != 2 {
		t.Fatalf("expected merge commit with 2 parents, got %d", len(mergeCommit.ParentHashes))
	}

	mainRef, err := repo.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		t.Fatalf("main ref: %v", err)
	}
	if mainRef.Hash().String() != mergeSHA {
		t.Fatalf("expected main to point to merge commit, got %s", mainRef.Hash().String())
	}
}

func TestGetMergeBase(t *testing.T) {
	repoPath := initBareRepoWithDivergentBranches(t)

	mergeBase, err := GetMergeBase(repoPath, "main", "feature")
	if err != nil {
		t.Fatalf("GetMergeBase: %v", err)
	}
	if mergeBase == "" {
		t.Fatal("expected merge base SHA")
	}
}

func TestMergeDetectsConflict(t *testing.T) {
	repoPath := initBareRepoWithConflictingBranches(t)

	_, err := Merge(repoPath, "main", "feature", "merge")
	if !errors.Is(err, domainservice.ErrMergeConflict) {
		t.Fatalf("expected merge conflict, got %v", err)
	}
}
