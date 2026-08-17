package builder

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
	"github.com/tot-ra/blog-engine/internal/related"
)

func TestRelatedForPageManualOverridesComputedAndHideWins(t *testing.T) {
	b := NewSiteBuilder(config.DefaultConfig())
	b.relatedEntries = []related.Entry{{Path: "en/blog/manual.md", URL: "/manual/", Title: "Manual"}}
	b.relatedByPageID = map[string][]related.RelatedMatch{"q": {{Path: "computed", Title: "Computed"}}}
	page := &Page{ID: "q", Frontmatter: &parser.Frontmatter{Related: []string{"manual"}}}
	got := b.relatedForPage(page)
	if len(got) != 1 || got[0].Title != "Manual" {
		t.Fatalf("manual override = %#v", got)
	}
	page.Frontmatter.HideRelated = true
	if got := b.relatedForPage(page); len(got) != 0 {
		t.Fatalf("hidden = %#v", got)
	}
}

func TestPrepareRelatedArticlesMissingCacheIsFailSoft(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Build.ContentDir = t.TempDir()
	cfg.Related.CachePath = filepath.Join(t.TempDir(), "missing.json")
	b := NewSiteBuilder(cfg)
	b.prepareRelatedArticles()
	if len(b.relatedByPageID) != 0 {
		t.Fatalf("related = %#v", b.relatedByPageID)
	}
}

func TestRelatedArticlesForPageUsesSharedCardHelpers(t *testing.T) {
	date := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	target := &Page{
		URL: "/blog/target/", Title: "Target title", Description: "Fallback description",
		SourcePath: "/content/blog/target.html", RawContent: `<style>.leak { color: red; }</style><p>Visible preview text.</p>`,
		Content: `<p><img src="/images/target.jpg" alt="Cover"></p>`, Frontmatter: &parser.Frontmatter{Date: date},
	}
	b := NewSiteBuilder(config.DefaultConfig())
	b.pagesByURL[target.URL] = target
	b.relatedByPageID = map[string][]related.RelatedMatch{"current": {{URL: target.URL, Score: 0.75}}}

	got := b.relatedArticlesForPage(&Page{ID: "current", Frontmatter: &parser.Frontmatter{}})
	if len(got) != 1 {
		t.Fatalf("related articles = %#v", got)
	}
	article := got[0]
	if article.Title != target.Title || article.URL != target.URL || !article.Date.Equal(date) || article.ImageURL != "/images/target.jpg" || article.Score != 0.75 {
		t.Fatalf("related article metadata = %#v", article)
	}
	if article.Excerpt != "Visible preview text." || strings.Contains(article.Excerpt, ".leak") {
		t.Fatalf("related excerpt must use shared HTML preview helper, got %q", article.Excerpt)
	}
}
