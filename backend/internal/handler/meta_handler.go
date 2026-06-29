package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type BuildInfo struct {
	AppName     string
	Version     string
	GitCommit   string
	BuildDate   string
	LicenseName string
	SourceURL   string
}

type LicenseEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
	URL     string `json:"url"`
}

type MetaHandler struct {
	info       BuildInfo
	thirdParty []LicenseEntry
}

func NewMetaHandler(info BuildInfo, thirdParty []LicenseEntry) *MetaHandler {
	entries := make([]LicenseEntry, 0)
	if thirdParty != nil {
		entries = append(entries, thirdParty...)
	}
	return &MetaHandler{
		info:       info,
		thirdParty: entries,
	}
}

func (h *MetaHandler) GetMeta(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"app_name":   h.info.AppName,
		"version":    h.info.Version,
		"git_commit": h.info.GitCommit,
		"build_date": h.info.BuildDate,
		"license":    h.info.LicenseName,
		"source_url": h.info.SourceURL,
	})
}

type licensesResponse struct {
	AppLicense string         `json:"app_license"`
	ThirdParty []LicenseEntry `json:"third_party"`
}

func (h *MetaHandler) GetLicenses(c echo.Context) error {
	return c.JSON(http.StatusOK, licensesResponse{
		AppLicense: h.info.LicenseName,
		ThirdParty: h.thirdParty,
	})
}

func (h *MetaHandler) RegisterRoutes(e *echo.Echo) {
	e.GET("/api/meta", h.GetMeta)
	e.GET("/api/licenses", h.GetLicenses)
}
