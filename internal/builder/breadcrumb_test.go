package builder

import (
	"testing"

	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestBreadcrumb_RootPage(t *testing.T) {
	page := &Page{URL: "/", Title: "Home", Language: "en", Frontmatter: &parser.Frontmatter{}}
	gen := NewBreadcrumbGenerator(map[string]struct{}{"en": {}, "ru": {}})
	crumbs := gen.Generate(page, nil)

	if len(crumbs) != 1 {
		t.Fatalf("Expected 1 crumb for root, got %d", len(crumbs))
	}
	if crumbs[0].Title != "Home" {
		t.Errorf("Expected 'Home', got '%s'", crumbs[0].Title)
	}
}

func TestBreadcrumb_NestedPage(t *testing.T) {
	page := &Page{URL: "/en/blog/tech/golang/", Title: "Golang", Language: "en", Frontmatter: &parser.Frontmatter{}}
	gen := NewBreadcrumbGenerator(map[string]struct{}{"en": {}, "ru": {}})
	crumbs := gen.Generate(page, nil)

	// Home > Blog > Tech > Golang
	if len(crumbs) != 4 {
		t.Fatalf("Expected 4 crumbs, got %d", len(crumbs))
	}
	if crumbs[0].Title != "Home" || crumbs[0].URL != "/en/" {
		t.Errorf("First crumb should be Home, got %+v", crumbs[0])
	}
	if crumbs[1].Title != "Blog" {
		t.Errorf("Second crumb should be Blog, got %s", crumbs[1].Title)
	}
	if crumbs[3].IsCurrent != true {
		t.Error("Last crumb should be current")
	}
	if crumbs[3].URL != "" {
		t.Error("Current page crumb should have empty URL")
	}
}

func TestBreadcrumb_DefaultLanguageUnprefixedPage(t *testing.T) {
	page := &Page{URL: "/docs/web-app/", Title: "Web App", Language: "en", Frontmatter: &parser.Frontmatter{}}
	gen := NewBreadcrumbGeneratorWithDefault(map[string]struct{}{"en": {}, "et": {}}, "en")
	crumbs := gen.Generate(page, nil)

	if len(crumbs) != 3 {
		t.Fatalf("Expected 3 crumbs, got %d", len(crumbs))
	}
	if crumbs[0].URL != "/" {
		t.Fatalf("Expected default-language home URL to be '/', got %q", crumbs[0].URL)
	}
	if crumbs[1].URL != "/docs/" {
		t.Fatalf("Expected docs breadcrumb to stay unprefixed, got %q", crumbs[1].URL)
	}
}

func TestBreadcrumb_DefaultLanguagePrefixedPageKeepsLangHome(t *testing.T) {
	pages := map[string]*Page{
		"about": {
			ID:          "about",
			Title:       "Обо мне",
			URL:         "/rus/about/",
			Language:    "rus",
			Frontmatter: &parser.Frontmatter{},
		},
	}
	tree := NewNavigationBuilder().BuildTree(pages)
	gen := NewDefaultAwareBreadcrumbGenerator(map[string]struct{}{"rus": {}, "est": {}}, "rus")
	crumbs := gen.Generate(pages["about"], tree)

	if len(crumbs) != 2 {
		t.Fatalf("expected 2 crumbs, got %d", len(crumbs))
	}
	if crumbs[0].URL != "/rus/" {
		t.Fatalf("expected prefixed default-language home URL, got %q", crumbs[0].URL)
	}
	if crumbs[1].Title != "Обо мне" || !crumbs[1].IsCurrent {
		t.Fatalf("expected current about crumb from nav tree, got %+v", crumbs[1])
	}
}

func TestBreadcrumb_WithNavTree(t *testing.T) {
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "My Post", URL: "/en/blog/my-post/", Language: "en", Type: TypeBlog, Frontmatter: &parser.Frontmatter{}},
	}
	nb := NewNavigationBuilder()
	tree := nb.BuildTree(pages)

	page := pages["p1"]
	gen := NewBreadcrumbGenerator(map[string]struct{}{"en": {}, "ru": {}})
	crumbs := gen.Generate(page, tree)

	// Home > Blog > My Post
	if len(crumbs) != 3 {
		t.Fatalf("Expected 3 crumbs, got %d", len(crumbs))
	}
	// The last crumb should use the page title from the nav tree
	if crumbs[2].Title != "My Post" {
		t.Errorf("Expected 'My Post', got '%s'", crumbs[2].Title)
	}
}

func TestBreadcrumb_UTF8FallbackTitle(t *testing.T) {
	page := &Page{URL: "/ru/blog/вера/", Title: "Вера", Language: "ru", Frontmatter: &parser.Frontmatter{}}
	gen := NewBreadcrumbGenerator(map[string]struct{}{"en": {}, "ru": {}})
	crumbs := gen.Generate(page, nil)

	if len(crumbs) != 3 {
		t.Fatalf("Expected 3 crumbs, got %d", len(crumbs))
	}
	if crumbs[2].Title != "Вера" {
		t.Fatalf("Expected UTF-8 title 'Вера', got '%s'", crumbs[2].Title)
	}
}

func TestBreadcrumb_UsesLocalizedContentSectionLabels(t *testing.T) {
	pages := map[string]*Page{
		"sensor": {
			ID:         "sensor",
			Title:      "Mesitaru andurid",
			URL:        "/et/docs/beehive-sensors/beehive-sensors/",
			Language:   "et",
			SourcePath: "content/et/docs/beehive-sensors/beehive-sensors.md",
			Type:       TypeDoc,
			Frontmatter: &parser.Frontmatter{
				NavTitle: "Mesitaru andurid",
			},
		},
		"install": {
			ID:          "install",
			Title:       "Installation",
			URL:         "/et/docs/beehive-sensors/installation/",
			Language:    "et",
			SourcePath:  "content/et/docs/beehive-sensors/installation.md",
			Type:        TypeDoc,
			Frontmatter: &parser.Frontmatter{},
		},
	}
	tree := NewNavigationBuilder().BuildTree(pages)
	if node := tree.ByPath["/et/docs/beehive-sensors/"]; node == nil || node.Title != "Mesitaru andurid" {
		t.Fatalf("expected nav tree section label, got %#v", node)
	}
	gen := NewBreadcrumbGenerator(map[string]struct{}{"en": {}, "et": {}})

	crumbs := gen.Generate(pages["install"], tree)
	if len(crumbs) != 4 {
		t.Fatalf("expected 4 crumbs, got %d", len(crumbs))
	}
	if crumbs[2].Title != "Mesitaru andurid" {
		t.Fatalf("expected content-derived segment label, got %q", crumbs[2].Title)
	}
}

func TestBreadcrumb_DefaultLanguageUsesRootURLs(t *testing.T) {
	page := &Page{URL: "/research/papers/bee-study/", Title: "Bee Study", Language: "en", Frontmatter: &parser.Frontmatter{}}
	gen := NewDefaultAwareBreadcrumbGenerator(map[string]struct{}{"en": {}, "ru": {}}, "en")
	crumbs := gen.Generate(page, nil)

	if len(crumbs) != 4 {
		t.Fatalf("Expected 4 crumbs, got %d", len(crumbs))
	}
	if crumbs[0].URL != "/" {
		t.Fatalf("expected default-language home crumb at root, got %+v", crumbs[0])
	}
	if crumbs[1].URL != "/research/" || crumbs[2].URL != "/research/papers/" {
		t.Fatalf("expected default-language breadcrumb URLs at root, got %+v", crumbs)
	}
}
