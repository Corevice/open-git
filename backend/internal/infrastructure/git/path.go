package git

import (
	"errors"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

var gitPathSegmentRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func SafeRepoPath(root, ownerLogin, repoName string) (string, error) {
	if ownerLogin == "" || repoName == "" {
		return "", errors.New("invalid repository path")
	}
	if strings.Contains(ownerLogin, "\x00") || strings.Contains(repoName, "\x00") {
		return "", errors.New("invalid repository path")
	}

	decodedOwner, err := url.PathUnescape(ownerLogin)
	if err != nil {
		return "", errors.New("invalid repository path")
	}
	decodedRepo, err := url.PathUnescape(repoName)
	if err != nil {
		return "", errors.New("invalid repository path")
	}
	decodedRepo = strings.TrimSuffix(decodedRepo, ".git")

	if !gitPathSegmentRegex.MatchString(decodedOwner) || !gitPathSegmentRegex.MatchString(decodedRepo) {
		return "", errors.New("invalid repository path")
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	diskPath := filepath.Join(rootAbs, decodedOwner, decodedRepo+".git")
	cleanPath := filepath.Clean(diskPath)
	if cleanPath != rootAbs && !strings.HasPrefix(cleanPath, rootAbs+string(filepath.Separator)) {
		return "", errors.New("invalid repository path")
	}
	return cleanPath, nil
}
