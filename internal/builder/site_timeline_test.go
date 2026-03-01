package builder

import (
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestBuildBlogTimeline_GroupsByYearOnly(t *testing.T) {
	posts := []*Page{
		{
			Title: "Newest",
			URL:   "/en/blog/newest/",
			Frontmatter: &parser.Frontmatter{
				Date: time.Date(2026, time.January, 12, 10, 0, 0, 0, time.UTC),
			},
		},
		{
			Title: "Middle",
			URL:   "/en/blog/middle/",
			Frontmatter: &parser.Frontmatter{
				Date: time.Date(2026, time.March, 9, 10, 0, 0, 0, time.UTC),
			},
		},
		{
			Title: "Older Year",
			URL:   "/en/blog/older/",
			Frontmatter: &parser.Frontmatter{
				Date: time.Date(2025, time.December, 1, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	timeline := buildBlogTimeline(posts, 10)
	if len(timeline) != 2 {
		t.Fatalf("expected 2 years, got %d", len(timeline))
	}

	if timeline[0].Year != 2026 {
		t.Fatalf("expected first year 2026, got %d", timeline[0].Year)
	}
	if len(timeline[0].Items) != 2 {
		t.Fatalf("expected 2 items in 2026 group, got %d", len(timeline[0].Items))
	}
	if timeline[0].Items[0].Title != "Middle" || timeline[0].Items[1].Title != "Newest" {
		t.Fatalf("expected date-desc order inside year, got %#v", timeline[0].Items)
	}
	if timeline[1].Year != 2025 || len(timeline[1].Items) != 1 {
		t.Fatalf("expected second group for 2025 with one item, got %#v", timeline[1])
	}
}
