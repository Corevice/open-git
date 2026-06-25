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

func initEmptyRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := gogit.PlainInit(dir, false); err != nil {
		t.Fatalf("init repo: %v", err)
	}
	return dir
}

func headSHA(t *testing.T, repoPath string) string {
	t.Helper()
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	return ref.Hash().String()
}

func TestGetBranches_OK(t *testing.T) {
	repoPath := initRepoWithCommits(t, 2)

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/branches", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var branches []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &branches); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(branches) == 0 {
		t.Fatal("expected at least one branch")
	}
	if branches[0].Name == "" {
		t.Fatal("expected branch name")
	}
	if branches[0].Commit.SHA == "" {
		t.Fatal("expected branch commit sha")
	}
}

func TestGetTags_Empty(t *testing.T) {
	repoPath := initRepoWithCommits(t, 1)

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/tags", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var tags []json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &tags); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("expected empty tags array, got %d items", len(tags))
	}
}

func TestGetCommitDetail_OK(t *testing.T) {
	repoPath := initRepoWithCommits(t, 2)
	sha := headSHA(t, repoPath)

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/commits/"+sha, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name  string `json:"name"`
				Email string `json:"email"`
				Date  string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
		Files []struct {
			Filename  string  `json:"filename"`
			Status    string  `json:"status"`
			Additions int     `json:"additions"`
			Deletions int     `json:"deletions"`
			Patch     *string `json:"patch"`
		} `json:"files"`
		Stats struct {
			Total     int `json:"total"`
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.SHA != sha {
		t.Fatalf("sha = %q, want %q", resp.SHA, sha)
	}
	if resp.Commit.Message == "" {
		t.Fatal("expected commit message")
	}
	if resp.Commit.Author.Name == "" {
		t.Fatal("expected commit author name")
	}
	if len(resp.Files) == 0 {
		t.Fatal("expected files in commit detail")
	}
	if resp.Stats.Total == 0 && resp.Stats.Additions == 0 && resp.Stats.Deletions == 0 {
		t.Fatal("expected non-zero stats")
	}
}

func TestGetCommitDetail_ShortSHA(t *testing.T) {
	repoPath := initRepoWithCommits(t, 1)
	sha := headSHA(t, repoPath)
	shortSHA := sha[:7]

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/commits/"+shortSHA, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.HasPrefix(resp.SHA, shortSHA) {
		t.Fatalf("sha = %q, want prefix %q", resp.SHA, shortSHA)
	}
}

func TestGetCommitDetail_NotFound(t *testing.T) {
	repoPath := initRepoWithCommits(t, 1)

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/commits/unknown123", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestGetContents_PathTraversal_Returns400(t *testing.T) {
	repoPath := initRepoWithCommits(t, 1)

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/contents?path=../etc/passwd", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestGetContents_NullByte_Returns400(t *testing.T) {
	repoPath := initRepoWithCommits(t, 1)

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/contents?path=foo%00bar", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestGetContents_AbsolutePath_Returns400(t *testing.T) {
	repoPath := initRepoWithCommits(t, 1)

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/contents?path=/etc/passwd", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestGetCommits_EmptyRepo_ReturnsEmptyArray(t *testing.T) {
	repoPath := initEmptyRepo(t)

	h := handler.NewContentHandler(&contentStubResolver{
		repo: &handler.ResolvedGitRepository{
			ID:       uuid.New(),
			DiskPath: repoPath,
		},
	})

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/commits", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var commits []json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &commits); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected empty commits array, got %d items", len(commits))
	}
}
