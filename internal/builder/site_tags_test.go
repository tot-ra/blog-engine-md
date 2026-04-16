package builder

import (
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestCollectTagPagesIncludesNonBlogContent(t *testing.T) {
	b := &SiteBuilder{
		pages: map[string]*Page{
			"study-page": {
				URL:        "/est/study/mahtra_pohikool/2klass/palve_muusikas/",
				Language:   "est",
				SourcePath: "content/est/study/mahtra_pohikool/2klass/palve_muusikas/index.md",
				Type:       TypePage,
				Frontmatter: &parser.Frontmatter{
					Tags: []string{"palve-muusikas"},
				},
			},
			"blog-page": {
				URL:        "/est/blog/example/",
				Language:   "est",
				SourcePath: "content/est/blog/example.md",
				Type:       TypeBlog,
				Frontmatter: &parser.Frontmatter{
					Date: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
					Tags: []string{"palve-muusikas"},
				},
			},
			"generated-tag-page": {
				URL:      "/est/tags/palve-muusikas/",
				Language: "est",
				Type:     TypePage,
				Frontmatter: &parser.Frontmatter{
					Tags: []string{"palve-muusikas"},
				},
			},
			"untagged-page": {
				URL:        "/est/study/mahtra_pohikool/2klass/untagged/",
				Language:   "est",
				SourcePath: "content/est/study/mahtra_pohikool/2klass/untagged/index.md",
				Type:       TypePage,
				Frontmatter: &parser.Frontmatter{},
			},
		},
	}

	pages := b.collectTagPages()
	if len(pages) != 2 {
		t.Fatalf("expected 2 source-backed tagged pages, got %d", len(pages))
	}
	if pages[0].URL != "/est/blog/example/" {
		t.Fatalf("expected dated blog page first, got %q", pages[0].URL)
	}
	if pages[1].URL != "/est/study/mahtra_pohikool/2klass/palve_muusikas/" {
		t.Fatalf("expected study page to be included, got %q", pages[1].URL)
	}
}
