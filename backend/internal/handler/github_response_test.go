package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
)

func TestRespondGitHubError401(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.RespondGitHubError(c, http.StatusUnauthorized, "Bad credentials", nil); err != nil {
		t.Fatalf("RespondGitHubError: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("body keys = %d, want 2", len(body))
	}
	if _, ok := body["message"]; !ok {
		t.Fatalf("missing message key in %s", rec.Body.String())
	}
	if _, ok := body["documentation_url"]; !ok {
		t.Fatalf("missing documentation_url key in %s", rec.Body.String())
	}
	if _, ok := body["errors"]; ok {
		t.Fatalf("unexpected errors key in %s", rec.Body.String())
	}

	var message string
	if err := json.Unmarshal(body["message"], &message); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if message != "Bad credentials" {
		t.Fatalf("message = %q, want %q", message, "Bad credentials")
	}

	var docURL string
	if err := json.Unmarshal(body["documentation_url"], &docURL); err != nil {
		t.Fatalf("unmarshal documentation_url: %v", err)
	}
	if docURL != "https://docs.github.com/rest" {
		t.Fatalf("documentation_url = %q, want %q", docURL, "https://docs.github.com/rest")
	}
}

func TestRespondGitHubError422WithFieldErrors(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	fieldErrors := []handler.GitHubFieldError{
		{Resource: "Repository", Field: "name", Code: "already_exists"},
	}

	if err := handler.RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", fieldErrors); err != nil {
		t.Fatalf("RespondGitHubError: %v", err)
	}

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}

	var body struct {
		Message string                    `json:"message"`
		Errors  []handler.GitHubFieldError `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Message != "Validation Failed" {
		t.Fatalf("message = %q, want %q", body.Message, "Validation Failed")
	}
	if len(body.Errors) != 1 {
		t.Fatalf("errors len = %d, want 1", len(body.Errors))
	}
	if body.Errors[0].Resource != "Repository" || body.Errors[0].Field != "name" || body.Errors[0].Code != "already_exists" {
		t.Fatalf("errors[0] = %+v, want {Resource:Repository Field:name Code:already_exists}", body.Errors[0])
	}
}

func TestRespondGitHubOK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.RespondGitHubOK(c, map[string]string{"login": "alice"}); err != nil {
		t.Fatalf("RespondGitHubOK: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["login"] != "alice" {
		t.Fatalf("login = %q, want %q", body["login"], "alice")
	}
	if _, ok := body["data"]; ok {
		t.Fatalf("unexpected data wrapper in %s", rec.Body.String())
	}
}
