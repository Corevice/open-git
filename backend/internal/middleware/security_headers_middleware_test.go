package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestSecurityHeaders_SetsAllHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	nextCalled := false
	handler := middleware.SecurityHeaders()(func(c echo.Context) error {
		nextCalled = true
		return c.NoContent(http.StatusOK)
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	headerChecks := map[string]string{
		echo.HeaderStrictTransportSecurity: "max-age=31536000; includeSubDomains",
		echo.HeaderXContentTypeOptions:     "nosniff",
		echo.HeaderXFrameOptions:           "DENY",
		echo.HeaderXXSSProtection:          "1; mode=block",
		echo.HeaderReferrerPolicy:          "strict-origin-when-cross-origin",
	}
	for name, want := range headerChecks {
		if got := rec.Header().Get(name); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}
