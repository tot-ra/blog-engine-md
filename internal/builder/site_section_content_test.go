package builder

import (
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestSectionChildrenContent_RendersVideoPosterGrid(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{},
		pagesByURL: map[string]*Page{
			"/ru/talks/": {
				URL:      "/ru/talks/",
				Language: "ru",
				Title:    "Доклады",
				Frontmatter: &parser.Frontmatter{
					ShowChildren: true,
				},
			},
			"/ru/talks/bees/": {
				URL:        "/ru/talks/bees/",
				Language:   "ru",
				Title:      "Bees talk",
				RawContent: `<iframe src="https://www.youtube.com/embed/MtCDgqzdYnM"></iframe>`,
			},
			"/ru/talks/oauth/": {
				URL:        "/ru/talks/oauth/",
				Language:   "ru",
				Title:      "OAuth talk",
				RawContent: `<iframe src="https://player.vimeo.com/video/17136348?h=0b6ff5aa77"></iframe>`,
			},
			"/ru/talks/notes/": {
				URL:      "/ru/talks/notes/",
				Language: "ru",
				Title:    "Notes only",
			},
		},
		navTree: &NavTree{
			ByPath: map[string]*NavNode{
				"/ru/talks/": {
					URL: "/ru/talks/",
					Children: []*NavNode{
						{Title: "Bees talk", URL: "/ru/talks/bees/"},
						{Title: "OAuth talk", URL: "/ru/talks/oauth/"},
						{Title: "Notes only", URL: "/ru/talks/notes/"},
					},
				},
			},
		},
	}

	got := b.sectionChildrenContent(b.pagesByURL["/ru/talks/"])
	if strings.Contains(got, `<ul class="section-index">`) {
		t.Fatalf("expected media poster grid instead of plain list, got %q", got)
	}
	if !strings.Contains(got, `class="section-video-preview-list"`) {
		t.Fatalf("expected section-video-preview-list, got %q", got)
	}
	if !strings.Contains(got, `href="/ru/talks/bees/"`) || !strings.Contains(got, "https://i.ytimg.com/vi/MtCDgqzdYnM/hqdefault.jpg") {
		t.Fatalf("expected YouTube poster linking to talk page, got %q", got)
	}
	if !strings.Contains(got, `href="/ru/talks/oauth/"`) || !strings.Contains(got, "https://vumbnail.com/17136348.jpg") {
		t.Fatalf("expected Vimeo poster linking to talk page, got %q", got)
	}
	if strings.Contains(got, "youtube.com/embed") || strings.Contains(got, "player.vimeo.com") {
		t.Fatalf("expected image posters only, not embeds, got %q", got)
	}
	if !strings.Contains(got, "Notes only") {
		t.Fatalf("expected title-only child to remain in grid, got %q", got)
	}
}

func TestSectionChildrenContent_UsesConfiguredRecentEmbeds(t *testing.T) {
	hideChildren := false
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Navigation: config.Navigation{
				Sidebar: config.SidebarConfig{
					Sections: map[string]config.SidebarSectionConfig{
						"students_performance": {
							ShowChildrenList: &hideChildren,
							RecentEmbeds: config.RecentEmbedsConfig{
								Enabled:  true,
								Provider: "youtube",
								Limit:    2,
								SortBy:   "date",
								Title:    "Последние YouTube видео",
							},
						},
					},
				},
			},
		},
		pagesByURL: map[string]*Page{
			"/rus/students_performance/": {
				URL:      "/rus/students_performance/",
				Language: "rus",
				Title:    "Ученики",
				Frontmatter: &parser.Frontmatter{
					ShowChildren: true,
				},
			},
			"/rus/students_performance/a/": {
				URL:        "/rus/students_performance/a/",
				Language:   "rus",
				Title:      "Older",
				RawContent: "::youtube[aaaaaaaaaaa]",
				Frontmatter: &parser.Frontmatter{
					Date: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
				},
				ModifiedTime: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			},
			"/rus/students_performance/b/": {
				URL:        "/rus/students_performance/b/",
				Language:   "rus",
				Title:      "Newer",
				RawContent: "::youtube[bbbbbbbbbbb]",
				Frontmatter: &parser.Frontmatter{
					Date: time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC),
				},
				ModifiedTime: time.Date(2024, time.February, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		navTree: &NavTree{
			ByPath: map[string]*NavNode{
				"/rus/students_performance/": {
					URL: "/rus/students_performance/",
					Children: []*NavNode{
						{Title: "Category A", URL: "/rus/students_performance/category-a/"},
					},
				},
			},
		},
	}

	got := b.sectionChildrenContent(b.pagesByURL["/rus/students_performance/"])
	if !strings.Contains(got, "Последние YouTube видео") {
		t.Fatalf("expected configured recent embeds title, got %q", got)
	}
	if !strings.Contains(got, "Newer") || !strings.Contains(got, "Older") {
		t.Fatalf("expected recent embed entries in output, got %q", got)
	}
	if strings.Contains(got, "Category A") {
		t.Fatalf("expected child list to be hidden by config, got %q", got)
	}
}

func TestSectionChildrenContent_RecentEmbedsCanSortByModified(t *testing.T) {
	showChildren := false
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Navigation: config.Navigation{
				Sidebar: config.SidebarConfig{
					Sections: map[string]config.SidebarSectionConfig{
						"students_performance": {
							ShowChildrenList: &showChildren,
							RecentEmbeds: config.RecentEmbedsConfig{
								Enabled:  true,
								Provider: "youtube",
								Limit:    2,
								SortBy:   "modified",
								Title:    "Latest embeds",
							},
						},
					},
				},
			},
		},
		pagesByURL: map[string]*Page{
			"/rus/students_performance/": {
				URL:      "/rus/students_performance/",
				Language: "rus",
				Title:    "Ученики",
				Frontmatter: &parser.Frontmatter{
					ShowChildren: true,
				},
			},
			"/rus/students_performance/a/": {
				URL:        "/rus/students_performance/a/",
				Language:   "rus",
				Title:      "Old date new file",
				RawContent: "::youtube[aaaaaaaaaaa]",
				Frontmatter: &parser.Frontmatter{
					Date: time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC),
				},
				ModifiedTime: time.Date(2024, time.March, 2, 0, 0, 0, 0, time.UTC),
			},
			"/rus/students_performance/b/": {
				URL:        "/rus/students_performance/b/",
				Language:   "rus",
				Title:      "New date old file",
				RawContent: "::youtube[bbbbbbbbbbb]",
				Frontmatter: &parser.Frontmatter{
					Date: time.Date(2025, time.April, 3, 0, 0, 0, 0, time.UTC),
				},
				ModifiedTime: time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
			},
		},
		navTree: &NavTree{
			ByPath: map[string]*NavNode{
				"/rus/students_performance/": {
					URL:      "/rus/students_performance/",
					Children: []*NavNode{{Title: "Category A", URL: "/rus/students_performance/category-a/"}},
				},
			},
		},
	}

	got := b.sectionChildrenContent(b.pagesByURL["/rus/students_performance/"])
	if !strings.Contains(got, "Old date new file") || !strings.Contains(got, "New date old file") {
		t.Fatalf("expected both entries in output, got %q", got)
	}
	if strings.Index(got, "Old date new file") > strings.Index(got, "New date old file") {
		t.Fatalf("expected modified sort order, got %q", got)
	}
}
