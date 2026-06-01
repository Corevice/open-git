package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
)

type fakeRepoFinder struct {
	repo *handler.GitRepositoryRef
}

func (f *fakeRepoFinder) FindByOwnerAndName(_ context.Context, _ string, _ string) (*handler.GitRepositoryRef, error) {
	return f.repo, nil
}

type fakePermissions struct {
	read  bool
	write bool
}

func (p *fakePermissions) HasRead(_ context.Context, _ int64, _ int64) (bool, error) {
	return p.read, nil
}

func (p *fakePermissions) HasWrite(_ context.Context, _ int64, _ int64) (bool, error) {
	return p.write, nil
}

type fakeProtections struct {
	protected map[string]*handler.GitBranchProtection
}

func (p *fakeProtections) FindForRef(_ context.Context, _ int64, ref string) (*handler.GitBranchProtection, error) {
	if p.protected == nil {
		return nil, nil
	}
	return p.protected[ref], nil
}

type fakeGitServer struct {
	advertiseService string
	advertised       bool
	receiveCalled    bool
	forcePushFor     map[string]bool
}

func (s *fakeGitServer) AdvertiseRefs(w http.ResponseWriter, _ string, service string) error {
	s.advertiseService = service
	s.advertised = true
	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))
	w.Header().Set("Cache-Control", "no-cache")
	_, err := w.Write([]byte("0000"))
	return err
}

func (s *fakeGitServer) ServeUploadPack(w http.ResponseWriter, _ *http.Request, _ string) error {
	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	return nil
}

func (s *fakeGitServer) ServeReceivePack(w http.ResponseWriter, _ *http.Request, _ string) error {
	s.receiveCalled = true
	w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
	return nil
}

func (s *fakeGitServer) IsForcePush(_ string, ref, _ string, _ string) (bool, error) {
	if s.forcePushFor == nil {
		return false, nil
	}
	return s.forcePushFor[ref], nil
}

func newTestEcho(h *handler.GitHTTPHandler) *echo.Echo {
	e := echo.New()
	h.RegisterRoutes(e)
	return e
}

func pktLine(payload string) string {
	return fmt.Sprintf("%04x%s", len(payload)+4, payload)
}

func TestInfoRefsContentType(t *testing.T) {
	repo := &handler.GitRepositoryRef{ID: 1, OwnerLogin: "alice", Name: "demo", StoragePath: "alice/demo.git"}
	git := &fakeGitServer{}
	h := handler.NewGitHTTPHandler(
		&fakeRepoFinder{repo: repo},
		&fakePermissions{read: true, write: true},
		&fakeProtections{},
		git,
		nil,
	)
	e := newTestEcho(h)

	req := httptest.NewRequest(http.MethodGet, "/alice/demo.git/info/refs?service=git-upload-pack", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/x-git-upload-pack-advertisement" {
		t.Fatalf("unexpected Content-Type %q", ct)
	}
	if !git.advertised || git.advertiseService != "upload-pack" {
		t.Fatalf("expected AdvertiseRefs(upload-pack) to be invoked, got %q", git.advertiseService)
	}
}

func TestForceRejectProtectedBranch(t *testing.T) {
	repo := &handler.GitRepositoryRef{ID: 1, OwnerLogin: "alice", Name: "demo", StoragePath: "alice/demo.git"}
	git := &fakeGitServer{forcePushFor: map[string]bool{"refs/heads/main": true}}
	protections := &fakeProtections{
		protected: map[string]*handler.GitBranchProtection{
			"refs/heads/main": {Pattern: "refs/heads/main", AllowForcePushes: false},
		},
	}
	h := handler.NewGitHTTPHandler(
		&fakeRepoFinder{repo: repo},
		&fakePermissions{read: true, write: true},
		protections,
		git,
		func(c echo.Context) int64 { return 42 },
	)
	e := newTestEcho(h)

	body := buildReceivePackBody(t,
		"1111111111111111111111111111111111111111",
		"2222222222222222222222222222222222222222",
		"refs/heads/main",
	)
	req := httptest.NewRequest(http.MethodPost, "/alice/demo.git/git-receive-pack", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-git-receive-pack-request")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "refs/heads/main") {
		t.Fatalf("expected error to mention protected ref, got %s", rec.Body.String())
	}
	if git.receiveCalled {
		t.Fatalf("git receive-pack should not be invoked when force-push is rejected")
	}
}

func TestReceivePackAnonymousReturns401(t *testing.T) {
	repo := &handler.GitRepositoryRef{ID: 1, OwnerLogin: "alice", Name: "demo", StoragePath: "alice/demo.git"}
	h := handler.NewGitHTTPHandler(
		&fakeRepoFinder{repo: repo},
		&fakePermissions{read: true, write: true},
		&fakeProtections{},
		&fakeGitServer{},
		func(c echo.Context) int64 { return 0 },
	)
	e := newTestEcho(h)

	body := buildReceivePackBody(t,
		"1111111111111111111111111111111111111111",
		"2222222222222222222222222222222222222222",
		"refs/heads/feature",
	)
	req := httptest.NewRequest(http.MethodPost, "/alice/demo.git/git-receive-pack", strings.NewReader(body))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestNonProtectedForcePushAllowed(t *testing.T) {
	repo := &handler.GitRepositoryRef{ID: 1, OwnerLogin: "alice", Name: "demo", StoragePath: "alice/demo.git"}
	git := &fakeGitServer{forcePushFor: map[string]bool{"refs/heads/feature": true}}
	h := handler.NewGitHTTPHandler(
		&fakeRepoFinder{repo: repo},
		&fakePermissions{read: true, write: true},
		&fakeProtections{},
		git,
		func(c echo.Context) int64 { return 42 },
	)
	e := newTestEcho(h)

	body := buildReceivePackBody(t,
		"1111111111111111111111111111111111111111",
		"2222222222222222222222222222222222222222",
		"refs/heads/feature",
	)
	req := httptest.NewRequest(http.MethodPost, "/alice/demo.git/git-receive-pack", strings.NewReader(body))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !git.receiveCalled {
		t.Fatalf("expected git receive-pack to be invoked for non-protected ref")
	}
}

func buildReceivePackBody(t *testing.T, oldOID, newOID, ref string) string {
	t.Helper()
	command := fmt.Sprintf("%s %s %s\x00report-status\n", oldOID, newOID, ref)
	return pktLine(command) + "0000"
}
