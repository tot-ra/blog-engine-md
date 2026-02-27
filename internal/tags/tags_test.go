package tags

import (
	"testing"
	"time"
)

func TestBuildTagIndex(t *testing.T) {
	pages := []PageSummary{
		{
			Title: "Go Basics",
			URL:   "/blog/go-basics/",
			Date:  time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
			Tags:  []string{"go", "tutorial"},
			Type:  "blog",
		},
		{
			Title: "Advanced Go",
			URL:   "/blog/advanced-go/",
			Date:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			Tags:  []string{"go", "advanced"},
			Type:  "blog",
		},
		{
			Title: "CSS Guide",
			URL:   "/blog/css-guide/",
			Date:  time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC),
			Tags:  []string{"css", "tutorial"},
			Type:  "blog",
		},
		{
			Title: "No Tags",
			URL:   "/blog/no-tags/",
			Date:  time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
			Tags:  nil,
			Type:  "blog",
		},
	}

	idx := BuildTagIndex(pages)

	// Check tag count
	if idx.Count("go") != 2 {
		t.Errorf("expected 2 pages with 'go', got %d", idx.Count("go"))
	}
	if idx.Count("tutorial") != 2 {
		t.Errorf("expected 2 pages with 'tutorial', got %d", idx.Count("tutorial"))
	}
	if idx.Count("css") != 1 {
		t.Errorf("expected 1 page with 'css', got %d", idx.Count("css"))
	}
	if idx.Count("nonexistent") != 0 {
		t.Errorf("expected 0 pages with 'nonexistent', got %d", idx.Count("nonexistent"))
	}

	// Check sorted tags list
	allTags := idx.Tags()
	expected := []string{"advanced", "css", "go", "tutorial"}
	if len(allTags) != len(expected) {
		t.Fatalf("expected %d tags, got %d: %v", len(expected), len(allTags), allTags)
	}
	for i, tag := range expected {
		if allTags[i] != tag {
			t.Errorf("tag[%d] = %q, want %q", i, allTags[i], tag)
		}
	}

	// Check date ordering (newest first) within "go" tag
	goPages := idx["go"]
	if goPages[0].Title != "Advanced Go" {
		t.Errorf("expected newest first, got %q", goPages[0].Title)
	}
}

func TestBuildTagIndexEmpty(t *testing.T) {
	idx := BuildTagIndex(nil)
	if len(idx.Tags()) != 0 {
		t.Error("empty input should produce empty tag index")
	}
}

func TestBuildTagIndexIgnoresEmptyTags(t *testing.T) {
	pages := []PageSummary{
		{Title: "Test", Tags: []string{"", "go", ""}},
	}
	idx := BuildTagIndex(pages)
	if len(idx.Tags()) != 1 {
		t.Errorf("expected 1 tag, got %d: %v", len(idx.Tags()), idx.Tags())
	}
}
