package workflow

import "testing"

func testCtx() *EvalContext {
	return &EvalContext{
		Contexts: map[string]map[string]string{
			"github":  {"ref_name": "main", "event_name": "push", "sha": "abc123", "run_number": "7"},
			"env":     {"STAGE": "prod", "COUNT": "3"},
			"secrets": {"TOKEN": "s3cr3t"},
			"matrix":  {"os": "linux", "go": "1.22"},
			"runner":  {"os": "Linux"},
		},
	}
}

func TestInterpolateString(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"sha=${{ github.sha }}", "sha=abc123"},
		{"${{ env.STAGE }}-${{ matrix.os }}", "prod-linux"},
		{"plain text", "plain text"},
		{"n=${{ github.run_number }}", "n=7"},
		{"${{ format('{0}/{1}', matrix.os, matrix.go) }}", "linux/1.22"},
		{"missing=${{ env.NOPE }}", "missing="},
		{"bool=${{ env.STAGE == 'prod' }}", "bool=true"},
	}
	ctx := testCtx()
	for _, tc := range cases {
		got, err := InterpolateString(tc.in, ctx)
		if err != nil {
			t.Errorf("InterpolateString(%q) error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("InterpolateString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEvaluateCondition(t *testing.T) {
	cases := []struct {
		cond string
		want bool
	}{
		{"", true},
		{"github.ref_name == 'main'", true},
		{"github.ref_name == 'dev'", false},
		{"${{ github.event_name == 'push' }}", true},
		{"env.STAGE == 'prod' && github.ref_name == 'main'", true},
		{"env.STAGE == 'prod' && github.ref_name == 'dev'", false},
		{"env.STAGE == 'staging' || github.ref_name == 'main'", true},
		{"!(github.ref_name == 'main')", false},
		{"contains(github.ref_name, 'mai')", true},
		{"startsWith(github.sha, 'abc')", true},
		{"endsWith(github.sha, 'zzz')", false},
		{"github.run_number > 5", true},
		{"github.run_number >= 8", false},
	}
	ctx := testCtx()
	for _, tc := range cases {
		got, err := EvaluateCondition(tc.cond, ctx)
		if err != nil {
			t.Errorf("EvaluateCondition(%q) error: %v", tc.cond, err)
			continue
		}
		if got != tc.want {
			t.Errorf("EvaluateCondition(%q) = %v, want %v", tc.cond, got, tc.want)
		}
	}
}

func TestConditionStatusFunctions(t *testing.T) {
	base := testCtx()
	base.Failed = false
	if ok, _ := EvaluateCondition("success()", base); !ok {
		t.Error("success() should be true when nothing failed")
	}
	if ok, _ := EvaluateCondition("failure()", base); ok {
		t.Error("failure() should be false when nothing failed")
	}
	if ok, _ := EvaluateCondition("always()", base); !ok {
		t.Error("always() should be true")
	}

	failed := testCtx()
	failed.Failed = true
	if ok, _ := EvaluateCondition("success()", failed); ok {
		t.Error("success() should be false after a failure")
	}
	if ok, _ := EvaluateCondition("failure()", failed); !ok {
		t.Error("failure() should be true after a failure")
	}
	if ok, _ := EvaluateCondition("always()", failed); !ok {
		t.Error("always() should be true even after a failure")
	}
}
