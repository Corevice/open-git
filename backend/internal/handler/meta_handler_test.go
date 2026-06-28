package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
)

func TestGetMetaOK(t *testing.T) {
	e := echo.New()
	metaHandler := handler.NewMetaHandler(handler.BuildInfo{
		AppName:     "OpenGit",
		Version:     "1.0.0",
		GitCommit:   "abc1234",
		BuildDate:   "2025-01-01T00:00:00Z",
		LicenseName: "Apache-2.0",
		SourceURL:   "https://example.org/org/repo",
	}, nil)
	metaHandler.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"app_name", "version", "git_commit", "build_date", "license", "source_url"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("%s key missing", key)
		}
	}
}

func TestGetLicensesWithEntries(t *testing.T) {
	e := echo.New()
	entry := handler.LicenseEntry{
		Name:    "github.com/labstack/echo/v4",
		Version: "v4.11.0",
		License: "MIT",
		URL:     "https://github.com/labstack/echo",
	}
	metaHandler := handler.NewMetaHandler(handler.BuildInfo{
		LicenseName: "Apache-2.0",
	}, []handler.LicenseEntry{entry})
	metaHandler.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, "/api/licenses", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		AppLicense string                 `json:"app_license"`
		ThirdParty []handler.LicenseEntry `json:"third_party"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.AppLicense != "Apache-2.0" {
		t.Fatalf("app_license = %q, want Apache-2.0", body.AppLicense)
	}
	if len(body.ThirdParty) != 1 {
		t.Fatalf("third_party len = %d, want 1", len(body.ThirdParty))
	}
	if body.ThirdParty[0].Name != entry.Name {
		t.Fatalf("third_party[0].name = %q, want %q", body.ThirdParty[0].Name, entry.Name)
	}
}

func TestGetLicensesEmpty(t *testing.T) {
	e := echo.New()
	metaHandler := handler.NewMetaHandler(handler.BuildInfo{
		LicenseName: "Apache-2.0",
	}, nil)
	metaHandler.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, "/api/licenses", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	raw := rec.Body.String()
	if raw == "" {
		t.Fatal("response body is empty")
	}
	if !json.Valid([]byte(raw)) {
		t.Fatalf("response is not valid JSON: %s", raw)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	thirdParty, ok := body["third_party"]
	if !ok {
		t.Fatal("third_party key missing")
	}
	if string(thirdParty) != "[]" {
		t.Fatalf("third_party = %s, want []", string(thirdParty))
	}
}

func TestGetMetaDevVersion(t *testing.T) {
	e := echo.New()
	metaHandler := handler.NewMetaHandler(handler.BuildInfo{
		AppName: "OpenGit",
		Version: "dev",
	}, nil)
	metaHandler.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["version"] != "dev" {
		t.Fatalf("version = %q, want dev", body["version"])
	}
}
