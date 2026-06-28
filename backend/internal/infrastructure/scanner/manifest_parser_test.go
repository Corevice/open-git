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

func TestParseGoMod(t *testing.T) {
	deps, err := ParseManifest(ManifestGoMod, readTestdata(t, "sample.mod"))
	if err != nil {
		t.Fatalf("ParseManifest(go.mod) returned error: %v", err)
	}

	want := []Dependency{
		{Name: "github.com/gin-gonic/gin", Version: "v1.9.1", Ecosystem: "Go"},
		{Name: "golang.org/x/mod", Version: "v0.14.0", Ecosystem: "Go"},
	}

	got := sortDeps(deps)
	expected := sortDeps(want)
	if len(got) != len(expected) {
		t.Fatalf("dependency count: got %d, want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("dependency[%d]: got %+v, want %+v", i, got[i], expected[i])
		}
	}
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

	got := sortDeps(deps)
	expected := sortDeps(want)
	if len(got) != len(expected) {
		t.Fatalf("dependency count: got %d, want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("dependency[%d]: got %+v, want %+v", i, got[i], expected[i])
		}
	}
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

	got := sortDeps(deps)
	expected := sortDeps(want)
	if len(got) != len(expected) {
		t.Fatalf("dependency count: got %d, want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("dependency[%d]: got %+v, want %+v", i, got[i], expected[i])
		}
	}
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
