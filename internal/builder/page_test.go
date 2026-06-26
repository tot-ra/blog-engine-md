package builder

import (
	"testing"

	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestURLGenerator_SelfNamedMarkdownFileUsesParentDirectory(t *testing.T) {
	g := NewURLGenerator("https://example.com")

	got := g.Generate("docs/beehive-sensors/beehive-sensors.md", &parser.Frontmatter{})
	want := "/docs/beehive-sensors/"
	if got != want {
		t.Fatalf("Generate() = %q, want %q", got, want)
	}
}

func TestURLGenerator_SlugEquivalentSelfNamedMarkdownFileUsesParentDirectory(t *testing.T) {
	g := NewURLGenerator("https://example.com")

	got := g.Generate("docs/web_app/web_app.md", &parser.Frontmatter{})
	want := "/docs/web_app/"
	if got != want {
		t.Fatalf("Generate() = %q, want %q", got, want)
	}
}

func TestURLGenerator_SelfNamedMarkdownFileRespectsExplicitSlug(t *testing.T) {
	g := NewURLGenerator("https://example.com")

	got := g.Generate("docs/beehive-sensors/beehive-sensors.md", &parser.Frontmatter{Slug: "custom"})
	want := "/docs/beehive-sensors/custom/"
	if got != want {
		t.Fatalf("Generate() = %q, want %q", got, want)
	}
}
