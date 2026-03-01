package builder

import (
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestPrevNext_BlogByDate(t *testing.T) {
	now := time.Now()
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "Newest", URL: "/blog/newest/", Language: "en", Type: TypeBlog,
			Frontmatter: &parser.Frontmatter{Date: now}},
		"p2": {ID: "p2", Title: "Middle", URL: "/blog/middle/", Language: "en", Type: TypeBlog,
			Frontmatter: &parser.Frontmatter{Date: now.Add(-24 * time.Hour)}},
		"p3": {ID: "p3", Title: "Oldest", URL: "/blog/oldest/", Language: "en", Type: TypeBlog,
			Frontmatter: &parser.Frontmatter{Date: now.Add(-48 * time.Hour)}},
	}

	gen := NewPrevNextGenerator()

	// Middle post: prev=Newest, next=Oldest
	links := gen.Generate(pages["p2"], pages, nil)
	if links == nil {
		t.Fatal("Expected prev/next links")
	}
	if links.Prev == nil || links.Prev.Title != "Newest" {
		t.Errorf("Expected prev=Newest, got %+v", links.Prev)
	}
	if links.Next == nil || links.Next.Title != "Oldest" {
		t.Errorf("Expected next=Oldest, got %+v", links.Next)
	}

	// Newest post: no prev, next=Middle
	links = gen.Generate(pages["p1"], pages, nil)
	if links.Prev != nil {
		t.Error("Newest post should have no prev")
	}
	if links.Next == nil || links.Next.Title != "Middle" {
		t.Errorf("Expected next=Middle, got %+v", links.Next)
	}

	// Oldest post: prev=Middle, no next
	links = gen.Generate(pages["p3"], pages, nil)
	if links.Prev == nil || links.Prev.Title != "Middle" {
		t.Errorf("Expected prev=Middle, got %+v", links.Prev)
	}
	if links.Next != nil {
		t.Error("Oldest post should have no next")
	}
}

func TestPrevNext_DocsByTreeOrder(t *testing.T) {
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "First", URL: "/docs/first/", Language: "en", Type: TypeDoc,
			Frontmatter: &parser.Frontmatter{Order: 1}},
		"p2": {ID: "p2", Title: "Second", URL: "/docs/second/", Language: "en", Type: TypeDoc,
			Frontmatter: &parser.Frontmatter{Order: 2}},
	}

	nb := NewNavigationBuilder()
	tree := nb.BuildTree(pages)

	gen := NewPrevNextGenerator()
	links := gen.Generate(pages["p1"], pages, tree)

	if links == nil {
		t.Fatal("Expected prev/next links")
	}
	if links.Prev != nil {
		t.Error("First doc should have no prev")
	}
	if links.Next == nil || links.Next.Title != "Second" {
		t.Errorf("Expected next=Second, got %+v", links.Next)
	}
}
