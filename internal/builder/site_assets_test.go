package builder

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
	"github.com/tot-ra/blog-engine/internal/renderer"
)

func TestCopyAssetsCopiesStaticAssetsAndTriangleJSOnly(t *testing.T) {
	contentDir := t.TempDir()
	outputDir := t.TempDir()

	files := map[string]string{
		"downloads/file.pdf":     "pdf",
		"styles/site.css":        "css",
		"scripts/app.js":         "js",
		"triangle/widget.js":     "triangle js",
		"triangle/component.mjs": "triangle mjs",
	}
	index := &ContentIndex{}
	for rel, body := range files {
		path := filepath.Join(contentDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
		index.AssetFiles = append(index.AssetFiles, ContentFile{Path: path, RelativePath: filepath.FromSlash(rel), ContentType: TypeAsset})
	}

	b := NewSiteBuilder(config.DefaultConfig())
	b.config.Build.OutputDir = outputDir
	b.config.Build.ParallelWorkers = 1

	if err := b.copyAssets(index); err != nil {
		t.Fatalf("copy assets: %v", err)
	}

	assertFileContent(t, filepath.Join(outputDir, "assets", "downloads", "file.pdf"), "pdf")
	assertFileContent(t, filepath.Join(outputDir, "assets", "triangle", "widget.js"), "triangle js")
	assertFileContent(t, filepath.Join(outputDir, "assets", "triangle", "component.mjs"), "triangle mjs")
	assertFileMissing(t, filepath.Join(outputDir, "assets", "styles", "site.css"))
	assertFileMissing(t, filepath.Join(outputDir, "assets", "scripts", "app.js"))
}

func TestCopyConfiguredLogoCopiesOnlyAssetLogo(t *testing.T) {
	contentDir := t.TempDir()
	outputDir := t.TempDir()
	logoPath := filepath.Join(contentDir, "assets", "img", "logo.svg")
	if err := os.MkdirAll(filepath.Dir(logoPath), 0755); err != nil {
		t.Fatalf("mkdir logo dir: %v", err)
	}
	if err := os.WriteFile(logoPath, []byte("<svg/>"), 0644); err != nil {
		t.Fatalf("write logo: %v", err)
	}

	b := NewSiteBuilder(config.DefaultConfig())
	b.config.Build.ContentDir = contentDir
	b.config.Build.OutputDir = outputDir
	b.config.Site.Logo = "/assets/img/logo.svg"

	if err := b.copyConfiguredLogo(); err != nil {
		t.Fatalf("copy configured logo: %v", err)
	}
	assertFileContent(t, filepath.Join(outputDir, "assets", "img", "logo.svg"), "<svg/>")

	b.config.Site.Logo = "/img/not-an-asset.svg"
	if err := b.copyConfiguredLogo(); err != nil {
		t.Fatalf("non-asset logo should be ignored: %v", err)
	}
}

func TestCopyTriangleModulesCopiesOnlyTopLevelModules(t *testing.T) {
	contentDir := t.TempDir()
	outputDir := t.TempDir()
	triangleDir := filepath.Join(contentDir, "triangle")
	if err := os.MkdirAll(filepath.Join(triangleDir, "nested"), 0755); err != nil {
		t.Fatalf("mkdir triangle dir: %v", err)
	}
	for rel, body := range map[string]string{
		"widget.js":      "js",
		"module.mjs":     "mjs",
		"style.css":      "css",
		"nested/skip.js": "nested",
	} {
		path := filepath.Join(triangleDir, filepath.FromSlash(rel))
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	b := NewSiteBuilder(config.DefaultConfig())
	b.config.Build.ContentDir = contentDir
	b.config.Build.OutputDir = outputDir

	if err := b.copyTriangleModules(); err != nil {
		t.Fatalf("copy triangle modules: %v", err)
	}
	assertFileContent(t, filepath.Join(outputDir, "assets", "triangle", "widget.js"), "js")
	assertFileContent(t, filepath.Join(outputDir, "assets", "triangle", "module.mjs"), "mjs")
	assertFileMissing(t, filepath.Join(outputDir, "assets", "triangle", "style.css"))
	assertFileMissing(t, filepath.Join(outputDir, "assets", "triangle", "nested", "skip.js"))
}

func TestCollectBlogPostsAndByLanguageSortByDateDescending(t *testing.T) {
	older := datedPage("older", "/blog/older/", "en", TypeBlog, time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC))
	newer := datedPage("newer", "/blog/newer/", "en", TypeBlog, time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC))
	russian := datedPage("ru", "/ru/blog/post/", "ru", TypeBlog, time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC))
	doc := datedPage("doc", "/docs/page/", "en", TypeDoc, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC))
	redirect := datedPage("redirect", "/blog/2024-03-01-newer/", "en", TypeBlog, time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC))
	redirect.Frontmatter.HideNav = true
	redirect.Frontmatter.RedirectURL = "/blog/newer/"
	b := &SiteBuilder{pages: map[string]*Page{
		older.ID:    older,
		newer.ID:    newer,
		russian.ID:  russian,
		doc.ID:      doc,
		redirect.ID: redirect,
	}}

	posts := b.collectBlogPosts()
	if got := pageIDs(posts); !reflect.DeepEqual(got, []string{"newer", "ru", "older"}) {
		t.Fatalf("collectBlogPosts order = %#v", got)
	}

	byLanguage := b.collectBlogPostsByLanguage()
	if got := pageIDs(byLanguage["en"]); !reflect.DeepEqual(got, []string{"newer", "older"}) {
		t.Fatalf("English posts order = %#v", got)
	}
	if got := pageIDs(byLanguage["ru"]); !reflect.DeepEqual(got, []string{"ru"}) {
		t.Fatalf("Russian posts = %#v", got)
	}
}

func TestSidebarExcludeURLsLocalizesRulesForMatchingPages(t *testing.T) {
	b := NewSiteBuilder(config.DefaultConfig())
	b.config.I18n.Default = "en"
	b.config.Navigation.Sidebar.ExcludeRules = []config.SidebarExcludeRule{
		{MatchPaths: []string{"docs"}, ExcludePaths: []string{"docs/private", "/shared/hidden/"}},
		{MatchPaths: []string{"blog"}, ExcludePaths: []string{"blog/drafts"}},
	}

	excludes := b.sidebarExcludeURLs(&Page{URL: "/et/docs/intro/", Language: "et"})
	want := []string{"/et/docs/private/", "/et/shared/hidden/"}
	if !reflect.DeepEqual(excludes, want) {
		t.Fatalf("localized excludes = %#v, want %#v", excludes, want)
	}
	if got := b.sidebarExcludeURLs(&Page{URL: "/docs/intro/", Language: "en"}); !reflect.DeepEqual(got, []string{"/docs/private/", "/shared/hidden/"}) {
		t.Fatalf("default-language excludes = %#v", got)
	}
	if got := b.sidebarExcludeURLs(&Page{URL: "/about/", Language: "en"}); len(got) != 0 {
		t.Fatalf("unmatched page excludes = %#v", got)
	}
}
func TestBuildSectionTimelineReturnsNilWithoutRootURL(t *testing.T) {
	b := &SiteBuilder{pages: map[string]*Page{
		"page": datedPage("page", "/en/projects/page/", "en", TypePage, time.Date(2024, time.May, 1, 0, 0, 0, 0, time.UTC)),
	}}

	if got := b.buildSectionTimeline(nil, "en", 10); got != nil {
		t.Fatalf("nil root timeline = %#v, want nil", got)
	}
	if got := b.buildSectionTimeline(&renderer.NavNode{}, "en", 10); got != nil {
		t.Fatalf("empty root timeline = %#v, want nil", got)
	}
}

func TestBuildSectionTimelineFiltersAndLimitsSectionPages(t *testing.T) {
	newer := datedPage("newer", "/en/projects/newer/", "en", TypePage, time.Date(2024, time.May, 1, 0, 0, 0, 0, time.UTC))
	older := datedPage("older", "/en/projects/older/", "en", TypePage, time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC))
	hidden := datedPage("hidden", "/en/projects/hidden/", "en", TypePage, time.Date(2024, time.June, 1, 0, 0, 0, 0, time.UTC))
	hidden.Frontmatter.HideNav = true
	redirect := datedPage("redirect", "/en/projects/redirect/", "en", TypePage, time.Date(2024, time.July, 1, 0, 0, 0, 0, time.UTC))
	redirect.Frontmatter.RedirectURL = "https://example.com"
	wrongLanguage := datedPage("wrong-lang", "/et/projects/post/", "et", TypePage, time.Date(2024, time.August, 1, 0, 0, 0, 0, time.UTC))
	rootPage := datedPage("root", "/en/projects/", "en", TypePage, time.Date(2024, time.September, 1, 0, 0, 0, 0, time.UTC))
	undated := &Page{ID: "undated", URL: "/en/projects/undated/", Language: "en", Frontmatter: &parser.Frontmatter{}}

	b := &SiteBuilder{pages: map[string]*Page{
		newer.ID: newer, older.ID: older, hidden.ID: hidden, redirect.ID: redirect,
		wrongLanguage.ID: wrongLanguage, rootPage.ID: rootPage, undated.ID: undated,
	}}

	timeline := b.buildSectionTimeline(&renderer.NavNode{URL: "/en/projects/"}, "en", 1)
	if len(timeline) != 1 {
		t.Fatalf("timeline year count = %d, want 1: %#v", len(timeline), timeline)
	}
	if timeline[0].Year != 2024 || len(timeline[0].Items) != 1 || timeline[0].Items[0].Title != "newer" {
		t.Fatalf("timeline = %#v, want only newest visible project", timeline)
	}
}

func datedPage(id, url, language string, pageType PageType, date time.Time) *Page {
	return &Page{
		ID:       id,
		URL:      url,
		Language: language,
		Title:    id,
		Type:     pageType,
		Frontmatter: &parser.Frontmatter{
			Date: date,
		},
	}
}

func pageIDs(pages []*Page) []string {
	ids := make([]string, 0, len(pages))
	for _, page := range pages {
		ids = append(ids, page.ID)
	}
	return ids
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, string(data), want)
	}
}

func assertFileMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, stat err = %v", path, err)
	}
}
