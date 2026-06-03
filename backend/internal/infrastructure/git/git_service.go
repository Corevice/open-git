package git

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
)

// Service provides Git repository operations backed by go-git.
type Service struct {
	repoRoot string
}

// NewService creates a new git Service that stores repositories under repoRoot.
func NewService(repoRoot string) *Service {
	return &Service{repoRoot: repoRoot}
}

// repoPath returns the filesystem path for the given repository ID.
func (s *Service) repoPath(repoID uuid.UUID) string {
	return filepath.Join(s.repoRoot, repoID.String()+".git")
}

// InitBare initialises a new bare Git repository at path.
func InitBare(path string) error {
	_, err := gogit.PlainInit(path, true)
	return err
}

// ServeUploadPack proxies a git-upload-pack (clone/fetch) request.
func ServeUploadPack(w http.ResponseWriter, r *http.Request, repoPath string) {
	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	w.Header().Set("Cache-Control", "no-cache")
	cmd := exec.CommandContext(r.Context(), "git", "upload-pack", "--stateless-rpc", repoPath)
	cmd.Stdin = r.Body
	cmd.Stdout = w
	_ = cmd.Run()
}

// ServeReceivePack proxies a git-receive-pack (push) request.
func ServeReceivePack(w http.ResponseWriter, r *http.Request, repoPath string) {
	w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
	w.Header().Set("Cache-Control", "no-cache")
	cmd := exec.CommandContext(r.Context(), "git", "receive-pack", "--stateless-rpc", repoPath)
	cmd.Stdin = r.Body
	cmd.Stdout = w
	_ = cmd.Run()
}

// BranchExists reports whether branch exists in the given repository.
func (s *Service) BranchExists(ctx context.Context, repoID uuid.UUID, branch string) (bool, error) {
	repo, err := gogit.PlainOpen(s.repoPath(repoID))
	if err != nil {
		return false, err
	}
	ref := plumbing.NewBranchReferenceName(branch)
	_, err = repo.Reference(ref, true)
	if err == plumbing.ErrReferenceNotFound {
		return false, nil
	}
	return err == nil, err
}

// ResolveRef returns the commit SHA that ref points to.
func (s *Service) ResolveRef(ctx context.Context, repoID uuid.UUID, ref string) (string, error) {
	repo, err := gogit.PlainOpen(s.repoPath(repoID))
	if err != nil {
		return "", err
	}
	h, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return "", err
	}
	return h.String(), nil
}

// Merge merges head into base using the given method.
func (s *Service) Merge(ctx context.Context, repoID uuid.UUID, base, head, method string) error {
	return fmt.Errorf("merge not implemented: use git CLI")
}
