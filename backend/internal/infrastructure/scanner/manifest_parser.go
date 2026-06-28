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
	Name      string
	Version   string
	Ecosystem string
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

	// Indirect dependencies are included intentionally: they are part of the
	// module's resolved dependency graph and may carry vulnerabilities even
	// when not directly imported by application code.
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
		if _, exists := seen[name]; !exists {
			seen[name] = version
		}
	}

	deps := make([]Dependency, 0, len(seen))
	for name, version := range seen {
		deps = append(deps, Dependency{
			Name:      name,
			Version:   stripSemverPrefix(version),
			Ecosystem: "npm",
		})
	}
	return deps, nil
}

// requirementOperators lists PEP 440-style operators in longest-match-first order.
var requirementOperators = []string{"==", "!=", ">=", "<=", "~=", ">", "<"}

func parseRequirementLine(line string) (name, version string, ok bool) {
	for _, op := range requirementOperators {
		idx := strings.Index(line, op)
		if idx <= 0 {
			continue
		}
		name = strings.TrimSpace(line[:idx])
		version = strings.TrimSpace(line[idx+len(op):])
		if name == "" || version == "" {
			continue
		}
		return name, version, true
	}
	return "", "", false
}

func stripSemverPrefix(version string) string {
	version = strings.TrimSpace(version)
	for {
		changed := false
		switch {
		case strings.HasPrefix(version, ">="):
			version = strings.TrimSpace(version[2:])
			changed = true
		case strings.HasPrefix(version, "<="):
			version = strings.TrimSpace(version[2:])
			changed = true
		case strings.HasPrefix(version, "!="):
			version = strings.TrimSpace(version[2:])
			changed = true
		case strings.HasPrefix(version, "~"):
			version = strings.TrimSpace(version[1:])
			changed = true
		case strings.HasPrefix(version, "^"):
			version = strings.TrimSpace(version[1:])
			changed = true
		case strings.HasPrefix(version, ">"):
			version = strings.TrimSpace(version[1:])
			changed = true
		case strings.HasPrefix(version, "<"):
			version = strings.TrimSpace(version[1:])
			changed = true
		case strings.HasPrefix(version, "="):
			version = strings.TrimSpace(version[1:])
			changed = true
		case strings.HasPrefix(version, "v") && len(version) > 1 && version[1] >= '0' && version[1] <= '9':
			version = strings.TrimSpace(version[1:])
			changed = true
		}
		if !changed {
			break
		}
	}
	return version
}

func parseRequirementsTxt(content []byte) ([]Dependency, error) {
	lines := strings.Split(string(content), "\n")
	deps := make([]Dependency, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
			if line == "" {
				continue
			}
		}

		name, version, ok := parseRequirementLine(line)
		if !ok {
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
