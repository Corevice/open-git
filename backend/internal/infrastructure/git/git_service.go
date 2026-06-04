package git

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

// InitBare creates a new bare repository at path.
func InitBare(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_, err := gogit.PlainInit(path, true)
	return err
}

func repoServer(repoPath string) (*server.Server, *transport.Endpoint, error) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, nil, err
	}

	root := filepath.Dir(abs)
	name := filepath.Base(abs)
	loader := server.NewFilesystemLoader(osfs.New(root))
	svr := server.NewServer(loader)

	ep, err := transport.NewEndpoint(name)
	if err != nil {
		return nil, nil, fmt.Errorf("transport endpoint: %w", err)
	}
	return svr, ep, nil
}

// ServeUploadPack proxies git-upload-pack for a bare repository.
func ServeUploadPack(w http.ResponseWriter, r *http.Request, repoPath string) error {
	svr, ep, err := repoServer(repoPath)
	if err != nil {
		return err
	}

	sess, err := svr.NewUploadPackSession(ep, nil)
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close() }()

	req := packp.NewUploadPackRequest()
	if err := req.Decode(r.Body); err != nil {
		return err
	}

	resp, err := sess.UploadPack(r.Context(), req)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	return resp.Encode(w)
}

// ServeReceivePack proxies git-receive-pack for a bare repository.
func ServeReceivePack(w http.ResponseWriter, r *http.Request, repoPath string) error {
	svr, ep, err := repoServer(repoPath)
	if err != nil {
		return err
	}

	sess, err := svr.NewReceivePackSession(ep, nil)
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close() }()

	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(r.Body); err != nil {
		return err
	}

	report, err := sess.ReceivePack(r.Context(), req)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
	return report.Encode(w)
}
