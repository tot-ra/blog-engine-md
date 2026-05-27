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

func TestRenderPage_CategoryOnlySidebarSectionUsesRegularSidebar(t *testing.T) {
	cfg := sidebarRenderTestConfig(t)
	cfg.Navigation.Sidebar.Sections = map[string]config.SidebarSectionConfig{
		"docs": {
			DefaultMode: "categories",
			EnableTime:  false,
			EnableGraph: false,
			MatchPaths:  []string{"docs"},
			SidebarRoot: "docs",
		},
	}

	page := &Page{
		ID:          "docs-intro",
		URL:         "/en/docs/intro/",
		Language:    "en",
		Title:       "Intro",
		Content:     "<p>Intro</p>",
		Type:        TypeDoc,
		Frontmatter: &parser.Frontmatter{},
	}
	b := sidebarRenderTestBuilder(t, cfg, page)

	if err := b.renderPage(page); err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := readRenderedPage(t, cfg, "en/docs/intro/index.html")
	sidebar := sidebarNavHTML(t, html)
	if !strings.Contains(sidebar, `<nav class="sidebar"`) {
		t.Fatalf("expected regular sidebar, got %q", sidebar)
	}
	if strings.Contains(sidebar, "data-sidebar-mode") || strings.Contains(sidebar, "sidebar-mode-pane") {
		t.Fatalf("expected category-only section to avoid mode-sidebar markup, got %q", sidebar)
	}
}

func TestRenderPage_BlogSidebarStillUsesConfiguredModes(t *testing.T) {
	cfg := sidebarRenderTestConfig(t)
	cfg.Navigation.Sidebar.Sections = map[string]config.SidebarSectionConfig{
		"blog": {
			DefaultMode: "time",
			EnableTime:  true,
			EnableGraph: true,
			MatchPaths:  []string{"blog"},
			SidebarRoot: "blog",
		},
	}

	date := time.Date(2026, time.May, 27, 12, 0, 0, 0, time.UTC)
	page := &Page{
		ID:       "blog-post",
		URL:      "/en/blog/post/",
		Language: "en",
		Title:    "Post",
		Content:  "<p>Post</p>",
		Type:     TypeBlog,
		Frontmatter: &parser.Frontmatter{
			Date: date,
		},
	}
	b := sidebarRenderTestBuilder(t, cfg, page)
	b.blogTimeline = map[string][]renderer.TimelineYear{
		"en": {
			{
				Year: 2026,
				Items: []renderer.TimelineItem{
					{Title: "Post", URL: "/en/blog/post/", Date: date},
				},
			},
		},
	}

	if err := b.renderPage(page); err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := readRenderedPage(t, cfg, "en/blog/post/index.html")
	sidebar := sidebarNavHTML(t, html)
	for _, expected := range []string{
		`data-sidebar-mode="time"`,
		`data-sidebar-mode-btn="time"`,
		`data-sidebar-mode-btn="graph"`,
	} {
		if !strings.Contains(sidebar, expected) {
			t.Fatalf("expected blog sidebar to contain %q, got %q", expected, sidebar)
		}
	}
}

func sidebarRenderTestConfig(t *testing.T) *config.SiteConfig {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Build.OutputDir = t.TempDir()
	cfg.Site.URL = "https://example.com"
	cfg.I18n.Default = "en"
	cfg.I18n.Languages = []config.LanguageConfig{{Code: "en", Label: "English"}}
	cfg.Navigation.Header.Enabled = false
	cfg.Navigation.Breadcrumbs.Enabled = false
	cfg.Navigation.PrevNext.Enabled = false
	cfg.Navigation.Sidebar.Collapsed = true
	cfg.Navigation.Sidebar.MaxDepth = 4
	return cfg
}

func sidebarRenderTestBuilder(t *testing.T, cfg *config.SiteConfig, pages ...*Page) *SiteBuilder {
	t.Helper()

	b := NewSiteBuilder(cfg)
	templates := renderer.NewTemplateEngine()
	if err := templates.LoadTemplates(filepath.Join(t.TempDir(), "missing-templates")); err != nil {
		t.Fatalf("load templates: %v", err)
	}
	b.templates = templates
	b.pages = make(map[string]*Page, len(pages))
	b.pagesByURL = make(map[string]*Page, len(pages))
	for _, page := range pages {
		b.pages[page.ID] = page
		b.pagesByURL[page.URL] = page
	}
	b.navTree = NewNavigationBuilder().BuildTree(b.pages)
	return b
}

func readRenderedPage(t *testing.T, cfg *config.SiteConfig, rel string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(cfg.Build.OutputDir, rel))
	if err != nil {
		t.Fatalf("read rendered page: %v", err)
	}
	return string(data)
}

func sidebarNavHTML(t *testing.T, html string) string {
	t.Helper()

	start := strings.Index(html, `<nav class="sidebar`)
	if start < 0 {
		t.Fatalf("expected sidebar nav in rendered HTML")
	}
	end := strings.Index(html[start:], `</nav>`)
	if end < 0 {
		t.Fatalf("expected sidebar nav closing tag in rendered HTML")
	}
	return html[start : start+end+len(`</nav>`)]
}
