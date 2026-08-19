package renderer

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/i18n"
)

func TestTemplateEngine_RenderBeforeLoadReturnsError(t *testing.T) {
	engine := NewTemplateEngine()

	_, err := engine.Render("page", PageData{})
	if err == nil {
		t.Fatal("expected an error when rendering before templates are loaded")
	}
	if !strings.Contains(err.Error(), "templates not loaded") {
		t.Fatalf("expected templates not loaded error, got %v", err)
	}
}

func TestTemplateEngine_LoadTemplatesUsesDefaultsWhenDirectoryMissing(t *testing.T) {
	engine := NewTemplateEngine()
	missingDir := filepath.Join(t.TempDir(), "missing")

	if err := engine.LoadTemplates(missingDir); err != nil {
		t.Fatalf("LoadTemplates() returned error: %v", err)
	}

	html, err := engine.RenderPage(PageData{
		Site:    config.SiteConfig{Site: config.Site{Title: "Example", Language: "en"}},
		Page:    Page{Title: "About"},
		Content: template.HTML("<p>Hello from content</p>"),
	})
	if err != nil {
		t.Fatalf("RenderPage() with default template returned error: %v", err)
	}

	for _, want := range []string{
		"<!DOCTYPE html>",
		"<title>About | Example</title>",
		"<h1>About</h1>",
		"<p>Hello from content</p>",
		"th, td { border: 1px solid var(--nav-border); padding: 8px;",
		`querySelectorAll("body img:not(.navbar-logo img)")`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected rendered HTML to contain %q, got:\n%s", want, html)
		}
	}
}

func TestTemplateEngine_HomepageRendersMarkdownAlternative(t *testing.T) {
	engine := NewTemplateEngine()
	missingDir := filepath.Join(t.TempDir(), "missing")
	if err := engine.LoadTemplates(missingDir); err != nil {
		t.Fatalf("LoadTemplates() returned error: %v", err)
	}

	html, err := engine.RenderPage(PageData{
		Page:        Page{Title: "Home", Layout: "homepage"},
		MarkdownURL: "/index.md",
	})
	if err != nil {
		t.Fatalf("RenderPage() homepage returned error: %v", err)
	}
	want := `<link rel="alternate" type="text/markdown" href="/index.md" title="Markdown source">`
	if !strings.Contains(html, want) {
		t.Fatalf("expected homepage to contain %q, got:\n%s", want, html)
	}
	for _, want := range []string{
		`document.documentElement.classList.add("js")`,
		`querySelectorAll("body img:not(.navbar-logo img)")`,
		`img[data-load-sequence]:not(.is-loaded)`,
		`requestAnimationFrame(function()`,
		`transition: opacity 120ms ease-out`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected homepage image loading behavior to contain %q, got:\n%s", want, html)
		}
	}
}

func TestTemplateEngine_DefaultTemplateRendersTagLinks(t *testing.T) {
	engine := NewTemplateEngine()
	missingDir := filepath.Join(t.TempDir(), "missing")

	if err := engine.LoadTemplates(missingDir); err != nil {
		t.Fatalf("LoadTemplates() returned error: %v", err)
	}

	html, err := engine.RenderPage(PageData{
		Site:        config.SiteConfig{Site: config.Site{Title: "Example", Language: "en"}},
		Page:        Page{Title: "Tagged"},
		Frontmatter: Frontmatter{Tags: []string{"Hello World"}},
		GraphTagURL: func(tag string) string {
			return "/graph/?tag=Hello+World"
		},
	})
	if err != nil {
		t.Fatalf("RenderPage() with default template tags returned error: %v", err)
	}
	if !strings.Contains(html, `<a class="tag" href="/graph/?tag=Hello&#43;World">#Hello World</a>`) {
		t.Fatalf("expected rendered tag link, got:\n%s", html)
	}
}

func TestTemplateEngine_LoadTemplatesUsesDefaultsWhenDirectoryIsEmpty(t *testing.T) {
	engine := NewTemplateEngine()
	templateDir := t.TempDir()

	if err := engine.LoadTemplates(templateDir); err != nil {
		t.Fatalf("LoadTemplates() returned error: %v", err)
	}

	html, err := engine.RenderPage(PageData{
		Site:    config.SiteConfig{Site: config.Site{Title: "Empty Dir Site", Language: "en"}},
		Page:    Page{Title: "Fallback Page"},
		Content: template.HTML("<p>Fallback content</p>"),
	})
	if err != nil {
		t.Fatalf("RenderPage() with default template returned error: %v", err)
	}
	if !strings.Contains(html, "<title>Fallback Page | Empty Dir Site</title>") {
		t.Fatalf("expected default template output, got:\n%s", html)
	}
}

func TestTemplateEngine_LoadTemplatesCustomDirectoryAndFunctions(t *testing.T) {
	engine := NewTemplateEngine()
	templateDir := t.TempDir()
	nestedDir := filepath.Join(templateDir, "layouts")
	if err := os.Mkdir(nestedDir, 0o755); err != nil {
		t.Fatalf("failed to create nested template dir: %v", err)
	}
	customTemplate := `Title={{.Page.Title}} Slug={{slugify "Hello, World!"}} Date={{formatDate .Frontmatter.Date "2006"}} Lower={{lower "LOUD"}} Content={{.Content}}`
	if err := os.WriteFile(filepath.Join(nestedDir, "custom.html"), []byte(customTemplate), 0o644); err != nil {
		t.Fatalf("failed to write custom template: %v", err)
	}

	if err := engine.LoadTemplates(templateDir); err != nil {
		t.Fatalf("LoadTemplates() returned error: %v", err)
	}

	html, err := engine.RenderPage(PageData{
		Page:        Page{Title: "Custom Page", Layout: "custom"},
		Frontmatter: Frontmatter{Date: time.Date(2024, time.June, 25, 0, 0, 0, 0, time.UTC)},
		Content:     template.HTML("<strong>safe</strong>"),
	})
	if err != nil {
		t.Fatalf("RenderPage() with custom template returned error: %v", err)
	}

	for _, want := range []string{
		"Title=Custom Page",
		"Slug=hello-world",
		"Date=2024",
		"Lower=loud",
		"Content=<strong>safe</strong>",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected custom rendered HTML to contain %q, got %q", want, html)
		}
	}
}

func TestTemplateEngine_LoadTemplatesReportsReadAndParseErrors(t *testing.T) {
	t.Run("parse error", func(t *testing.T) {
		engine := NewTemplateEngine()
		templateDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(templateDir, "broken.html"), []byte(`{{if}}`), 0o644); err != nil {
			t.Fatalf("failed to write broken template: %v", err)
		}

		err := engine.LoadTemplates(templateDir)
		if err == nil {
			t.Fatal("expected parse error")
		}
		if !strings.Contains(err.Error(), "failed to parse template") {
			t.Fatalf("expected parse error context, got %v", err)
		}
	})

	t.Run("read error", func(t *testing.T) {
		engine := NewTemplateEngine()
		templateDir := t.TempDir()
		unreadable := filepath.Join(templateDir, "unreadable.html")
		if err := os.WriteFile(unreadable, []byte(`hello`), 0o000); err != nil {
			t.Fatalf("failed to write unreadable template: %v", err)
		}
		defer os.Chmod(unreadable, 0o644)

		err := engine.LoadTemplates(templateDir)
		if err == nil {
			t.Fatal("expected read error")
		}
		if !strings.Contains(err.Error(), "failed to read template") {
			t.Fatalf("expected read error context, got %v", err)
		}
	})
}

func TestTemplateEngine_RenderReturnsTemplateExecutionError(t *testing.T) {
	engine := NewTemplateEngine()
	templateDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(templateDir, "needs-field.html"), []byte(`{{.Missing.Field}}`), 0o644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}
	if err := engine.LoadTemplates(templateDir); err != nil {
		t.Fatalf("LoadTemplates() returned error: %v", err)
	}

	_, err := engine.Render("needs-field", PageData{})
	if err == nil {
		t.Fatal("expected template execution error")
	}
	if !strings.Contains(err.Error(), "failed to render template needs-field") {
		t.Fatalf("expected render error context, got %v", err)
	}
}

func TestTemplateEngine_DefaultTemplateRendersRelatedArticles(t *testing.T) {
	engine := NewTemplateEngine()
	if err := engine.LoadTemplates(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("LoadTemplates() returned error: %v", err)
	}

	html, err := engine.RenderPage(PageData{
		Site:    config.SiteConfig{Site: config.Site{Title: "Example", Language: "en"}},
		Page:    Page{Title: "Current article", Language: "en"},
		UI:      i18n.UI("en"),
		Content: template.HTML("<p>Article body</p>"),
		Related: []RelatedArticle{{
			Title: "Related title", URL: "/blog/related/", Date: time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC),
			Excerpt: "A short preview.", ImageURL: "/images/related.jpg", Score: 0.8123456,
		}},
		PrevNext: &PrevNextLinks{Next: &NavLink{Title: "Next article", URL: "/blog/next/"}},
	})
	if err != nil {
		t.Fatalf("RenderPage() returned error: %v", err)
	}

	start := strings.Index(html, `<section class="related-articles"`)
	if start < 0 {
		t.Fatalf("related container not found in:\n%s", html)
	}
	end := strings.Index(html[start:], `</section>`)
	if end < 0 {
		t.Fatalf("related container is not closed in:\n%s", html)
	}
	container := html[start : start+end]
	for _, want := range []string{
		`<h2 id="related-articles-title">Related articles</h2>`,
		`class="section-article-preview" data-related-score="0.812346"`,
		`href="/blog/related/"`,
		`src="/images/related.jpg"`,
		`<h3><a href="/blog/related/">Related title</a></h3>`,
		`A short preview.`,
	} {
		if !strings.Contains(container, want) {
			t.Errorf("related container missing %q:\n%s", want, container)
		}
	}
	if strings.Index(html, `class="related-articles"`) > strings.Index(html, `class="prev-next"`) {
		t.Error("related articles must render before prev/next navigation")
	}
}

func TestTemplateEngine_DefaultTemplateOmitsEmptyRelatedArticles(t *testing.T) {
	engine := NewTemplateEngine()
	if err := engine.LoadTemplates(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("LoadTemplates() returned error: %v", err)
	}

	html, err := engine.RenderPage(PageData{
		Site: config.SiteConfig{Site: config.Site{Title: "Example", Language: "ru"}},
		Page: Page{Title: "Статья", Language: "ru"}, UI: i18n.UI("ru"),
	})
	if err != nil {
		t.Fatalf("RenderPage() returned error: %v", err)
	}
	if strings.Contains(html, `class="related-articles"`) || strings.Contains(html, "Похожие статьи") {
		t.Fatalf("empty related block rendered:\n%s", html)
	}
}
