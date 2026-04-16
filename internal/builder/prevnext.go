package builder

import (
	"path"
	"sort"
	"strings"
)

// NavLink represents a navigation link (previous or next)
type NavLink struct {
	Title string
	URL   string
	Type  string // "blog" | "doc"
}

// PrevNextLinks holds previous and next navigation links
type PrevNextLinks struct {
	Prev *NavLink
	Next *NavLink
}

// PrevNextGenerator generates previous/next page links
type PrevNextGenerator struct {
	sameCategoryOnly bool
}

// NewPrevNextGenerator creates a new prev/next generator
func NewPrevNextGenerator(sameCategoryOnly bool) *PrevNextGenerator {
	return &PrevNextGenerator{sameCategoryOnly: sameCategoryOnly}
}

// Generate creates prev/next links for a page.
// Blog posts are sorted by date (newest first).
// Docs follow nav tree depth-first order.
func (g *PrevNextGenerator) Generate(page *Page, allPages map[string]*Page, tree *NavTree) *PrevNextLinks {
	if page.Type == TypeBlog {
		return g.generateForBlog(page, allPages)
	}
	return g.generateForDocs(page, allPages, tree)
}

// generateForBlog generates prev/next for blog posts sorted by date (newest first)
func (g *PrevNextGenerator) generateForBlog(page *Page, allPages map[string]*Page) *PrevNextLinks {
	// Collect all blog posts
	var blogPosts []*Page
	for _, p := range allPages {
		if p.Type == TypeBlog && p.Language == page.Language {
			if p.Frontmatter != nil {
				if p.Frontmatter.HideNav || strings.TrimSpace(p.Frontmatter.RedirectURL) != "" {
					continue
				}
			}
			if g.sameCategoryOnly && siblingGroupURL(p.URL) != siblingGroupURL(page.URL) {
				continue
			}
			blogPosts = append(blogPosts, p)
		}
	}

	// Sort by date descending (newest first)
	sort.SliceStable(blogPosts, func(i, j int) bool {
		a := blogPosts[i].Frontmatter.Date
		b := blogPosts[j].Frontmatter.Date
		return a.After(b)
	})

	// Find current page index
	idx := -1
	for i, p := range blogPosts {
		if p.ID == page.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}

	links := &PrevNextLinks{}

	// Previous = newer post (lower index)
	if idx > 0 {
		p := blogPosts[idx-1]
		links.Prev = &NavLink{Title: p.Title, URL: p.URL, Type: "blog"}
	}

	// Next = older post (higher index)
	if idx < len(blogPosts)-1 {
		p := blogPosts[idx+1]
		links.Next = &NavLink{Title: p.Title, URL: p.URL, Type: "blog"}
	}

	return links
}

// generateForDocs generates prev/next for docs following nav tree order
func (g *PrevNextGenerator) generateForDocs(page *Page, allPages map[string]*Page, tree *NavTree) *PrevNextLinks {
	if tree == nil {
		return nil
	}

	pages := tree.FlattenPages()
	byURL := make(map[string]*Page, len(allPages))
	for _, p := range allPages {
		byURL[p.URL] = p
	}
	filtered := make([]*NavNode, 0, len(pages))
	for _, n := range pages {
		p, ok := byURL[n.URL]
		if !ok {
			continue
		}
		if p.Type != TypeBlog && p.Language == page.Language {
			if g.sameCategoryOnly && siblingGroupURL(p.URL) != siblingGroupURL(page.URL) {
				continue
			}
			filtered = append(filtered, n)
		}
	}
	pages = filtered

	idx := -1
	for i, node := range pages {
		if node.URL == page.URL {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}

	links := &PrevNextLinks{}
	if idx > 0 {
		p := pages[idx-1]
		links.Prev = &NavLink{Title: p.Title, URL: p.URL, Type: "doc"}
	}
	if idx < len(pages)-1 {
		p := pages[idx+1]
		links.Next = &NavLink{Title: p.Title, URL: p.URL, Type: "doc"}
	}

	return links
}

func siblingGroupURL(pageURL string) string {
	trimmed := strings.Trim(strings.TrimSpace(pageURL), "/")
	if trimmed == "" {
		return "/"
	}

	dir := path.Dir("/" + trimmed)
	if dir == "." || dir == "/" {
		return "/"
	}
	return strings.TrimSuffix(dir, "/") + "/"
}
