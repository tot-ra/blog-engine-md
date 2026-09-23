package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/feed"
	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestBlogSectionFeedHelpers(t *testing.T) {
	if got := BlogSectionFeedRelativePath("/ru/blog/"); got != "/ru/blog/rss.xml" {
		t.Fatalf("BlogSectionFeedRelativePath = %q", got)
	}
	if got := BlogSectionURLFromPageURL("/ru/blog/tech/post/"); got != "/ru/blog/" {
		t.Fatalf("BlogSectionURLFromPageURL = %q", got)
	}

	index := &Page{
		URL:  "/ru/blog/",
		Type: TypeBlog,
	}
	post := &Page{
		URL:         "/ru/blog/post/",
		Type:        TypeBlog,
		Frontmatter: &parser.Frontmatter{Date: time.Now()},
	}
	if !IsBlogSectionIndexPage(index) {
		t.Fatal("expected blog index page")
	}
	if IsBlogSectionIndexPage(post) {
		t.Fatal("blog post should not be treated as section index")
	}
}

func TestGenerateBlogSectionFeeds(t *testing.T) {
	tmp := t.TempDir()
	date := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	cfg := config.DefaultConfig()
	cfg.Site.URL = "https://example.com"
	cfg.Site.Title = "Example"
	cfg.Build.OutputDir = tmp
	cfg.Feeds.RSS.Enabled = true
	cfg.Feeds.RSS.Items = 10

	b := &SiteBuilder{
		config: cfg,
		pages: map[string]*Page{
			"index": {
				ID:          "blog-index",
				URL:         "/ru/blog/",
				Language:    "ru",
				Title:       "Блог",
				Type:        TypeBlog,
				Frontmatter: &parser.Frontmatter{},
			},
			"new": {
				ID:           "new",
				URL:          "/ru/blog/new/",
				Language:     "ru",
				Title:        "Newest",
				Type:         TypeBlog,
				SourcePath:   "content/ru/blog/new.md",
				Description:  "desc",
				Frontmatter:  &parser.Frontmatter{Date: date, Tags: []string{"tech"}},
			},
			"old": {
				ID:           "old",
				URL:          "/ru/blog/old/",
				Language:     "ru",
				Title:        "Older",
				Type:         TypeBlog,
				SourcePath:   "content/ru/blog/old.md",
				Description:  "desc",
				Frontmatter:  &parser.Frontmatter{Date: date.Add(-24 * time.Hour)},
			},
		},
	}

	gen := feedGeneratorForTest(cfg)
	if err := b.generateBlogSectionFeeds(gen); err != nil {
		t.Fatalf("generateBlogSectionFeeds: %v", err)
	}

	feedPath := filepath.Join(tmp, "ru", "blog", "rss.xml")
	data, err := os.ReadFile(feedPath)
	if err != nil {
		t.Fatalf("read feed: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "https://example.com/ru/blog/rss.xml") {
		t.Fatalf("feed missing self link: %s", content)
	}
	if !strings.Contains(content, "https://example.com/ru/blog/new/") {
		t.Fatalf("feed missing newest item: %s", content)
	}
	if strings.Index(content, "Newest") > strings.Index(content, "Older") {
		t.Fatalf("expected newest item before older in feed")
	}
}

func feedGeneratorForTest(cfg *config.SiteConfig) *feed.FeedGenerator {
	return feed.NewFeedGenerator(
		cfg.Site.Title,
		cfg.Site.URL,
		cfg.Site.Language,
		cfg.Author.Name,
		cfg.Author.Email,
	)
}
