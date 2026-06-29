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

type contributorsStubResolver struct {
	repo *handler.ResolvedGitRepository
}

func (s *contributorsStubResolver) Resolve(_ context.Context, _, _ string) (*handler.ResolvedGitRepository, error) {
	return s.repo, nil
}

type contributorsStubMemberships struct {
	hasReadAccess bool
}

func (s *contributorsStubMemberships) HasReadAccess(_ context.Context, _ int64, _ uuid.UUID) (bool, error) {
	return s.hasReadAccess, nil
}

func (s *contributorsStubMemberships) HasWriteAccess(_ context.Context, _ int64, _ uuid.UUID) (bool, error) {
	return s.hasReadAccess, nil
}

func initRepoWithMultipleAuthors(t *testing.T, authorCount, commitsPerAuthor int) string {
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

	commitIdx := 0
	for i := 0; i < authorCount; i++ {
		author := &object.Signature{
			Name:  fmt.Sprintf("User%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
		for j := 0; j < commitsPerAuthor; j++ {
			name := fmt.Sprintf("file%d.txt", commitIdx)
			if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
				t.Fatalf("write file: %v", err)
			}
			if _, err := wt.Add(name); err != nil {
				t.Fatalf("add file: %v", err)
			}
			if _, err := wt.Commit(fmt.Sprintf("commit %d", commitIdx), &gogit.CommitOptions{Author: author}); err != nil {
				t.Fatalf("commit: %v", err)
			}
			commitIdx++
		}
	}
	return dir
}

func TestGetContributorsPublicRepo(t *testing.T) {
	repoPath := initRepoWithCommits(t, 5)

	h := handler.NewContributorsHandler(
		&contributorsStubResolver{
			repo: &handler.ResolvedGitRepository{
				Visibility: "public",
				DiskPath:   repoPath,
			},
		},
		&contributorsStubMemberships{},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/contributors", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []struct {
		Login         string `json:"login"`
		Contributions int    `json:"contributions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) == 0 {
		t.Fatal("expected non-empty contributors array")
	}
	if resp[0].Contributions <= 0 {
		t.Fatalf("expected contributions > 0, got %d", resp[0].Contributions)
	}
}

func TestGetContributorsPerPageClamped(t *testing.T) {
	repoPath := initRepoWithMultipleAuthors(t, 150, 1)

	h := handler.NewContributorsHandler(
		&contributorsStubResolver{
			repo: &handler.ResolvedGitRepository{
				Visibility: "public",
				DiskPath:   repoPath,
			},
		},
		&contributorsStubMemberships{},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/contributors?per_page=200", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) > 100 {
		t.Fatalf("expected at most 100 entries, got %d", len(resp))
	}
}

func TestGetContributorsPaginationLinkHeader(t *testing.T) {
	repoPath := initRepoWithMultipleAuthors(t, 5, 1)

	h := handler.NewContributorsHandler(
		&contributorsStubResolver{
			repo: &handler.ResolvedGitRepository{
				Visibility: "public",
				DiskPath:   repoPath,
			},
		},
		&contributorsStubMemberships{},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/contributors?page=1&per_page=2", nil)
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

func TestGetContributorsPrivateRepoNoAuth(t *testing.T) {
	repoPath := initRepoWithCommits(t, 1)

	h := handler.NewContributorsHandler(
		&contributorsStubResolver{
			repo: &handler.ResolvedGitRepository{
				Visibility: "private",
				DiskPath:   repoPath,
			},
		},
		&contributorsStubMemberships{},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/contributors", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}
