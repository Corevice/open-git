package docs

import (
	"regexp"
	"strings"
)

var nonAlphaNumRun = regexp.MustCompile(`[^a-z0-9]+`)

// DocSection represents one h2 section parsed from CONTRIBUTING.md.
type DocSection struct {
	Slug    string
	Title   string
	Order   int
	Content string
}

// ParseSections splits markdown on h2 headings (`\n## ` boundaries).
func ParseSections(markdown string) []DocSection {
	if strings.TrimSpace(markdown) == "" {
		return []DocSection{}
	}

	text := markdown
	if strings.HasPrefix(text, "## ") {
		// first chunk is already a section heading
	} else if idx := strings.Index(text, "\n## "); idx >= 0 {
		text = text[idx+1:]
	} else {
		return []DocSection{}
	}

	parts := strings.Split(text, "\n## ")
	sections := make([]DocSection, 0, len(parts))
	order := 1
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		title, content := splitSectionPart(part)
		title = strings.TrimSpace(strings.TrimPrefix(title, "## "))
		if title == "" {
			continue
		}

		sections = append(sections, DocSection{
			Slug:    slugify(title),
			Title:   title,
			Order:   order,
			Content: content,
		})
		order++
	}
	return sections
}

func splitSectionPart(part string) (string, string) {
	lines := strings.SplitN(part, "\n", 2)
	title := lines[0]
	if len(lines) == 1 {
		return title, ""
	}
	return title, lines[1]
}

func slugify(title string) string {
	s := strings.ToLower(title)
	s = nonAlphaNumRun.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
