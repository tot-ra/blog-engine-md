package builder

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tot-ra/blog-engine/internal/parser"
)

// SectionChild represents a child entry on a section index page
type SectionChild struct {
	Title       string
	URL         string
	Description string
	Date        *time.Time
	IsSection   bool
}

// SectionIndexGenerator generates index pages for directories without explicit index.md
type SectionIndexGenerator struct{}

// NewSectionIndexGenerator creates a new section index generator
func NewSectionIndexGenerator() *SectionIndexGenerator {
	return &SectionIndexGenerator{}
}

// GenerateMissing creates index pages for any section nodes that don't already have a page
func (g *SectionIndexGenerator) GenerateMissing(pages map[string]*Page, tree *NavTree) []*Page {
	var generated []*Page

	var walk func(node *NavNode)
	walk = func(node *NavNode) {
		if node.Type != "section" {
			return
		}

		// Check if an index page already exists for this section
		exists := false
		for _, p := range pages {
			if p.URL == node.URL {
				exists = true
				break
			}
		}

		if !exists && node.URL != "" {
			page := g.generateIndexPage(node)
			if page != nil {
				generated = append(generated, page)
			}
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	for _, child := range tree.Root.Children {
		walk(child)
	}

	return generated
}

// generateIndexPage creates an index page for a section node
func (g *SectionIndexGenerator) generateIndexPage(node *NavNode) *Page {
	// Collect children info
	var children []SectionChild
	for _, child := range node.Children {
		sc := SectionChild{
			Title:     child.Title,
			URL:       child.URL,
			IsSection: child.Type == "section",
		}
		children = append(children, sc)
	}

	// Sort: sections first, then alphabetical
	sort.SliceStable(children, func(i, j int) bool {
		if children[i].IsSection != children[j].IsSection {
			return children[i].IsSection
		}
		return strings.ToLower(children[i].Title) < strings.ToLower(children[j].Title)
	})

	// Build simple HTML listing
	var sb strings.Builder
	sb.WriteString("<ul class=\"section-index\">\n")
	for _, child := range children {
		icon := "📄"
		if child.IsSection {
			icon = "📁"
		}
		sb.WriteString(fmt.Sprintf("  <li>%s <a href=\"%s\">%s</a></li>\n", icon, child.URL, child.Title))
	}
	sb.WriteString("</ul>\n")

	pageType := determinePageType(strings.TrimPrefix(node.URL, "/"))

	return &Page{
		ID:           node.ID + "-index",
		URL:          node.URL,
		Title:        node.Title,
		Description:  fmt.Sprintf("Index of %s", node.Title),
		Content:      sb.String(),
		Frontmatter:  &parser.Frontmatter{},
		TOC:          nil,
		Type:         pageType,
		ModifiedTime: time.Now(),
	}
}
