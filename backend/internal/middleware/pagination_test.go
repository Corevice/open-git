package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestParsePaginationDefaults(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		t.Fatalf("ParsePaginationParams: %v", err)
	}
	if page != 1 {
		t.Fatalf("page = %d, want 1", page)
	}
	if perPage != 30 {
		t.Fatalf("perPage = %d, want 30", perPage)
	}
}

func TestParsePaginationClampPerPage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?per_page=150", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		t.Fatalf("ParsePaginationParams: %v", err)
	}
	if page != 1 {
		t.Fatalf("page = %d, want 1", page)
	}
	if perPage != 100 {
		t.Fatalf("perPage = %d, want 100", perPage)
	}
}

func TestParsePaginationNegativePerPage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?per_page=-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, _, err := middleware.ParsePaginationParams(c)
	if err == nil {
		t.Fatal("expected error for negative per_page")
	}

	var he *echo.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", he.Code, http.StatusUnprocessableEntity)
	}
}

func TestParsePaginationNonNumeric(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?per_page=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, _, err := middleware.ParsePaginationParams(c)
	if err == nil {
		t.Fatal("expected error for non-numeric per_page")
	}

	var he *echo.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", he.Code, http.StatusUnprocessableEntity)
	}
}
