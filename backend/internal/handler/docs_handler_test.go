package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
	docsuc "github.com/open-git/backend/internal/usecase/docs"
)

func TestDocsHandler(t *testing.T) {
	dir := t.TempDir()
	content := "# Contributing\n\n## Section One\nFirst section body.\n\n## Section Two\nSecond section body.\n"
	if err := os.WriteFile(filepath.Join(dir, "CONTRIBUTING.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write CONTRIBUTING.md: %v", err)
	}

	treeUC := docsuc.NewGetDocTreeUsecase(dir)
	sectionUC := docsuc.NewGetDocSectionUsecase(treeUC)
	docsHandler := handler.NewDocsHandler(treeUC, sectionUC, "https://example.com/edit/main")

	e := echo.New()
	v1 := e.Group("/api/v1")
	docsHandler.RegisterRoutes(v1)

	t.Run("tree", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/docs/contributing", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, "section-one") || !strings.Contains(body, "section-two") {
			t.Fatalf("expected both slugs in response, got %s", body)
		}
	})

	t.Run("section", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/docs/contributing/section-one", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		contentMarkdown, _ := resp["content_markdown"].(string)
		if contentMarkdown == "" {
			t.Fatalf("expected non-empty content_markdown")
		}
		editURL, _ := resp["edit_url"].(string)
		if editURL == "" {
			t.Fatalf("expected non-empty edit_url")
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/docs/contributing/does-not-exist", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid slug", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/docs/contributing/INVALID", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})
}
