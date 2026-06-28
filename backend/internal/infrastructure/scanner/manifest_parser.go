package scanner

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/mod/modfile"
)

const (
	ManifestGoMod           = "go.mod"
	ManifestPackageJSON     = "package.json"
	ManifestRequirementsTxt = "requirements.txt"
)

var ErrUnknownManifest = errors.New("unknown manifest type")

type Dependency struct {
	Name       string
	Version    string
	Ecosystem  string
}

func ParseManifest(manifestType string, content []byte) ([]Dependency, error) {
	switch manifestType {
	case ManifestGoMod:
		return parseGoMod(content)
	case ManifestPackageJSON:
		return parsePackageJSON(content)
	case ManifestRequirementsTxt:
		return parseRequirementsTxt(content)
	default:
		return nil, ErrUnknownManifest
	}
}

func parseGoMod(content []byte) ([]Dependency, error) {
	mf, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}

	deps := make([]Dependency, 0, len(mf.Require))
	for _, req := range mf.Require {
		if req == nil {
			continue
		}
		deps = append(deps, Dependency{
			Name:      req.Mod.Path,
			Version:   req.Mod.Version,
			Ecosystem: "Go",
		})
	}
	return deps, nil
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func parsePackageJSON(content []byte) ([]Dependency, error) {
	var pkg packageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	seen := make(map[string]string)
	for name, version := range pkg.Dependencies {
		seen[name] = version
	}
	for name, version := range pkg.DevDependencies {
		seen[name] = version
	}

	deps := make([]Dependency, 0, len(seen))
	for name, version := range seen {
		deps = append(deps, Dependency{
			Name:      name,
			Version:   version,
			Ecosystem: "npm",
		})
	}
	return deps, nil
}

func parseRequirementsTxt(content []byte) ([]Dependency, error) {
	lines := strings.Split(string(content), "\n")
	deps := make([]Dependency, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "==", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])
		if name == "" || version == "" {
			continue
		}

		deps = append(deps, Dependency{
			Name:      name,
			Version:   version,
			Ecosystem: "PyPI",
		})
	}
	return deps, nil
}
