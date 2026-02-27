package renderer

import (
	"fmt"
	"html/template"
	"strings"
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

// RenderSidebar renders a navigation tree as sidebar HTML, marking the current page as active
func RenderSidebar(root *NavNode, currentPath string, maxDepth int) template.HTML {
	if root == nil || len(root.Children) == 0 {
		return ""
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}

	var sb strings.Builder
	sb.WriteString("<nav class=\"sidebar\" aria-label=\"Navigation\">\n")
	renderSidebarList(&sb, root.Children, currentPath, 1, maxDepth)
	sb.WriteString("</nav>\n")

	return template.HTML(sb.String())
}

func renderSidebarList(sb *strings.Builder, nodes []*NavNode, currentPath string, depth, maxDepth int) {
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

		if node.Type == "section" {
			classes = append(classes, "sidebar-section")
		}
		if isActive {
			classes = append(classes, "active")
		}
		if isAncestor || isActive {
			classes = append(classes, "expanded")
		}

		sb.WriteString(fmt.Sprintf("  <li class=\"%s\">\n", strings.Join(classes, " ")))

		if isActive {
			sb.WriteString(fmt.Sprintf("    <a href=\"%s\" aria-current=\"page\">%s</a>\n", node.URL, template.HTMLEscapeString(node.Title)))
		} else {
			sb.WriteString(fmt.Sprintf("    <a href=\"%s\">%s</a>\n", node.URL, template.HTMLEscapeString(node.Title)))
		}

		if len(node.Children) > 0 {
			renderSidebarList(sb, node.Children, currentPath, depth+1, maxDepth)
		}

		sb.WriteString("  </li>\n")
	}

	sb.WriteString("</ul>\n")
}
