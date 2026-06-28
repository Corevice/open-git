package compat_test

import (
	"os"
	"strings"
	"testing"

	"github.com/open-git/backend/internal/compat"
)

func TestLoadGoldenMissingFile(t *testing.T) {
	_, err := compat.LoadGolden("GET", "/missing-endpoint")
	if err == nil {
		t.Fatal("expected error for missing golden file")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected not exist error, got %v", err)
	}
}

func TestDiffGoldenIdentical(t *testing.T) {
	expected := []byte(`{"login":"octocat","id":1}`)
	actual := []byte(`{"login":"octocat","id":1}`)
	if diff := compat.DiffGolden(expected, actual); diff != "" {
		t.Fatalf("expected empty diff, got %q", diff)
	}
}

func TestDiffGoldenMissingField(t *testing.T) {
	expected := []byte(`{"login":"octocat","id":1,"type":"User"}`)
	actual := []byte(`{"login":"octocat","id":1}`)
	diff := compat.DiffGolden(expected, actual)
	if diff == "" {
		t.Fatal("expected diff for missing field")
	}
	if !strings.Contains(diff, "missing field: type") {
		t.Fatalf("expected missing field diff, got %q", diff)
	}
}

func TestDiffGoldenAddedField(t *testing.T) {
	expected := []byte(`{"login":"octocat"}`)
	actual := []byte(`{"login":"octocat","id":1}`)
	diff := compat.DiffGolden(expected, actual)
	if diff == "" {
		t.Fatal("expected diff for added field")
	}
	if !strings.Contains(diff, "added field: id") {
		t.Fatalf("expected added field diff, got %q", diff)
	}
}

func TestDiffGoldenTypeChanged(t *testing.T) {
	expected := []byte(`{"id":1}`)
	actual := []byte(`{"id":"1"}`)
	diff := compat.DiffGolden(expected, actual)
	if diff == "" {
		t.Fatal("expected diff for type change")
	}
	if !strings.Contains(diff, "type changed: id") {
		t.Fatalf("expected type changed diff, got %q", diff)
	}
}

func TestGoldenPath(t *testing.T) {
	got := compat.GoldenPath("GET", "/user")
	want := "testdata/golden/GET_user.json"
	if got != want {
		t.Fatalf("GoldenPath()=%q, want %q", got, want)
	}
}

func TestLoadGoldenExistingFile(t *testing.T) {
	data, err := compat.LoadGolden("GET", "/user")
	if err != nil {
		t.Fatalf("LoadGolden() error: %v", err)
	}
	if !strings.Contains(string(data), "octocat") {
		t.Fatalf("unexpected golden content: %s", data)
	}
}

func TestJUnitReport(t *testing.T) {
	results := map[string][]compat.CheckResult{
		"GET /user": {
			{Name: "status_code", Passed: true},
			{Name: "rate_limit_headers", Passed: false, Diff: "missing headers"},
		},
	}
	out := compat.JUnitReport("compat", results)
	if !strings.HasPrefix(string(out), "<?xml") {
		t.Fatalf("expected XML header, got %q", string(out[:min(len(out), 40)]))
	}
	if !strings.Contains(string(out), "<testsuites") {
		t.Fatal("expected testsuites element")
	}
}

func TestJSONReport(t *testing.T) {
	results := map[string][]compat.CheckResult{
		"GET /user": {
			{Name: "status_code", Passed: true},
		},
	}
	out := compat.JSONReport(10, 8, 1, 1, results)
	body := string(out)
	for _, field := range []string{"coverage_rate", "total_endpoints", "passing", "failing", "unimplemented"} {
		if !strings.Contains(body, field) {
			t.Fatalf("expected JSON field %q in %s", field, body)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
