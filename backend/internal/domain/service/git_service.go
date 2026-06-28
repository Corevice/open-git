package service

type FileDiff struct {
	Filename         string
	PreviousFilename string
	Status           string
	Patch            string
	Additions        int
	Deletions        int
	Binary           bool
}

type GitService interface {
	BranchExists(repoPath, branch string) (bool, error)
	ResolveRef(repoPath, ref string) (string, error)
	Merge(repoPath, base, head, method string) (string, error)
	GetDiff(repoPath, base, head string, maxFiles int) ([]FileDiff, bool, error)
	GetMergeBase(repoPath, base, head string) (string, error)
}
