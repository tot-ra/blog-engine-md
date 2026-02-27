package builder

import (
	"sort"
	"strings"
)

// NavTree represents the full navigation hierarchy
type NavTree struct {
	Root   *NavNode
	ByPath map[string]*NavNode // Quick lookup by URL path
}

// NavNode represents a single node in the navigation tree
type NavNode struct {
	ID       string
	Title    string
	URL      string
	Children []*NavNode
	Parent   *NavNode
	Order    int
	Hidden   bool
	Type     string // "section" | "page"
}

// FlattenPages returns all leaf (page) nodes in depth-first order
func (t *NavTree) FlattenPages() []*NavNode {
	var pages []*NavNode
	var walk func(node *NavNode)
	walk = func(node *NavNode) {
		if node.Type == "page" {
			pages = append(pages, node)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, child := range t.Root.Children {
		walk(child)
	}
	return pages
}

// NavigationBuilder builds navigation trees from pages
type NavigationBuilder struct{}

// NewNavigationBuilder creates a new navigation builder
func NewNavigationBuilder() *NavigationBuilder {
	return &NavigationBuilder{}
}

// BuildTree builds a NavTree from a set of pages
func (nb *NavigationBuilder) BuildTree(pages map[string]*Page) *NavTree {
	tree := &NavTree{
		Root: &NavNode{
			ID:       "root",
			Title:    "Root",
			Type:     "section",
			Children: make([]*NavNode, 0),
		},
		ByPath: make(map[string]*NavNode),
	}

	// Insert each page into the tree
	for _, page := range pages {
		if page.Frontmatter != nil && page.Frontmatter.HideNav {
			continue
		}
		nb.insertPage(tree, page)
	}

	// Sort all children recursively
	nb.sortChildren(tree.Root)

	return tree
}

// insertPage inserts a page into the navigation tree, creating intermediate section nodes as needed
func (nb *NavigationBuilder) insertPage(tree *NavTree, page *Page) {
	// Parse the URL into segments: /blog/tech/post/ → ["blog", "tech", "post"]
	url := strings.Trim(page.URL, "/")
	if url == "" {
		return // Skip root page for nav tree
	}
	segments := strings.Split(url, "/")

	current := tree.Root
	pathSoFar := ""

	for i, seg := range segments {
		pathSoFar += seg + "/"
		fullPath := "/" + pathSoFar
		isLast := i == len(segments)-1

		// Check if child already exists
		var child *NavNode
		for _, c := range current.Children {
			if c.URL == fullPath {
				child = c
				break
			}
		}

		if child == nil {
			if isLast {
				// Create page node
				order := 0
				if page.Frontmatter != nil {
					order = page.Frontmatter.Order
				}
				child = &NavNode{
					ID:       page.ID,
					Title:    page.Title,
					URL:      page.URL,
					Parent:   current,
					Order:    order,
					Type:     "page",
					Children: make([]*NavNode, 0),
				}
			} else {
				// Create intermediate section node
				title := capitalizeFirst(seg)
				child = &NavNode{
					ID:       strings.Trim(fullPath, "/"),
					Title:    title,
					URL:      fullPath,
					Parent:   current,
					Type:     "section",
					Children: make([]*NavNode, 0),
				}
			}
			current.Children = append(current.Children, child)
			tree.ByPath[fullPath] = child
		}

		current = child
	}
}

// sortChildren recursively sorts children by Order, then by Title
func (nb *NavigationBuilder) sortChildren(node *NavNode) {
	sort.SliceStable(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		// Sections first, then pages
		if a.Type != b.Type {
			return a.Type == "section"
		}
		// By Order (0 treated as unordered, goes after ordered items)
		if a.Order != b.Order {
			if a.Order == 0 {
				return false
			}
			if b.Order == 0 {
				return true
			}
			return a.Order < b.Order
		}
		// Alphabetical by title
		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	})
	for _, child := range node.Children {
		nb.sortChildren(child)
	}
}
