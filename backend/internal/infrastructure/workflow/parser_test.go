package workflow

import (
	"errors"
	"testing"
)

func TestParseValidWorkflow(t *testing.T) {
	yamlSrc := []byte(`name: CI
on: [push, pull_request]
jobs:
  build:
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Run tests
        run: go test ./...
  lint:
    steps:
      - name: Lint
        run: golangci-lint run
`)

	wf, err := ParseWorkflow(yamlSrc)
	if err != nil {
		t.Fatalf("ParseWorkflow returned unexpected error: %v", err)
	}
	if wf == nil {
		t.Fatal("ParseWorkflow returned nil workflow")
	}
	if wf.Name != "CI" {
		t.Errorf("Name: got %q, want %q", wf.Name, "CI")
	}
	if len(wf.Jobs) != 2 {
		t.Fatalf("Jobs: got %d, want 2", len(wf.Jobs))
	}
	build, ok := wf.Jobs["build"]
	if !ok {
		t.Fatal("expected job 'build' to be present")
	}
	if len(build.Steps) != 2 {
		t.Fatalf("build steps: got %d, want 2", len(build.Steps))
	}
	if build.Steps[0].Uses != "actions/checkout@v4" {
		t.Errorf("first step uses: got %q, want %q", build.Steps[0].Uses, "actions/checkout@v4")
	}
	if build.Steps[1].Run != "go test ./..." {
		t.Errorf("second step run: got %q, want %q", build.Steps[1].Run, "go test ./...")
	}
}

func TestParseInvalidYAML(t *testing.T) {
	// Unclosed flow sequence — produces a definite yaml syntax error with line info.
	yamlSrc := []byte("name: CI\njobs:\n  build:\n    steps: [\n      - run: echo ok\n")

	wf, err := ParseWorkflow(yamlSrc)
	if err == nil {
		t.Fatal("expected ParseWorkflow to return error for malformed YAML")
	}
	if wf != nil {
		t.Fatal("expected nil workflow on parse error")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if pe.Line <= 0 {
		t.Errorf("expected ParseError.Line > 0, got %d (msg=%s)", pe.Line, pe.Message)
	}
}

func TestParseEmptyDocument(t *testing.T) {
	_, err := ParseWorkflow([]byte(""))
	if err == nil {
		t.Fatal("expected error for empty document")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T", err)
	}
}

func TestParseMissingJobs(t *testing.T) {
	yamlSrc := []byte("name: CI\non: push\n")
	_, err := ParseWorkflow(yamlSrc)
	if err == nil {
		t.Fatal("expected error when jobs is missing")
	}
}
