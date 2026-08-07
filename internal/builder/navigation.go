package builder

import (
	"sort"
	"strings"

	"github.com/tot-ra/blog-engine/internal/parser"
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
type NavigationBuilder struct {
	routeTitles map[string]string
}

// NewNavigationBuilder creates a new navigation builder
func NewNavigationBuilder() *NavigationBuilder {
	return &NavigationBuilder{}
}

// BuildTree builds a NavTree from a set of pages
func (nb *NavigationBuilder) BuildTree(pages map[string]*Page) *NavTree {
	nb.routeTitles = collectRouteTitles(pages)
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
		if page.Frontmatter != nil {
			if page.Frontmatter.HideNav || strings.TrimSpace(page.Frontmatter.RedirectURL) != "" {
				continue
			}
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
					Title:    displayTitleForPage(page),
					URL:      page.URL,
					Parent:   current,
					Order:    order,
					Type:     "page",
					Children: make([]*NavNode, 0),
				}
			} else {
				// Create intermediate section node. Prefer titles discovered from
				// localized content pages so section labels remain content metadata,
				// not global config translations.
				title := nb.titleForPath(fullPath, page.Language, seg)
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
		} else if !isLast && child.Type == "section" {
			if title := nb.routeTitleForPath(fullPath); title != "" {
				child.Title = title
			}
		}

		if isLast {
			if child.Type == "section" {
				child.ID = page.ID
				child.Title = displayTitleForPage(page)
				if page.Frontmatter != nil {
					child.Order = page.Frontmatter.Order
				}
			}
		}

		current = child
	}
}

// sortChildren recursively sorts children by Order, then by Title
func (nb *NavigationBuilder) sortChildren(node *NavNode) {
	sort.SliceStable(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
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
		// Sections first, then pages when neither side has a stronger explicit order
		if a.Type != b.Type {
			return a.Type == "section"
		}
		// Alphabetical by title
		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	})
	for _, child := range node.Children {
		nb.sortChildren(child)
	}
}

func (nb *NavigationBuilder) titleForPath(fullPath, lang, segment string) string {
	if title := nb.routeTitleForPath(fullPath); title != "" {
		return title
	}
	if title := segmentLabel(lang, segment); title != "" {
		return title
	}
	return humanizeSegment(segment)
}

func (nb *NavigationBuilder) routeTitleForPath(fullPath string) string {
	if nb == nil || len(nb.routeTitles) == 0 {
		return ""
	}
	return strings.TrimSpace(nb.routeTitles[normalizeNavURL(fullPath)])
}

func collectRouteTitles(pages map[string]*Page) map[string]string {
	if len(pages) == 0 {
		return nil
	}

	type routeTitle struct {
		title    string
		priority int
	}

	candidates := make(map[string]routeTitle, len(pages))
	for _, page := range pages {
		if page == nil || strings.TrimSpace(page.URL) == "" {
			continue
		}
		if page.Frontmatter != nil && strings.TrimSpace(page.Frontmatter.RedirectURL) != "" {
			// Redirect placeholders often point to a canonical language while keeping
			// untranslated titles. Do not let them override labels for localized trees.
			continue
		}
		title := displayTitleForPage(page)
		if title == "" {
			continue
		}
		for _, candidate := range pageTitleCandidateURLs(page) {
			url := normalizeNavURL(candidate.URL)
			if url == "" {
				continue
			}
			candidateTitle := strings.TrimSpace(candidate.title)
			if candidateTitle == "" {
				candidateTitle = title
			}
			// Prefer actual self-page titles when a section stores its overview as
			// folder_name/folder_name.md. This keeps generated cards like
			// /products/web_app/ labeled “📱 Web-app” instead of “Web app”.
			if existing, exists := candidates[url]; exists {
				if existing.priority > candidate.Priority || (existing.priority == candidate.Priority && !preferRouteTitle(candidateTitle, existing.title)) {
					continue
				}
			}
			candidates[url] = routeTitle{
				title:    candidateTitle,
				priority: candidate.Priority,
			}
		}
	}

	out := make(map[string]string, len(candidates))
	for url, candidate := range candidates {
		out[url] = candidate.title
	}
	return out
}

type pageTitleCandidate struct {
	URL      string
	Priority int
	title    string
}

func preferRouteTitle(candidate, existing string) bool {
	candidate = strings.TrimSpace(candidate)
	existing = strings.TrimSpace(existing)
	if candidate == "" {
		return false
	}
	if existing == "" {
		return true
	}
	// Emoji/frontmatter titles are usually more intentional than fallback
	// generated index labels for the same route.
	return startsWithNonASCII(candidate) && !startsWithNonASCII(existing)
}

func startsWithNonASCII(s string) bool {
	for _, r := range strings.TrimSpace(s) {
		return r > 127
	}
	return false
}

func pageTitleCandidateURLs(page *Page) []pageTitleCandidate {
	url := normalizeNavURL(page.URL)
	if url == "" {
		return nil
	}

	source := strings.TrimSpace(page.SourcePath)
	directPriority := 2
	if source == "" {
		// Generated section indexes should not override labels discovered from
		// real content pages such as products/web_app/web_app.md.
		directPriority = 0
	}
	candidates := []pageTitleCandidate{{URL: url, Priority: directPriority}}
	if source == "" {
		return candidates
	}

	name := strings.TrimSuffix(filepathBaseSlash(source), filepathExtSlash(source))
	sourceDir := filepathDirSlash(source)
	parentDirName := filepathBaseSlash(sourceDir)
	parent := strings.Trim(strings.TrimSuffix(url, "/"), "/")
	parts := []string{}
	if parent != "" {
		parts = strings.Split(parent, "/")
	}
	if len(parts) == 0 {
		return candidates
	}

	last := parts[len(parts)-1]
	slug := parser.GenerateSlug(name)
	dirSlug := parser.GenerateSlug(parentDirName)
	isSelfNamed := strings.EqualFold(parentDirName, name) || (dirSlug != "" && strings.EqualFold(dirSlug, slug))
	if !isSelfNamed || len(parts) < 2 {
		return candidates
	}

	// Promote a self-named page title to its containing section only when the
	// page still lives at a child slug under that section:
	//   section/section.md -> /section/section/  => also label /section/
	//   web_app/web_app.md -> /web_app/web-app/   => also label /web_app/
	// If URL generation already collapsed the file onto the section directory
	// (blog/EDC/EDC.md -> /blog/EDC/), the page already labels that section.
	// Climbing one more level would wrongly retitle the grandparent (/blog/).
	lastIsFileSlug := strings.EqualFold(name, last) || (slug != "" && strings.EqualFold(slug, last))
	if !lastIsFileSlug {
		return candidates
	}
	secondLast := parts[len(parts)-2]
	secondLastIsSourceDir := strings.EqualFold(secondLast, parentDirName) ||
		(dirSlug != "" && strings.EqualFold(secondLast, dirSlug))
	if !secondLastIsSourceDir {
		return candidates
	}

	candidates = append(candidates, pageTitleCandidate{
		URL:      "/" + strings.Join(parts[:len(parts)-1], "/") + "/",
		Priority: 1,
	})
	return candidates
}

func displayTitleForPage(page *Page) string {
	if page == nil {
		return ""
	}
	if page.Frontmatter != nil {
		if title := strings.TrimSpace(page.Frontmatter.NavTitle); title != "" {
			return title
		}
	}
	return strings.TrimSpace(page.Title)
}

func normalizeNavURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	trimmed = "/" + strings.Trim(trimmed, "/")
	if trimmed == "/" {
		return trimmed
	}
	return trimmed + "/"
}

func filepathBaseSlash(path string) string {
	path = strings.TrimRight(strings.ReplaceAll(path, "\\", "/"), "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func filepathDirSlash(path string) string {
	path = strings.TrimRight(strings.ReplaceAll(path, "\\", "/"), "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return ""
}

func filepathExtSlash(path string) string {
	base := filepathBaseSlash(path)
	if idx := strings.LastIndex(base, "."); idx >= 0 {
		return base[idx:]
	}
	return ""
}

func humanizeSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return ""
	}
	segment = strings.NewReplacer("-", " ", "_", " ").Replace(segment)
	segment = strings.Join(strings.Fields(segment), " ")
	return capitalizeFirst(segment)
}
