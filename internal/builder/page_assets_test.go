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
