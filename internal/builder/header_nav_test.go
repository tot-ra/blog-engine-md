package builder

import (
	"testing"

	"github.com/tot-ra/blog-engine/internal/config"
)

func TestHeaderItemVisibleForLanguage(t *testing.T) {
	item := config.HeaderItem{
		Title:     "Sheet Archive",
		Languages: []string{"rus"},
	}

	if !headerItemVisibleForLanguage(item, "rus") {
		t.Fatal("expected item to be visible for allowed language")
	}
	if headerItemVisibleForLanguage(item, "est") {
		t.Fatal("expected item to be hidden for other languages")
	}
	if !headerItemVisibleForLanguage(config.HeaderItem{Title: "About"}, "est") {
		t.Fatal("expected item without language restriction to remain visible")
	}
}
