package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestParsePaginationValidPageAndPerPage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?page=2&per_page=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		t.Fatalf("ParsePaginationParams: %v", err)
	}
	if page != 2 {
		t.Fatalf("page = %d, want 2", page)
	}
	if perPage != 10 {
		t.Fatalf("perPage = %d, want 10", perPage)
	}
}

func TestParsePaginationClampPerPage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?per_page=200", nil)
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

func TestParsePaginationZeroPerPageClampsToOne(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?per_page=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		t.Fatalf("ParsePaginationParams: %v", err)
	}
	if page != 1 {
		t.Fatalf("page = %d, want 1", page)
	}
	if perPage != 1 {
		t.Fatalf("perPage = %d, want 1", perPage)
	}
}

func TestParsePaginationZeroPageReturns422(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?page=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, _, err := middleware.ParsePaginationParams(c)
	assertPaginationValidationError(t, err, "page")
}

func TestParsePaginationNonNumericPageReturns422(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?page=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, _, err := middleware.ParsePaginationParams(c)
	assertPaginationValidationError(t, err, "page")
}

func TestParsePaginationNonNumericPerPageReturns422(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?per_page=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, _, err := middleware.ParsePaginationParams(c)
	assertPaginationValidationError(t, err, "per_page")
}

func TestBuildLinkHeaderEmptyWhenTotalZero(t *testing.T) {
	link := middleware.BuildLinkHeader("https://example.com/items", 1, 10, 0)
	if link != "" {
		t.Fatalf("BuildLinkHeader = %q, want empty string", link)
	}
}

func TestBuildLinkHeaderFirstPageContainsNextAndLast(t *testing.T) {
	link := middleware.BuildLinkHeader("https://example.com/items", 1, 10, 25)
	if link == "" {
		t.Fatal("expected non-empty Link header")
	}
	if !strings.Contains(link, `rel="next"`) {
		t.Fatalf("Link header missing rel=\"next\": %q", link)
	}
	if !strings.Contains(link, `rel="last"`) {
		t.Fatalf("Link header missing rel=\"last\": %q", link)
	}
	if strings.Contains(link, `rel="prev"`) {
		t.Fatalf("Link header should not contain rel=\"prev\": %q", link)
	}
}

func assertPaginationValidationError(t *testing.T, err error, wantField string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected validation error")
	}

	var he *echo.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *echo.HTTPError, got %T", err)
	}
	if he.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", he.Code, http.StatusUnprocessableEntity)
	}

	body, ok := he.Message.(map[string]any)
	if !ok {
		t.Fatalf("expected map message body, got %T", he.Message)
	}
	if body["message"] != "Validation Failed" {
		t.Fatalf("message = %v, want Validation Failed", body["message"])
	}

	errorsRaw, ok := body["errors"].([]map[string]string)
	if !ok {
		errorsAny, ok := body["errors"].([]any)
		if !ok || len(errorsAny) == 0 {
			t.Fatalf("expected errors array, got %v", body["errors"])
		}
		first, ok := errorsAny[0].(map[string]string)
		if !ok {
			t.Fatalf("expected map error entry, got %T", errorsAny[0])
		}
		if first["field"] != wantField {
			t.Fatalf("field = %q, want %q", first["field"], wantField)
		}
		return
	}
	if len(errorsRaw) == 0 {
		t.Fatal("expected at least one field error")
	}
	if errorsRaw[0]["field"] != wantField {
		t.Fatalf("field = %q, want %q", errorsRaw[0]["field"], wantField)
	}
	if errorsRaw[0]["resource"] != "pagination" {
		t.Fatalf("resource = %q, want pagination", errorsRaw[0]["resource"])
	}
	if errorsRaw[0]["code"] != "invalid" {
		t.Fatalf("code = %q, want invalid", errorsRaw[0]["code"])
	}
}
