package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type AutoInitOpts struct {
	Readme            string
	GitIgnoreTemplate string
	LicenseTemplate   string
}

func AutoInitRepository(bareRepoPath string, opts AutoInitOpts) error {
	if err := os.MkdirAll(filepath.Dir(bareRepoPath), 0o755); err != nil {
		return fmt.Errorf("create bare repo parent dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "open-git-autoinit-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := gogit.PlainInitWithOptions(tmpDir, &gogit.PlainInitOptions{
		InitBranch: plumbing.NewBranchReferenceName("main").Short(),
	})
	if err != nil {
		return fmt.Errorf("init working repo: %w", err)
	}

	wroteFiles := false
	if opts.Readme != "" {
		if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(readmeContent(opts.Readme)), 0o644); err != nil {
			return fmt.Errorf("write README.md: %w", err)
		}
		wroteFiles = true
	}
	if opts.GitIgnoreTemplate != "" {
		if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignoreContent(opts.GitIgnoreTemplate)), 0o644); err != nil {
			return fmt.Errorf("write .gitignore: %w", err)
		}
		wroteFiles = true
	}
	if opts.LicenseTemplate != "" {
		if err := os.WriteFile(filepath.Join(tmpDir, "LICENSE"), []byte(licenseContent(opts.LicenseTemplate)), 0o644); err != nil {
			return fmt.Errorf("write LICENSE: %w", err)
		}
		wroteFiles = true
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("open worktree: %w", err)
	}

	if wroteFiles {
		if _, err := wt.Add("."); err != nil {
			return fmt.Errorf("stage files: %w", err)
		}
	}

	author := &object.Signature{
		Name:  "OpenHub",
		Email: "noreply@localhost",
		When:  time.Now().UTC(),
	}
	if _, err := wt.Commit("Initial commit", &gogit.CommitOptions{
		Author:            author,
		Committer:         author,
		AllowEmptyCommits: !wroteFiles,
	}); err != nil {
		return fmt.Errorf("create initial commit: %w", err)
	}

	absTmpDir, err := filepath.Abs(tmpDir)
	if err != nil {
		return fmt.Errorf("resolve temp dir: %w", err)
	}

	if _, err := gogit.PlainClone(bareRepoPath, true, &gogit.CloneOptions{
		URL:           "file://" + filepath.ToSlash(absTmpDir),
		ReferenceName: plumbing.NewBranchReferenceName("main"),
		SingleBranch:  true,
	}); err != nil {
		return fmt.Errorf("clone bare repo: %w", err)
	}

	return nil
}

func readmeContent(name string) string {
	title := strings.TrimSpace(name)
	if title == "" {
		title = "Repository"
	}
	return fmt.Sprintf("# %s\n\nInitial repository.\n", title)
}

func gitignoreContent(template string) string {
	switch strings.ToLower(strings.TrimSpace(template)) {
	case "go":
		return "# Binaries\n*.exe\n*.exe~\n*.dll\n*.so\n*.dylib\n\n# Test binary\ntest\n*.test\n\n# Output\n*.out\n"
	default:
		return fmt.Sprintf("# %s gitignore template\n", template)
	}
}

func licenseContent(template string) string {
	switch strings.ToLower(strings.TrimSpace(template)) {
	case "mit":
		return "MIT License\n\nCopyright (c) OpenHub\n\nPermission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the \"Software\"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:\n"
	default:
		return fmt.Sprintf("%s license template\n", template)
	}
}
