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

func TestRenderPage_HideSidebarSectionOmitsLeftNav(t *testing.T) {
	cfg := sidebarRenderTestConfig(t)
	cfg.Navigation.Sidebar.Sections = map[string]config.SidebarSectionConfig{
		"about": {
			MatchPaths:  []string{"about"},
			HideSidebar: true,
		},
	}

	page := &Page{
		ID:          "about",
		URL:         "/en/about/",
		Language:    "en",
		Title:       "About",
		Content:     "<p>About</p>",
		Type:        TypeDoc,
		Frontmatter: &parser.Frontmatter{},
	}
	child := &Page{
		ID:          "about-job",
		URL:         "/en/about/work/",
		Language:    "en",
		Title:       "Work",
		Content:     "<p>Work</p>",
		Type:        TypeDoc,
		Frontmatter: &parser.Frontmatter{},
	}
	b := sidebarRenderTestBuilder(t, cfg, page, child)

	if err := b.renderPage(page); err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := readRenderedPage(t, cfg, "en/about/index.html")
	if strings.Contains(html, `<nav class="sidebar"`) {
		t.Fatalf("expected hideSidebar section to omit left nav, got sidebar in HTML")
	}
}

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

func TestRenderPage_DefaultLanguageDocsSidebarUsesUnprefixedLinks(t *testing.T) {
	cfg := sidebarRenderTestConfig(t)
	cfg.I18n.Languages = []config.LanguageConfig{
		{Code: "en", Label: "English"},
		{Code: "et", Label: "Eesti"},
	}
	cfg.Navigation.Sidebar.Sections = map[string]config.SidebarSectionConfig{
		"docs": {
			DefaultMode: "categories",
			EnableTime:  false,
			EnableGraph: false,
			MatchPaths:  []string{"docs"},
			SidebarRoot: "docs",
		},
	}

	docsIndex := &Page{
		ID:          "docs-index",
		URL:         "/docs/",
		Language:    "en",
		Title:       "Docs",
		Content:     "<p>Docs</p>",
		Type:        TypeDoc,
		Frontmatter: &parser.Frontmatter{},
	}
	child := &Page{
		ID:          "docs-beehive-sensors",
		URL:         "/docs/beehive-sensors/",
		Language:    "en",
		Title:       "Beehive sensors",
		Content:     "<p>Beehive sensors</p>",
		Type:        TypeDoc,
		Frontmatter: &parser.Frontmatter{},
	}
	b := sidebarRenderTestBuilder(t, cfg, docsIndex, child)

	if err := b.renderPage(docsIndex); err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := readRenderedPage(t, cfg, "docs/index.html")
	sidebar := sidebarNavHTML(t, html)
	if !strings.Contains(sidebar, `href="/docs/beehive-sensors/"`) {
		t.Fatalf("expected unprefixed docs child link in sidebar, got %q", sidebar)
	}
	if strings.Contains(sidebar, `href="/en/docs/`) {
		t.Fatalf("expected default-language docs sidebar to avoid /en/ links, got %q", sidebar)
	}
}

func TestRenderPage_TimeOnlySectionOmitsModeSwitch(t *testing.T) {
	falseVal := false
	cfg := sidebarRenderTestConfig(t)
	cfg.Navigation.Sidebar.Sections = map[string]config.SidebarSectionConfig{
		"events": {
			DefaultMode:      "time",
			EnableTime:       true,
			EnableCategories: &falseVal,
			EnableGraph:      false,
			MatchPaths:       []string{"events"},
			SidebarRoot:      "events",
		},
	}

	date := time.Date(2025, time.April, 28, 12, 0, 0, 0, time.UTC)
	index := &Page{
		ID:          "events-index",
		URL:         "/en/events/",
		Language:    "en",
		Title:       "Events",
		Content:     "<p>Events</p>",
		Type:        TypeDoc,
		Frontmatter: &parser.Frontmatter{},
	}
	post := &Page{
		ID:       "events-meetup",
		URL:      "/en/events/meetup/",
		Language: "en",
		Title:    "Meetup",
		Content:  "<p>Meetup</p>",
		Type:     TypeDoc,
		Frontmatter: &parser.Frontmatter{
			Date: date,
		},
	}
	b := sidebarRenderTestBuilder(t, cfg, index, post)

	if err := b.renderPage(index); err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := readRenderedPage(t, cfg, "en/events/index.html")
	sidebar := sidebarNavHTML(t, html)
	for _, want := range []string{
		`data-sidebar-mode="time"`,
		`data-sidebar-mode-pane="time"`,
		`href="/en/events/meetup/"`,
	} {
		if !strings.Contains(sidebar, want) {
			t.Fatalf("expected time-only events sidebar to contain %q, got %q", want, sidebar)
		}
	}
	for _, ban := range []string{
		`sidebar-mode-switch`,
		`data-sidebar-mode-btn=`,
		`data-sidebar-mode-pane="categories"`,
	} {
		if strings.Contains(sidebar, ban) {
			t.Fatalf("expected time-only events sidebar to omit %q, got %q", ban, sidebar)
		}
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

func TestRenderPage_SingleLanguageBlogPostKeepsLeftSidebar(t *testing.T) {
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

	date := time.Date(2026, time.June, 2, 12, 0, 0, 0, time.UTC)
	blogIndex := &Page{
		ID:          "blog-index",
		URL:         "/blog/",
		Language:    "en",
		Title:       "Blog",
		Content:     "<p>Blog index</p>",
		Type:        TypeBlog,
		Frontmatter: &parser.Frontmatter{},
	}
	post := &Page{
		ID:         "blog-post",
		URL:        "/blog/post/",
		Language:   "en",
		Title:      "Post",
		Content:    "<p>Post</p>",
		Type:       TypeBlog,
		SourcePath: "/tmp/post.md",
		Frontmatter: &parser.Frontmatter{
			Date: date,
		},
	}
	b := sidebarRenderTestBuilder(t, cfg, blogIndex, post)
	b.blogTimeline = map[string][]renderer.TimelineYear{
		"en": {
			{
				Year: 2026,
				Items: []renderer.TimelineItem{
					{Title: "Post", URL: "/blog/post/", Date: date},
				},
			},
		},
	}

	if err := b.renderPage(post); err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := readRenderedPage(t, cfg, "blog/post/index.html")
	sidebar := sidebarNavHTML(t, html)
	if !strings.Contains(sidebar, `class="sidebar blog-sidebar"`) {
		t.Fatalf("expected blog sidebar on single-language blog post, got %q", sidebar)
	}
	if !strings.Contains(sidebar, `href="/blog/post/" aria-current="page"`) {
		t.Fatalf("expected current blog post link inside left sidebar, got %q", sidebar)
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

func TestRenderPage_PublishesDiscoverableMarkdownAlternative(t *testing.T) {
	cfg := sidebarRenderTestConfig(t)
	cfg.Build.PublishMarkdown = true
	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "post.md")
	source := "---\ntitle: Agent-friendly post\ntags: [AI]\n---\n\n# Agent-friendly post\n\nOriginal markdown.\n"
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("write source markdown: %v", err)
	}

	page := &Page{
		ID:          "agent-friendly-post",
		URL:         "/en/blog/agent-friendly-post/",
		Language:    "en",
		SourcePath:  sourcePath,
		Title:       "Agent-friendly post",
		Content:     "<p>Original markdown.</p>",
		Type:        TypeBlog,
		Frontmatter: &parser.Frontmatter{},
	}
	b := sidebarRenderTestBuilder(t, cfg, page)

	if err := b.renderPage(page); err != nil {
		t.Fatalf("render page: %v", err)
	}

	html := readRenderedPage(t, cfg, "en/blog/agent-friendly-post/index.html")
	wantLink := `<link rel="alternate" type="text/markdown" href="/en/blog/agent-friendly-post/index.md" title="Markdown source">`
	if !strings.Contains(html, wantLink) {
		t.Fatalf("expected Markdown discovery link %q, got:\n%s", wantLink, html)
	}
	markdown := readRenderedPage(t, cfg, "en/blog/agent-friendly-post/index.md")
	if markdown != source {
		t.Fatalf("expected published Markdown to preserve the complete source; got %q", markdown)
	}
}

func TestRenderPage_DoesNotPublishMarkdownForGeneratedPage(t *testing.T) {
	cfg := sidebarRenderTestConfig(t)
	cfg.Build.PublishMarkdown = true
	page := &Page{
		ID:          "generated-blog-index",
		URL:         "/en/blog/",
		Language:    "en",
		Title:       "Blog",
		Content:     "<p>Generated listing.</p>",
		Type:        TypeBlog,
		Frontmatter: &parser.Frontmatter{},
	}
	b := sidebarRenderTestBuilder(t, cfg, page)

	if err := b.renderPage(page); err != nil {
		t.Fatalf("render generated page: %v", err)
	}

	html := readRenderedPage(t, cfg, "en/blog/index.html")
	if strings.Contains(html, `type="text/markdown"`) {
		t.Fatalf("generated page must not advertise unavailable Markdown, got:\n%s", html)
	}
	if _, err := os.Stat(filepath.Join(cfg.Build.OutputDir, "en/blog/index.md")); !os.IsNotExist(err) {
		t.Fatalf("generated page must not publish Markdown, stat error: %v", err)
	}
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
