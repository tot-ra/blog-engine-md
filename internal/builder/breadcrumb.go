package builder

import (
	"strings"

	"github.com/tot-ra/blog-engine/internal/i18n"
)

// BreadcrumbItem represents one entry in a breadcrumb trail
type BreadcrumbItem struct {
	Title     string
	URL       string
	IsCurrent bool
}

// BreadcrumbGenerator generates breadcrumb trails for pages
type BreadcrumbGenerator struct {
	langCodes     map[string]struct{}
	segmentLabels map[string]map[string]string
}

// NewBreadcrumbGenerator creates a new breadcrumb generator
func NewBreadcrumbGenerator(langCodes map[string]struct{}) *BreadcrumbGenerator {
	return &BreadcrumbGenerator{langCodes: langCodes}
}

// NewBreadcrumbGeneratorWithLabels creates a breadcrumb generator with
// site-defined URL segment labels.
func NewBreadcrumbGeneratorWithLabels(langCodes map[string]struct{}, segmentLabels map[string]map[string]string) *BreadcrumbGenerator {
	return &BreadcrumbGenerator{langCodes: langCodes, segmentLabels: segmentLabels}
}

// Generate creates a breadcrumb trail for a page using the nav tree for title lookups
func (g *BreadcrumbGenerator) Generate(page *Page, tree *NavTree) []BreadcrumbItem {
	homeLabel := i18n.UI(page.Language).Home
	homeURL := languageBasePath(page.URL, page.Language, page.Language)
	if len(g.langCodes) <= 1 {
		homeURL = "/"
	}
	crumbs := []BreadcrumbItem{
		{Title: homeLabel, URL: homeURL},
	}

	url := strings.Trim(page.URL, "/")
	if url == "" {
		return crumbs
	}
	segments := strings.Split(url, "/")
	start := 0
	if len(segments) > 0 {
		first := strings.ToLower(segments[0])
		if _, ok := g.langCodes[first]; ok {
			start = 1
		}
	}

	pathSoFar := ""
	for i := start; i < len(segments); i++ {
		seg := segments[i]
		pathSoFar += seg + "/"
		fullPath := homeURL + pathSoFar
		fullPath = "/" + strings.Trim(fullPath, "/") + "/"
		isLast := i == len(segments)-1

		title := segmentLabel(page.Language, seg, g.segmentLabels)
		if title == "" {
			title = capitalizeFirst(seg)
		}

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
