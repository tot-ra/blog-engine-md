package builder

import (
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
	"github.com/tot-ra/blog-engine/internal/renderer"
)

func TestCloneSidebarNodeWithoutURL_RemovesExcludedDescendants(t *testing.T) {
	root := &renderer.NavNode{
		Title: "Root",
		URL:   "/",
		Children: []*renderer.NavNode{
			{Title: "Keep", URL: "/keep/"},
			{Title: "Drop", URL: "/drop/"},
			{
				Title: "Parent",
				URL:   "/parent/",
				Children: []*renderer.NavNode{
					nil,
					{Title: "Nested drop", URL: "/drop/"},
					{Title: "Nested keep", URL: "/parent/keep/"},
				},
			},
		},
	}

	cloned := cloneSidebarNodeWithoutURL(root, "/drop/")
	if cloned == nil {
		t.Fatal("expected cloned root")
	}
	if cloned == root {
		t.Fatal("expected a distinct cloned root")
	}
	if len(cloned.Children) != 2 {
		t.Fatalf("expected top-level excluded child to be removed, got %d children", len(cloned.Children))
	}
	if cloned.Children[0].URL != "/keep/" || cloned.Children[1].URL != "/parent/" {
		t.Fatalf("unexpected top-level children: %#v", cloned.Children)
	}
	if len(cloned.Children[1].Children) != 1 || cloned.Children[1].Children[0].URL != "/parent/keep/" {
		t.Fatalf("expected nil and excluded nested children to be removed, got %#v", cloned.Children[1].Children)
	}
	if len(root.Children[2].Children) != 3 {
		t.Fatalf("expected original tree to be left untouched, got %#v", root.Children[2].Children)
	}
}

func TestRenderRecentEmbedHTML_RecognizesSupportedProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		raw      string
		want     string
	}{
		{
			name:     "youtube",
			provider: "youtube",
			raw:      "intro\n::youtube[abcDEF123_4]\noutro",
			want:     "https://www.youtube-nocookie.com/embed/abcDEF123_4",
		},
		{
			name:     "vimeo",
			provider: "vimeo",
			raw:      "::vimeo[123456789]",
			want:     "https://player.vimeo.com/video/123456789",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderRecentEmbedHTML(tt.provider, tt.raw)
			if got == "" {
				t.Fatal("expected embed HTML")
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("expected %q in %q", tt.want, got)
			}
		})
	}

	if got := renderRecentEmbedHTML("youtube", "::youtube[too-short]"); got != "" {
		t.Fatalf("expected invalid YouTube shortcode to be ignored, got %q", got)
	}
	if got := renderRecentEmbedHTML("unknown", "::youtube[abcDEF123_4]"); got != "" {
		t.Fatalf("expected unknown provider to be ignored, got %q", got)
	}
}

func TestMergeHomepageConfig_OverlaysNonZeroFields(t *testing.T) {
	base := config.HomepageConfig{
		Enabled: true,
		Hero: config.HeroConfig{
			Enabled:     true,
			Background:  "/base.jpg",
			Title:       "Base title",
			Subtitle:    "Base subtitle",
			Description: "Base description",
			VideoEmbed:  "base-video",
			CTAButtons:  []config.CTAButton{{Label: "Base", URL: "/base"}},
		},
		Chat: config.HomepageChatConfig{
			Enabled:          true,
			BaseURL:          "https://chat.example.com",
			RecipientAgentID: "base-agent",
			Title:            "Base chat",
		},
		BlogShowcase: config.BlogShowcaseConfig{
			Enabled: true,
			Limit:   4,
			Title:   "Latest posts",
		},
		Projects:    []config.ProjectConfig{{Title: "Base project"}},
		SocialLinks: []config.SocialLinkGroup{{Title: "Base socials"}},
		CustomHTML:  "<p>base</p>",
	}
	override := config.HomepageConfig{
		Hero: config.HeroConfig{
			Background: "/localized.jpg",
			Title:      "Localized title",
		},
		Chat: config.HomepageChatConfig{
			Title: "Localized chat",
		},
		BlogShowcase: config.BlogShowcaseConfig{
			Title: "Neueste Beiträge",
		},
		HideProjects: true,
		CustomHTML:   "<p>localized</p>",
	}

	got := mergeHomepageConfig(base, override)
	if !got.Enabled || !got.Hero.Enabled || !got.Chat.Enabled {
		t.Fatalf("expected enabled flags inherited from base, got %#v", got)
	}
	if got.Hero.Background != "/localized.jpg" || got.Hero.Title != "Localized title" {
		t.Fatalf("expected localized hero fields, got %#v", got.Hero)
	}
	if got.Hero.Subtitle != "Base subtitle" || got.Hero.Description != "Base description" || got.Hero.VideoEmbed != "base-video" {
		t.Fatalf("expected unspecified hero fields inherited, got %#v", got.Hero)
	}
	if got.Chat.BaseURL != "https://chat.example.com" || got.Chat.RecipientAgentID != "base-agent" || got.Chat.Title != "Localized chat" {
		t.Fatalf("expected chat fields merged, got %#v", got.Chat)
	}
	if !got.BlogShowcase.Enabled || got.BlogShowcase.Limit != 4 || got.BlogShowcase.Title != "Neueste Beiträge" {
		t.Fatalf("expected localized blog showcase fields merged, got %#v", got.BlogShowcase)
	}
	if !got.HideProjects || got.CustomHTML != "<p>localized</p>" {
		t.Fatalf("expected localized homepage fields, got %#v", got)
	}
	if got.Projects[0].Title != "Base project" || got.SocialLinks[0].Title != "Base socials" {
		t.Fatalf("expected collection fields inherited when override is empty, got %#v", got)
	}
}

func TestHomepageBlogShowcaseSelectsLatestLocalizedPosts(t *testing.T) {
	cfg := config.DefaultConfig()
	b := NewSiteBuilder(cfg)
	b.pages = map[string]*Page{
		"new": {
			Title: "Newest", URL: "/ru/blog/new/", Language: "ru", Type: TypeBlog, SourcePath: "/tmp/new.md",
			Content: `<p><img src="cover.jpg" alt="Cover"></p>`, RawContent: "Newest preview.",
			Frontmatter: &parser.Frontmatter{Date: time.Date(2026, time.January, 3, 0, 0, 0, 0, time.UTC)},
		},
		"old": {
			Title: "Older", URL: "/ru/blog/old/", Language: "ru", Type: TypeBlog, SourcePath: "/tmp/old.md",
			RawContent: "Older preview.", Frontmatter: &parser.Frontmatter{Date: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)},
		},
		"en": {
			Title: "English", URL: "/blog/en/", Language: "en", Type: TypeBlog, SourcePath: "/tmp/en.md",
			Frontmatter: &parser.Frontmatter{Date: time.Date(2026, time.January, 4, 0, 0, 0, 0, time.UTC)},
		},
	}

	got := b.homepageBlogShowcase("ru", 2)
	if len(got) != 2 || got[0].Title != "Newest" || got[1].Title != "Older" {
		t.Fatalf("expected two latest Russian posts, got %#v", got)
	}
	if !strings.Contains(string(got[0].ImageHTML), `src="cover.jpg"`) || got[0].Description != "Newest preview." {
		t.Fatalf("expected image and excerpt on newest card, got %#v", got[0])
	}
}

func TestSiteRenderHelpers_DetectAbsoluteAndConvert(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Site.URL = "https://example.com/root/"
	cfg.I18n.Default = "en"
	cfg.I18n.Languages = []config.LanguageConfig{{Code: "en"}, {Code: "et"}}
	b := NewSiteBuilder(cfg)

	if got := b.detectLanguageFromURL("/et/docs/page/"); got != "et" {
		t.Fatalf("expected language from URL, got %q", got)
	}
	if got := b.detectLanguageFromURL("/docs/page/"); got != "en" {
		t.Fatalf("expected default language fallback, got %q", got)
	}
	if got := b.absolutePageURL("/docs/page/"); got != "https://example.com/root/docs/page/" {
		t.Fatalf("unexpected absolute URL: %q", got)
	}
	if got := b.absolutePageURL("https://other.example/page"); got != "https://other.example/page" {
		t.Fatalf("expected absolute URLs to pass through, got %q", got)
	}

	toc := convertTocItems([]*TocItem{{Level: 2, Text: "Parent", Anchor: "parent", Children: []*TocItem{{Level: 3, Text: "Child", Anchor: "child"}}}})
	if len(toc) != 1 || toc[0].Text != "Parent" || len(toc[0].Children) != 1 || toc[0].Children[0].Anchor != "child" {
		t.Fatalf("unexpected converted TOC: %#v", toc)
	}

	breadcrumbs := convertBreadcrumbs([]BreadcrumbItem{{Title: "Docs", URL: "/docs/"}, {Title: "Page", URL: "/docs/page/", IsCurrent: true}})
	if len(breadcrumbs) != 2 || !breadcrumbs[1].IsCurrent || breadcrumbs[0].URL != "/docs/" {
		t.Fatalf("unexpected converted breadcrumbs: %#v", breadcrumbs)
	}

	prevNext := convertPrevNext(&PrevNextLinks{Prev: &NavLink{Title: "Prev", URL: "/prev/", Type: "doc"}, Next: &NavLink{Title: "Next", URL: "/next/", Type: "blog"}})
	if prevNext.Prev == nil || prevNext.Next == nil || prevNext.Prev.Title != "Prev" || prevNext.Next.Type != "blog" {
		t.Fatalf("unexpected converted prev/next: %#v", prevNext)
	}
}
