package seo

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateMeta_Basic(t *testing.T) {
	page := PageSEOInput{
		Title:       "Hello World",
		Description: "A test post",
		URL:         "/blog/hello/",
		Type:        "blog",
		Date:        time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		Tags:        []string{"go", "tutorial"},
	}

	cfg := Config{
		SiteTitle:     "My Blog",
		SiteURL:       "https://example.com",
		TitleTemplate: "%s | My Blog",
		AuthorName:    "Test Author",
	}

	meta := GenerateMeta(page, cfg)

	if meta.Title != "Hello World | My Blog" {
		t.Errorf("expected title 'Hello World | My Blog', got '%s'", meta.Title)
	}
	if meta.Description != "A test post" {
		t.Errorf("expected description 'A test post', got '%s'", meta.Description)
	}
	if meta.Canonical != "https://example.com/blog/hello/" {
		t.Errorf("expected canonical URL, got '%s'", meta.Canonical)
	}
	if meta.OG.Type != "article" {
		t.Errorf("expected og:type 'article', got '%s'", meta.OG.Type)
	}
	if meta.Twitter.Card != "summary" {
		t.Errorf("expected twitter card 'summary', got '%s'", meta.Twitter.Card)
	}
}

func TestGenerateMeta_WithImage(t *testing.T) {
	page := PageSEOInput{
		Title: "Post With Image",
		URL:   "/blog/post/",
		Type:  "blog",
		Image: "/img/cover.jpg",
	}

	cfg := Config{
		SiteTitle: "Blog",
		SiteURL:   "https://example.com",
	}

	meta := GenerateMeta(page, cfg)

	if meta.OG.Image != "https://example.com/img/cover.jpg" {
		t.Errorf("expected absolute OG image, got '%s'", meta.OG.Image)
	}
	if meta.Twitter.Card != "summary_large_image" {
		t.Errorf("expected large image card when image present, got '%s'", meta.Twitter.Card)
	}
}

func TestGenerateMeta_AutoDescription(t *testing.T) {
	page := PageSEOInput{
		Title:   "Post",
		URL:     "/post/",
		Type:    "page",
		Content: "# Header\n\nThis is the first paragraph of content that should be used as description.",
	}

	cfg := Config{
		SiteTitle: "Blog",
		SiteURL:   "https://example.com",
	}

	meta := GenerateMeta(page, cfg)

	if meta.Description == "" {
		t.Error("expected auto-generated description")
	}
	if strings.Contains(meta.Description, "# Header") {
		t.Error("description should not contain markdown headers")
	}
}

func TestRenderMetaTags(t *testing.T) {
	meta := &SEOMeta{
		Title:       "Test Page | Blog",
		Description: "A test page",
		Canonical:   "https://example.com/test/",
		Robots:      "index, follow",
		OG: OpenGraph{
			Title:    "Test Page",
			Type:     "website",
			URL:      "https://example.com/test/",
			SiteName: "Blog",
		},
		Twitter: TwitterCard{
			Card:  "summary",
			Title: "Test Page",
		},
	}

	html := RenderMetaTags(meta)

	if !strings.Contains(html, "<title>Test Page | Blog</title>") {
		t.Error("expected title tag")
	}
	if !strings.Contains(html, `rel="canonical"`) {
		t.Error("expected canonical link")
	}
	if !strings.Contains(html, `og:title`) {
		t.Error("expected og:title")
	}
	if !strings.Contains(html, `twitter:card`) {
		t.Error("expected twitter:card")
	}
}

func TestGenerateJSONLD(t *testing.T) {
	page := PageSEOInput{
		Title: "Test Post",
		URL:   "/blog/test/",
		Type:  "blog",
		Date:  time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		Tags:  []string{"go"},
	}

	cfg := Config{
		SiteURL:    "https://example.com",
		AuthorName: "Author",
	}

	meta := GenerateMeta(page, cfg)

	if meta.JSONLD == "" {
		t.Fatal("expected JSON-LD")
	}
	if !strings.Contains(meta.JSONLD, "BlogPosting") {
		t.Error("expected BlogPosting type in JSON-LD")
	}
	if !strings.Contains(meta.JSONLD, "Author") {
		t.Error("expected author in JSON-LD")
	}
	if !strings.Contains(meta.JSONLD, "2025-06-15") {
		t.Error("expected date in JSON-LD")
	}
}
