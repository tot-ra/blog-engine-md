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

// RenderBlogSidebar renders a 3-mode blog sidebar:
// categories (tree), time (timeline), graph (embedded graph view).
func RenderBlogSidebar(root *NavNode, currentPath string, maxDepth int, collapsed bool, timeline []TimelineYear, ui i18n.UIStrings, graphURL string) template.HTML {
	if root == nil || len(root.Children) == 0 {
		return ""
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<nav class=\"sidebar blog-sidebar\" aria-label=\"%s\" data-sidebar-mode=\"categories\">\n", template.HTMLEscapeString(ui.BlogNavigation)))
	sb.WriteString(fmt.Sprintf("  <div class=\"sidebar-mode-switch\" role=\"tablist\" aria-label=\"%s\">\n", template.HTMLEscapeString(ui.BlogViewMode)))
	sb.WriteString(fmt.Sprintf("    <button type=\"button\" class=\"sidebar-mode-btn is-active\" role=\"tab\" aria-selected=\"true\" data-sidebar-mode-btn=\"categories\">%s</button>\n", template.HTMLEscapeString(ui.Categories)))
	sb.WriteString(fmt.Sprintf("    <button type=\"button\" class=\"sidebar-mode-btn\" role=\"tab\" aria-selected=\"false\" data-sidebar-mode-btn=\"time\">%s</button>\n", template.HTMLEscapeString(ui.Time)))
	sb.WriteString(fmt.Sprintf("    <button type=\"button\" class=\"sidebar-mode-btn\" role=\"tab\" aria-selected=\"false\" data-sidebar-mode-btn=\"graph\">%s</button>\n", template.HTMLEscapeString(ui.Graph)))
	sb.WriteString("  </div>\n")

	sb.WriteString("  <div class=\"sidebar-mode-pane\" data-sidebar-mode-pane=\"categories\">\n")
	renderSidebarList(&sb, root.Children, currentPath, 1, maxDepth, collapsed, ui)
	sb.WriteString("  </div>\n")

	sb.WriteString("  <div class=\"sidebar-mode-pane\" data-sidebar-mode-pane=\"time\" hidden>\n")
	renderTimeline(&sb, timeline, currentPath)
	sb.WriteString("  </div>\n")

	sb.WriteString("  <div class=\"sidebar-mode-pane sidebar-graph-pane\" data-sidebar-mode-pane=\"graph\" hidden>\n")
	if graphURL == "" {
		graphURL = "/graph/"
	}
	sb.WriteString(fmt.Sprintf("    <iframe class=\"sidebar-graph-frame\" title=\"%s\" data-src=\"%s?embed=1\"></iframe>\n", template.HTMLEscapeString(ui.BlogGraphView), template.HTMLEscapeString(strings.TrimSuffix(graphURL, "/"))))
	sb.WriteString("  </div>\n")
	sb.WriteString("</nav>\n")

	return template.HTML(sb.String())
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

		if node.Type == "section" {
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

		if node.Type == "section" && hasChildren {
			sb.WriteString("    <div class=\"sidebar-section-head\">\n")
			sb.WriteString(fmt.Sprintf("      <button class=\"sidebar-toggle\" type=\"button\" aria-label=\"%s %s\" aria-expanded=\"%t\"></button>\n", template.HTMLEscapeString(ui.ToggleSectionOf), template.HTMLEscapeString(node.Title), expanded))
		}
		if isActive {
			sb.WriteString(fmt.Sprintf("      <a href=\"%s\" aria-current=\"page\">%s</a>\n", node.URL, template.HTMLEscapeString(node.Title)))
		} else {
			sb.WriteString(fmt.Sprintf("      <a href=\"%s\">%s</a>\n", node.URL, template.HTMLEscapeString(node.Title)))
		}
		if node.Type == "section" && hasChildren {
			sb.WriteString("    </div>\n")
		}

		if hasChildren {
			renderSidebarList(sb, node.Children, currentPath, depth+1, maxDepth, collapsed, ui)
		}

		sb.WriteString("  </li>\n")
	}

	sb.WriteString("</ul>\n")
}
