package git_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	git "github.com/open-git/backend/internal/infrastructure/git"
)

func makeRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()

	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if _, err := wt.Add(path); err != nil {
			t.Fatalf("add file: %v", err)
		}
	}

	_, err = wt.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	return dir
}

func commitChanges(t *testing.T, repoPath, message string, files map[string]string, deletePaths []string) {
	t.Helper()

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(repoPath, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if _, err := wt.Add(path); err != nil {
			t.Fatalf("add file: %v", err)
		}
	}

	for _, path := range deletePaths {
		if _, err := wt.Remove(path); err != nil {
			t.Fatalf("remove file: %v", err)
		}
	}

	_, err = wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func headSHA(t *testing.T, repoPath string) string {
	t.Helper()

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}

	return head.Hash().String()
}

func findFileDiff(files []git.FileDiff, name string) *git.FileDiff {
	for i := range files {
		if files[i].Filename == name {
			return &files[i]
		}
	}
	return nil
}

func TestGetBranches_HasDefaultBranch(t *testing.T) {
	repoPath := makeRepo(t, map[string]string{"README.md": "hello"})

	branches, err := git.GetBranches(repoPath)
	if err != nil {
		t.Fatalf("GetBranches: %v", err)
	}
	if len(branches) < 1 {
		t.Fatalf("expected at least one branch, got %d", len(branches))
	}
}

func TestGetBranches_EmptyRepo(t *testing.T) {
	dir := t.TempDir()

	_, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	branches, err := git.GetBranches(dir)
	if err != nil {
		t.Fatalf("GetBranches: %v", err)
	}
	if len(branches) != 0 {
		t.Fatalf("expected empty branch list, got %d", len(branches))
	}
}

func TestGetTags_NoTags(t *testing.T) {
	repoPath := makeRepo(t, map[string]string{"README.md": "hello"})

	tags, err := git.GetTags(repoPath)
	if err != nil {
		t.Fatalf("GetTags: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("expected no tags, got %d", len(tags))
	}
}

func TestGetCommitDetail_RootCommit(t *testing.T) {
	repoPath := makeRepo(t, map[string]string{"README.md": "hello"})
	fullSHA := headSHA(t, repoPath)

	detail, err := git.GetCommitDetail(repoPath, fullSHA)
	if err != nil {
		t.Fatalf("GetCommitDetail: %v", err)
	}
	if detail.SHA != fullSHA {
		t.Fatalf("expected SHA %s, got %s", fullSHA, detail.SHA)
	}

	diff := findFileDiff(detail.Files, "README.md")
	if diff == nil {
		t.Fatal("expected README.md in file diffs")
	}
	if diff.Status != "added" {
		t.Fatalf("expected status added, got %s", diff.Status)
	}
}

func TestGetCommitDetail_ShortSHA(t *testing.T) {
	repoPath := makeRepo(t, map[string]string{"README.md": "hello"})
	fullSHA := headSHA(t, repoPath)
	shortSHA := fullSHA[:7]

	detail, err := git.GetCommitDetail(repoPath, shortSHA)
	if err != nil {
		t.Fatalf("GetCommitDetail: %v", err)
	}
	if detail.SHA != fullSHA {
		t.Fatalf("expected SHA %s, got %s", fullSHA, detail.SHA)
	}
}

func TestGetCommitDetail_FileAdded(t *testing.T) {
	repoPath := makeRepo(t, map[string]string{"README.md": "hello"})
	commitChanges(t, repoPath, "add file", map[string]string{"new.txt": "new content"}, nil)

	detail, err := git.GetCommitDetail(repoPath, headSHA(t, repoPath))
	if err != nil {
		t.Fatalf("GetCommitDetail: %v", err)
	}

	diff := findFileDiff(detail.Files, "new.txt")
	if diff == nil {
		t.Fatal("expected new.txt in file diffs")
	}
	if diff.Status != "added" {
		t.Fatalf("expected status added, got %s", diff.Status)
	}
}

func TestGetCommitDetail_FileModified(t *testing.T) {
	repoPath := makeRepo(t, map[string]string{"README.md": "hello"})
	commitChanges(t, repoPath, "modify file", map[string]string{"README.md": "hello world"}, nil)

	detail, err := git.GetCommitDetail(repoPath, headSHA(t, repoPath))
	if err != nil {
		t.Fatalf("GetCommitDetail: %v", err)
	}

	diff := findFileDiff(detail.Files, "README.md")
	if diff == nil {
		t.Fatal("expected README.md in file diffs")
	}
	if diff.Status != "modified" {
		t.Fatalf("expected status modified, got %s", diff.Status)
	}
	if diff.Patch == nil {
		t.Fatal("expected non-nil patch for modified file")
	}
}

func TestGetCommitDetail_FileDeleted(t *testing.T) {
	repoPath := makeRepo(t, map[string]string{"README.md": "hello"})
	commitChanges(t, repoPath, "delete file", nil, []string{"README.md"})

	detail, err := git.GetCommitDetail(repoPath, headSHA(t, repoPath))
	if err != nil {
		t.Fatalf("GetCommitDetail: %v", err)
	}

	diff := findFileDiff(detail.Files, "README.md")
	if diff == nil {
		t.Fatal("expected README.md in file diffs")
	}
	if diff.Status != "deleted" {
		t.Fatalf("expected status deleted, got %s", diff.Status)
	}
}
