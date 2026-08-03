package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
	"github.com/tot-ra/blog-engine/internal/renderer"
)

func newDefaultRootI18nBuilder(t *testing.T) *SiteBuilder {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Site.URL = "https://example.com"
	cfg.Build.OutputDir = t.TempDir()
	cfg.I18n.Default = "en"
	cfg.I18n.Languages = []config.LanguageConfig{
		{Code: "en", Label: "English"},
		{Code: "ru", Label: "Русский"},
	}
	cfg.Navigation.Header.Enabled = false
	cfg.Navigation.Breadcrumbs.Enabled = false
	cfg.Navigation.PrevNext.Enabled = false

	templates := renderer.NewTemplateEngine()
	if err := templates.LoadTemplates(filepath.Join(t.TempDir(), "missing-templates")); err != nil {
		t.Fatalf("load templates: %v", err)
	}

	b := NewSiteBuilder(cfg)
	b.templates = templates
	b.navTree = NewNavigationBuilder().BuildTree(b.pages)
	return b
}

func TestLocalizeInternalLinksKeepsDefaultLanguageAtRoot(t *testing.T) {
	b := newDefaultRootI18nBuilder(t)
	html := `<a href="/blog/post/">Post</a><img src="/graph/data.json"><a href="/docs/">Docs</a>`

	got := b.localizeInternalLinks(html, "en", "/blog/post/")
	if strings.Contains(got, "/en/blog/") || strings.Contains(got, "/en/docs/") || strings.Contains(got, "/en/graph/") {
		t.Fatalf("expected default-language links to remain root-relative, got %q", got)
	}
	if !strings.Contains(got, `href="/blog/post/"`) || !strings.Contains(got, `href="/docs/"`) || !strings.Contains(got, `src="/graph/data.json"`) {
		t.Fatalf("expected root-relative links to be preserved, got %q", got)
	}
}

func TestLocalizeInternalLinksPrefixesNonDefaultLanguage(t *testing.T) {
	b := newDefaultRootI18nBuilder(t)
	html := `<a href="/blog/post/">Post</a><img src="/graph/data.json">`

	got := b.localizeInternalLinks(html, "ru", "/ru/blog/post/")
	if !strings.Contains(got, `href="/ru/blog/post/"`) || !strings.Contains(got, `src="/ru/graph/data.json"`) {
		t.Fatalf("expected non-default language links to be prefixed, got %q", got)
	}
}

func TestBuildLanguageOptionsUsesRootForDefaultLanguage(t *testing.T) {
	b := newDefaultRootI18nBuilder(t)
	b.pagesByURL["/blog/post/"] = &Page{URL: "/blog/post/", Language: "en"}
	b.pagesByURL["/ru/blog/"] = &Page{URL: "/ru/blog/", Language: "ru"}

	options := b.buildLanguageOptions(&Page{URL: "/blog/post/", Language: "en"})
	if len(options) != 2 {
		t.Fatalf("expected 2 language options, got %d", len(options))
	}
	if options[0].Code != "en" || options[0].URL != "/blog/post/" {
		t.Fatalf("expected default English option to use root post URL, got %+v", options[0])
	}
	if options[1].Code != "ru" || options[1].URL != "/ru/blog/" {
		t.Fatalf("expected Russian option to fall back to /ru/blog/, got %+v", options[1])
	}
}

func TestBuildLanguageOptionsUsesExistingPrefixedDefaultLanguagePage(t *testing.T) {
	b := newDefaultRootI18nBuilder(t)
	b.pagesByURL["/en/about/"] = &Page{URL: "/en/about/", Language: "en"}
	b.pagesByURL["/ru/about/"] = &Page{URL: "/ru/about/", Language: "ru"}

	options := b.buildLanguageOptions(&Page{URL: "/ru/about/", Language: "ru"})
	if len(options) != 2 {
		t.Fatalf("expected 2 language options, got %d", len(options))
	}
	if options[0].Code != "en" || options[0].URL != "/en/about/" {
		t.Fatalf("expected existing prefixed English page, got %+v", options[0])
	}
	if options[1].Code != "ru" || options[1].URL != "/ru/about/" {
		t.Fatalf("expected existing prefixed Russian page, got %+v", options[1])
	}
}

func TestGeneratedTagAndArchivePagesUseRootForDefaultLanguage(t *testing.T) {
	b := newDefaultRootI18nBuilder(t)
	postDate := time.Date(2026, time.June, 8, 0, 0, 0, 0, time.UTC)
	post := &Page{
		ID:       "post",
		URL:      "/blog/post/",
		Language: "en",
		Title:    "Post",
		Type:     TypeBlog,
		Frontmatter: &parser.Frontmatter{
			Date: postDate,
			Tags: []string{"research"},
		},
	}

	if err := b.generateTagPages([]*Page{post}); err != nil {
		t.Fatalf("generateTagPages returned error: %v", err)
	}
	if b.pagesByURL["/tags/"] == nil || b.pagesByURL["/tags/research/"] == nil {
		t.Fatalf("expected default-language tag pages at root, got URLs %#v", pageURLs(b.pagesByURL))
	}
	if b.pagesByURL["/en/tags/"] != nil || b.pagesByURL["/en/tags/research/"] != nil {
		t.Fatalf("did not expect default-language tag pages under /en/, got URLs %#v", pageURLs(b.pagesByURL))
	}

	if err := b.generateArchivePages([]*Page{post}); err != nil {
		t.Fatalf("generateArchivePages returned error: %v", err)
	}
	if b.pagesByURL["/archive/"] == nil || b.pagesByURL["/archive/2026/"] == nil {
		t.Fatalf("expected default-language archive pages at root, got URLs %#v", pageURLs(b.pagesByURL))
	}
	if b.pagesByURL["/en/archive/"] != nil || b.pagesByURL["/en/archive/2026/"] != nil {
		t.Fatalf("did not expect default-language archive pages under /en/, got URLs %#v", pageURLs(b.pagesByURL))
	}
}

func pageURLs(pages map[string]*Page) []string {
	urls := make([]string, 0, len(pages))
	for url := range pages {
		urls = append(urls, url)
	}
	return urls
}

func TestGeneratedGraphSkipsDefaultLanguageMirror(t *testing.T) {
	b := newDefaultRootI18nBuilder(t)
	b.pages["post"] = &Page{
		ID:         "post",
		URL:        "/blog/post/",
		Language:   "en",
		Title:      "Post",
		Type:       TypeBlog,
		RawContent: "",
		Frontmatter: &parser.Frontmatter{
			Tags: []string{"research"},
		},
	}

	if err := b.generateGraph(); err != nil {
		t.Fatalf("generateGraph returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(b.config.Build.OutputDir, "graph", "index.html")); err != nil {
		t.Fatalf("expected root graph page: %v", err)
	}
	if _, err := os.Stat(filepath.Join(b.config.Build.OutputDir, "en", "graph", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("did not expect default-language graph mirror under /en/, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(b.config.Build.OutputDir, "ru", "graph", "index.html")); err != nil {
		t.Fatalf("expected non-default graph mirror under /ru/: %v", err)
	}
}
