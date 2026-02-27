package builder

import (
	"strings"
)

// BreadcrumbItem represents one entry in a breadcrumb trail
type BreadcrumbItem struct {
	Title     string
	URL       string
	IsCurrent bool
}

// BreadcrumbGenerator generates breadcrumb trails for pages
type BreadcrumbGenerator struct {
	homeLabel string
}

// NewBreadcrumbGenerator creates a new breadcrumb generator
func NewBreadcrumbGenerator(homeLabel string) *BreadcrumbGenerator {
	if homeLabel == "" {
		homeLabel = "Home"
	}
	return &BreadcrumbGenerator{homeLabel: homeLabel}
}

// Generate creates a breadcrumb trail for a page using the nav tree for title lookups
func (g *BreadcrumbGenerator) Generate(page *Page, tree *NavTree) []BreadcrumbItem {
	crumbs := []BreadcrumbItem{
		{Title: g.homeLabel, URL: "/"},
	}

	url := strings.Trim(page.URL, "/")
	if url == "" {
		return crumbs
	}
	segments := strings.Split(url, "/")

	pathSoFar := ""
	for i, seg := range segments {
		pathSoFar += seg + "/"
		fullPath := "/" + pathSoFar
		isLast := i == len(segments)-1

		title := capitalizeFirst(seg)

		// Try to find a better title from the nav tree
		if tree != nil {
			if node, ok := tree.ByPath[fullPath]; ok {
				title = node.Title
			}
		}

		crumb := BreadcrumbItem{
			Title:     title,
			URL:       fullPath,
			IsCurrent: isLast,
		}
		if isLast {
			crumb.URL = "" // Current page has no link
		}
		crumbs = append(crumbs, crumb)
	}

	return crumbs
}
