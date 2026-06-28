package workflow

import (
	"errors"
	"strings"
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

func TestBackwardCompatParseWorkflow(t *testing.T) {
	t.Run("valid", TestParseValidWorkflow)
	t.Run("invalidYAML", TestParseInvalidYAML)
	t.Run("emptyDocument", TestParseEmptyDocument)
	t.Run("missingJobs", TestParseMissingJobs)
}

func TestParseWorkflowFull_FullSchema(t *testing.T) {
	yamlSrc := []byte(`name: Full CI
on:
  push:
    branches: [main]
env:
  NODE_ENV: test
jobs:
  build:
    runs-on: ubuntu-latest
    needs: [lint]
    if: github.ref == 'refs/heads/main'
    env:
      BUILD: "1"
    outputs:
      version: ${{ steps.ver.outputs.version }}
    timeout-minutes: 30
    container: node:18
    services:
      postgres: postgres:14
    strategy:
      matrix:
        node: ["18", "20"]
    steps:
      - id: checkout
        name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
        env:
          TOKEN: secret
        if: success()
  lint:
    runs-on: ubuntu-latest
    steps:
      - run: npm run lint
`)

	ir, diags, err := ParseWorkflowFull(yamlSrc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ir == nil {
		t.Fatal("expected non-nil IR")
	}
	if ir.On["push"] == nil {
		t.Error("expected push trigger in on map")
	}
	if ir.Env["NODE_ENV"] != "test" {
		t.Errorf("env NODE_ENV: got %q, want test", ir.Env["NODE_ENV"])
	}

	build, ok := ir.Jobs["build"]
	if !ok {
		t.Fatal("expected build job")
	}
	if build.RunsOn != "ubuntu-latest" {
		t.Errorf("runs-on: got %q, want ubuntu-latest", build.RunsOn)
	}
	if len(build.Needs) != 1 || build.Needs[0] != "lint" {
		t.Errorf("needs: got %v, want [lint]", build.Needs)
	}
	if build.Env["BUILD"] != "1" {
		t.Errorf("job env BUILD: got %q", build.Env["BUILD"])
	}
	if len(build.MatrixExpansion) != 2 {
		t.Errorf("matrix expansion: got %d combos, want 2", len(build.MatrixExpansion))
	}
	if len(build.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(build.Steps))
	}
	step := build.Steps[0]
	if step.ID != "checkout" {
		t.Errorf("step id: got %q", step.ID)
	}
	if step.UsesRef == nil || step.UsesRef.Kind != "remote" {
		t.Error("expected remote uses ref")
	}
	if len(step.IfAST) == 0 {
		t.Error("expected if expression AST on step")
	}

	hasError := false
	for _, d := range diags {
		if d.Severity == "error" {
			hasError = true
		}
	}
	if hasError {
		t.Errorf("unexpected error diagnostics: %v", diags)
	}
}

func TestOnTrigger_StringForm(t *testing.T) {
	yamlSrc := []byte(`on: push
jobs:
  j:
    steps:
      - run: echo ok
`)
	ir, _, _ := ParseWorkflowFull(yamlSrc)
	if ir == nil {
		t.Fatal("expected IR")
	}
	if _, ok := ir.On["push"]; !ok {
		t.Errorf("on map: got %v, want push key", ir.On)
	}
}

func TestOnTrigger_ArrayForm(t *testing.T) {
	yamlSrc := []byte(`on: [push, pull_request]
jobs:
  j:
    steps:
      - run: echo ok
`)
	ir, _, _ := ParseWorkflowFull(yamlSrc)
	if ir == nil {
		t.Fatal("expected IR")
	}
	if _, ok := ir.On["push"]; !ok {
		t.Error("expected push key")
	}
	if _, ok := ir.On["pull_request"]; !ok {
		t.Error("expected pull_request key")
	}
}

func TestOnTrigger_MapForm(t *testing.T) {
	yamlSrc := []byte(`on:
  push:
    branches: [main]
jobs:
  j:
    steps:
      - run: echo ok
`)
	ir, _, _ := ParseWorkflowFull(yamlSrc)
	if ir == nil {
		t.Fatal("expected IR")
	}
	push, ok := ir.On["push"].(map[string]any)
	if !ok {
		t.Fatalf("push trigger: got %T", ir.On["push"])
	}
	branches, ok := push["branches"].([]any)
	if !ok || len(branches) != 1 {
		t.Errorf("branches: got %v", push["branches"])
	}
}

func TestUsesResolution_Remote(t *testing.T) {
	ref, diag := resolveUses("actions/checkout@v4", 1)
	if diag != nil {
		t.Fatalf("unexpected diagnostic: %v", diag)
	}
	if ref.Kind != "remote" || ref.Owner != "actions" || ref.Name != "checkout" || ref.Ref != "v4" {
		t.Errorf("got %+v", ref)
	}
}

func TestUsesResolution_Local(t *testing.T) {
	ref, diag := resolveUses("./.github/actions/local", 1)
	if diag != nil {
		t.Fatalf("unexpected diagnostic: %v", diag)
	}
	if ref.Kind != "local" || ref.LocalPath != "./.github/actions/local" {
		t.Errorf("got %+v", ref)
	}
}

func TestUsesResolution_Docker(t *testing.T) {
	ref, diag := resolveUses("docker://node:18", 1)
	if diag != nil {
		t.Fatalf("unexpected diagnostic: %v", diag)
	}
	if ref.Kind != "docker" || ref.Image != "node:18" {
		t.Errorf("got %+v", ref)
	}
}

func TestUsesResolution_MissingRef(t *testing.T) {
	_, diag := resolveUses("actions/checkout", 5)
	if diag == nil {
		t.Fatal("expected diagnostic for missing @ref")
	}
	if diag.Severity != "error" || !strings.Contains(diag.Message, "missing @ref") {
		t.Errorf("got diagnostic: %+v", diag)
	}
}

func TestMatrixExpansion_IncludeExclude(t *testing.T) {
	yamlSrc := []byte(`on: push
jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu, windows]
        node-version: [18, 20]
        exclude:
          - os: windows
            node-version: 18
    steps:
      - run: echo ok
`)
	ir, diags, _ := ParseWorkflowFull(yamlSrc)
	for _, d := range diags {
		if d.Severity == "error" {
			t.Fatalf("unexpected error: %v", d)
		}
	}
	expansion := ir.Jobs["build"].MatrixExpansion
	// 2 os x 2 node = 4, minus 1 excluded = 3
	if len(expansion) != 3 {
		t.Errorf("matrix expansion count: got %d, want 3", len(expansion))
	}
}

func TestMatrixCap(t *testing.T) {
	yamlSrc := []byte(`on: push
jobs:
  build:
    strategy:
      matrix:
        a: [1,2,3,4,5]
        b: [1,2,3,4,5]
        c: [1,2,3,4,5]
        d: [1,2,3,4,5]
        e: [1,2,3,4,5]
        f: [1,2,3,4,5]
        g: [1,2,3,4,5]
    steps:
      - run: echo ok
`)
	_, diags, _ := ParseWorkflowFull(yamlSrc)
	found := false
	for _, d := range diags {
		if d.Severity == "error" && strings.Contains(d.Message, "256") {
			found = true
		}
	}
	if !found {
		t.Error("expected matrix cap diagnostic")
	}
}

func TestDAGOrder(t *testing.T) {
	yamlSrc := []byte(`on: push
jobs:
  a:
    needs: b
    steps:
      - run: echo a
  b:
    steps:
      - run: echo b
`)
	ir, diags, _ := ParseWorkflowFull(yamlSrc)
	for _, d := range diags {
		if d.Severity == "error" {
			t.Fatalf("unexpected error: %v", d)
		}
	}
	order := ir.DAG.Order
	if len(order) != 2 {
		t.Fatalf("order: got %v", order)
	}
	if order[0] != "b" || order[1] != "a" {
		t.Errorf("order: got %v, want [b a]", order)
	}
}

func TestDAGCycle(t *testing.T) {
	yamlSrc := []byte(`on: push
jobs:
  a:
    needs: b
    steps:
      - run: echo a
  b:
    needs: a
    steps:
      - run: echo b
`)
	_, diags, _ := ParseWorkflowFull(yamlSrc)
	found := false
	for _, d := range diags {
		if d.Severity == "error" && strings.Contains(d.Message, "cyclic dependency") && strings.Contains(d.Message, "→") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected cycle diagnostic, got %v", diags)
	}
}

func TestFileSizeLimit(t *testing.T) {
	data := make([]byte, 513*1024)
	copy(data, []byte("on: push\njobs:\n  j:\n    steps:\n      - run: echo\n"))
	ir, diags, err := ParseWorkflowFull(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ir != nil {
		t.Error("expected nil IR for oversized file")
	}
	found := false
	for _, d := range diags {
		if d.Severity == "error" && strings.Contains(d.Message, "maximum size") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected size limit diagnostic, got %v", diags)
	}
}

func TestStepUsesRunConflict(t *testing.T) {
	yamlSrc := []byte(`on: push
jobs:
  build:
    steps:
      - uses: actions/checkout@v4
        run: echo conflict
`)
	_, diags, _ := ParseWorkflowFull(yamlSrc)
	found := false
	for _, d := range diags {
		if d.Severity == "error" && strings.Contains(d.Message, "both 'uses' and 'run'") && d.Line > 0 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected uses+run conflict diagnostic with line, got %v", diags)
	}
}

func TestMissingOn(t *testing.T) {
	yamlSrc := []byte(`jobs:
  build:
    steps:
      - run: echo ok
`)
	_, diags, _ := ParseWorkflowFull(yamlSrc)
	found := false
	for _, d := range diags {
		if d.Severity == "error" && strings.Contains(d.Message, "trigger 'on'") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing on diagnostic, got %v", diags)
	}
}
