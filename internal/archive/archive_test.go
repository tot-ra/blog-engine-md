package archive

import (
	"testing"
	"time"
)

func TestBuildArchive(t *testing.T) {
	pages := []PageSummary{
		{
			Title: "Jan 2025 Post",
			URL:   "/blog/jan-2025/",
			Date:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			Title: "Dec 2024 Post",
			URL:   "/blog/dec-2024/",
			Date:  time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			Title: "Jan 2025 Second",
			URL:   "/blog/jan-2025-2/",
			Date:  time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			Title: "Mar 2024 Post",
			URL:   "/blog/mar-2024/",
			Date:  time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		},
	}

	archive := BuildArchive(pages)

	// Should have 2 years
	if len(archive) != 2 {
		t.Fatalf("expected 2 years, got %d", len(archive))
	}

	// Newest year first
	if archive[0].Year != 2025 {
		t.Errorf("expected 2025 first, got %d", archive[0].Year)
	}
	if archive[1].Year != 2024 {
		t.Errorf("expected 2024 second, got %d", archive[1].Year)
	}

	// 2025: 1 month (January), 2 posts
	if archive[0].Count != 2 {
		t.Errorf("2025 should have 2 posts, got %d", archive[0].Count)
	}
	if len(archive[0].Months) != 1 {
		t.Fatalf("2025 should have 1 month, got %d", len(archive[0].Months))
	}
	if archive[0].Months[0].Month != time.January {
		t.Errorf("2025 month should be January, got %v", archive[0].Months[0].Month)
	}

	// Check ordering within month (newest first)
	janPages := archive[0].Months[0].Pages
	if janPages[0].Title != "Jan 2025 Post" {
		t.Errorf("expected newest first in month, got %q", janPages[0].Title)
	}

	// 2024: 2 months (Dec, Mar), 2 posts total
	if archive[1].Count != 2 {
		t.Errorf("2024 should have 2 posts, got %d", archive[1].Count)
	}
	if len(archive[1].Months) != 2 {
		t.Fatalf("2024 should have 2 months, got %d", len(archive[1].Months))
	}
	// Months sorted descending
	if archive[1].Months[0].Month != time.December {
		t.Errorf("first month of 2024 should be December, got %v", archive[1].Months[0].Month)
	}
}

func TestBuildArchiveEmpty(t *testing.T) {
	result := BuildArchive(nil)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestBuildArchiveSkipsZeroDate(t *testing.T) {
	pages := []PageSummary{
		{Title: "No Date", URL: "/page/"},
		{Title: "Has Date", URL: "/blog/post/", Date: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	archive := BuildArchive(pages)
	if len(archive) != 1 {
		t.Fatalf("expected 1 year, got %d", len(archive))
	}
	if archive[0].Count != 1 {
		t.Errorf("expected 1 post, got %d", archive[0].Count)
	}
}
