package scanner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func validateTestdataName(name string) error {
	if name == "" || strings.Contains(name, "..") || filepath.IsAbs(name) {
		return fmt.Errorf("invalid testdata name: %q", name)
	}

	cleanName := filepath.Clean(name)
	if cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid testdata name: %q", name)
	}

	baseDir, err := filepath.Abs("testdata")
	if err != nil {
		return fmt.Errorf("resolve testdata dir: %w", err)
	}
	targetPath, err := filepath.Abs(filepath.Join(baseDir, cleanName))
	if err != nil {
		return fmt.Errorf("resolve testdata path: %w", err)
	}
	if targetPath != baseDir && !strings.HasPrefix(targetPath, baseDir+string(filepath.Separator)) {
		return fmt.Errorf("invalid testdata name: %q", name)
	}
	return nil
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()

	if err := validateTestdataName(name); err != nil {
		t.Fatal(err)
	}

	cleanName := filepath.Clean(name)
	targetPath, err := filepath.Abs(filepath.Join("testdata", cleanName))
	if err != nil {
		t.Fatalf("resolve testdata path: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	return content
}

func sortDeps(deps []Dependency) []Dependency {
	sorted := append([]Dependency(nil), deps...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Name != sorted[j].Name {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].Version < sorted[j].Version
	})
	return sorted
}

func assertDepsEqual(t *testing.T, got, want []Dependency) {
	t.Helper()

	got = sortDeps(got)
	want = sortDeps(want)
	if len(got) != len(want) {
		t.Fatalf("dependency count: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("dependency[%d]: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseGoMod(t *testing.T) {
	deps, err := ParseManifest(ManifestGoMod, readTestdata(t, "sample.mod"))
	if err != nil {
		t.Fatalf("ParseManifest(go.mod) returned error: %v", err)
	}

	want := []Dependency{
		{Name: "github.com/gin-gonic/gin", Version: "v1.9.1", Ecosystem: "Go"},
		{Name: "github.com/klauspost/cpuid/v2", Version: "v2.2.4", Ecosystem: "Go"},
		{Name: "golang.org/x/mod", Version: "v0.14.0", Ecosystem: "Go"},
	}

	assertDepsEqual(t, deps, want)
}

func TestParsePackageJSON(t *testing.T) {
	deps, err := ParseManifest(ManifestPackageJSON, readTestdata(t, "sample_package.json"))
	if err != nil {
		t.Fatalf("ParseManifest(package.json) returned error: %v", err)
	}

	// Expected versions reflect stripSemverPrefix output (^ and ~ prefixes removed).
	want := []Dependency{
		{Name: "express", Version: "4.18.2", Ecosystem: "npm"},
		{Name: "jest", Version: "29.7.0", Ecosystem: "npm"},
		{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"},
		{Name: "typescript", Version: "5.3.3", Ecosystem: "npm"},
	}

	assertDepsEqual(t, deps, want)
}

func TestParsePackageJSONDependenciesPrecedence(t *testing.T) {
	content := []byte(`{
		"dependencies": {"lodash": "4.17.21"},
		"devDependencies": {"lodash": "4.17.20"}
	}`)
	deps, err := ParseManifest(ManifestPackageJSON, content)
	if err != nil {
		t.Fatalf("ParseManifest(package.json) returned error: %v", err)
	}

	assertDepsEqual(t, deps, []Dependency{
		{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"},
	})
}

func TestParseRequirementsTxtCases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Dependency
		absent  []string
	}{
		{
			name:    "sample file",
			content: string(readTestdata(t, "sample_requirements.txt")),
			want: []Dependency{
				{Name: "django", Version: "4.2.7", Ecosystem: "PyPI"},
				{Name: "flask", Version: "3.0.0", Ecosystem: "PyPI"},
				{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
			},
		},
		{
			name:    "inline comment",
			content: "requests==2.31.0  # HTTP library\n",
			want: []Dependency{
				{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
			},
		},
		{
			name:    "non-strict specifiers",
			content: "requests>=2.31.0\nflask~=3.0.0\ndjango!=4.2.6\n",
			want: []Dependency{
				{Name: "flask", Version: "3.0.0", Ecosystem: "PyPI"},
				{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
			},
			absent: []string{"django"},
		},
		{
			name:    "compound specifiers",
			content: "requests>=2.31.0,<3.0\n",
			want: []Dependency{
				{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
			},
		},
		{
			name:    "compound specifiers prefer exact",
			content: "requests==2.31.0,>=2.0\n",
			want: []Dependency{
				{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
			},
		},
		{
			name:    "PEP508 extras",
			content: "requests[security]==2.31.0\n",
			want: []Dependency{
				{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
			},
		},
		{
			name:    "package name before comma-bound operator",
			content: "requests,>=2.31.0\n",
			want: []Dependency{
				{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
			},
		},
		{
			name:    "pip option lines skipped",
			content: "-r other.txt\n-c constraints.txt\n--index-url https://pypi.org/simple\nrequests==2.31.0\n",
			want: []Dependency{
				{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, err := ParseManifest(ManifestRequirementsTxt, []byte(tt.content))
			if err != nil {
				t.Fatalf("ParseManifest(requirements.txt) returned error: %v", err)
			}
			assertDepsEqual(t, deps, tt.want)
			for _, name := range tt.absent {
				for _, dep := range deps {
					if dep.Name == name {
						t.Errorf("dependency %q should be absent (exclusion or no concrete version)", name)
					}
				}
			}
		})
	}
}

func TestValidateTestdataNameRejectsPathTraversal(t *testing.T) {
	tests := []string{
		"../sample.mod",
		"../../etc/passwd",
		"/etc/passwd",
		"foo/../../secret",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateTestdataName(name); err == nil {
				t.Fatalf("validateTestdataName(%q) expected error, got nil", name)
			}
		})
	}

	if err := validateTestdataName("sample.mod"); err != nil {
		t.Fatalf("validateTestdataName(sample.mod) unexpected error: %v", err)
	}
}

func TestParsePackageJSONStripsSemverPrefix(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Dependency
		absent  []string
	}{
		{
			name:    "caret and tilde prefixes",
			content: `{"dependencies": {"lodash": "^4.17.21", "express": "~4.18.2"}}`,
			want: []Dependency{
				{Name: "express", Version: "4.18.2", Ecosystem: "npm"},
				{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"},
			},
		},
		{
			name:    "exclusion constraint omitted",
			content: `{"dependencies": {"lodash": "!=4.17.21", "express": "4.18.2"}}`,
			want: []Dependency{
				{Name: "express", Version: "4.18.2", Ecosystem: "npm"},
			},
			absent: []string{"lodash"},
		},
		{
			name:    "compound range omitted",
			content: `{"dependencies": {"lodash": ">=2.0,<3.0"}}`,
			want:    nil,
			absent:  []string{"lodash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, err := ParseManifest(ManifestPackageJSON, []byte(tt.content))
			if err != nil {
				t.Fatalf("ParseManifest(package.json) returned error: %v", err)
			}
			assertDepsEqual(t, deps, tt.want)
			for _, name := range tt.absent {
				for _, dep := range deps {
					if dep.Name == name {
						t.Errorf("dependency %q should be absent (no concrete version)", name)
					}
				}
			}
		})
	}
}

func TestParseEmptyManifests(t *testing.T) {
	t.Run("go.mod", func(t *testing.T) {
		deps, err := ParseManifest(ManifestGoMod, []byte("module example.com/empty\n\ngo 1.21\n"))
		if err != nil {
			t.Fatalf("ParseManifest(go.mod) returned error: %v", err)
		}
		assertDepsEqual(t, deps, nil)
	})

	t.Run("package.json", func(t *testing.T) {
		deps, err := ParseManifest(ManifestPackageJSON, []byte(`{"name":"empty"}`))
		if err != nil {
			t.Fatalf("ParseManifest(package.json) returned error: %v", err)
		}
		assertDepsEqual(t, deps, nil)
	})

	t.Run("requirements.txt", func(t *testing.T) {
		deps, err := ParseManifest(ManifestRequirementsTxt, []byte(""))
		if err != nil {
			t.Fatalf("ParseManifest(requirements.txt) returned error: %v", err)
		}
		assertDepsEqual(t, deps, nil)
	})
}

func TestParseUnknownManifest(t *testing.T) {
	_, err := ParseManifest("Cargo.toml", []byte("[package]\nname = \"sample\"\n"))
	if !errors.Is(err, ErrUnknownManifest) {
		t.Fatalf("ParseManifest(unknown) error: got %v, want ErrUnknownManifest", err)
	}
}

func TestParseMalformedPackageJSON(t *testing.T) {
	_, err := ParseManifest(ManifestPackageJSON, []byte(`{"dependencies":`))
	if err == nil {
		t.Fatal("ParseManifest(malformed package.json) expected error, got nil")
	}
}

func TestParseMalformedGoMod(t *testing.T) {
	_, err := ParseManifest(ManifestGoMod, []byte("modul broken\n"))
	if err == nil {
		t.Fatal("ParseManifest(malformed go.mod) expected error, got nil")
	}
}
