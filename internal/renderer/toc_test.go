package renderer

import (
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/i18n"
)

func TestRenderTOC_Empty(t *testing.T) {
	result := RenderTOC(nil, i18n.UI("en"))
	if result != "" {
		t.Errorf("Expected empty string for nil items, got %q", result)
	}
}

func TestRenderTOC_Basic(t *testing.T) {
	items := []*TocItem{
		{Level: 2, Text: "Introduction", Anchor: "introduction"},
		{Level: 2, Text: "Details", Anchor: "details", Children: []*TocItem{
			{Level: 3, Text: "Sub Detail", Anchor: "sub-detail"},
		}},
	}

	result := string(RenderTOC(items, i18n.UI("en")))

	if !strings.Contains(result, `class="toc"`) {
		t.Error("Expected toc class in output")
	}
	if !strings.Contains(result, `href="#introduction"`) {
		t.Error("Expected introduction anchor")
	}
	if !strings.Contains(result, `href="#details"`) {
		t.Error("Expected details anchor")
	}
	if !strings.Contains(result, `href="#sub-detail"`) {
		t.Error("Expected sub-detail anchor")
	}
	if !strings.Contains(result, `class="toc-sublist"`) {
		t.Error("Expected nested sublist")
	}
}

func TestRenderSidebar_Empty(t *testing.T) {
	result := RenderSidebar(nil, "/", 3, true, i18n.UI("en"))
	if result != "" {
		t.Errorf("Expected empty string for nil root, got %q", result)
	}
}

func TestRenderSidebar_ActiveState(t *testing.T) {
	root := &NavNode{
		ID:   "root",
		Type: "section",
		Children: []*NavNode{
			{
				ID: "docs", Title: "Docs", URL: "/docs/", Type: "section",
				Children: []*NavNode{
					{ID: "about", Title: "About", URL: "/docs/about/", Type: "page"},
					{ID: "faq", Title: "FAQ", URL: "/docs/faq/", Type: "page"},
				},
			},
		},
	}

	result := string(RenderSidebar(root, "/docs/about/", 3, true, i18n.UI("en")))

	// "about" should be active
	if !strings.Contains(result, `aria-current="page"`) {
		t.Error("Expected aria-current on active page")
	}
	// "docs" section should be expanded (ancestor of active)
	if !strings.Contains(result, "expanded") {
		t.Error("Expected expanded class on ancestor section")
	}
}

func TestRenderSidebar_SectionHeadKeepsToggleAfterLabel(t *testing.T) {
	root := &NavNode{
		ID:   "root",
		Type: "section",
		Children: []*NavNode{
			{
				ID: "docs", Title: "📚 Docs", URL: "/docs/", Type: "section",
				Children: []*NavNode{
					{ID: "intro", Title: "Intro", URL: "/docs/intro/", Type: "page"},
				},
			},
		},
	}

	result := string(RenderSidebar(root, "/docs/intro/", 3, true, i18n.UI("en")))
	labelIndex := strings.Index(result, `<a href="/docs/">📚 Docs</a>`)
	toggleIndex := strings.Index(result, `<button class="sidebar-toggle"`)

	if labelIndex == -1 || toggleIndex == -1 {
		t.Fatalf("Expected section label and toggle in output, got %s", result)
	}
	if labelIndex > toggleIndex {
		t.Error("Expected expandable folder label to render before its toggle button")
	}
}

func TestRenderSidebar_DefaultCollapsed(t *testing.T) {
	root := &NavNode{
		ID:   "root",
		Type: "section",
		Children: []*NavNode{
			{
				ID: "docs", Title: "Docs", URL: "/docs/", Type: "section",
				Children: []*NavNode{
					{
						ID: "guide", Title: "Guide", URL: "/docs/guide/", Type: "section",
						Children: []*NavNode{
							{ID: "start", Title: "Start", URL: "/docs/guide/start/", Type: "page"},
						},
					},
				},
			},
		},
	}

	result := string(RenderSidebar(root, "/blog/", 4, true, i18n.UI("en")))

	// First layer section should be collapsed by default.
	if !strings.Contains(result, `aria-label="Toggle section Docs" aria-expanded="false"`) {
		t.Error("Expected first layer section to be collapsed")
	}
	// Deeper section should start collapsed when not active.
	if !strings.Contains(result, `aria-label="Toggle section Guide" aria-expanded="false"`) {
		t.Error("Expected deeper section to be collapsed by default")
	}
}

func TestRenderModeSidebar_RendersAvailableModesAndDefaultTime(t *testing.T) {
	root := &NavNode{
		ID:   "root",
		Type: "section",
		Children: []*NavNode{
			{ID: "blog", Title: "Blog", URL: "/blog/", Type: "section"},
			{ID: "docs", Title: "Docs", URL: "/docs/", Type: "section"},
		},
	}
	timeline := []TimelineYear{
		{
			Year: 2024,
			Items: []TimelineItem{
				{Title: "Current <Post>", URL: "/blog/current/", Date: mustDate(t, "2024-06-01")},
				{Title: "Other Post", URL: "/blog/other/", Date: mustDate(t, "2024-05-01")},
				{Title: "Draft Without URL", Date: mustDate(t, "2024-04-01")},
			},
		},
	}

	result := string(RenderModeSidebar(root, "/blog/current/", 3, true, timeline, i18n.UI("en"), "/custom-graph/", "time", true))

	for _, want := range []string{
		`data-sidebar-mode="time"`,
		`data-sidebar-default-mode="time"`,
		`data-sidebar-mode-btn="categories"`,
		`data-sidebar-mode-btn="time"`,
		`data-sidebar-mode-btn="graph"`,
		`class="sidebar-mode-btn is-active" role="tab" aria-selected="true" data-sidebar-mode-btn="time"`,
		`data-sidebar-mode-pane="categories" hidden`,
		`data-sidebar-mode-pane="time"`,
		`data-sidebar-mode-pane="graph" hidden`,
		`<h4>2024</h4>`,
		`<li class="active"><a href="/blog/current/" aria-current="page">Current &lt;Post&gt;</a></li>`,
		`<li><a href="/blog/other/">Other Post</a></li>`,
		`data-src="/custom-graph?embed=1"`,
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected mode sidebar output to contain %q, got:\n%s", want, result)
		}
	}
	if strings.Contains(result, "Draft Without URL") {
		t.Fatalf("expected timeline items without URLs to be skipped, got:\n%s", result)
	}
}

func TestRenderModeSidebar_FallsBackFromUnavailableDefaults(t *testing.T) {
	root := &NavNode{
		ID:   "root",
		Type: "section",
		Children: []*NavNode{
			{ID: "docs", Title: "Docs", URL: "/docs/", Type: "section"},
		},
	}

	tests := []struct {
		name        string
		defaultMode string
		timeline    []TimelineYear
		showGraph   bool
	}{
		{name: "invalid default", defaultMode: "unknown"},
		{name: "time without timeline", defaultMode: "time"},
		{name: "graph disabled", defaultMode: "graph", showGraph: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(RenderModeSidebar(root, "/docs/", 3, true, tt.timeline, i18n.UI("en"), "", tt.defaultMode, tt.showGraph))
			if !strings.Contains(result, `data-sidebar-default-mode="categories"`) {
				t.Fatalf("expected fallback to categories mode, got:\n%s", result)
			}
			if strings.Contains(result, `sidebar-mode-switch`) {
				t.Fatalf("expected no mode switch when only categories mode is available, got:\n%s", result)
			}
		})
	}
}

func TestRenderModeSidebarOptions_TimeOnlyHidesModeSwitch(t *testing.T) {
	root := &NavNode{
		ID:   "events",
		Type: "section",
		Children: []*NavNode{
			{ID: "meetup", Title: "Meetup", URL: "/events/meetup/", Type: "page"},
		},
	}
	timeline := []TimelineYear{
		{
			Year: 2025,
			Items: []TimelineItem{
				{Title: "Meetup", URL: "/events/meetup/", Date: mustDate(t, "2025-04-28")},
			},
		},
	}

	result := string(RenderModeSidebarOptions(root, "/events/meetup/", 3, true, timeline, i18n.UI("en"), "", "time", false, false))

	for _, want := range []string{
		`data-sidebar-mode="time"`,
		`data-sidebar-default-mode="time"`,
		`data-sidebar-mode-pane="time"`,
		`<h4>2025</h4>`,
		`<li class="active"><a href="/events/meetup/" aria-current="page">Meetup</a></li>`,
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected time-only sidebar to contain %q, got:\n%s", want, result)
		}
	}
	for _, ban := range []string{
		`sidebar-mode-switch`,
		`data-sidebar-mode-btn=`,
		`data-sidebar-mode-pane="categories"`,
	} {
		if strings.Contains(result, ban) {
			t.Fatalf("expected time-only sidebar to omit %q, got:\n%s", ban, result)
		}
	}
}

func TestRenderModeSidebar_DefaultGraphUsesEmbeddedGraphURL(t *testing.T) {
	root := &NavNode{
		ID:   "root",
		Type: "section",
		Children: []*NavNode{
			{ID: "docs", Title: "Docs", URL: "/docs/", Type: "section"},
		},
	}

	result := string(RenderModeSidebar(root, "/docs/", 3, true, nil, i18n.UI("en"), "", "graph", true))

	for _, want := range []string{
		`data-sidebar-default-mode="graph"`,
		`data-sidebar-mode-btn="graph"`,
		`class="sidebar-mode-btn is-active" role="tab" aria-selected="true" data-sidebar-mode-btn="graph"`,
		`data-sidebar-mode-pane="categories" hidden`,
		`data-sidebar-mode-pane="graph"`,
		`data-src="/graph?embed=1"`,
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected graph mode sidebar output to contain %q, got:\n%s", want, result)
		}
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("failed to parse test date %q: %v", value, err)
	}
	return parsed
}
