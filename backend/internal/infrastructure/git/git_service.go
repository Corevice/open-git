package git

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Service wraps go-git and the git binary to host bare repositories over HTTP.
type Service struct {
	Root       string
	GitBinary  string
	Executable execFunc
}

type execFunc func(name string, args ...string) *exec.Cmd

// New returns a Service rooted at the given directory.
func New(root string) *Service {
	return &Service{
		Root:       root,
		GitBinary:  "git",
		Executable: exec.Command,
	}
}

// AbsRepoPath returns an absolute on-disk path for the supplied repo path.
func (s *Service) AbsRepoPath(repoPath string) string {
	if filepath.IsAbs(repoPath) {
		return repoPath
	}
	return filepath.Join(s.Root, repoPath)
}

// InitBare creates a new bare repository at the supplied path.
func (s *Service) InitBare(path string) error {
	abs := s.AbsRepoPath(path)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("git: create parent: %w", err)
	}
	if _, err := gogit.PlainInit(abs, true); err != nil {
		return fmt.Errorf("git: init bare: %w", err)
	}
	return nil
}

// ServeUploadPack proxies a stateless-rpc git-upload-pack request.
func (s *Service) ServeUploadPack(w http.ResponseWriter, r *http.Request, repoPath string) error {
	return s.serveRPC(w, r, repoPath, "upload-pack", "application/x-git-upload-pack-result")
}

// ServeReceivePack proxies a stateless-rpc git-receive-pack request.
func (s *Service) ServeReceivePack(w http.ResponseWriter, r *http.Request, repoPath string) error {
	return s.serveRPC(w, r, repoPath, "receive-pack", "application/x-git-receive-pack-result")
}

// AdvertiseRefs writes the smart-HTTP info/refs advertisement for the given service.
func (s *Service) AdvertiseRefs(w http.ResponseWriter, repoPath, service string) error {
	abs := s.AbsRepoPath(repoPath)
	if _, err := os.Stat(abs); err != nil {
		return err
	}

	contentType := fmt.Sprintf("application/x-git-%s-advertisement", service)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")

	announcement := fmt.Sprintf("# service=git-%s\n", service)
	if _, err := w.Write([]byte(pktLine(announcement))); err != nil {
		return err
	}
	if _, err := w.Write([]byte("0000")); err != nil {
		return err
	}

	cmd := s.Executable(s.GitBinary, service, "--stateless-rpc", "--advertise-refs", abs)
	cmd.Stdout = w
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// IsForcePush returns true when updating ref from oldOID to newOID is not a fast-forward.
// A new branch (old == zero) or deletion (new == zero) is never considered a force-push here.
func (s *Service) IsForcePush(repoPath, ref, oldOID, newOID string) (bool, error) {
	if isZeroOID(oldOID) || isZeroOID(newOID) {
		return false, nil
	}
	abs := s.AbsRepoPath(repoPath)
	repo, err := gogit.PlainOpen(abs)
	if err != nil {
		return false, err
	}

	oldHash := plumbing.NewHash(oldOID)
	newHash := plumbing.NewHash(newOID)

	newCommit, err := repo.CommitObject(newHash)
	if err != nil {
		// If the new commit isn't yet in the repo we can't prove fast-forward.
		return true, nil
	}
	oldCommit, err := repo.CommitObject(oldHash)
	if err != nil {
		return true, nil
	}

	isAncestor, err := oldCommit.IsAncestor(newCommit)
	if err != nil {
		return false, err
	}
	return !isAncestor, nil
}

func (s *Service) serveRPC(w http.ResponseWriter, r *http.Request, repoPath, service, contentType string) error {
	abs := s.AbsRepoPath(repoPath)
	if _, err := os.Stat(abs); err != nil {
		return err
	}

	body, err := decodeBody(r)
	if err != nil {
		return err
	}
	defer body.Close()

	cmd := s.Executable(s.GitBinary, service, "--stateless-rpc", abs)
	cmd.Stdin = body
	cmd.Stderr = io.Discard

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")

	cmd.Stdout = w
	return cmd.Run()
}

func decodeBody(r *http.Request) (io.ReadCloser, error) {
	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		return gz, nil
	}
	return r.Body, nil
}

func pktLine(payload string) string {
	return fmt.Sprintf("%04x%s", len(payload)+4, payload)
}

func isZeroOID(oid string) bool {
	if oid == "" {
		return true
	}
	for _, ch := range oid {
		if ch != '0' {
			return false
		}
	}
	return true
}

// ErrRepoNotFound is returned when a requested repository does not exist on disk.
var ErrRepoNotFound = errors.New("repository not found")
