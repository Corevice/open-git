package git

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
)

// TreeEntry describes a file or directory in a repository tree.
type TreeEntry struct {
	Name string
	Type string
	Size int64
	SHA  string
	Path string
}

// CommitSummary is a lightweight commit record for history listing.
type CommitSummary struct {
	SHA     string
	Message string
	Author  string
	Date    time.Time
}

const (
	TreeEntryTypeFile = "file"
	TreeEntryTypeDir  = "dir"
)

var ErrPathNotFound = errors.New("path not found")

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

func openRepository(repoPath string) (*gogit.Repository, error) {
	return gogit.PlainOpen(repoPath)
}

func resolveCommit(repo *gogit.Repository, ref string) (*object.Commit, error) {
	if ref == "" {
		ref = plumbing.HEAD.String()
	}
	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, fmt.Errorf("resolve ref %q: %w", ref, err)
	}
	return repo.CommitObject(*hash)
}

// GetTree lists entries at path (file returns a single entry; directory returns children).
func GetTree(repoPath, ref, path string) ([]TreeEntry, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return nil, err
	}

	commit, err := resolveCommit(repo, ref)
	if err != nil {
		return nil, err
	}

	root, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	path = strings.Trim(path, "/")
	target := root
	if path != "" {
		entry, err := root.FindEntry(path)
		if err != nil {
			return nil, ErrPathNotFound
		}
		if entry.Mode == filemode.Dir {
			target, err = repo.TreeObject(entry.Hash)
			if err != nil {
				return nil, err
			}
			return treeEntries(repo, target, path)
		}
		size, err := blobSize(repo, entry.Hash)
		if err != nil {
			return nil, err
		}
		return []TreeEntry{{
			Name: entry.Name,
			Type: TreeEntryTypeFile,
			Size: size,
			SHA:  entry.Hash.String(),
			Path: path,
		}}, nil
	}

	return treeEntries(repo, target, "")
}

func treeEntries(repo *gogit.Repository, tree *object.Tree, basePath string) ([]TreeEntry, error) {
	entries := make([]TreeEntry, 0, len(tree.Entries))
	for _, e := range tree.Entries {
		entryPath := e.Name
		if basePath != "" {
			entryPath = basePath + "/" + e.Name
		}
		entryType := TreeEntryTypeFile
		var size int64
		if e.Mode == filemode.Dir {
			entryType = TreeEntryTypeDir
		} else {
			size, _ = blobSize(repo, e.Hash)
		}
		entries = append(entries, TreeEntry{
			Name: e.Name,
			Type: entryType,
			Size: size,
			SHA:  e.Hash.String(),
			Path: entryPath,
		})
	}
	return entries, nil
}

func blobSize(repo *gogit.Repository, hash plumbing.Hash) (int64, error) {
	blob, err := repo.BlobObject(hash)
	if err != nil {
		return 0, err
	}
	return blob.Size, nil
}

// GetBlob returns file content at path for the given ref.
func GetBlob(repoPath, ref, path string) ([]byte, int64, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return nil, 0, err
	}

	commit, err := resolveCommit(repo, ref)
	if err != nil {
		return nil, 0, err
	}

	root, err := commit.Tree()
	if err != nil {
		return nil, 0, err
	}

	path = strings.Trim(path, "/")
	if path == "" {
		return nil, 0, ErrPathNotFound
	}

	entry, err := root.FindEntry(path)
	if err != nil {
		return nil, 0, ErrPathNotFound
	}
	if entry.Mode == filemode.Dir {
		return nil, 0, fmt.Errorf("path is a directory")
	}

	blob, err := repo.BlobObject(entry.Hash)
	if err != nil {
		return nil, 0, err
	}
	reader, err := blob.Reader()
	if err != nil {
		return nil, 0, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, err
	}
	return data, int64(len(data)), nil
}

// GetBlobBySHA returns raw blob bytes by object SHA.
func GetBlobBySHA(repoPath, sha string) ([]byte, int64, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return nil, 0, err
	}
	hash := plumbing.NewHash(sha)
	blob, err := repo.BlobObject(hash)
	if err != nil {
		return nil, 0, err
	}
	reader, err := blob.Reader()
	if err != nil {
		return nil, 0, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, err
	}
	return data, blob.Size, nil
}

// GetCommits returns a page of commit history and the total commit count.
func GetCommits(repoPath, branch string, page, perPage int) ([]CommitSummary, int, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return nil, 0, err
	}

	start, err := resolveCommit(repo, branch)
	if err != nil {
		return nil, 0, err
	}

	iter, err := repo.Log(&gogit.LogOptions{From: start.Hash})
	if err != nil {
		return nil, 0, err
	}
	defer iter.Close()

	all := make([]CommitSummary, 0)
	err = iter.ForEach(func(c *object.Commit) error {
		all = append(all, CommitSummary{
			SHA:     c.Hash.String(),
			Message: strings.TrimSpace(c.Message),
			Author:  c.Author.Name,
			Date:    c.Author.When,
		})
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	total := len(all)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	startIdx := (page - 1) * perPage
	if startIdx >= total {
		return []CommitSummary{}, total, nil
	}
	endIdx := startIdx + perPage
	if endIdx > total {
		endIdx = total
	}
	return all[startIdx:endIdx], total, nil
}
