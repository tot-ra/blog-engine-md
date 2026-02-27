package renderer

import (
	"fmt"
	"html/template"
	"strings"
)

// TocItem mirrors builder.TocItem for rendering
type TocItem struct {
	Level    int
	Text     string
	Anchor   string
	Children []*TocItem
}

// RenderTOC renders a table of contents as HTML
func RenderTOC(items []*TocItem) template.HTML {
	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<nav class=\"toc\" aria-label=\"Table of contents\">\n")
	sb.WriteString("  <h2 class=\"toc-title\">On this page</h2>\n")
	renderTocList(&sb, items, "toc-list")
	sb.WriteString("</nav>\n")

	return template.HTML(sb.String())
}

func renderTocList(sb *strings.Builder, items []*TocItem, class string) {
	sb.WriteString("<ul class=\"" + class + "\">\n")
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("  <li class=\"toc-item\">\n"))
		sb.WriteString(fmt.Sprintf("    <a href=\"#%s\">%s</a>\n", item.Anchor, template.HTMLEscapeString(item.Text)))
		if len(item.Children) > 0 {
			renderTocList(sb, item.Children, "toc-sublist")
		}
		sb.WriteString("  </li>\n")
	}
	sb.WriteString("</ul>\n")
}
