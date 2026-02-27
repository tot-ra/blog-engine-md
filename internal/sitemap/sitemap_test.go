package sitemap

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	entries := []SitemapEntry{
		{
			URL:        "https://example.com/",
			LastMod:    time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			ChangeFreq: "weekly",
			Priority:   1.0,
		},
		{
			URL:        "https://example.com/blog/",
			LastMod:    time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			ChangeFreq: "weekly",
			Priority:   0.9,
		},
		{
			URL:        "https://example.com/blog/post/",
			LastMod:    time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
			ChangeFreq: "never",
			Priority:   0.8,
		},
	}

	result, err := Generate(entries)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify XML header
	if !strings.HasPrefix(result, xml.Header) {
		t.Error("Sitemap should start with XML header")
	}

	// Verify structure
	checks := []string{
		`xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`,
		"<url>",
		"<loc>https://example.com/</loc>",
		"<lastmod>2025-01-15</lastmod>",
		"<changefreq>weekly</changefreq>",
		"<priority>1</priority>",
		"<loc>https://example.com/blog/post/</loc>",
		"<changefreq>never</changefreq>",
		"<priority>0.8</priority>",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Sitemap missing expected content: %s\nGot:\n%s", check, result)
		}
	}

	// Verify valid XML
	var us urlSet
	if err := xml.Unmarshal([]byte(strings.TrimPrefix(result, xml.Header)), &us); err != nil {
		t.Fatalf("Sitemap is not valid XML: %v", err)
	}

	if len(us.URLs) != 3 {
		t.Errorf("expected 3 URLs, got %d", len(us.URLs))
	}
}

func TestGenerateEmpty(t *testing.T) {
	result, err := Generate(nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(result, "<urlset") {
		t.Error("Empty sitemap should still have urlset element")
	}
}

func TestGenerateNoLastMod(t *testing.T) {
	entries := []SitemapEntry{
		{
			URL:        "https://example.com/",
			ChangeFreq: "weekly",
			Priority:   1.0,
		},
	}

	result, err := Generate(entries)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if strings.Contains(result, "<lastmod>") {
		t.Error("Sitemap should not include empty lastmod")
	}
}
