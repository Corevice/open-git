package actions_test

import (
	"testing"

	"github.com/open-git/backend/internal/usecase/actions"
)

func TestMatchLabels(t *testing.T) {
	tests := []struct {
		name      string
		requested []string
		available []string
		want      bool
	}{
		{
			name:      "empty requested matches any available",
			requested: nil,
			available: []string{"self-hosted", "linux"},
			want:      true,
		},
		{
			name:      "empty requested matches empty available",
			requested: []string{},
			available: []string{},
			want:      true,
		},
		{
			name:      "available superset of requested",
			requested: []string{"linux"},
			available: []string{"self-hosted", "linux"},
			want:      true,
		},
		{
			name:      "available missing requested label",
			requested: []string{"self-hosted", "linux"},
			available: []string{"linux"},
			want:      false,
		},
		{
			name:      "exact label match",
			requested: []string{"self-hosted", "linux"},
			available: []string{"self-hosted", "linux"},
			want:      true,
		},
		{
			name:      "disjoint labels",
			requested: []string{"windows"},
			available: []string{"linux"},
			want:      false,
		},
		{
			name:      "requested larger than available",
			requested: []string{"self-hosted", "linux", "x64"},
			available: []string{"self-hosted", "linux"},
			want:      false,
		},
		{
			name:      "single requested label present",
			requested: []string{"ubuntu-latest"},
			available: []string{"ubuntu-latest", "large"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actions.MatchLabels(tt.requested, tt.available)
			if got != tt.want {
				t.Fatalf("MatchLabels(%v, %v) = %v, want %v", tt.requested, tt.available, got, tt.want)
			}
		})
	}
}

func TestIsGitHubHosted(t *testing.T) {
	tests := []struct {
		label string
		want  bool
	}{
		{label: "ubuntu-latest", want: true},
		{label: "ubuntu-22.04", want: true},
		{label: "ubuntu-20.04", want: true},
		{label: "windows-latest", want: true},
		{label: "macos-latest", want: true},
		{label: "self-hosted", want: false},
		{label: "linux", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got := actions.IsGitHubHosted(tt.label)
			if got != tt.want {
				t.Fatalf("IsGitHubHosted(%q) = %v, want %v", tt.label, got, tt.want)
			}
		})
	}
}

func TestUsesActAdapter(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   bool
	}{
		{name: "empty labels", labels: nil, want: true},
		{name: "ubuntu-latest", labels: []string{"ubuntu-latest"}, want: true},
		{name: "ubuntu-22.04", labels: []string{"ubuntu-22.04"}, want: true},
		{name: "ubuntu-20.04", labels: []string{"ubuntu-20.04"}, want: true},
		{name: "self-hosted", labels: []string{"self-hosted", "linux"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actions.UsesActAdapter(tt.labels)
			if got != tt.want {
				t.Fatalf("UsesActAdapter(%v) = %v, want %v", tt.labels, got, tt.want)
			}
		})
	}
}
