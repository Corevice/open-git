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

func isValidPackageName(name string) bool {
	if name == "" || strings.ContainsAny(name, " \t\n\r\f") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.', r == '@', r == '/':
		default:
			return false
		}
	}
	return true
}

func isValidVersionString(version string) bool {
	if version == "" {
		return false
	}
	for _, r := range version {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func parsePackageJSON(content []byte) ([]Dependency, error) {
	var pkg packageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	seen := make(map[string]string)
	for name, version := range pkg.Dependencies {
		if !isValidPackageName(name) || !isValidVersionString(version) {
			continue
		}
		seen[name] = version
	}
	for name, version := range pkg.DevDependencies {
		if !isValidPackageName(name) || !isValidVersionString(version) {
			continue
		}
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

func stripPEP508Extras(name string) string {
	if idx := strings.Index(name, "["); idx >= 0 {
		if end := strings.Index(name[idx:], "]"); end >= 0 {
			name = name[:idx] + name[idx+end+1:]
		}
	}
	return strings.TrimSpace(name)
}

func findRequirementOperator(spec string) (op string, idx int) {
	for _, candidate := range requirementOperators {
		if i := strings.Index(spec, candidate); i >= 0 {
			return candidate, i
		}
	}
	return "", -1
}

func parseRequirementLine(line string) (name, version string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", false
	}

	parts := strings.Split(line, ",")
	var pkgName string
	var exactVersion string
	var lowerBound string
	var compatibleVersion string

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		op, opIdx := findRequirementOperator(part)
		if opIdx < 0 {
			if i == 0 && pkgName == "" {
				pkgName = stripPEP508Extras(part)
			}
			continue
		}

		var partName string
		partVersion := strings.TrimSpace(part[opIdx+len(op):])
		if partVersion == "" {
			continue
		}

		if opIdx > 0 {
			partName = stripPEP508Extras(strings.TrimSpace(part[:opIdx]))
			if i == 0 {
				if partName == "" {
					continue
				}
				pkgName = partName
			}
		} else if i > 0 {
			// Continuation segment such as ",<3.0" in "requests>=2.31.0,<3.0".
		} else {
			continue
		}

		switch op {
		case "==":
			exactVersion = partVersion
		case "!=":
			// Exclusion constraints do not identify an installed version.
		case ">=", ">":
			if lowerBound == "" {
				lowerBound = partVersion
			}
		case "~=":
			if compatibleVersion == "" {
				compatibleVersion = partVersion
			}
		case "<=", "<":
			// Upper bounds alone do not identify an installed version.
		}
	}

	if pkgName == "" {
		return "", "", false
	}

	switch {
	case exactVersion != "":
		version = exactVersion
	case compatibleVersion != "":
		version = compatibleVersion
	case lowerBound != "":
		version = lowerBound
	default:
		return "", "", false
	}

	return pkgName, version, true
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
