package handler

import (
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// IsForcePushForTest exposes the unexported isForcePush helper for unit tests
// in the handler_test package.
func IsForcePushForTest(repo *gogit.Repository, oldHash, newHash plumbing.Hash) (bool, error) {
	return isForcePush(repo, oldHash, newHash)
}
