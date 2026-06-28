package scanner

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("testdata", name))
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

func TestParseRequirementsTxt(t *testing.T) {
	deps, err := ParseManifest(ManifestRequirementsTxt, readTestdata(t, "sample_requirements.txt"))
	if err != nil {
		t.Fatalf("ParseManifest(requirements.txt) returned error: %v", err)
	}

	want := []Dependency{
		{Name: "django", Version: "4.2.7", Ecosystem: "PyPI"},
		{Name: "flask", Version: "3.0.0", Ecosystem: "PyPI"},
		{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
	}

	assertDepsEqual(t, deps, want)
}

func TestParseRequirementsTxtInlineComment(t *testing.T) {
	content := []byte("requests==2.31.0  # HTTP library\n")
	deps, err := ParseManifest(ManifestRequirementsTxt, content)
	if err != nil {
		t.Fatalf("ParseManifest(requirements.txt) returned error: %v", err)
	}

	assertDepsEqual(t, deps, []Dependency{
		{Name: "requests", Version: "2.31.0", Ecosystem: "PyPI"},
	})
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
