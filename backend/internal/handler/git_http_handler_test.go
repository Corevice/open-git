package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
	infragit "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/middleware"
)

type stubResolver struct {
	repo *handler.ResolvedGitRepository
}

func (s *stubResolver) Resolve(_ context.Context, _, _ string) (*handler.ResolvedGitRepository, error) {
	return s.repo, nil
}

type stubMembership struct {
	write bool
}

func (s *stubMembership) HasReadAccess(_ context.Context, _ int64, _ uuid.UUID) (bool, error) {
	return s.write, nil
}

func (s *stubMembership) HasWriteAccess(_ context.Context, _ int64, _ uuid.UUID) (bool, error) {
	return s.write, nil
}

type stubProtection struct {
	protected map[string]bool
}

func (s *stubProtection) IsBranchProtected(_ context.Context, _ uuid.UUID, branch string) (bool, error) {
	return s.protected[branch], nil
}

func TestInfoRefsContentType(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "alice", "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}

	repoID := uuid.New()
	h := handler.NewGitHTTPHandler(
		root,
		&stubResolver{repo: &handler.ResolvedGitRepository{
			ID:       repoID,
			DiskPath: repoPath,
		}},
		nil,
		nil,
		nil,
		nil,
	)

	e := echo.New()
	h.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, "/alice/demo.git/info/refs?service="+transport.UploadPackServiceName, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := rec.Header().Get("Content-Type")
	want := "application/x-git-upload-pack-advertisement"
	if got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
}

func TestForceRejectProtectedBranch(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "alice", "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}
	if err := seedMainBranch(t, repoPath); err != nil {
		t.Fatalf("seed main branch: %v", err)
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	mainRef, err := repo.Reference(plumbing.HEAD, true)
	if err != nil {
		t.Fatalf("head ref: %v", err)
	}
	oldHash := mainRef.Hash()

	// Unrelated commit (non fast-forward) to simulate force-push.
	newHash := createCommit(t, repo, "other", plumbing.ZeroHash)

	repoID := uuid.New()
	orgID := uuid.New()
	h := handler.NewGitHTTPHandler(
		root,
		&stubResolver{repo: &handler.ResolvedGitRepository{
			ID:             repoID,
			OrganizationID: orgID,
			OwnerID:        42,
			DiskPath:       repoPath,
		}},
		&stubMembership{write: true},
		&stubProtection{protected: map[string]bool{"main": true}},
		nil,
		nil,
	)

	body := encodeReceivePackRequest(t, oldHash, newHash)

	e := echo.New()
	h.RegisterRoutes(e)

	t.Run("anonymous returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/alice/demo.git/git-receive-pack", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("force push to protected main returns 422", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/alice/demo.git/git-receive-pack", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:owner/:repo.git/git-receive-pack")
		c.SetParamNames("owner", "repo")
		c.SetParamValues("alice", "demo")
		middleware.SetAuthContext(c, 42, []string{"repo"})

		err := h.ReceivePack(c)
		if err == nil {
			t.Fatal("expected error")
		}
		he, ok := err.(*echo.HTTPError)
		if !ok {
			t.Fatalf("error type = %T, want *echo.HTTPError", err)
		}
		if he.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want %d", he.Code, http.StatusUnprocessableEntity)
		}
	})
}

func TestIsForcePushFallsBackToFalseOnMissingObjects(t *testing.T) {
	root := t.TempDir()
	repoPath := filepath.Join(root, "alice", "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}
	if err := seedMainBranch(t, repoPath); err != nil {
		t.Fatalf("seed main branch: %v", err)
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	mainRef, err := repo.Reference(plumbing.HEAD, true)
	if err != nil {
		t.Fatalf("head ref: %v", err)
	}
	existing := mainRef.Hash()
	missing := plumbing.NewHash("deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	t.Run("missing newHash returns non-force with error", func(t *testing.T) {
		forced, err := handler.IsForcePushForTest(repo, existing, missing)
		if err == nil {
			t.Fatal("expected error for missing new commit, got nil")
		}
		if forced {
			t.Fatalf("forced = true, want false on missing object")
		}
	})

	t.Run("missing oldHash returns non-force with error", func(t *testing.T) {
		forced, err := handler.IsForcePushForTest(repo, missing, existing)
		if err == nil {
			t.Fatal("expected error for missing old commit, got nil")
		}
		if forced {
			t.Fatalf("forced = true, want false on missing object")
		}
	})

	t.Run("zero hash on either side is never force", func(t *testing.T) {
		forced, err := handler.IsForcePushForTest(repo, plumbing.ZeroHash, existing)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if forced {
			t.Fatal("forced = true, want false when oldHash is zero (ref creation)")
		}
		forced, err = handler.IsForcePushForTest(repo, existing, plumbing.ZeroHash)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if forced {
			t.Fatal("forced = true, want false when newHash is zero (ref deletion)")
		}
	})

	t.Run("fast-forward is non-force", func(t *testing.T) {
		child := createCommit(t, repo, "follow-up", existing)
		forced, err := handler.IsForcePushForTest(repo, existing, child)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if forced {
			t.Fatal("forced = true, want false for fast-forward update")
		}
	})

	t.Run("unrelated history is force", func(t *testing.T) {
		unrelated := createCommit(t, repo, "unrelated", plumbing.ZeroHash)
		forced, err := handler.IsForcePushForTest(repo, existing, unrelated)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !forced {
			t.Fatal("forced = false, want true for non-fast-forward update")
		}
	})
}

func TestReceivePackAllowsPushWithUnwrittenNewObject(t *testing.T) {
	// Regression: rejectProtectedForcePush runs before the packfile in the
	// receive-pack body is processed, so the new commit object may not yet
	// exist in the bare repo. The protection check must not reject such a
	// push as a force-push.
	root := t.TempDir()
	repoPath := filepath.Join(root, "alice", "demo.git")
	if err := infragit.InitBare(repoPath); err != nil {
		t.Fatalf("init bare repo: %v", err)
	}
	if err := seedMainBranch(t, repoPath); err != nil {
		t.Fatalf("seed main branch: %v", err)
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	mainRef, err := repo.Reference(plumbing.HEAD, true)
	if err != nil {
		t.Fatalf("head ref: %v", err)
	}
	oldHash := mainRef.Hash()
	// A hash that has no corresponding object in the repo, simulating the new
	// commit carried inside a packfile that hasn't been processed yet.
	newHash := plumbing.NewHash("cafebabecafebabecafebabecafebabecafebabe")

	repoID := uuid.New()
	orgID := uuid.New()
	h := handler.NewGitHTTPHandler(
		root,
		&stubResolver{repo: &handler.ResolvedGitRepository{
			ID:             repoID,
			OrganizationID: orgID,
			OwnerID:        42,
			DiskPath:       repoPath,
		}},
		&stubMembership{write: true},
		&stubProtection{protected: map[string]bool{"main": true}},
		nil,
		nil,
	)

	body := encodeReceivePackRequest(t, oldHash, newHash)

	e := echo.New()
	h.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodPost, "/alice/demo.git/git-receive-pack", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:owner/:repo.git/git-receive-pack")
	c.SetParamNames("owner", "repo")
	c.SetParamValues("alice", "demo")
	middleware.SetAuthContext(c, 42, []string{"repo"})

	err = h.ReceivePack(c)
	// Downstream ServeReceivePack may still fail because the request body
	// carries no real packfile in this test — what matters is that the
	// failure is NOT the 422 force-push rejection from the protection check.
	if err != nil {
		if he, ok := err.(*echo.HTTPError); ok && he.Code == http.StatusUnprocessableEntity {
			t.Fatalf("got 422 force-push rejection for push with unwritten new object: %v", err)
		}
	}
}

func seedMainBranch(t *testing.T, repoPath string) error {
	t.Helper()
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return err
	}
	commit := createCommit(t, repo, "initial", plumbing.ZeroHash)
	ref := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/main"), commit)
	if err := repo.Storer.SetReference(ref); err != nil {
		return err
	}
	head := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/main"))
	return repo.Storer.SetReference(head)
}

func createCommit(t *testing.T, repo *gogit.Repository, message string, parent plumbing.Hash) plumbing.Hash {
	t.Helper()
	now := time.Now().UTC()
	commit := &object.Commit{
		Message: message,
		Author: object.Signature{
			Name:  "test",
			Email: "test@example.com",
			When:  now,
		},
		Committer: object.Signature{
			Name:  "test",
			Email: "test@example.com",
			When:  now,
		},
	}
	if parent != plumbing.ZeroHash {
		commit.ParentHashes = []plumbing.Hash{parent}
	}

	encoded := repo.Storer.NewEncodedObject()
	if err := commit.Encode(encoded); err != nil {
		t.Fatalf("encode commit: %v", err)
	}
	hash, err := repo.Storer.SetEncodedObject(encoded)
	if err != nil {
		t.Fatalf("store commit: %v", err)
	}
	return hash
}

func encodeReceivePackRequest(t *testing.T, oldHash, newHash plumbing.Hash) []byte {
	t.Helper()
	req := packp.NewReferenceUpdateRequest()
	req.Commands = []*packp.Command{
		{
			Name: plumbing.ReferenceName("refs/heads/main"),
			Old:  oldHash,
			New:  newHash,
		},
	}
	var buf bytes.Buffer
	if err := req.Encode(&buf); err != nil {
		t.Fatalf("encode receive-pack request: %v", err)
	}
	return buf.Bytes()
}
