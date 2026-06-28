package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	infragit "github.com/open-git/backend/internal/infrastructure/git"
)

type mockBranchResolver struct {
	repo *handler.ResolvedGitRepository
}

func (m *mockBranchResolver) Resolve(_ context.Context, _, _ string) (*handler.ResolvedGitRepository, error) {
	return m.repo, nil
}

type mockBranchMemberships struct {
	read  bool
	write bool
}

func (m *mockBranchMemberships) HasReadAccess(_ context.Context, _ int64, _ uuid.UUID) (bool, error) {
	return m.read, nil
}

func (m *mockBranchMemberships) HasWriteAccess(_ context.Context, _ int64, _ uuid.UUID) (bool, error) {
	return m.write, nil
}

type mockBranchRepoRepo struct {
	defaultBranch string
}

func (m *mockBranchRepoRepo) Create(context.Context, *entity.Repository) error { return nil }

func (m *mockBranchRepoRepo) GetByOwnerAndName(context.Context, uuid.UUID, string) (*entity.Repository, error) {
	return nil, nil
}

func (m *mockBranchRepoRepo) GetByOwnerLoginAndName(_ context.Context, _, _ string) (*entity.Repository, error) {
	return &entity.Repository{DefaultBranch: m.defaultBranch}, nil
}

func (m *mockBranchRepoRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, int, error) {
	return nil, 0, nil
}

func (m *mockBranchRepoRepo) CountByOrg(context.Context, uuid.UUID) (int, error) { return 0, nil }

func (m *mockBranchRepoRepo) ListByOwner(context.Context, uuid.UUID, int, int) ([]*entity.Repository, int, error) {
	return nil, 0, nil
}

func (m *mockBranchRepoRepo) CountByOwner(context.Context, uuid.UUID) (int, error) { return 0, nil }

func (m *mockBranchRepoRepo) UpdateVisibility(context.Context, uuid.UUID, string) error { return nil }

func (m *mockBranchRepoRepo) UpdateName(context.Context, uuid.UUID, string) error { return nil }

func (m *mockBranchRepoRepo) UpdateDefaultBranch(context.Context, uuid.UUID, string) error { return nil }

func (m *mockBranchRepoRepo) Delete(context.Context, uuid.UUID) error { return nil }

func newBranchHandlerEcho(t *testing.T, repoPath string, memberships *mockBranchMemberships, defaultBranch string) *echo.Echo {
	t.Helper()

	h := handler.NewBranchHandler(
		&mockBranchResolver{repo: &handler.ResolvedGitRepository{
			ID:             uuid.New(),
			OrganizationID: uuid.New(),
			OwnerID:        42,
			Visibility:     entity.VisibilityPublic,
			DiskPath:       repoPath,
		}},
		&mockBranchRepoRepo{defaultBranch: defaultBranch},
		memberships,
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, authMiddleware(42))
	return e
}

func TestBranchHandler_ListBranches_empty(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}

	e := newBranchHandlerEcho(t, repoPath, &mockBranchMemberships{}, "main")

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/branches", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("branches = %#v, want empty slice", resp)
	}
}

func TestBranchHandler_ListBranches_populated(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}
	if err := seedMainBranch(t, repoPath); err != nil {
		t.Fatalf("seed main branch: %v", err)
	}

	e := newBranchHandlerEcho(t, repoPath, &mockBranchMemberships{}, "main")

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/branches", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("branches = %#v, want one branch", resp)
	}
	if resp[0]["name"] != "main" {
		t.Fatalf("branch name = %v, want main", resp[0]["name"])
	}
	commit, ok := resp[0]["commit"].(map[string]any)
	if !ok || commit["sha"] == "" {
		t.Fatalf("commit.sha missing: %#v", resp[0])
	}
}

func TestBranchHandler_CreateRef_success(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}
	if err := seedMainBranch(t, repoPath); err != nil {
		t.Fatalf("seed main branch: %v", err)
	}

	gitRepo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	mainRef, err := gitRepo.Reference(plumbing.Head, true)
	if err != nil {
		t.Fatalf("head ref: %v", err)
	}

	e := newBranchHandlerEcho(t, repoPath, &mockBranchMemberships{write: true}, "main")

	body, err := json.Marshal(map[string]string{
		"ref": "refs/heads/feature",
		"sha": mainRef.Hash().String(),
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/git/refs", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestBranchHandler_CreateRef_badPrefix_422(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}

	e := newBranchHandlerEcho(t, repoPath, &mockBranchMemberships{write: true}, "main")

	body, err := json.Marshal(map[string]string{
		"ref": "refs/tags/foo",
		"sha": plumbing.ZeroHash.String(),
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/git/refs", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestBranchHandler_DeleteRef_success(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}
	if err := seedMainBranch(t, repoPath); err != nil {
		t.Fatalf("seed main branch: %v", err)
	}

	gitRepo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	mainRef, err := gitRepo.Reference(plumbing.Head, true)
	if err != nil {
		t.Fatalf("head ref: %v", err)
	}
	featureRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/feature"), mainRef.Hash())
	if err := gitRepo.Storer.SetReference(featureRef); err != nil {
		t.Fatalf("create feature branch: %v", err)
	}

	e := newBranchHandlerEcho(t, repoPath, &mockBranchMemberships{write: true}, "main")

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/git/refs/heads/feature", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestBranchHandler_DeleteRef_isDefault_422(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}
	if err := seedMainBranch(t, repoPath); err != nil {
		t.Fatalf("seed main branch: %v", err)
	}

	e := newBranchHandlerEcho(t, repoPath, &mockBranchMemberships{write: true}, "main")

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/git/refs/heads/main", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}
