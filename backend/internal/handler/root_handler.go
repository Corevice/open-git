package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type RootHandler struct{}

func NewRootHandler() *RootHandler {
	return &RootHandler{}
}

func (h *RootHandler) Get(c echo.Context) error {
	base := "https://" + c.Request().Host
	return c.JSON(http.StatusOK, map[string]string{
		"current_user_url":   base + "/user",
		"repository_url":     base + "/repos/{owner}/{repo}",
		"repositories_url":   base + "/user/repos",
		"issues_url":         base + "/repos/{owner}/{repo}/issues",
		"pulls_url":          base + "/repos/{owner}/{repo}/pulls",
	})
}
