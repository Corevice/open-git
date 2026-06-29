package git

import (
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func TestAutoInitRepositoryCreatesInitialCommitOnMain(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bareRepoPath := filepath.Join(dir, "test.git")

	if err := AutoInitRepository(bareRepoPath, AutoInitOpts{Readme: "test-repo"}); err != nil {
		t.Fatalf("AutoInitRepository: %v", err)
	}

	repo, err := gogit.PlainOpen(bareRepoPath)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}

	ref, err := repo.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		t.Fatalf("reference main: %v", err)
	}
	if ref.Hash().IsZero() {
		t.Fatal("expected non-zero commit hash on main")
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	if commit.NumParents() != 0 {
		t.Fatalf("expected root commit, got %d parents", commit.NumParents())
	}
}

func TestAutoInitRepositoryWithReadmeContainsREADME(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bareRepoPath := filepath.Join(dir, "readme.git")

	if err := AutoInitRepository(bareRepoPath, AutoInitOpts{Readme: "my-project"}); err != nil {
		t.Fatalf("AutoInitRepository: %v", err)
	}

	repo, err := gogit.PlainOpen(bareRepoPath)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}

	ref, err := repo.Reference(plumbing.NewBranchReferenceName("main"), true)
	if err != nil {
		t.Fatalf("reference main: %v", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}

	entry, err := tree.FindEntry("README.md")
	if err != nil {
		t.Fatalf("FindEntry README.md: %v", err)
	}
	if entry.Name != "README.md" {
		t.Fatalf("expected README.md, got %q", entry.Name)
	}
}
