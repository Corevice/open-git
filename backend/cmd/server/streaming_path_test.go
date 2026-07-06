package main

import "testing"

// TestIsStreamingPath guards the timeout-middleware skipper. SSE endpoints must
// be skipped (their ResponseWriter must stay an http.Flusher); everything else
// must keep the timeout.
func TestIsStreamingPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// Job-only SSE stream consumed by the web UI.
		{"/api/v3/repos/alice/demo/actions/jobs/42/logs", true},
		// Run-scoped SSE stream.
		{"/api/v1/repos/alice/demo/actions/runs/7/jobs/42/logs/stream", true},
		{"/api/repos/alice/demo/actions/runs/7/jobs/42/logs/stream", true},
		// Run-scoped JSON logs (has a run in the path, not job-only) — timeout OK.
		{"/api/v1/repos/alice/demo/actions/runs/7/jobs/42/logs", false},
		// Ordinary endpoints must keep the timeout.
		{"/api/v3/user", false},
		{"/api/v3/repos/alice/demo", false},
		{"/api/v3/repos/alice/demo/actions/runs", false},
		{"/healthz", false},
	}
	for _, tc := range cases {
		if got := isStreamingPath(tc.path); got != tc.want {
			t.Errorf("isStreamingPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
