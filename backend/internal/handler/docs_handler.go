package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	docsuc "github.com/open-git/backend/internal/usecase/docs"
)

// DocsHandler serves CONTRIBUTING.md section endpoints.
type DocsHandler struct {
	getTree     *docsuc.GetDocTreeUsecase
	getSection  *docsuc.GetDocSectionUsecase
	editBaseURL string
}

func NewDocsHandler(
	getTree *docsuc.GetDocTreeUsecase,
	getSection *docsuc.GetDocSectionUsecase,
	editBaseURL string,
) *DocsHandler {
	return &DocsHandler{
		getTree:     getTree,
		getSection:  getSection,
		editBaseURL: editBaseURL,
	}
}

func (h *DocsHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/docs/contributing", h.GetContributingTree)
	g.GET("/docs/contributing/:slug", h.GetContributingSection)
}

type docSectionTreeItem struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Order int    `json:"order"`
}

type docTreeResponse struct {
	Sections []docSectionTreeItem `json:"sections"`
}

type docSectionResponse struct {
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	ContentMarkdown string `json:"content_markdown"`
	UpdatedAt       string `json:"updated_at"`
	EditURL         string `json:"edit_url"`
}

func (h *DocsHandler) GetContributingTree(c echo.Context) error {
	sections, err := h.getTree.Execute(c.Request().Context())
	if err != nil {
		return err
	}

	items := make([]docSectionTreeItem, 0, len(sections))
	for _, section := range sections {
		items = append(items, docSectionTreeItem{
			Slug:  section.Slug,
			Title: section.Title,
			Order: section.Order,
		})
	}
	return c.JSON(http.StatusOK, docTreeResponse{Sections: items})
}

func (h *DocsHandler) GetContributingSection(c echo.Context) error {
	slug := c.Param("slug")

	section, err := h.getSection.Execute(c.Request().Context(), slug)
	if err != nil {
		if errors.Is(err, docsuc.ErrInvalidSlug) {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}
		if errors.Is(err, domain.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"message":           "Not Found",
				"documentation_url": "/api/v1/docs/contributing",
			})
		}
		return err
	}

	updatedAt, err := h.getTree.UpdatedAt()
	if err != nil {
		return err
	}

	editURL := ""
	if h.editBaseURL != "" {
		editURL = h.editBaseURL + "/CONTRIBUTING.md"
	}

	return c.JSON(http.StatusOK, docSectionResponse{
		Slug:            section.Slug,
		Title:           section.Title,
		ContentMarkdown: section.Content,
		UpdatedAt:       updatedAt.Format(time.RFC3339),
		EditURL:         editURL,
	})
}
