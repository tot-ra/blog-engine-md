package renderer

import (
	"fmt"
	"html/template"
	"strings"
	"time"
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

// TimelineMonth groups timeline items by month.
type TimelineMonth struct {
	MonthLabel string
	Items      []TimelineItem
}

// TimelineYear groups timeline items by year.
type TimelineYear struct {
	Year   int
	Months []TimelineMonth
}

// RenderSidebar renders a navigation tree as sidebar HTML, marking the current page as active
func RenderSidebar(root *NavNode, currentPath string, maxDepth int, collapsed bool) template.HTML {
	if root == nil || len(root.Children) == 0 {
		return ""
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}

	var sb strings.Builder
	sb.WriteString("<nav class=\"sidebar\" aria-label=\"Navigation\">\n")
	renderSidebarList(&sb, root.Children, currentPath, 1, maxDepth, collapsed)
	sb.WriteString("</nav>\n")

	return template.HTML(sb.String())
}

// RenderBlogSidebar renders a 3-mode blog sidebar:
// categories (tree), time (timeline), graph (embedded graph view).
func RenderBlogSidebar(root *NavNode, currentPath string, maxDepth int, collapsed bool, timeline []TimelineYear) template.HTML {
	if root == nil || len(root.Children) == 0 {
		return ""
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}

	var sb strings.Builder
	sb.WriteString("<nav class=\"sidebar blog-sidebar\" aria-label=\"Blog navigation\" data-sidebar-mode=\"categories\">\n")
	sb.WriteString("  <div class=\"sidebar-mode-switch\" role=\"tablist\" aria-label=\"Blog view mode\">\n")
	sb.WriteString("    <button type=\"button\" class=\"sidebar-mode-btn is-active\" role=\"tab\" aria-selected=\"true\" data-sidebar-mode-btn=\"categories\">Categories</button>\n")
	sb.WriteString("    <button type=\"button\" class=\"sidebar-mode-btn\" role=\"tab\" aria-selected=\"false\" data-sidebar-mode-btn=\"time\">Time</button>\n")
	sb.WriteString("    <button type=\"button\" class=\"sidebar-mode-btn\" role=\"tab\" aria-selected=\"false\" data-sidebar-mode-btn=\"graph\">Graph</button>\n")
	sb.WriteString("  </div>\n")

	sb.WriteString("  <div class=\"sidebar-mode-pane\" data-sidebar-mode-pane=\"categories\">\n")
	renderSidebarList(&sb, root.Children, currentPath, 1, maxDepth, collapsed)
	sb.WriteString("  </div>\n")

	sb.WriteString("  <div class=\"sidebar-mode-pane\" data-sidebar-mode-pane=\"time\" hidden>\n")
	renderTimeline(&sb, timeline, currentPath)
	sb.WriteString("  </div>\n")

	sb.WriteString("  <div class=\"sidebar-mode-pane sidebar-graph-pane\" data-sidebar-mode-pane=\"graph\" hidden>\n")
	sb.WriteString("    <iframe class=\"sidebar-graph-frame\" title=\"Blog graph view\" data-src=\"/graph/?embed=1\"></iframe>\n")
	sb.WriteString("  </div>\n")
	sb.WriteString("</nav>\n")

	return template.HTML(sb.String())
}

func renderTimeline(sb *strings.Builder, years []TimelineYear, currentPath string) {
	sb.WriteString("    <div class=\"sidebar-timeline\">\n")
	for _, year := range years {
		sb.WriteString(fmt.Sprintf("      <section class=\"timeline-year\"><h4>%d</h4>\n", year.Year))
		for _, month := range year.Months {
			sb.WriteString(fmt.Sprintf("        <h5>%s</h5>\n", template.HTMLEscapeString(month.MonthLabel)))
			sb.WriteString("        <ul class=\"timeline-list\">\n")
			for _, item := range month.Items {
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
		}
		sb.WriteString("      </section>\n")
	}
	sb.WriteString("    </div>\n")
}

func renderSidebarList(sb *strings.Builder, nodes []*NavNode, currentPath string, depth, maxDepth int, collapsed bool) {
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
			sb.WriteString(fmt.Sprintf("      <button class=\"sidebar-toggle\" type=\"button\" aria-label=\"Toggle %s\" aria-expanded=\"%t\"></button>\n", template.HTMLEscapeString(node.Title), expanded))
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
			renderSidebarList(sb, node.Children, currentPath, depth+1, maxDepth, collapsed)
		}

		sb.WriteString("  </li>\n")
	}

	sb.WriteString("</ul>\n")
}
