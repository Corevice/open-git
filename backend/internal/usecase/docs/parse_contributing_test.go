package docs

import (
	"testing"
)

func TestParseSectionsEmpty(t *testing.T) {
	sections := ParseSections("")
	if len(sections) != 0 {
		t.Fatalf("expected empty slice, got %d sections", len(sections))
	}
}

func TestParseSectionsSingleHeading(t *testing.T) {
	sections := ParseSections("## Development Setup\n\nRun make setup.\n")
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0].Title != "Development Setup" {
		t.Fatalf("unexpected title: %q", sections[0].Title)
	}
	if sections[0].Slug != "development-setup" {
		t.Fatalf("unexpected slug: %q", sections[0].Slug)
	}
	if sections[0].Content != "\nRun make setup.\n" {
		t.Fatalf("unexpected content: %q", sections[0].Content)
	}
	if sections[0].Order != 1 {
		t.Fatalf("unexpected order: %d", sections[0].Order)
	}
}

func TestParseSectionsMultipleHeadings(t *testing.T) {
	markdown := "# Contributing\n\n## Section One\nFirst body.\n\n## Section Two\nSecond body.\n"
	sections := ParseSections(markdown)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].Slug != "section-one" || sections[0].Order != 1 {
		t.Fatalf("unexpected first section: %+v", sections[0])
	}
	if sections[1].Slug != "section-two" || sections[1].Order != 2 {
		t.Fatalf("unexpected second section: %+v", sections[1])
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Development Setup": "development-setup",
		"C# Style":          "c-style",
	}
	for title, want := range cases {
		if got := slugify(title); got != want {
			t.Fatalf("slugify(%q) = %q, want %q", title, got, want)
		}
	}
}
