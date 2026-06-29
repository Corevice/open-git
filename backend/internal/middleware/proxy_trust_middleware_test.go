package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestSetupProxyTrust_EmptyCIDR(t *testing.T) {
	e := echo.New()
	if err := middleware.SetupProxyTrust(e, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.IPExtractor != nil {
		t.Fatal("expected IPExtractor to remain nil")
	}
}

func TestSetupProxyTrust_ValidCIDR(t *testing.T) {
	e := echo.New()
	if err := middleware.SetupProxyTrust(e, "172.16.0.0/12"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.IPExtractor == nil {
		t.Fatal("expected IPExtractor to be set")
	}
}

func TestSetupProxyTrust_MultipleCIDRs(t *testing.T) {
	e := echo.New()
	if err := middleware.SetupProxyTrust(e, "10.0.0.0/8, 172.16.0.0/12"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.IPExtractor == nil {
		t.Fatal("expected IPExtractor to be set")
	}
}

func TestSetupProxyTrust_InvalidCIDR(t *testing.T) {
	e := echo.New()
	err := middleware.SetupProxyTrust(e, "not-a-cidr")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not-a-cidr") {
		t.Fatalf("error %q should contain not-a-cidr", err.Error())
	}
}

func TestSetupProxyTrust_XFFTrusted(t *testing.T) {
	e := echo.New()
	if err := middleware.SetupProxyTrust(e, "172.16.0.0/12"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var gotIP string
	e.GET("/", func(c echo.Context) error {
		gotIP = c.RealIP()
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.RemoteAddr = "172.16.0.1:12345"

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if gotIP != "1.2.3.4" {
		t.Fatalf("RealIP() = %q, want %q", gotIP, "1.2.3.4")
	}
}
