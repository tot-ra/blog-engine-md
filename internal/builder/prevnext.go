package builder

import (
	"sort"
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
type PrevNextGenerator struct{}

// NewPrevNextGenerator creates a new prev/next generator
func NewPrevNextGenerator() *PrevNextGenerator {
	return &PrevNextGenerator{}
}

// Generate creates prev/next links for a page.
// Blog posts are sorted by date (newest first).
// Docs follow nav tree depth-first order.
func (g *PrevNextGenerator) Generate(page *Page, allPages map[string]*Page, tree *NavTree) *PrevNextLinks {
	if page.Type == TypeBlog {
		return g.generateForBlog(page, allPages)
	}
	return g.generateForDocs(page, tree)
}

// generateForBlog generates prev/next for blog posts sorted by date (newest first)
func (g *PrevNextGenerator) generateForBlog(page *Page, allPages map[string]*Page) *PrevNextLinks {
	// Collect all blog posts
	var blogPosts []*Page
	for _, p := range allPages {
		if p.Type == TypeBlog {
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
func (g *PrevNextGenerator) generateForDocs(page *Page, tree *NavTree) *PrevNextLinks {
	if tree == nil {
		return nil
	}

	pages := tree.FlattenPages()

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
