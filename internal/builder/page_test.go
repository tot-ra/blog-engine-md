package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestURLGenerator_SelfNamedMarkdownFileUsesParentDirectory(t *testing.T) {
	g := NewURLGenerator("https://example.com")

	got := g.Generate("docs/beehive-sensors/beehive-sensors.md", &parser.Frontmatter{})
	want := "/docs/beehive-sensors/"
	if got != want {
		t.Fatalf("Generate() = %q, want %q", got, want)
	}
}

func TestURLGenerator_SlugEquivalentSelfNamedMarkdownFileUsesParentDirectory(t *testing.T) {
	g := NewURLGenerator("https://example.com")

	got := g.Generate("docs/web_app/web_app.md", &parser.Frontmatter{})
	want := "/docs/web_app/"
	if got != want {
		t.Fatalf("Generate() = %q, want %q", got, want)
	}
}

func TestURLGenerator_SelfNamedMarkdownFileRespectsExplicitSlug(t *testing.T) {
	g := NewURLGenerator("https://example.com")

	got := g.Generate("docs/beehive-sensors/beehive-sensors.md", &parser.Frontmatter{Slug: "custom"})
	want := "/docs/beehive-sensors/custom/"
	if got != want {
		t.Fatalf("Generate() = %q, want %q", got, want)
	}
}

func TestPageBuilderBuildsHTMLPartialWithoutMarkdownRendering(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "interactive.html")
	source := `<!--
---
title: Interactive Article
description: Trusted HTML partial
---
-->
<section class="demo"><h2>Try it</h2></section>
<style>.demo { color: red; }</style>
<script>window.demoLoaded = true;</script>`
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	page, err := NewPageBuilder("https://example.com", "en", map[string]struct{}{"en": {}}).Build(ContentFile{
		Path: path, RelativePath: "en/blog/interactive.html", ContentType: TypeHTML,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	for _, want := range []string{`<section class="demo">`, "<style>", "<script>"} {
		if !strings.Contains(page.Content, want) {
			t.Fatalf("HTML content missing %q: %s", want, page.Content)
		}
	}
	if strings.Contains(page.Content, "frontmatter") || len(page.TOC) != 0 {
		t.Fatalf("HTML metadata or TOC leaked: content=%s toc=%#v", page.Content, page.TOC)
	}
}
