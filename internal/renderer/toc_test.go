package renderer

import (
	"strings"
	"testing"
)

func TestRenderTOC_Empty(t *testing.T) {
	result := RenderTOC(nil)
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

	result := string(RenderTOC(items))

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
	result := RenderSidebar(nil, "/", 3)
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

	result := string(RenderSidebar(root, "/docs/about/", 3))

	// "about" should be active
	if !strings.Contains(result, `aria-current="page"`) {
		t.Error("Expected aria-current on active page")
	}
	// "docs" section should be expanded (ancestor of active)
	if !strings.Contains(result, "expanded") {
		t.Error("Expected expanded class on ancestor section")
	}
}
