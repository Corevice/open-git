package handler

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

var ownerRepoNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateOwnerRepo rejects malformed owner or repository path parameters.
func ValidateOwnerRepo(owner, repo string) error {
	for _, name := range []string{owner, repo} {
		if err := validateOwnerRepoName(name); err != nil {
			return err
		}
	}
	return nil
}

func validateOwnerRepoName(name string) error {
	if name == "" {
		return invalidOwnerRepoError()
	}
	if strings.Contains(name, "..") {
		return invalidOwnerRepoError()
	}
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return invalidOwnerRepoError()
	}
	if !ownerRepoNamePattern.MatchString(name) {
		return invalidOwnerRepoError()
	}
	return nil
}

func invalidOwnerRepoError() *echo.HTTPError {
	return echo.NewHTTPError(http.StatusBadRequest, map[string]string{
		"message": "Invalid owner or repository name",
	})
}
