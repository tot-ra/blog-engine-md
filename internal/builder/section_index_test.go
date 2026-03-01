package builder

import (
	"testing"

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

