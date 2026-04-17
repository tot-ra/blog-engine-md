package builder

import (
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestSectionChildrenContent_UsesConfigDrivenRecentEmbeds(t *testing.T) {
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
								TitleI18n: map[string]string{
									"rus": "Последние YouTube видео",
								},
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
