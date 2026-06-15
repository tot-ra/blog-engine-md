package builder

import (
	"testing"

	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestBuildTree_BasicStructure(t *testing.T) {
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "Post One", URL: "/blog/tech/post-one/", Type: TypeBlog, Frontmatter: &parser.Frontmatter{}},
		"p2": {ID: "p2", Title: "Post Two", URL: "/blog/tech/post-two/", Type: TypeBlog, Frontmatter: &parser.Frontmatter{}},
		"p3": {ID: "p3", Title: "About", URL: "/docs/about/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{}},
	}

	nb := NewNavigationBuilder()
	tree := nb.BuildTree(pages)

	if tree.Root == nil {
		t.Fatal("Expected root node")
	}
	if len(tree.Root.Children) != 2 {
		t.Fatalf("Expected 2 top-level sections (blog, docs), got %d", len(tree.Root.Children))
	}

	// Check ByPath lookups
	if _, ok := tree.ByPath["/blog/"]; !ok {
		t.Error("Expected /blog/ in ByPath")
	}
	if _, ok := tree.ByPath["/docs/"]; !ok {
		t.Error("Expected /docs/ in ByPath")
	}
	if _, ok := tree.ByPath["/blog/tech/"]; !ok {
		t.Error("Expected /blog/tech/ in ByPath")
	}
}

func TestBuildTree_Ordering(t *testing.T) {
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "Zebra", URL: "/docs/zebra/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{}},
		"p2": {ID: "p2", Title: "Alpha", URL: "/docs/alpha/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{Order: 2}},
		"p3": {ID: "p3", Title: "Beta", URL: "/docs/beta/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{Order: 1}},
	}

	nb := NewNavigationBuilder()
	tree := nb.BuildTree(pages)

	docsNode := tree.ByPath["/docs/"]
	if docsNode == nil {
		t.Fatal("Expected /docs/ node")
	}

	// Order should be: Beta (order=1), Alpha (order=2), Zebra (order=0 = unordered, last)
	if len(docsNode.Children) != 3 {
		t.Fatalf("Expected 3 children under docs, got %d", len(docsNode.Children))
	}
	if docsNode.Children[0].Title != "Beta" {
		t.Errorf("Expected first child 'Beta', got '%s'", docsNode.Children[0].Title)
	}
	if docsNode.Children[1].Title != "Alpha" {
		t.Errorf("Expected second child 'Alpha', got '%s'", docsNode.Children[1].Title)
	}
	if docsNode.Children[2].Title != "Zebra" {
		t.Errorf("Expected third child 'Zebra', got '%s'", docsNode.Children[2].Title)
	}
}

func TestBuildTree_OrderingPrefersExplicitOrderOverNodeType(t *testing.T) {
	pages := map[string]*Page{
		"root":          {ID: "root", Title: "Root", URL: "/docs/mixed/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{}},
		"section-child": {ID: "section-child", Title: "Section Child", URL: "/docs/mixed/section-child/nested/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{Order: 2}},
		"page-child":    {ID: "page-child", Title: "Page Child", URL: "/docs/mixed/page-child/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{Order: 1}},
	}

	nb := NewNavigationBuilder()
	tree := nb.BuildTree(pages)

	node := tree.ByPath["/docs/mixed/"]
	if node == nil {
		t.Fatal("Expected /docs/mixed/ node")
	}
	if len(node.Children) != 2 {
		t.Fatalf("Expected 2 children under /docs/mixed/, got %d", len(node.Children))
	}
	if node.Children[0].Title != "Page Child" {
		t.Fatalf("Expected first child 'Page Child', got %q", node.Children[0].Title)
	}
	if node.Children[1].Title != "Section-child" {
		t.Fatalf("Expected second child 'Section Child', got %q", node.Children[1].Title)
	}
}

func TestBuildTree_HideNav(t *testing.T) {
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "Visible", URL: "/docs/visible/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{}},
		"p2": {ID: "p2", Title: "Hidden", URL: "/docs/hidden/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{HideNav: true}},
	}

	nb := NewNavigationBuilder()
	tree := nb.BuildTree(pages)

	docsNode := tree.ByPath["/docs/"]
	if docsNode == nil {
		t.Fatal("Expected /docs/ node")
	}
	if len(docsNode.Children) != 1 {
		t.Fatalf("Expected 1 child (hidden excluded), got %d", len(docsNode.Children))
	}
	if docsNode.Children[0].Title != "Visible" {
		t.Errorf("Expected 'Visible', got '%s'", docsNode.Children[0].Title)
	}
}

func TestFlattenPages(t *testing.T) {
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "A", URL: "/docs/a/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{Order: 1}},
		"p2": {ID: "p2", Title: "B", URL: "/docs/b/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{Order: 2}},
		"p3": {ID: "p3", Title: "C", URL: "/docs/sub/c/", Type: TypeDoc, Frontmatter: &parser.Frontmatter{Order: 1}},
	}

	nb := NewNavigationBuilder()
	tree := nb.BuildTree(pages)
	flat := tree.FlattenPages()

	if len(flat) != 3 {
		t.Fatalf("Expected 3 flat pages, got %d", len(flat))
	}
}

func TestBuildTree_SectionTitleUTF8(t *testing.T) {
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "Post", URL: "/blog/вера/post/", Type: TypeBlog, Frontmatter: &parser.Frontmatter{}},
	}

	nb := NewNavigationBuilder()
	tree := nb.BuildTree(pages)

	node := tree.ByPath["/blog/вера/"]
	if node == nil {
		t.Fatal("Expected /blog/вера/ node")
	}
	if node.Title != "Вера" {
		t.Fatalf("Expected UTF-8 title 'Вера', got '%s'", node.Title)
	}
}

func TestBuildTree_UsesConfiguredSegmentLabels(t *testing.T) {
	pages := map[string]*Page{
		"sensor": {
			ID:          "sensor",
			Title:       "Installation",
			URL:         "/et/docs/beehive-sensors/installation/",
			Language:    "et",
			Type:        TypeDoc,
			Frontmatter: &parser.Frontmatter{},
		},
	}
	labels := map[string]map[string]string{
		"et": {
			"beehive-sensors": "Mesitaru andurid",
		},
	}

	tree := NewNavigationBuilderWithLabels(labels).BuildTree(pages)
	node := tree.ByPath["/et/docs/beehive-sensors/"]
	if node == nil {
		t.Fatal("expected configured section node")
	}
	if node.Title != "Mesitaru andurid" {
		t.Fatalf("expected configured segment label, got %q", node.Title)
	}
}
