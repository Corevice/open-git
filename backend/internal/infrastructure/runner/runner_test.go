package runner

import (
	"context"
	"strings"
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestValidateActJobName_RejectsInjection(t *testing.T) {
	cases := []string{"", "job;rm", "job name", "--help", " $(id)"}
	for _, name := range cases {
		if err := validateActJobName(name); err == nil {
			t.Fatalf("validateActJobName(%q) expected error", name)
		}
	}
}

func TestValidateActJobName_AcceptsValidNames(t *testing.T) {
	cases := []string{"build", "lint-test", "deploy_prod"}
	for _, name := range cases {
		if err := validateActJobName(name); err != nil {
			t.Fatalf("validateActJobName(%q) unexpected error: %v", name, err)
		}
	}
}

func TestWriteSecretsFile_RejectsNewlines(t *testing.T) {
	_, err := writeSecretsFile(map[string]string{"KEY\nINJECT": "value"})
	if err == nil {
		t.Fatal("expected error for injected secret key")
	}
	_, err = writeSecretsFile(map[string]string{"KEY": "value\nINJECT"})
	if err == nil {
		t.Fatal("expected error for injected secret value")
	}
}

func TestBuildWorkflowYAML_QuotesSpecialCharacters(t *testing.T) {
	job := &entity.WorkflowJob{Name: "build"}
	steps := []*Step{{Name: `say "hello"`}}
	yaml := buildWorkflowYAML(job, steps)
	if !strings.Contains(yaml, `name: "say \"hello\""`) {
		t.Fatalf("expected YAML-quoted step name, got:\n%s", yaml)
	}
	if strings.Contains(yaml, `%q`) {
		t.Fatalf("expected YAML quoting, not Go quoting: %s", yaml)
	}
}

func TestYamlDoubleQuote_EscapesControlCharacters(t *testing.T) {
	got := yamlDoubleQuote("a\nb")
	want := `"a\nb"`
	if got != want {
		t.Fatalf("yamlDoubleQuote() = %q, want %q", got, want)
	}
}

func TestDockerActRunner_RejectsInvalidJobName(t *testing.T) {
	runner := NewDockerActRunner("act")
	err := runner.ExecuteJob(
		context.Background(),
		&entity.WorkflowJob{Name: "bad;job"},
		nil,
		nil,
		func(int64, string) {},
	)
	if err == nil {
		t.Fatal("expected invalid job name error")
	}
}
