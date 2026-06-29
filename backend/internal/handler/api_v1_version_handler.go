package handler

import (
	"github.com/labstack/echo/v4"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

type APIV1VersionHandler struct{}

func NewAPIV1VersionHandler() *APIV1VersionHandler {
	return &APIV1VersionHandler{}
}

func (h *APIV1VersionHandler) Handle(c echo.Context) error {
	return RespondOK(c, map[string]string{
		"version":   Version,
		"commit":    Commit,
		"buildDate": BuildDate,
	})
}
