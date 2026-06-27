package handler

import (
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

func RespondGitHubError(c echo.Context, status int, message string, fieldErrors []GitHubFieldError) error {
	body := GitHubError{Message: message}
	if len(fieldErrors) > 0 {
		body.Errors = fieldErrors
	}
	return c.JSON(status, body)
}

func RespondGitHubOK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, data)
}
