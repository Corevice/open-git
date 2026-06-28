package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type GitHubFieldError struct {
	Resource string `json:"resource"`
	Field    string `json:"field"`
	Code     string `json:"code"`
}

type GitHubError struct {
	Message          string             `json:"message"`
	DocumentationURL string             `json:"documentation_url,omitempty"`
	Errors           []GitHubFieldError `json:"errors,omitempty"`
}

func RespondGitHubError(c echo.Context, status int, message, docsURL string, fieldErrors []GitHubFieldError) error {
	body := GitHubError{Message: message}
	if docsURL != "" {
		body.DocumentationURL = docsURL
	}
	if len(fieldErrors) > 0 {
		body.Errors = fieldErrors
	}
	return c.JSON(status, body)
}

func setGitHubJSONHeaders(c echo.Context, data any) {
	c.Response().Header().Set("X-GitHub-Media-Type", "github.v3; format=json")
	if data == nil {
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	hash := md5.Sum(b)
	c.Response().Header().Set("ETag", fmt.Sprintf(`W/"%s"`, hex.EncodeToString(hash[:])))
}

func RespondGitHubOK(c echo.Context, data any) error {
	setGitHubJSONHeaders(c, data)
	return c.JSON(http.StatusOK, data)
}

func RespondGitHubCreated(c echo.Context, data any) error {
	setGitHubJSONHeaders(c, data)
	return c.JSON(http.StatusCreated, data)
}

func RespondGitHubNotFound(c echo.Context, docsURL string) error {
	return RespondGitHubError(c, http.StatusNotFound, "Not Found", docsURL, nil)
}

func RespondGitHubValidationFailed(c echo.Context, docsURL string, fieldErrors []GitHubFieldError) error {
	return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", docsURL, fieldErrors)
}
