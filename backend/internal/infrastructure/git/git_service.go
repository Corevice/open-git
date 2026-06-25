package git

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	formatpatch "github.com/go-git/go-git/v5/plumbing/format/patch"
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

// BranchInfo describes a repository branch reference.
type BranchInfo struct {
	Name string
	SHA  string
}

// TagInfo describes a repository tag reference.
type TagInfo struct {
	Name string
	SHA  string
}

// FileDiff describes a single file change within a commit.
type FileDiff struct {
	Filename  string
	Status    string
	Additions int
	Deletions int
	Patch     *string
}

// CommitStats aggregates line change counts for a commit.
type CommitStats struct {
	Total     int
	Additions int
	Deletions int
}

// CommitDetail is a commit with its file-level diff.
type CommitDetail struct {
	SHA     string
	Message string
	Author  string
	Email   string
	Date    time.Time
	Files   []FileDiff
	Stats   CommitStats
}

const (
	TreeEntryTypeFile = "file"
	TreeEntryTypeDir  = "dir"
)

var (
	ErrPathNotFound    = errors.New("path not found")
	ErrEmptyRepository = errors.New("empty repository")
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

const maxPatchBytes = 102400

// GetBranches lists all branch references in the repository.
func GetBranches(repoPath string) ([]BranchInfo, error) {
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		if strings.Contains(err.Error(), "reference not found") {
			return []BranchInfo{}, nil
		}
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		if strings.Contains(err.Error(), "reference not found") || errors.Is(err, plumbing.ErrReferenceNotFound) {
			return []BranchInfo{}, nil
		}
		return nil, err
	}
	if head.Hash().IsZero() {
		return []BranchInfo{}, nil
	}

	iter, err := repo.Branches()
	if err != nil {
		return nil, err
	}

	branches := make([]BranchInfo, 0)
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, BranchInfo{
			Name: ref.Name().Short(),
			SHA:  ref.Hash().String(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Name < branches[j].Name
	})
	return branches, nil
}

// GetTags lists all tag references in the repository.
func GetTags(repoPath string) ([]TagInfo, error) {
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		if strings.Contains(err.Error(), "reference not found") {
			return []TagInfo{}, nil
		}
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		if strings.Contains(err.Error(), "reference not found") || errors.Is(err, plumbing.ErrReferenceNotFound) {
			return []TagInfo{}, nil
		}
		return nil, err
	}
	if head.Hash().IsZero() {
		return []TagInfo{}, nil
	}

	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}

	tags := make([]TagInfo, 0)
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		hash := ref.Hash()
		if tagObj, err := repo.TagObject(hash); err == nil {
			hash = tagObj.Target
		}
		tags = append(tags, TagInfo{
			Name: ref.Name().Short(),
			SHA:  hash.String(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})
	return tags, nil
}

// GetCommitDetail returns a commit and its file-level diff.
func GetCommitDetail(repoPath, sha string) (*CommitDetail, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return nil, err
	}

	commit, err := resolveCommit(repo, sha)
	if err != nil {
		return nil, err
	}

	detail := &CommitDetail{
		SHA:     commit.Hash.String(),
		Message: strings.TrimSpace(commit.Message),
		Author:  commit.Author.Name,
		Email:   commit.Author.Email,
		Date:    commit.Author.When,
	}

	var files []FileDiff
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return nil, err
		}
		patch, err := commit.Patch(parent)
		if err != nil {
			return nil, err
		}
		files, err = fileDiffsFromPatch(patch)
		if err != nil {
			return nil, err
		}
	} else {
		tree, err := commit.Tree()
		if err != nil {
			return nil, err
		}
		emptyTree := &object.Tree{}
		patch, err := emptyTree.Patch(tree)
		if err != nil {
			return nil, err
		}
		files, err = fileDiffsFromPatch(patch)
		if err != nil {
			return nil, err
		}
	}

	detail.Files = files
	detail.Stats = commitStatsFromFiles(files)
	return detail, nil
}

func fileDiffsFromPatch(patch *object.Patch) ([]FileDiff, error) {
	stats := patch.Stats()
	files := make([]FileDiff, 0, len(stats))

	for _, fp := range patch.FilePatches() {
		from, to := fp.Files()
		name, status := classifyFileChange(from, to)

		stat := stats[name]
		additions := stat.Additions
		deletions := stat.Deletions

		var patchPtr *string
		var buf bytes.Buffer
		if err := formatpatch.Encode(&buf, fp); err == nil {
			patchText := buf.String()
			if len(patchText) <= maxPatchBytes {
				patchPtr = &patchText
			}
		}

		files = append(files, FileDiff{
			Filename:  name,
			Status:    status,
			Additions: additions,
			Deletions: deletions,
			Patch:     patchPtr,
		})
	}

	return files, nil
}

func classifyFileChange(from, to object.File) (string, string) {
	switch {
	case from == nil && to != nil:
		return to.Name(), "added"
	case from != nil && to == nil:
		return from.Name(), "deleted"
	case to != nil:
		return to.Name(), "modified"
	default:
		return from.Name(), "modified"
	}
}

func commitStatsFromFiles(files []FileDiff) CommitStats {
	stats := CommitStats{Total: len(files)}
	for _, f := range files {
		stats.Additions += f.Additions
		stats.Deletions += f.Deletions
	}
	return stats
}
