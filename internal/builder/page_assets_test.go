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
	page, err := builder.Build(ContentFile{Path: pagePath, RelativePath: filepath.ToSlash(filepath.Join("docs", "paper.md"))})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	for _, want := range []string{
		`href="/assets/docs/pdfs/paper.pdf"`,
		`href="/assets/docs/downloads/archive.zip?download=1#top"`,
		`href="https://example.com/file.pdf"`,
		`href="/files/root.pdf"`,
		`href="other-page.md"`,
		`data="/assets/docs/pdfs/embed.pdf"`,
	} {
		if !strings.Contains(page.Content, want) {
			t.Fatalf("expected rendered content to contain %s:\n%s", want, page.Content)
		}
	}
}
