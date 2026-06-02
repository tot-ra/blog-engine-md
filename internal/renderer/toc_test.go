package renderer

import (
	"strings"
	"testing"

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
