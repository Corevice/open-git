package git_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/transport"

	infragit "github.com/open-git/backend/internal/infrastructure/git"
)

// TestServePackCLI_IncrementalPush is the regression guard for the thin-pack
// push bug: go-git's pure-Go receive-pack rejected the thin packs real git
// clients send for incremental commits ("reference delta not found"), so the
// second push to a repo failed with HTTP 500. ServePackCLI shells out to the
// git binary, which resolves ref-deltas against the existing object store.
//
// The test drives a real `git` client through a minimal smart-HTTP server
// wired directly to AdvertiseRefsCLI/ServePackCLI, and asserts BOTH the initial
// push and a follow-up incremental push (which sends a thin pack) succeed, plus
// that a fresh clone sees the incremental commit.
func TestServePackCLI_IncrementalPush(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	root := t.TempDir()
	bare := filepath.Join(root, "srv.git")
	runGit(t, root, "init", "--quiet", "--bare", "-b", "main", bare)

	srv := httptest.NewServer(smartHTTPHandler(t, bare))
	defer srv.Close()

	// Clone (empty), commit, push — the first pack is self-contained.
	work := filepath.Join(root, "work")
	runGit(t, root, "clone", "--quiet", srv.URL+"/srv.git", work)
	gitConfig(t, work)
	writeFile(t, filepath.Join(work, "a.txt"), "one\n")
	runGit(t, work, "add", "-A")
	runGit(t, work, "commit", "--quiet", "-m", "first")
	runGit(t, work, "push", "--quiet", "origin", "HEAD:refs/heads/main")

	// Second commit deltas against objects already on the server, so git sends
	// a THIN pack — this is exactly what used to 500.
	writeFile(t, filepath.Join(work, "a.txt"), "one\ntwo\nthree\n")
	runGit(t, work, "add", "-A")
	runGit(t, work, "commit", "--quiet", "-m", "second (thin pack)")
	if out, err := runGitErr(work, "push", "origin", "HEAD:refs/heads/main"); err != nil {
		t.Fatalf("incremental push failed (thin-pack regression): %v\n%s", err, out)
	}

	// A fresh clone must contain the incremental commit.
	clone2 := filepath.Join(root, "clone2")
	runGit(t, root, "clone", "--quiet", srv.URL+"/srv.git", clone2)
	got := readFile(t, filepath.Join(clone2, "a.txt"))
	if got != "one\ntwo\nthree\n" {
		t.Fatalf("cloned content = %q, want incremental content", got)
	}
}

// smartHTTPHandler is a minimal git smart-HTTP server backed by the CLI helpers.
func smartHTTPHandler(t *testing.T, bare string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/srv.git/info/refs", func(w http.ResponseWriter, r *http.Request) {
		service := r.URL.Query().Get("service")
		w.Header().Set("Content-Type", "application/x-"+service+"-advertisement")
		enc := pktline.NewEncoder(w)
		_ = enc.EncodeString("# service=" + service + "\n")
		_ = enc.Flush()
		if err := infragit.AdvertiseRefsCLI(r.Context(), w, bare, service); err != nil {
			t.Errorf("AdvertiseRefsCLI(%s): %v", service, err)
		}
	})

	serve := func(service string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-"+service+"-result")
			if err := infragit.ServePackCLI(r.Context(), w, r.Body, bare, service); err != nil {
				// Surface as 500 exactly like the real handler would.
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
	mux.HandleFunc("/srv.git/git-receive-pack", serve(transport.ReceivePackServiceName))
	mux.HandleFunc("/srv.git/git-upload-pack", serve(transport.UploadPackServiceName))
	return mux
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := runGitErr(dir, args...); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func runGitErr(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func gitConfig(t *testing.T, dir string) {
	runGit(t, dir, "config", "user.email", "t@example.com")
	runGit(t, dir, "config", "user.name", "tester")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
