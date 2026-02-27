package feed

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestGenerateRSS(t *testing.T) {
	gen := NewFeedGenerator("Test Blog", "https://example.com", "en", "John", "john@example.com")

	items := []FeedItem{
		{
			Title:       "First Post",
			URL:         "https://example.com/blog/first/",
			Date:        time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Description: "<p>Hello world</p>",
			Categories:  []string{"go", "tech"},
			GUID:        "https://example.com/blog/first/",
		},
		{
			Title:       "Second Post",
			URL:         "https://example.com/blog/second/",
			Date:        time.Date(2025, 1, 10, 8, 0, 0, 0, time.UTC),
			Description: "<p>Another post</p>",
			Categories:  nil,
		},
	}

	result, err := gen.GenerateRSS(items, "rss.xml")
	if err != nil {
		t.Fatalf("GenerateRSS failed: %v", err)
	}

	// Verify XML header
	if !strings.HasPrefix(result, xml.Header) {
		t.Error("RSS should start with XML header")
	}

	// Verify basic structure
	checks := []string{
		`<rss version="2.0"`,
		`xmlns:atom="http://www.w3.org/2005/Atom"`,
		"<channel>",
		"<title>Test Blog</title>",
		"<link>https://example.com</link>",
		"<language>en</language>",
		"<lastBuildDate>",
		`<atom:link href="https://example.com/rss.xml" rel="self" type="application/rss+xml"`,
		"<item>",
		"<title>First Post</title>",
		"<link>https://example.com/blog/first/</link>",
		`isPermaLink="true"`,
		"<category>go</category>",
		"<category>tech</category>",
		"<title>Second Post</title>",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("RSS missing expected content: %s", check)
		}
	}

	// Verify it's valid XML
	var rss rssRoot
	if err := xml.Unmarshal([]byte(strings.TrimPrefix(result, xml.Header)), &rss); err != nil {
		t.Fatalf("RSS is not valid XML: %v", err)
	}

	if len(rss.Channel.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(rss.Channel.Items))
	}
}

func TestGenerateAtom(t *testing.T) {
	gen := NewFeedGenerator("Test Blog", "https://example.com", "en", "John", "john@example.com")

	items := []FeedItem{
		{
			Title:       "First Post",
			URL:         "https://example.com/blog/first/",
			Date:        time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Description: "<p>Hello world</p>",
			Categories:  []string{"go"},
			GUID:        "https://example.com/blog/first/",
		},
	}

	result, err := gen.GenerateAtom(items, "atom.xml")
	if err != nil {
		t.Fatalf("GenerateAtom failed: %v", err)
	}

	// Verify XML header
	if !strings.HasPrefix(result, xml.Header) {
		t.Error("Atom should start with XML header")
	}

	// Verify basic structure
	checks := []string{
		`xmlns="http://www.w3.org/2005/Atom"`,
		"<title>Test Blog</title>",
		"<id>https://example.com/</id>",
		`<link href="https://example.com/" rel="alternate"`,
		`<link href="https://example.com/atom.xml" rel="self" type="application/atom+xml"`,
		"<name>John</name>",
		"<email>john@example.com</email>",
		"<entry>",
		"<title>First Post</title>",
		`<link href="https://example.com/blog/first/"`,
		`<category term="go"`,
		"<published>2025-01-15T10:00:00Z</published>",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Atom missing expected content: %s", check)
		}
	}
}

func TestGenerateRSSEmptyItems(t *testing.T) {
	gen := NewFeedGenerator("Test", "https://example.com", "en", "", "")
	result, err := gen.GenerateRSS(nil, "rss.xml")
	if err != nil {
		t.Fatalf("GenerateRSS failed: %v", err)
	}
	if !strings.Contains(result, "<channel>") {
		t.Error("RSS with no items should still have channel")
	}
}

func TestGenerateAtomNoAuthor(t *testing.T) {
	gen := NewFeedGenerator("Test", "https://example.com", "en", "", "")
	result, err := gen.GenerateAtom(nil, "atom.xml")
	if err != nil {
		t.Fatalf("GenerateAtom failed: %v", err)
	}
	if strings.Contains(result, "<author>") {
		t.Error("Atom with no author should not contain author element")
	}
}
