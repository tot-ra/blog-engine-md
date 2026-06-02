package renderer

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/tot-ra/blog-engine/internal/i18n"
)

// NavNode mirrors builder.NavNode for rendering (avoids circular dependency)
type NavNode struct {
	ID       string
	Title    string
	URL      string
	Children []*NavNode
	Order    int
	Hidden   bool
	Type     string // "section" | "page"
}

// TimelineItem represents one blog post entry for the timeline sidebar mode.
type TimelineItem struct {
	Title string
	URL   string
	Date  time.Time
}

// TimelineYear groups timeline items by year.
type TimelineYear struct {
	Year  int
	Items []TimelineItem
}

// RenderSidebar renders a navigation tree as sidebar HTML, marking the current page as active
func RenderSidebar(root *NavNode, currentPath string, maxDepth int, collapsed bool, ui i18n.UIStrings) template.HTML {
	if root == nil || len(root.Children) == 0 {
		return ""
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}

	var sb strings.Builder
	sb.WriteString("<nav class=\"sidebar\" aria-label=\"Navigation\">\n")
	renderSidebarList(&sb, filterRegularSidebarNodes(root.Children), currentPath, 1, maxDepth, collapsed, ui)
	sb.WriteString("</nav>\n")

	return template.HTML(sb.String())
}

func filterRegularSidebarNodes(nodes []*NavNode) []*NavNode {
	filtered := make([]*NavNode, 0, len(nodes))
	for _, node := range nodes {
		trimmed := strings.TrimSuffix(node.URL, "/")
		if strings.HasSuffix(trimmed, "/blog") {
			continue
		}
		filtered = append(filtered, node)
	}
	return filtered
}

// RenderModeSidebar renders a navigation sidebar with optional categories/time/graph modes.
func RenderModeSidebar(root *NavNode, currentPath string, maxDepth int, collapsed bool, timeline []TimelineYear, ui i18n.UIStrings, graphURL string, defaultMode string, showGraph bool) template.HTML {
	return renderModeSidebar(root, currentPath, maxDepth, collapsed, timeline, ui, graphURL, defaultMode, showGraph, ui.Navigation, ui.ViewMode, ui.Graph)
}

func renderModeSidebar(root *NavNode, currentPath string, maxDepth int, collapsed bool, timeline []TimelineYear, ui i18n.UIStrings, graphURL, defaultMode string, showGraph bool, navigationLabel, viewModeLabel, graphTitle string) template.HTML {
	if root == nil || len(root.Children) == 0 {
		return ""
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if defaultMode == "" {
		defaultMode = "categories"
	}
	if defaultMode != "categories" && defaultMode != "time" && defaultMode != "graph" {
		defaultMode = "categories"
	}
	if defaultMode == "graph" && !showGraph {
		defaultMode = "categories"
	}
	if defaultMode == "time" && len(timeline) == 0 {
		defaultMode = "categories"
	}
	modeCount := 1
	if len(timeline) > 0 {
		modeCount++
	}
	if showGraph {
		modeCount++
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<nav class=\"sidebar blog-sidebar\" aria-label=\"%s\" data-sidebar-mode=\"%s\" data-sidebar-default-mode=\"%s\">\n", template.HTMLEscapeString(navigationLabel), template.HTMLEscapeString(defaultMode), template.HTMLEscapeString(defaultMode)))
	if modeCount > 1 {
		sb.WriteString(fmt.Sprintf("  <div class=\"sidebar-mode-switch\" role=\"tablist\" aria-label=\"%s\">\n", template.HTMLEscapeString(viewModeLabel)))
		sb.WriteString(fmt.Sprintf("    <button type=\"button\" class=\"sidebar-mode-btn%s\" role=\"tab\" aria-selected=\"%t\" data-sidebar-mode-btn=\"categories\">%s</button>\n", activeModeClass(defaultMode, "categories"), defaultMode == "categories", template.HTMLEscapeString(ui.Categories)))
		if len(timeline) > 0 {
			sb.WriteString(fmt.Sprintf("    <button type=\"button\" class=\"sidebar-mode-btn%s\" role=\"tab\" aria-selected=\"%t\" data-sidebar-mode-btn=\"time\">%s</button>\n", activeModeClass(defaultMode, "time"), defaultMode == "time", template.HTMLEscapeString(ui.Time)))
		}
		if showGraph {
			sb.WriteString(fmt.Sprintf("    <button type=\"button\" class=\"sidebar-mode-btn%s\" role=\"tab\" aria-selected=\"%t\" data-sidebar-mode-btn=\"graph\">%s</button>\n", activeModeClass(defaultMode, "graph"), defaultMode == "graph", template.HTMLEscapeString(ui.Graph)))
		}
		sb.WriteString("  </div>\n")
	}

	sb.WriteString(fmt.Sprintf("  <div class=\"sidebar-mode-pane\" data-sidebar-mode-pane=\"categories\"%s>\n", hiddenAttr(defaultMode != "categories")))
	renderSidebarList(&sb, root.Children, currentPath, 1, maxDepth, collapsed, ui)
	sb.WriteString("  </div>\n")

	if len(timeline) > 0 {
		sb.WriteString(fmt.Sprintf("  <div class=\"sidebar-mode-pane\" data-sidebar-mode-pane=\"time\"%s>\n", hiddenAttr(defaultMode != "time")))
		renderTimeline(&sb, timeline, currentPath)
		sb.WriteString("  </div>\n")
	}

	if showGraph {
		sb.WriteString(fmt.Sprintf("  <div class=\"sidebar-mode-pane sidebar-graph-pane\" data-sidebar-mode-pane=\"graph\"%s>\n", hiddenAttr(defaultMode != "graph")))
		if graphURL == "" {
			graphURL = "/graph/"
		}
		sb.WriteString(fmt.Sprintf("    <iframe class=\"sidebar-graph-frame\" title=\"%s\" data-src=\"%s?embed=1\"></iframe>\n", template.HTMLEscapeString(graphTitle), template.HTMLEscapeString(strings.TrimSuffix(graphURL, "/"))))
		sb.WriteString("  </div>\n")
	}
	sb.WriteString("</nav>\n")

	return template.HTML(sb.String())
}

func activeModeClass(currentMode, mode string) string {
	if currentMode == mode {
		return " is-active"
	}
	return ""
}

func hiddenAttr(hidden bool) string {
	if hidden {
		return " hidden"
	}
	return ""
}

func renderTimeline(sb *strings.Builder, years []TimelineYear, currentPath string) {
	sb.WriteString("    <div class=\"sidebar-timeline\">\n")
	for _, year := range years {
		sb.WriteString(fmt.Sprintf("      <section class=\"timeline-year\"><h4>%d</h4>\n", year.Year))
		sb.WriteString("        <ul class=\"timeline-list\">\n")
		for _, item := range year.Items {
			if item.URL == "" {
				continue
			}
			if currentPath == item.URL {
				sb.WriteString(fmt.Sprintf("          <li class=\"active\"><a href=\"%s\" aria-current=\"page\">%s</a></li>\n",
					item.URL,
					template.HTMLEscapeString(item.Title),
				))
				continue
			}
			sb.WriteString(fmt.Sprintf("          <li><a href=\"%s\">%s</a></li>\n",
				item.URL,
				template.HTMLEscapeString(item.Title),
			))
		}
		sb.WriteString("        </ul>\n")
		sb.WriteString("      </section>\n")
	}
	sb.WriteString("    </div>\n")
}

func renderSidebarList(sb *strings.Builder, nodes []*NavNode, currentPath string, depth, maxDepth int, collapsed bool, ui i18n.UIStrings) {
	if depth > maxDepth {
		return
	}

	class := "sidebar-menu"
	if depth > 1 {
		class = "sidebar-submenu"
	}
	sb.WriteString(fmt.Sprintf("<ul class=\"%s\">\n", class))

	for _, node := range nodes {
		if node.Hidden {
			continue
		}

		classes := []string{"sidebar-item"}
		isActive := currentPath == node.URL
		isAncestor := !isActive && currentPath != "" && strings.HasPrefix(currentPath, node.URL)
		hasChildren := len(node.Children) > 0

		isSection := node.Type == "section" || hasChildren
		if isSection {
			classes = append(classes, "sidebar-section")
		}
		if isActive {
			classes = append(classes, "active")
		}
		expanded := isAncestor || isActive || (!collapsed && depth == 1)
		if expanded {
			classes = append(classes, "expanded")
		}

		sb.WriteString(fmt.Sprintf("  <li class=\"%s\">\n", strings.Join(classes, " ")))

		if isSection && hasChildren {
			sb.WriteString("    <div class=\"sidebar-section-head\">\n")
			// Keep the label first so expandable folders align with regular menu items;
			// CSS pushes the toggle button to the row end.
		}
		if isActive {
			sb.WriteString(fmt.Sprintf("      <a href=\"%s\" aria-current=\"page\">%s</a>\n", node.URL, template.HTMLEscapeString(node.Title)))
		} else {
			sb.WriteString(fmt.Sprintf("      <a href=\"%s\">%s</a>\n", node.URL, template.HTMLEscapeString(node.Title)))
		}
		if isSection && hasChildren {
			sb.WriteString(fmt.Sprintf("      <button class=\"sidebar-toggle\" type=\"button\" aria-label=\"%s %s\" aria-expanded=\"%t\"></button>\n", template.HTMLEscapeString(ui.ToggleSectionOf), template.HTMLEscapeString(node.Title), expanded))
			sb.WriteString("    </div>\n")
		}

		if hasChildren {
			renderSidebarList(sb, node.Children, currentPath, depth+1, maxDepth, collapsed, ui)
		}

		sb.WriteString("  </li>\n")
	}

	sb.WriteString("</ul>\n")
}
