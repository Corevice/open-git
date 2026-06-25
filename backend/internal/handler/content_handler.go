package handler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/labstack/echo/v4"

	infragit "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/middleware"
)

const maxContentBytes = 1 << 20 // 1MB

// ContentHandler serves repository content browsing endpoints.
type ContentHandler struct {
	resolver GitRepositoryResolver
}

func NewContentHandler(resolver GitRepositoryResolver) *ContentHandler {
	return &ContentHandler{resolver: resolver}
}

func (h *ContentHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/repos/:owner/:repo/contents", h.GetContents, middleware.OptionalAuth())
	g.GET("/repos/:owner/:repo/git/blobs/:sha", h.GetGitBlob, middleware.OptionalAuth())
	g.GET("/repos/:owner/:repo/commits", h.GetCommits, middleware.OptionalAuth())
	g.GET("/repos/:owner/:repo/branches", h.GetBranches, middleware.OptionalAuth())
	g.GET("/repos/:owner/:repo/tags", h.GetTags, middleware.OptionalAuth())
	g.GET("/repos/:owner/:repo/commits/:sha", h.GetCommitDetail, middleware.OptionalAuth())
}

type contentItemResponse struct {
	Name        string  `json:"name"`
	Path        string  `json:"path"`
	SHA         string  `json:"sha"`
	Size        int64   `json:"size"`
	Type        string  `json:"type"`
	Content     *string `json:"content,omitempty"`
	Encoding    string  `json:"encoding,omitempty"`
	Truncated   bool    `json:"truncated,omitempty"`
	RawURL      string  `json:"raw_url,omitempty"`
	DownloadURL string  `json:"download_url,omitempty"`
}

type blobResponse struct {
	SHA       string `json:"sha"`
	Size      int64  `json:"size"`
	Content   string `json:"content"`
	Encoding  string `json:"encoding"`
	Truncated bool   `json:"truncated,omitempty"`
	RawURL    string `json:"raw_url,omitempty"`
}

type commitAuthorResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type commitResponse struct {
	SHA    string               `json:"sha"`
	Commit commitDetailResponse `json:"commit"`
}

type commitDetailResponse struct {
	Message string               `json:"message"`
	Author  commitAuthorResponse `json:"author"`
}

type branchCommitResponse struct {
	SHA string `json:"sha"`
}

type branchResponse struct {
	Name   string               `json:"name"`
	Commit branchCommitResponse `json:"commit"`
}

type tagResponse struct {
	Name   string               `json:"name"`
	Commit branchCommitResponse `json:"commit"`
}

type filesResponse struct {
	Filename  string  `json:"filename"`
	Status    string  `json:"status"`
	Additions int     `json:"additions"`
	Deletions int     `json:"deletions"`
	Patch     *string `json:"patch"`
	TooLarge  bool    `json:"too_large,omitempty"`
}

type statsResponse struct {
	Total     int `json:"total"`
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
}

type commitDetailFullResponse struct {
	SHA    string               `json:"sha"`
	Commit commitDetailResponse `json:"commit"`
	Files  []filesResponse      `json:"files"`
	Stats  statsResponse        `json:"stats"`
}

func isPathSafe(p string) bool {
	if strings.Contains(p, "..") {
		return false
	}
	if strings.Contains(p, "\x00") {
		return false
	}
	if strings.HasPrefix(p, "/") {
		return false
	}
	return true
}

func (h *ContentHandler) GetContents(c echo.Context) error {
	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	ref := c.QueryParam("ref")
	if ref == "" {
		ref = "HEAD"
	}
	path := c.QueryParam("path")
	if !isPathSafe(path) {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid path"})
	}

	entries, err := infragit.GetTree(resolved.DiskPath, ref, path)
	if err != nil {
		if errors.Is(err, infragit.ErrEmptyRepository) {
			return c.JSON(http.StatusOK, []contentItemResponse{})
		}
		if errors.Is(err, infragit.ErrPathNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": err.Error()})
	}

	if len(entries) == 1 && entries[0].Type == infragit.TreeEntryTypeFile {
		return h.respondFileContent(c, resolved, ref, entries[0])
	}

	items := make([]contentItemResponse, 0, len(entries))
	for _, e := range entries {
		items = append(items, contentItemResponse{
			Name: e.Name,
			Path: e.Path,
			SHA:  e.SHA,
			Size: e.Size,
			Type: e.Type,
		})
	}
	return c.JSON(http.StatusOK, items)
}

func (h *ContentHandler) respondFileContent(c echo.Context, resolved *ResolvedGitRepository, ref string, entry infragit.TreeEntry) error {
	data, size, err := infragit.GetBlob(resolved.DiskPath, ref, entry.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": err.Error()})
	}

	rawURL := rawBlobURL(c, resolved, entry.SHA)
	resp := contentItemResponse{
		Name:        entry.Name,
		Path:        entry.Path,
		SHA:         entry.SHA,
		Size:        size,
		Type:        infragit.TreeEntryTypeFile,
		Encoding:    "base64",
		RawURL:      rawURL,
		DownloadURL: rawURL,
	}

	if size > maxContentBytes {
		resp.Truncated = true
		resp.Content = nil
	} else {
		encoded := encodeContent(data)
		resp.Content = &encoded
		if isBinaryContent(data) {
			resp.Encoding = "base64"
		}
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *ContentHandler) GetGitBlob(c echo.Context) error {
	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	sha := c.Param("sha")
	data, size, err := infragit.GetBlobBySHA(resolved.DiskPath, sha)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	rawURL := rawBlobURL(c, resolved, sha)
	resp := blobResponse{
		SHA:      sha,
		Size:     size,
		Encoding: "base64",
		RawURL:   rawURL,
	}

	if size > maxContentBytes {
		resp.Truncated = true
		preview := data[:maxContentBytes]
		resp.Content = base64.StdEncoding.EncodeToString(preview)
	} else {
		resp.Content = encodeContent(data)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *ContentHandler) GetCommits(c echo.Context) error {
	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	branch := c.QueryParam("sha")
	if branch == "" {
		branch = "HEAD"
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	commits, total, err := infragit.GetCommits(resolved.DiskPath, branch, page, perPage)
	if err != nil {
		if errors.Is(err, infragit.ErrEmptyRepository) {
			return c.JSON(http.StatusOK, []commitResponse{})
		}
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": err.Error()})
	}

	base := c.Scheme() + "://" + c.Request().Host + c.Request().URL.Path
	if q := c.Request().URL.RawQuery; q != "" {
		u := *c.Request().URL
		qv := u.Query()
		qv.Del("page")
		qv.Set("per_page", strconv.Itoa(perPage))
		if branch != "" && branch != "HEAD" {
			qv.Set("sha", branch)
		}
		u.RawQuery = qv.Encode()
		base = c.Scheme() + "://" + c.Request().Host + u.Path
		if u.RawQuery != "" {
			base += "?" + u.RawQuery
		}
	} else {
		params := []string{"per_page=" + strconv.Itoa(perPage)}
		if branch != "" && branch != "HEAD" {
			params = append(params, "sha="+branch)
		}
		base += "?" + strings.Join(params, "&")
	}

	if link := middleware.BuildLinkHeader(base, page, perPage, total); link != "" {
		c.Response().Header().Set("Link", link)
	}

	out := make([]commitResponse, 0, len(commits))
	for _, cm := range commits {
		out = append(out, commitResponse{
			SHA: cm.SHA,
			Commit: commitDetailResponse{
				Message: cm.Message,
				Author: commitAuthorResponse{
					Name: cm.Author,
					Date: cm.Date.UTC().Format("2006-01-02T15:04:05Z"),
				},
			},
		})
	}
	return c.JSON(http.StatusOK, out)
}

func (h *ContentHandler) GetBranches(c echo.Context) error {
	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	branches, err := infragit.GetBranches(resolved.DiskPath)
	if err != nil {
		if errors.Is(err, infragit.ErrEmptyRepository) {
			return c.JSON(http.StatusOK, []branchResponse{})
		}
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": err.Error()})
	}

	out := make([]branchResponse, 0, len(branches))
	for _, b := range branches {
		out = append(out, branchResponse{
			Name: b.Name,
			Commit: branchCommitResponse{
				SHA: b.SHA,
			},
		})
	}
	return c.JSON(http.StatusOK, out)
}

func (h *ContentHandler) GetTags(c echo.Context) error {
	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	tags, err := infragit.GetTags(resolved.DiskPath)
	if err != nil {
		if errors.Is(err, infragit.ErrEmptyRepository) {
			return c.JSON(http.StatusOK, []tagResponse{})
		}
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": err.Error()})
	}

	out := make([]tagResponse, 0, len(tags))
	for _, t := range tags {
		out = append(out, tagResponse{
			Name: t.Name,
			Commit: branchCommitResponse{
				SHA: t.SHA,
			},
		})
	}
	return c.JSON(http.StatusOK, out)
}

func (h *ContentHandler) GetCommitDetail(c echo.Context) error {
	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	detail, err := infragit.GetCommitDetail(resolved.DiskPath, c.Param("sha"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	files := make([]filesResponse, 0, len(detail.Files))
	for _, f := range detail.Files {
		file := filesResponse{
			Filename:  f.Filename,
			Status:    f.Status,
			Additions: f.Additions,
			Deletions: f.Deletions,
			Patch:     f.Patch,
		}
		if f.Patch == nil {
			file.TooLarge = true
		}
		files = append(files, file)
	}

	return c.JSON(http.StatusOK, commitDetailFullResponse{
		SHA: detail.SHA,
		Commit: commitDetailResponse{
			Message: detail.Message,
			Author: commitAuthorResponse{
				Name:  detail.Author,
				Email: detail.Email,
				Date:  detail.Date.UTC().Format("2006-01-02T15:04:05Z"),
			},
		},
		Files: files,
		Stats: statsResponse{
			Total:     detail.Stats.Total,
			Additions: detail.Stats.Additions,
			Deletions: detail.Stats.Deletions,
		},
	})
}

func encodeContent(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func isBinaryContent(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return !utf8.Valid(data)
}

func rawBlobURL(c echo.Context, resolved *ResolvedGitRepository, sha string) string {
	owner := c.Param("owner")
	repo := c.Param("repo")
	return c.Scheme() + "://" + c.Request().Host + "/repos/" + owner + "/" + repo + "/git/blobs/" + sha
}
