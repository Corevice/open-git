package git

import (
	"context"
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
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
	domainservice "github.com/open-git/backend/internal/domain/service"
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

var (
	ErrPathNotFound      = errors.New("path not found")
	ErrRefAlreadyExists  = errors.New("reference already exists")
)

// BranchSummary describes a branch or tag reference.
type BranchSummary struct {
	Name      string
	CommitSHA string
}

// FileDiff describes a single file change between two refs.
type FileDiff struct {
	OldPath string
	NewPath string
	Patch   string
	Status  string
}

// InitBare creates a new bare repository at path.
func InitBare(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_, err := gogit.PlainInit(path, true)
	return err
}

func repoServer(repoPath string) (transport.Transport, *transport.Endpoint, error) {
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

// GetBranches returns all branch references in the repository.
func GetBranches(repoPath string) ([]BranchSummary, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return nil, err
	}

	iter, err := repo.Branches()
	if err != nil {
		return nil, err
	}

	branches := make([]BranchSummary, 0)
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, BranchSummary{
			Name:      strings.TrimPrefix(ref.Name().String(), "refs/heads/"),
			CommitSHA: ref.Hash().String(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return branches, nil
}

// GetTags returns all tag references in the repository.
func GetTags(repoPath string) ([]BranchSummary, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return nil, err
	}

	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}

	tags := make([]BranchSummary, 0)
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tags = append(tags, BranchSummary{
			Name:      strings.TrimPrefix(ref.Name().String(), "refs/tags/"),
			CommitSHA: ref.Hash().String(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// CreateBranch creates a new branch pointing at fromRef.
func CreateBranch(repoPath, name, fromRef string) error {
	repo, err := openRepository(repoPath)
	if err != nil {
		return err
	}

	refName := plumbing.NewBranchReferenceName(name)
	if _, err := repo.Storer.Reference(refName); err == nil {
		return ErrRefAlreadyExists
	} else if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return err
	}

	commit, err := resolveCommit(repo, fromRef)
	if err != nil {
		return err
	}

	ref := plumbing.NewHashReference(refName, commit.Hash)
	return repo.Storer.SetReference(ref)
}

// DeleteBranch removes a branch reference.
func DeleteBranch(repoPath, name string) error {
	repo, err := openRepository(repoPath)
	if err != nil {
		return err
	}

	refName := plumbing.NewBranchReferenceName(name)
	if _, err := repo.Reference(refName, true); err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return ErrPathNotFound
		}
		return err
	}

	err = repo.Storer.RemoveReference(refName)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return ErrPathNotFound
		}
		return err
	}
	return nil
}

// SetDefaultBranch updates HEAD to point at the named branch.
func SetDefaultBranch(repoPath, name string) error {
	repo, err := openRepository(repoPath)
	if err != nil {
		return err
	}

	ref := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName(name))
	return repo.Storer.SetReference(ref)
}

// GetDiff returns file-level diffs between baseRef and headRef.
func GetDiff(repoPath, baseRef, headRef string) ([]FileDiff, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return nil, err
	}

	baseCommit, err := resolveCommit(repo, baseRef)
	if err != nil {
		return nil, err
	}
	headCommit, err := resolveCommit(repo, headRef)
	if err != nil {
		return nil, err
	}

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return nil, err
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, err
	}

	changes, err := object.DiffTreeContext(context.Background(), baseTree, headTree)
	if err != nil {
		return nil, err
	}

	patch, err := changes.Patch()
	if err != nil {
		return nil, err
	}

	filePatches := patch.FilePatches()
	diffs := make([]FileDiff, 0, len(filePatches))
	for _, fp := range filePatches {
		from, to := fp.Files()
		oldPath := ""
		newPath := ""
		if from != nil {
			oldPath = from.Path()
		}
		if to != nil {
			newPath = to.Path()
		}

		status := ""
		var patchContent strings.Builder
		for _, chunk := range fp.Chunks() {
			switch chunk.Type() {
			case fdiff.Add:
				if status == "" {
					status = "add"
				}
			case fdiff.Delete:
				status = "delete"
			case fdiff.Equal:
				continue
			default:
				if status != "delete" {
					status = "modify"
				}
			}
			if chunk.Type() != fdiff.Equal {
				patchContent.WriteString(chunk.Content())
			}
		}
		if status == "" {
			status = "modify"
		}

		diffs = append(diffs, FileDiff{
			OldPath: oldPath,
			NewPath: newPath,
			Patch:   patchContent.String(),
			Status:  status,
		})
	}

	return diffs, nil
}

// BranchExists reports whether a branch exists locally or as origin/<branch>.
func BranchExists(repoPath, branch string) (bool, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return false, err
	}

	refNames := []plumbing.ReferenceName{
		plumbing.NewBranchReferenceName(branch),
		plumbing.NewRemoteReferenceName("origin", branch),
	}
	for _, refName := range refNames {
		if _, err := repo.Reference(refName, true); err == nil {
			return true, nil
		}
	}
	return false, nil
}

// ResolveRef resolves a ref to its commit SHA.
func ResolveRef(repoPath, ref string) (string, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return "", err
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return "", fmt.Errorf("resolve ref %q: %w", ref, err)
	}
	return hash.String(), nil
}

// GetMergeBase returns the best common ancestor of base and head.
func GetMergeBase(repoPath, base, head string) (string, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return "", err
	}

	baseCommit, err := resolveCommit(repo, base)
	if err != nil {
		return "", err
	}
	headCommit, err := resolveCommit(repo, head)
	if err != nil {
		return "", err
	}

	mergeBases, err := baseCommit.MergeBase(headCommit)
	if err != nil {
		return "", err
	}
	if len(mergeBases) == 0 {
		return "", errors.New("no merge base found")
	}
	return mergeBases[0].Hash.String(), nil
}

// Merge merges head into base using merge, squash, or rebase strategy.
func Merge(repoPath, base, head, method string) (string, error) {
	repo, err := openRepository(repoPath)
	if err != nil {
		return "", err
	}

	baseCommit, err := resolveCommit(repo, base)
	if err != nil {
		return "", err
	}
	headCommit, err := resolveCommit(repo, head)
	if err != nil {
		return "", err
	}

	baseRefName := branchRefName(base)
	if baseRefName == "" {
		return "", fmt.Errorf("invalid base ref %q", base)
	}

	switch method {
	case "squash":
		return mergeSquash(repo, baseCommit, headCommit, baseRefName)
	case "rebase":
		return mergeRebase(repo, baseCommit, headCommit, baseRefName, head)
	default:
		return mergeMerge(repo, baseCommit, headCommit, baseRefName, head)
	}
}

func branchRefName(ref string) plumbing.ReferenceName {
	ref = strings.TrimPrefix(ref, "refs/heads/")
	return plumbing.NewBranchReferenceName(ref)
}

func isFastForward(base, head *object.Commit) (bool, error) {
	if base.Hash == head.Hash {
		return true, nil
	}
	mergeBases, err := base.MergeBase(head)
	if err != nil {
		return false, err
	}
	for _, mb := range mergeBases {
		if mb.Hash == base.Hash {
			return true, nil
		}
	}
	return false, nil
}

func mergeMerge(repo *gogit.Repository, baseCommit, headCommit *object.Commit, baseRef plumbing.ReferenceName, headRef string) (string, error) {
	ff, err := isFastForward(baseCommit, headCommit)
	if err != nil {
		return "", err
	}
	if ff {
		ref := plumbing.NewHashReference(baseRef, headCommit.Hash)
		if err := repo.Storer.SetReference(ref); err != nil {
			return "", err
		}
		return headCommit.Hash.String(), nil
	}

	mergedTreeHash, err := mergeCommitTrees(repo, baseCommit, headCommit)
	if err != nil {
		return "", err
	}

	mergeCommit := &object.Commit{
		Message:      fmt.Sprintf("Merge branch '%s'", strings.TrimPrefix(headRef, "refs/heads/")),
		TreeHash:     mergedTreeHash,
		ParentHashes: []plumbing.Hash{baseCommit.Hash, headCommit.Hash},
		Author:       headCommit.Author,
		Committer:    headCommit.Committer,
	}
	mergeHash, err := storeCommit(repo, mergeCommit)
	if err != nil {
		return "", err
	}

	ref := plumbing.NewHashReference(baseRef, mergeHash)
	if err := repo.Storer.SetReference(ref); err != nil {
		return "", err
	}
	return mergeHash.String(), nil
}

func mergeSquash(repo *gogit.Repository, baseCommit, headCommit *object.Commit, baseRef plumbing.ReferenceName) (string, error) {
	mergedTreeHash, err := mergeCommitTrees(repo, baseCommit, headCommit)
	if err != nil {
		return "", err
	}

	squashCommit := &object.Commit{
		Message:      fmt.Sprintf("Squashed commit of '%s'", headCommit.Hash.String()[:7]),
		TreeHash:     mergedTreeHash,
		ParentHashes: []plumbing.Hash{baseCommit.Hash},
		Author:       headCommit.Author,
		Committer:    headCommit.Committer,
	}
	squashHash, err := storeCommit(repo, squashCommit)
	if err != nil {
		return "", err
	}

	ref := plumbing.NewHashReference(baseRef, squashHash)
	if err := repo.Storer.SetReference(ref); err != nil {
		return "", err
	}
	return squashHash.String(), nil
}

func mergeRebase(repo *gogit.Repository, baseCommit, headCommit *object.Commit, baseRef plumbing.ReferenceName, headRef string) (string, error) {
	ff, err := isFastForward(baseCommit, headCommit)
	if err != nil {
		return "", err
	}
	if ff {
		ref := plumbing.NewHashReference(baseRef, headCommit.Hash)
		if err := repo.Storer.SetReference(ref); err != nil {
			return "", err
		}
		return headCommit.Hash.String(), nil
	}

	mergeBases, err := baseCommit.MergeBase(headCommit)
	if err != nil {
		return "", err
	}
	if len(mergeBases) == 0 {
		return "", errors.New("no merge base found")
	}

	commits, err := commitsSinceBase(repo, mergeBases[0], headCommit)
	if err != nil {
		return "", err
	}
	if len(commits) == 0 {
		ref := plumbing.NewHashReference(baseRef, baseCommit.Hash)
		if err := repo.Storer.SetReference(ref); err != nil {
			return "", err
		}
		return baseCommit.Hash.String(), nil
	}

	currentBase := baseCommit
	var lastHash plumbing.Hash
	for _, commit := range commits {
		rebasedTreeHash, err := replayCommitOnto(repo, currentBase, commit)
		if err != nil {
			return "", err
		}

		rebasedCommit := &object.Commit{
			Message:      commit.Message,
			TreeHash:     rebasedTreeHash,
			ParentHashes: []plumbing.Hash{currentBase.Hash},
			Author:       commit.Author,
			Committer:    commit.Committer,
		}
		rebasedHash, err := storeCommit(repo, rebasedCommit)
		if err != nil {
			return "", err
		}

		currentBase, err = repo.CommitObject(rebasedHash)
		if err != nil {
			return "", err
		}
		lastHash = rebasedHash
	}

	if err := repo.Storer.SetReference(plumbing.NewHashReference(baseRef, lastHash)); err != nil {
		return "", err
	}

	headBranchRef := branchRefName(headRef)
	if headBranchRef != "" {
		_ = repo.Storer.SetReference(plumbing.NewHashReference(headBranchRef, lastHash))
	}

	return lastHash.String(), nil
}

func commitsSinceBase(repo *gogit.Repository, mergeBase, head *object.Commit) ([]*object.Commit, error) {
	commits := make([]*object.Commit, 0)
	current := head
	for current.Hash != mergeBase.Hash {
		commits = append(commits, current)
		if len(current.ParentHashes) == 0 {
			break
		}
		parent, err := repo.CommitObject(current.ParentHashes[0])
		if err != nil {
			return nil, err
		}
		current = parent
	}

	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}
	return commits, nil
}

func replayCommitOnto(repo *gogit.Repository, newBase, commit *object.Commit) (plumbing.Hash, error) {
	if len(commit.ParentHashes) == 0 {
		return commit.TreeHash, nil
	}

	parent, err := repo.CommitObject(commit.ParentHashes[0])
	if err != nil {
		return plumbing.ZeroHash, err
	}

	ancestorTree, err := parent.Tree()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	baseTree, err := newBase.Tree()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	headTree, err := commit.Tree()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	mergedEntries, err := threeWayMergeTrees(ancestorTree, baseTree, headTree)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	mergedTree := &object.Tree{Entries: mergedEntries}
	return storeTree(repo, mergedTree)
}

func mergeCommitTrees(repo *gogit.Repository, baseCommit, headCommit *object.Commit) (plumbing.Hash, error) {
	mergeBases, err := baseCommit.MergeBase(headCommit)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	if len(mergeBases) == 0 {
		return plumbing.ZeroHash, errors.New("no merge base found")
	}

	mergeBaseTree, err := mergeBases[0].Tree()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	mergedEntries, err := threeWayMergeTrees(mergeBaseTree, baseTree, headTree)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	mergedTree := &object.Tree{Entries: mergedEntries}
	return storeTree(repo, mergedTree)
}

type treeEntryState struct {
	entry  object.TreeEntry
	exists bool
}

func treeEntryStateFromMap(entries map[string]object.TreeEntry, name string) treeEntryState {
	entry, ok := entries[name]
	return treeEntryState{entry: entry, exists: ok}
}

func treeEntriesEqual(a, b treeEntryState) bool {
	if !a.exists && !b.exists {
		return true
	}
	if a.exists != b.exists {
		return false
	}
	return a.entry.Hash == b.entry.Hash && a.entry.Mode == b.entry.Mode
}

func threeWayMergeTrees(ancestorTree, baseTree, headTree *object.Tree) ([]object.TreeEntry, error) {
	ancestorEntries := treeEntryMap(ancestorTree)
	baseEntries := treeEntryMap(baseTree)
	headEntries := treeEntryMap(headTree)

	names := make(map[string]struct{})
	for name := range ancestorEntries {
		names[name] = struct{}{}
	}
	for name := range baseEntries {
		names[name] = struct{}{}
	}
	for name := range headEntries {
		names[name] = struct{}{}
	}

	mergedEntries := make([]object.TreeEntry, 0, len(names))
	for name := range names {
		ancestor := treeEntryStateFromMap(ancestorEntries, name)
		base := treeEntryStateFromMap(baseEntries, name)
		head := treeEntryStateFromMap(headEntries, name)

		baseChanged := !treeEntriesEqual(ancestor, base)
		headChanged := !treeEntriesEqual(ancestor, head)

		switch {
		case baseChanged && headChanged:
			if !treeEntriesEqual(base, head) {
				return nil, domainservice.ErrMergeConflict
			}
			mergedEntries = append(mergedEntries, head.entry)
		case baseChanged:
			if base.exists {
				mergedEntries = append(mergedEntries, base.entry)
			}
		case headChanged:
			if head.exists {
				mergedEntries = append(mergedEntries, head.entry)
			}
		default:
			if ancestor.exists {
				mergedEntries = append(mergedEntries, ancestor.entry)
			}
		}
	}

	sortTreeEntries(mergedEntries)
	return mergedEntries, nil
}

func treeEntryMap(tree *object.Tree) map[string]object.TreeEntry {
	m := make(map[string]object.TreeEntry, len(tree.Entries))
	for _, e := range tree.Entries {
		m[e.Name] = e
	}
	return m
}

func sortTreeEntries(entries []object.TreeEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
}

func storeTree(repo *gogit.Repository, tree *object.Tree) (plumbing.Hash, error) {
	obj := repo.Storer.NewEncodedObject()
	if err := tree.Encode(obj); err != nil {
		return plumbing.ZeroHash, err
	}
	return repo.Storer.SetEncodedObject(obj)
}

func storeCommit(repo *gogit.Repository, commit *object.Commit) (plumbing.Hash, error) {
	obj := repo.Storer.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return plumbing.ZeroHash, err
	}
	return repo.Storer.SetEncodedObject(obj)
}

type GitServiceAdapter struct{}

func NewGitServiceAdapter() *GitServiceAdapter {
	return &GitServiceAdapter{}
}

func (GitServiceAdapter) BranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return BranchExists(repoPath, branch)
}

func (GitServiceAdapter) ResolveRef(ctx context.Context, repoPath, ref string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return ResolveRef(repoPath, ref)
}

func (GitServiceAdapter) Merge(ctx context.Context, repoPath, base, head, method string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	sha, err := Merge(repoPath, base, head, method)
	if errors.Is(err, domainservice.ErrMergeConflict) {
		return "", domainservice.ErrMergeConflict
	}
	return sha, err
}

func (GitServiceAdapter) GetDiff(ctx context.Context, repoPath, base, head string, maxFiles int) ([]domainservice.FileDiff, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	diffs, err := GetDiff(repoPath, base, head)
	if err != nil {
		return nil, false, err
	}
	truncated := false
	if maxFiles > 0 && len(diffs) > maxFiles {
		diffs = diffs[:maxFiles]
		truncated = true
	}
	out := make([]domainservice.FileDiff, 0, len(diffs))
	for _, d := range diffs {
		out = append(out, domainservice.FileDiff{
			Filename:         d.NewPath,
			PreviousFilename: d.OldPath,
			Status:           d.Status,
			Patch:            d.Patch,
		})
	}
	return out, truncated, nil
}

func (GitServiceAdapter) GetMergeBase(ctx context.Context, repoPath, base, head string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return GetMergeBase(repoPath, base, head)
}
