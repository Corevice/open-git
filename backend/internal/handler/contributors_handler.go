package handler

import (
	"net/http"
	"sort"
	"strconv"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

// ContributorsHandler serves repository contributor aggregation endpoints.
type ContributorsHandler struct {
	resolver    GitRepositoryResolver
	memberships GitMembershipAccess
}

func NewContributorsHandler(resolver GitRepositoryResolver, memberships GitMembershipAccess) *ContributorsHandler {
	return &ContributorsHandler{resolver: resolver, memberships: memberships}
}

func (h *ContributorsHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/repos/:owner/:repo/contributors", h.GetContributors, middleware.OptionalAuth())
}

type contributorResponse struct {
	Login         string `json:"login"`
	ID            int64  `json:"id"`
	AvatarURL     string `json:"avatar_url"`
	Contributions int    `json:"contributions"`
	Type          string `json:"type"`
}

type contributorAgg struct {
	login         string
	contributions int
}

func (h *ContributorsHandler) GetContributors(c echo.Context) error {
	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	if resolved.Visibility == "private" {
		userUUID := middleware.UserUUIDFromContext(c)
		if userUUID == uuid.Nil {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		userID := middleware.UserIDFromContext(c)
		ok, err := h.memberships.HasReadAccess(c.Request().Context(), userID, resolved.OrganizationID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check permissions"})
		}
		if !ok {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
	}

	repo, err := gogit.PlainOpen(resolved.DiskPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	iter, _ := repo.Log(&gogit.LogOptions{})
	agg := make(map[string]*contributorAgg)
	if iter != nil {
		_ = iter.ForEach(func(commit *object.Commit) error {
			email := commit.Author.Email
			entry, ok := agg[email]
			if !ok {
				entry = &contributorAgg{login: commit.Author.Name}
				agg[email] = entry
			}
			entry.contributions++
			return nil
		})
	}

	contributors := make([]contributorResponse, 0, len(agg))
	for _, entry := range agg {
		contributors = append(contributors, contributorResponse{
			Login:         entry.login,
			ID:            0,
			AvatarURL:     "",
			Contributions: entry.contributions,
			Type:          "User",
		})
	}

	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Contributions > contributors[j].Contributions
	})

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

	total := len(contributors)
	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}

	base := c.Scheme() + "://" + c.Request().Host + c.Request().URL.Path
	if c.Request().URL.RawQuery != "" {
		u := *c.Request().URL
		qv := u.Query()
		qv.Del("page")
		qv.Set("per_page", strconv.Itoa(perPage))
		u.RawQuery = qv.Encode()
		base = c.Scheme() + "://" + c.Request().Host + u.Path
		if u.RawQuery != "" {
			base += "?" + u.RawQuery
		}
	} else {
		base += "?per_page=" + strconv.Itoa(perPage)
	}

	if link := middleware.BuildLinkHeader(base, page, perPage, total); link != "" {
		c.Response().Header().Set("Link", link)
	}

	return c.JSON(http.StatusOK, contributors[start:end])
}
