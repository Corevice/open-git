package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
)

type contentStubResolver struct {
	repo *handler.ResolvedGitRepository
}

func (s *contentStubResolver) Resolve(_ context.Context, _, _ string) (*handler.ResolvedGitRepository, error) {
	return s.repo, nil
}

func initRepoWithCommits(t *testing.T, commitCount int) string {
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

	for i := 0; i < commitCount; i++ {
		name := fmt.Sprintf("file%d.txt", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if _, err := wt.Add(name); err != nil {
			t.Fatalf("add file: %v", err)
		}
		if _, err := wt.Commit(fmt.Sprintf("commit %d", i), &gogit.CommitOptions{
			Author: &object.Signature{Name: "Alice", Email: "alice@example.com"},
		}); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}
	return dir
}

const maxBlobTestSize = 1 << 20

func TestGetCommitsPaginationLinkHeader(t *testing.T) {
	repoPath := initRepoWithCommits(t, 25)

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/commits?page=1&per_page=10", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	link := rec.Header().Get("Link")
	if !strings.Contains(link, `rel="next"`) {
		t.Fatalf("Link header missing rel=next: %q", link)
	}
}

func TestBlobTruncation(t *testing.T) {
	dir := t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	largePath := filepath.Join(dir, "large.bin")
	if err := os.WriteFile(largePath, make([]byte, maxBlobTestSize+1024), 0o644); err != nil {
		t.Fatalf("write large file: %v", err)
	}
	if _, err := wt.Add("large.bin"); err != nil {
		t.Fatalf("add large file: %v", err)
	}
	commitHash, err := wt.Commit("large blob", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Alice", Email: "alice@example.com"},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	commit, err := repo.CommitObject(commitHash)
	if err != nil {
		t.Fatalf("commit object: %v", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("tree: %v", err)
	}
	entry, err := tree.FindEntry("large.bin")
	if err != nil {
		t.Fatalf("find entry: %v", err)
	}

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: dir,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/git/blobs/"+entry.Hash.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Truncated bool   `json:"truncated"`
		RawURL    string `json:"raw_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Truncated {
		t.Fatal("expected truncated=true for blob >1MB")
	}
	if resp.RawURL == "" {
		t.Fatal("expected raw_url for truncated blob")
	}
}
