package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPageBuilder_RewritesLocalStaticAssetLinks(t *testing.T) {
	tmpDir := t.TempDir()
	pagePath := filepath.Join(tmpDir, "docs", "paper.md")
	if err := os.MkdirAll(filepath.Dir(pagePath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `---
title: Paper
---
[PDF](pdfs/paper.pdf)
[Archive](./downloads/archive.zip?download=1#top)
[External](https://example.com/file.pdf)
[Root](/files/root.pdf)
[Page](other-page.md)
<object data={require('./pdfs/embed.pdf').default} type="application/pdf"></object>`
	if err := os.WriteFile(pagePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	builder := NewPageBuilder("https://example.com", "en", map[string]struct{}{"en": {}})
	builder.SetMarkdownLinkResolver(func(destination, pageRelPath string) (string, bool) {
		return resolveLocalMarkdownLink(destination, pageRelPath, map[string]string{
			"docs/other-page.md": "/docs/other-page/",
		})
	})
	page, err := builder.Build(ContentFile{Path: pagePath, RelativePath: filepath.ToSlash(filepath.Join("docs", "paper.md"))})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	for _, want := range []string{
		`href="/assets/docs/pdfs/paper.pdf"`,
		`href="/assets/docs/downloads/archive.zip?download=1#top"`,
		`href="https://example.com/file.pdf"`,
		`href="/files/root.pdf"`,
		`href="/docs/other-page/"`,
		`<iframe class="pdf-embed" src="/assets/docs/pdfs/embed.pdf" title="PDF preview" loading="lazy" height="800"></iframe>`,
	} {
		if !strings.Contains(page.Content, want) {
			t.Fatalf("expected rendered content to contain %s:\n%s", want, page.Content)
		}
	}
}

func TestPageBuilder_StripsDuplicateTitleHeading(t *testing.T) {
	tmpDir := t.TempDir()
	pagePath := filepath.Join(tmpDir, "content", "feature.md")
	if err := os.MkdirAll(filepath.Dir(pagePath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `---
title: 🎥 Video streaming via API
---

# 🎥 Video streaming via API

## Purpose
Body text.`
	if err := os.WriteFile(pagePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	builder := NewPageBuilder("https://example.com", "en", map[string]struct{}{"en": {}})
	page, err := builder.Build(ContentFile{Path: pagePath, RelativePath: "feature.md"})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if strings.Contains(page.Content, "<h1") {
		t.Fatalf("duplicate title h1 was rendered:\n%s", page.Content)
	}
	if !strings.Contains(page.Content, `<h2 id="purpose">Purpose</h2>`) {
		t.Fatalf("expected subsequent headings to remain:\n%s", page.Content)
	}
	if strings.Contains(page.RawContent, "# 🎥 Video streaming via API") {
		t.Fatalf("duplicate title h1 remained in RawContent: %q", page.RawContent)
	}
	if len(page.TOC) != 1 || page.TOC[0].Text != "Purpose" {
		t.Fatalf("expected TOC to start at Purpose, got %#v", page.TOC)
	}
}

func TestPageBuilder_StripsDuplicateTitleHeadingAfterHeroBlock(t *testing.T) {
	tmpDir := t.TempDir()
	pagePath := filepath.Join(tmpDir, "content", "ethics.md")
	if err := os.MkdirAll(filepath.Dir(pagePath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `---
title: Ethics
---
<div>
![](hero.jpg)
</div>

# Ethics

## Bee Welfare
Body text.`
	if err := os.WriteFile(pagePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	builder := NewPageBuilder("https://example.com", "en", map[string]struct{}{"en": {}})
	page, err := builder.Build(ContentFile{Path: pagePath, RelativePath: "ethics.md"})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if strings.Contains(page.Content, `<h1 id="ethics">Ethics</h1>`) {
		t.Fatalf("duplicate title h1 after hero was rendered:\n%s", page.Content)
	}
	if !strings.Contains(page.Content, `<h2 id="bee-welfare">Bee Welfare</h2>`) {
		t.Fatalf("expected later headings to remain:\n%s", page.Content)
	}
}

func TestPageBuilder_KeepsDistinctLeadingH1(t *testing.T) {
	tmpDir := t.TempDir()
	pagePath := filepath.Join(tmpDir, "content", "feature.md")
	if err := os.MkdirAll(filepath.Dir(pagePath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `---
title: Page Title
---

# Different Heading

Body text.`
	if err := os.WriteFile(pagePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	builder := NewPageBuilder("https://example.com", "en", map[string]struct{}{"en": {}})
	page, err := builder.Build(ContentFile{Path: pagePath, RelativePath: "feature.md"})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !strings.Contains(page.Content, "Different Heading") {
		t.Fatalf("distinct h1 should remain:\n%s", page.Content)
	}
}

func TestResolveLocalMarkdownLink(t *testing.T) {
	pathToURL := map[string]string{
		"research/papers/Apis mellifera Bee Verification with IoT and Graph Neural Network.md": "/research/papers/apis-mellifera-bee-verification-with-iot-and-graph-neural-network/",
		"papers/Apis mellifera Bee Verification with IoT and Graph Neural Network.md":          "/research/papers/apis-mellifera-bee-verification-with-iot-and-graph-neural-network/",
	}

	got, ok := resolveLocalMarkdownLink("papers/Apis%20mellifera%20Bee%20Verification%20with%20IoT%20and%20Graph%20Neural%20Network.md", "research/papers/index.md", pathToURL)
	if !ok {
		t.Fatal("expected local markdown link to resolve")
	}
	want := "/research/papers/apis-mellifera-bee-verification-with-iot-and-graph-neural-network/"
	if got != want {
		t.Fatalf("resolved URL = %q, want %q", got, want)
	}
}
