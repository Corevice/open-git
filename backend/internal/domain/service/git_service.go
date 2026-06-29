package service

import (
	"context"
	"errors"
)

var ErrMergeConflict = errors.New("merge conflict")

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
	BranchExists(ctx context.Context, repoPath, branch string) (bool, error)
	ResolveRef(ctx context.Context, repoPath, ref string) (string, error)
	Merge(ctx context.Context, repoPath, base, head, method string) (string, error)
	GetDiff(ctx context.Context, repoPath, base, head string, maxFiles int) ([]FileDiff, bool, error)
	GetMergeBase(ctx context.Context, repoPath, base, head string) (string, error)
}
