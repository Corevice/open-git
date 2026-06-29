package scanner

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
	if strings.HasPrefix(name, "@") {
		slashIdx := strings.Index(name, "/")
		if slashIdx <= 1 || strings.Contains(name[slashIdx+1:], "/") {
			return false
		}
	} else if strings.Contains(name, "/") {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return false
		}
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
		// Runtime dependencies take precedence: skip dev-only entries when the
		// same package is already listed under dependencies.
		if _, exists := seen[name]; !exists {
			seen[name] = version
		}
	}

	deps := make([]Dependency, 0, len(seen))
	for name, version := range seen {
		stripped := stripSemverPrefix(version)
		if stripped == "" {
			continue
		}
		deps = append(deps, Dependency{
			Name:      name,
			Version:   stripped,
			Ecosystem: "npm",
		})
	}
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].Name < deps[j].Name
	})
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
	bestIdx := -1
	bestLen := 0
	for i := 0; i < len(spec); i++ {
		for _, candidate := range requirementOperators {
			if !strings.HasPrefix(spec[i:], candidate) {
				continue
			}
			if len(candidate) > bestLen || (len(candidate) == bestLen && (bestIdx < 0 || i < bestIdx)) {
				op = candidate
				bestIdx = i
				bestLen = len(candidate)
			}
		}
	}
	if bestIdx >= 0 {
		return op, bestIdx
	}
	return "", -1
}

func parseRequirementLine(line string) (name, version string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", false
	}

	parts := splitRequirementSpecifiers(line)
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
	case lowerBound != "":
		version = lowerBound
	case compatibleVersion != "":
		version = compatibleVersion
	default:
		return "", "", false
	}

	return pkgName, version, true
}

// splitRequirementSpecifiers splits PEP 440 compound specifiers on commas while
// preserving commas inside URL-based dependency lines.
func splitRequirementSpecifiers(line string) []string {
	if strings.Contains(line, "://") {
		return []string{line}
	}

	raw := strings.Split(line, ",")
	if len(raw) <= 1 {
		return raw
	}

	parts := make([]string, 0, len(raw))
	current := strings.TrimSpace(raw[0])
	for _, segment := range raw[1:] {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if _, opIdx := findRequirementOperator(segment); opIdx >= 0 {
			if current != "" {
				parts = append(parts, current)
			}
			current = segment
			continue
		}
		if current == "" {
			current = segment
		} else {
			current += "," + segment
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// npmExactVersionPrefixes lists npm prefixes that can be stripped to yield a
// concrete version for OSV queries. Range operators (>=, <=, >, <) are excluded
// because they denote constraints, not installed versions.
var npmExactVersionPrefixes = []string{"^", "~", "="}

func stripSemverPrefix(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || strings.Contains(version, ",") || strings.HasPrefix(version, "!=") {
		return ""
	}
	for _, rangePrefix := range []string{">=", "<=", ">", "<"} {
		if strings.HasPrefix(version, rangePrefix) {
			return ""
		}
	}
	for _, prefix := range npmExactVersionPrefixes {
		if strings.HasPrefix(version, prefix) {
			stripped := strings.TrimSpace(version[len(prefix):])
			if stripped == "" {
				return ""
			}
			return stripped
		}
	}
	if strings.HasPrefix(version, "v") && len(version) > 1 && version[1] >= '0' && version[1] <= '9' {
		return strings.TrimSpace(version[1:])
	}
	return version
}

func isRequirementsTxtOptionLine(line string) bool {
	if !strings.HasPrefix(line, "-") {
		return false
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return true
	}
	switch fields[0] {
	case "-r", "--requirement", "-c", "--constraint", "-e", "--editable",
		"-i", "--index-url", "--extra-index-url", "--find-links", "-f",
		"--no-index", "--pre", "--use-feature":
		return true
	default:
		return false
	}
}

func parseRequirementsTxt(content []byte) ([]Dependency, error) {
	lines := strings.Split(string(content), "\n")
	deps := make([]Dependency, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if isRequirementsTxtOptionLine(line) {
			continue
		}

		if strings.Contains(line, "://") {
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
