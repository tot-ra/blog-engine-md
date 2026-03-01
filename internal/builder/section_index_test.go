package builder

import (
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestSectionIndexGenerator_GeneratesNestedSectionsUnderPageNode(t *testing.T) {
	pages := map[string]*Page{
		"en-home": {
			ID:          "en-home",
			URL:         "/en/",
			Language:    "en",
			Title:       "Home",
			Frontmatter: &parser.Frontmatter{},
			Type:        TypePage,
		},
		"en-docs-about": {
			ID:          "en-docs-about",
			URL:         "/en/docs/about/",
			Language:    "en",
			Title:       "About",
			Frontmatter: &parser.Frontmatter{},
			Type:        TypeDoc,
		},
		"en-blog-welcome": {
			ID:          "en-blog-welcome",
			URL:         "/en/blog/welcome/",
			Language:    "en",
			Title:       "Welcome",
			Frontmatter: &parser.Frontmatter{},
			Type:        TypeBlog,
		},
	}

	tree := NewNavigationBuilder().BuildTree(pages)
	languages := map[string]struct{}{
		"en": {},
		"ru": {},
	}

	generated := NewSectionIndexGenerator().GenerateMissing(pages, tree, "ru", languages)

	found := map[string]bool{}
	for _, p := range generated {
		found[p.URL] = true
	}

	if !found["/en/docs/"] {
		t.Fatalf("expected generated section index for /en/docs/, got: %#v", found)
	}
	if !found["/en/blog/"] {
		t.Fatalf("expected generated section index for /en/blog/, got: %#v", found)
	}
}

func TestSectionIndexGenerator_BlogSectionUsesPostPreviewsInDateOrder(t *testing.T) {
	latestDate := time.Date(2026, time.February, 20, 9, 0, 0, 0, time.UTC)
	olderDate := time.Date(2025, time.December, 10, 10, 0, 0, 0, time.UTC)
	oldestDate := time.Date(2024, time.June, 11, 10, 0, 0, 0, time.UTC)

	pages := map[string]*Page{
		"en-blog-latest": {
			ID:         "en-blog-latest",
			URL:        "/en/blog/latest/",
			Language:   "en",
			Title:      "Latest Post",
			SourcePath: "/tmp/latest.md",
			RawContent: "## Heading to skip\nFirst sentence for preview. Second sentence remains here.\n![Hidden image](img.png)\n| col | val |\n|---|---|\n<iframe src=\"https://example.com/embed\"></iframe>",
			Frontmatter: &parser.Frontmatter{
				Date: latestDate,
			},
			Type: TypeBlog,
		},
		"en-blog-older": {
			ID:         "en-blog-older",
			URL:        "/en/blog/older/",
			Language:   "en",
			Title:      "Older Post",
			SourcePath: "/tmp/older.md",
			RawContent: "Older story first line. Another sentence for the summary.",
			Frontmatter: &parser.Frontmatter{
				Date: olderDate,
			},
			Type: TypeBlog,
		},
		"en-blog-oldest": {
			ID:         "en-blog-oldest",
			URL:        "/en/blog/oldest/",
			Language:   "en",
			Title:      "Oldest Post",
			SourcePath: "/tmp/oldest.md",
			RawContent: "Old post text only.",
			Frontmatter: &parser.Frontmatter{
				Date: oldestDate,
			},
			Type: TypeBlog,
		},
	}

	tree := NewNavigationBuilder().BuildTree(pages)
	languages := map[string]struct{}{
		"en": {},
	}
	generated := NewSectionIndexGenerator().GenerateMissing(pages, tree, "en", languages)

	var blogIndex *Page
	for _, page := range generated {
		if page.URL == "/en/blog/" {
			blogIndex = page
			break
		}
	}
	if blogIndex == nil {
		t.Fatalf("expected generated /en/blog/ index page")
	}

	content := blogIndex.Content
	if !strings.Contains(content, "section-article-preview") {
		t.Fatalf("expected article preview markup, got: %s", content)
	}
	if strings.Contains(content, "Heading to skip") || strings.Contains(content, "Hidden image") || strings.Contains(content, "example.com/embed") {
		t.Fatalf("expected filtered preview text without heading/image/embed, got: %s", content)
	}

	latestPos := strings.Index(content, "Latest Post")
	olderPos := strings.Index(content, "Older Post")
	oldestPos := strings.Index(content, "Oldest Post")
	if latestPos == -1 || olderPos == -1 || oldestPos == -1 {
		t.Fatalf("expected all posts in generated content, got: %s", content)
	}
	if !(latestPos < olderPos && olderPos < oldestPos) {
		t.Fatalf("expected date-desc order Latest -> Older -> Oldest, got: %s", content)
	}
}
