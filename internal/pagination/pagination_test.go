package pagination

import (
	"testing"
)

func TestPaginateBasic(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e", "f", "g"}

	// Page 1, size 3
	result := Paginate(items, 3, 1, "/blog/")
	if len(result.Items) != 3 {
		t.Errorf("page 1: expected 3 items, got %d", len(result.Items))
	}
	if result.Items[0] != "a" || result.Items[2] != "c" {
		t.Errorf("page 1: unexpected items: %v", result.Items)
	}
	if result.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", result.TotalPages)
	}
	if result.TotalItems != 7 {
		t.Errorf("expected 7 total items, got %d", result.TotalItems)
	}
	if result.HasPrev {
		t.Error("page 1 should not have prev")
	}
	if !result.HasNext {
		t.Error("page 1 should have next")
	}
	if result.NextURL != "/blog/page/2/" {
		t.Errorf("unexpected next URL: %s", result.NextURL)
	}

	// Page 2
	result = Paginate(items, 3, 2, "/blog/")
	if len(result.Items) != 3 {
		t.Errorf("page 2: expected 3 items, got %d", len(result.Items))
	}
	if result.Items[0] != "d" {
		t.Errorf("page 2: expected 'd' first, got %q", result.Items[0])
	}
	if !result.HasPrev || !result.HasNext {
		t.Error("page 2 should have both prev and next")
	}
	if result.PrevURL != "/blog/" {
		t.Errorf("page 2 prev URL should be /blog/, got %s", result.PrevURL)
	}
	if result.NextURL != "/blog/page/3/" {
		t.Errorf("page 2 next URL: %s", result.NextURL)
	}

	// Page 3 (last)
	result = Paginate(items, 3, 3, "/blog/")
	if len(result.Items) != 1 {
		t.Errorf("page 3: expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0] != "g" {
		t.Errorf("page 3: expected 'g', got %q", result.Items[0])
	}
	if !result.HasPrev {
		t.Error("page 3 should have prev")
	}
	if result.HasNext {
		t.Error("page 3 should not have next")
	}
}

func TestPaginateEmpty(t *testing.T) {
	result := Paginate([]string{}, 10, 1, "/blog/")
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
	if result.TotalPages != 1 {
		t.Errorf("expected 1 total page, got %d", result.TotalPages)
	}
	if result.HasPrev || result.HasNext {
		t.Error("empty list should have no prev/next")
	}
}

func TestPaginateExactFit(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	result := Paginate(items, 3, 1, "/items/")
	if result.TotalPages != 2 {
		t.Errorf("expected 2 pages, got %d", result.TotalPages)
	}
	result = Paginate(items, 3, 2, "/items/")
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items on page 2, got %d", len(result.Items))
	}
}

func TestPaginatePageBeyondRange(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := Paginate(items, 2, 100, "/test/")
	if result.CurrentPage != 2 {
		t.Errorf("expected page clamped to 2, got %d", result.CurrentPage)
	}
}

func TestPaginatePageZero(t *testing.T) {
	items := []string{"a"}
	result := Paginate(items, 10, 0, "/test/")
	if result.CurrentPage != 1 {
		t.Errorf("expected page 1, got %d", result.CurrentPage)
	}
}

func TestPaginateURLNoTrailingSlash(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := Paginate(items, 1, 2, "/blog")
	if result.PrevURL != "/blog/" {
		t.Errorf("expected /blog/, got %s", result.PrevURL)
	}
	if result.NextURL != "/blog/page/3/" {
		t.Errorf("expected /blog/page/3/, got %s", result.NextURL)
	}
}
