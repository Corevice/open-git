package middleware

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	defaultPage    = 1
	defaultPerPage = 30
	maxPerPage     = 100
)

func paginationValidationError(field string) *echo.HTTPError {
	return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]any{
		"message": "Validation Failed",
		"errors": []map[string]string{{
			"resource": "pagination",
			"field":    field,
			"code":     "invalid",
		}},
	})
}

// ParsePaginationParams reads GitHub-style ?page and ?per_page query params.
func ParsePaginationParams(c echo.Context) (page, perPage int, err error) {
	page = defaultPage
	perPage = defaultPerPage

	if pageStr := c.QueryParam("page"); pageStr != "" {
		parsedPage, parseErr := strconv.Atoi(pageStr)
		if parseErr != nil {
			return 0, 0, paginationValidationError("page")
		}
		if parsedPage < 1 {
			return 0, 0, paginationValidationError("page")
		}
		page = parsedPage
	}

	if perPageStr := c.QueryParam("per_page"); perPageStr != "" {
		parsedPerPage, parseErr := strconv.Atoi(perPageStr)
		if parseErr != nil {
			return 0, 0, paginationValidationError("per_page")
		}
		if parsedPerPage < 1 {
			parsedPerPage = 1
		}
		if parsedPerPage > maxPerPage {
			parsedPerPage = maxPerPage
		}
		perPage = parsedPerPage
	}

	return page, perPage, nil
}

// BuildLinkHeader builds a GitHub-style Link header (rel=next/prev/last).
func BuildLinkHeader(base string, page, perPage, total int) string {
	if perPage <= 0 || total <= 0 {
		return ""
	}

	lastPage := (total + perPage - 1) / perPage
	if lastPage < 1 {
		lastPage = 1
	}

	u, err := url.Parse(base)
	if err != nil {
		return ""
	}

	query := u.Query()
	links := make([]string, 0, 4)

	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(perPage))
	u.RawQuery = query.Encode()
	links = append(links, `<`+u.String()+`>; rel="self"`)

	if page > 1 {
		query.Set("page", "1")
		u.RawQuery = query.Encode()
		links = append(links, `<`+u.String()+`>; rel="first"`)
		query.Set("page", strconv.Itoa(page-1))
		u.RawQuery = query.Encode()
		links = append(links, `<`+u.String()+`>; rel="prev"`)
	}
	if page < lastPage {
		query.Set("page", strconv.Itoa(page+1))
		u.RawQuery = query.Encode()
		links = append(links, `<`+u.String()+`>; rel="next"`)
		query.Set("page", strconv.Itoa(lastPage))
		u.RawQuery = query.Encode()
		links = append(links, `<`+u.String()+`>; rel="last"`)
	}

	return strings.Join(links, ", ")
}

// BuildAbsoluteLinkHeader builds a GitHub-style Link header with absolute URLs.
func BuildAbsoluteLinkHeader(c echo.Context, page, perPage, total int) string {
	req := c.Request()
	scheme := req.URL.Scheme
	if scheme == "" {
		scheme = c.Scheme()
	}
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}

	base := scheme + "://" + host + req.URL.RequestURI()
	return BuildLinkHeader(base, page, perPage, total)
}
