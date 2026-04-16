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

func TestSectionIndexGenerator_UsesFrontmatterOrderForSectionChildren(t *testing.T) {
	pages := map[string]*Page{
		"en-home": {
			ID:          "en-home",
			URL:         "/en/",
			Language:    "en",
			Title:       "Home",
			Frontmatter: &parser.Frontmatter{},
			Type:        TypePage,
		},
		"en-docs-zebra": {
			ID:       "en-docs-zebra",
			URL:      "/en/docs/zebra/",
			Language: "en",
			Title:    "Zebra",
			Frontmatter: &parser.Frontmatter{
				Order: 3,
			},
			Type: TypeDoc,
		},
		"en-docs-alpha": {
			ID:       "en-docs-alpha",
			URL:      "/en/docs/alpha/",
			Language: "en",
			Title:    "Alpha",
			Frontmatter: &parser.Frontmatter{
				Order: 2,
			},
			Type: TypeDoc,
		},
		"en-docs-beta": {
			ID:       "en-docs-beta",
			URL:      "/en/docs/beta/",
			Language: "en",
			Title:    "Beta",
			Frontmatter: &parser.Frontmatter{
				Order: 1,
			},
			Type: TypeDoc,
		},
	}

	tree := NewNavigationBuilder().BuildTree(pages)
	languages := map[string]struct{}{"en": {}}
	generated := NewSectionIndexGenerator().GenerateMissing(pages, tree, "en", languages)

	var docsIndex *Page
	for _, page := range generated {
		if page.URL == "/en/docs/" {
			docsIndex = page
			break
		}
	}
	if docsIndex == nil {
		t.Fatalf("expected generated /en/docs/ index page")
	}

	betaPos := strings.Index(docsIndex.Content, ">Beta<")
	alphaPos := strings.Index(docsIndex.Content, ">Alpha<")
	zebraPos := strings.Index(docsIndex.Content, ">Zebra<")
	if betaPos == -1 || alphaPos == -1 || zebraPos == -1 {
		t.Fatalf("expected ordered children in generated content, got: %s", docsIndex.Content)
	}
	if !(betaPos < alphaPos && alphaPos < zebraPos) {
		t.Fatalf("expected order Beta -> Alpha -> Zebra, got: %s", docsIndex.Content)
	}
}

func TestSectionChildrenHTML_MatrixOnlyIncludesClassLikeChildren(t *testing.T) {
	children := []SectionChild{
		{Title: "1-3 klass", URL: "/est/study/EG/1klass/"},
		{Title: "4-6 klass", URL: "/est/study/EG/4klass/"},
		{Title: "7 klass", URL: "/est/study/EG/7klass/"},
		{Title: "8 klass", URL: "/est/study/EG/8klass/"},
		{Title: "Laulud", URL: "/est/study/EG/playlist/"},
	}

	content := sectionChildrenHTML(children)
	if !strings.Contains(content, "1-3 klass") || !strings.Contains(content, "8 klass") {
		t.Fatalf("expected class-like children in matrix, got: %s", content)
	}
	if strings.Contains(content, "Laulud") {
		t.Fatalf("expected non-class child to be omitted from matrix, got: %s", content)
	}
}

func TestSectionChildrenHTML_UsesMatrixForTwoClassChildren(t *testing.T) {
	children := []SectionChild{
		{Title: "2 klass", URL: "/est/study/mahtra_pohikool/2klass/"},
		{Title: "3 klass", URL: "/est/study/mahtra_pohikool/3klass/"},
	}

	content := sectionChildrenHTML(children)
	if !strings.Contains(content, "section-index-card") {
		t.Fatalf("expected matrix card markup for two class children, got: %s", content)
	}
	if strings.Contains(content, "<ul class=\"section-index\">") {
		t.Fatalf("expected matrix instead of list for two class children, got: %s", content)
	}
}

func TestSectionChildrenHTML_UsesMatrixForClassURLsWithoutSpacedTitles(t *testing.T) {
	children := []SectionChild{
		{Title: "2klass", URL: "/est/study/mahtra_pohikool/2klass/"},
		{Title: "3klass", URL: "/est/study/mahtra_pohikool/3klass/"},
	}

	content := sectionChildrenHTML(children)
	if !strings.Contains(content, "section-index-card") {
		t.Fatalf("expected matrix card markup for class-like URLs, got: %s", content)
	}
	if strings.Contains(content, "<ul class=\"section-index\">") {
		t.Fatalf("expected matrix instead of list for class-like URLs, got: %s", content)
	}
}

func TestSectionBlogPostsHTML_RendersArticlePreviews(t *testing.T) {
	pages := map[string]*Page{
		"post-1": {
			ID:         "post-1",
			URL:        "/rus/blog/post-1/",
			Title:      "Post 1",
			Type:       TypeBlog,
			SourcePath: "/tmp/post-1.md",
			RawContent: "First sentence. Second sentence.",
			Frontmatter: &parser.Frontmatter{
				Date: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		"post-2": {
			ID:         "post-2",
			URL:        "/rus/blog/post-2/",
			Title:      "Post 2",
			Type:       TypeBlog,
			SourcePath: "/tmp/post-2.md",
			RawContent: "Another preview here.",
			Frontmatter: &parser.Frontmatter{
				Date: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	content := sectionBlogPostsHTML("/rus/blog/", pages)
	if !strings.Contains(content, "section-article-preview") {
		t.Fatalf("expected article preview markup, got: %s", content)
	}
	if strings.Contains(content, "<ul class=\"section-index\">") {
		t.Fatalf("expected preview list instead of generic section list, got: %s", content)
	}
	if !(strings.Index(content, "Post 1") < strings.Index(content, "Post 2")) {
		t.Fatalf("expected newest post first, got: %s", content)
	}
}
