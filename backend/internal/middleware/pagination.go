package middleware

import (
	"net/url"
	"strconv"
	"strings"
)

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
