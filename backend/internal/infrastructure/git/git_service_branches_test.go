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
)

func initBareRepoWithMain(t *testing.T) string {
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

	obj := repo.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(4)
	w, err := obj.Writer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("test")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	blobHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		t.Fatal(err)
	}

	tree := &object.Tree{
		Entries: []object.TreeEntry{
			{Name: "README", Mode: filemode.Regular, Hash: blobHash},
		},
	}
	treeObj := repo.Storer.NewEncodedObject()
	if err := tree.Encode(treeObj); err != nil {
		t.Fatal(err)
	}
	treeHash, err := repo.Storer.SetEncodedObject(treeObj)
	if err != nil {
		t.Fatal(err)
	}

	commit := &object.Commit{
		Message:  "init",
		TreeHash: treeHash,
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
		t.Fatal(err)
	}
	commitHash, err := repo.Storer.SetEncodedObject(commitObj)
	if err != nil {
		t.Fatal(err)
	}

	mainRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), commitHash)
	if err := repo.Storer.SetReference(mainRef); err != nil {
		t.Fatal(err)
	}
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := repo.Storer.SetReference(headRef); err != nil {
		t.Fatal(err)
	}

	return repoPath
}

func TestGetBranches_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "test.git")
	if err := InitBare(repoPath); err != nil {
		t.Fatalf("InitBare: %v", err)
	}

	branches, err := GetBranches(repoPath)
	if err != nil {
		t.Fatalf("GetBranches: %v", err)
	}
	if len(branches) != 0 {
		t.Fatalf("expected empty branches, got %v", branches)
	}
}

func TestCreateBranch_RejectsIfExists(t *testing.T) {
	repoPath := initBareRepoWithMain(t)

	if err := CreateBranch(repoPath, "feature", "main"); err != nil {
		t.Fatalf("first CreateBranch: %v", err)
	}

	err := CreateBranch(repoPath, "feature", "main")
	if !errors.Is(err, ErrRefAlreadyExists) {
		t.Fatalf("expected ErrRefAlreadyExists, got %v", err)
	}
}

func TestDeleteBranch_NotFound(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "test.git")
	if err := InitBare(repoPath); err != nil {
		t.Fatalf("InitBare: %v", err)
	}

	err := DeleteBranch(repoPath, "missing")
	if !errors.Is(err, ErrPathNotFound) {
		t.Fatalf("expected ErrPathNotFound, got %v", err)
	}
}

func TestGetTags_Empty(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "test.git")
	if err := InitBare(repoPath); err != nil {
		t.Fatalf("InitBare: %v", err)
	}

	tags, err := GetTags(repoPath)
	if err != nil {
		t.Fatalf("GetTags: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("expected empty tags, got %v", tags)
	}
}

func TestGetDiff_NoChanges(t *testing.T) {
	repoPath := initBareRepoWithMain(t)

	diffs, err := GetDiff(repoPath, "main", "main")
	if err != nil {
		t.Fatalf("GetDiff: %v", err)
	}
	if len(diffs) != 0 {
		t.Fatalf("expected empty diff, got %d entries", len(diffs))
	}
}
