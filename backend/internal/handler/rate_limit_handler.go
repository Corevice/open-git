package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type RateLimitHandler struct{}

func NewRateLimitHandler() *RateLimitHandler {
	return &RateLimitHandler{}
}

func (h *RateLimitHandler) Get(c echo.Context) error {
	reset := time.Now().UTC().Add(time.Hour).Unix()
	limitEntry := map[string]any{
		"limit":     5000,
		"remaining": 4999,
		"reset":     reset,
		"used":      1,
	}
	return c.JSON(http.StatusOK, map[string]any{
		"resources": map[string]any{
			"core": limitEntry,
		},
		"rate": limitEntry,
	})
}
