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

	gen := NewPrevNextGenerator(false)

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

	gen := NewPrevNextGenerator(false)
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

func TestPrevNext_DocsSameCategoryOnly(t *testing.T) {
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "First", URL: "/est/study/mahtra_pohikool/2klass/first/", Language: "est", Type: TypeDoc,
			Frontmatter: &parser.Frontmatter{Order: 1}},
		"p2": {ID: "p2", Title: "Second", URL: "/est/study/mahtra_pohikool/2klass/second/", Language: "est", Type: TypeDoc,
			Frontmatter: &parser.Frontmatter{Order: 2}},
		"p3": {ID: "p3", Title: "Third", URL: "/est/study/mahtra_pohikool/3klass/third/", Language: "est", Type: TypeDoc,
			Frontmatter: &parser.Frontmatter{Order: 1}},
	}

	tree := NewNavigationBuilder().BuildTree(pages)
	gen := NewPrevNextGenerator(true)

	links := gen.Generate(pages["p2"], pages, tree)
	if links == nil {
		t.Fatal("Expected prev/next links")
	}
	if links.Prev == nil || links.Prev.Title != "First" {
		t.Fatalf("Expected prev=First, got %+v", links.Prev)
	}
	if links.Next != nil {
		t.Fatalf("Expected no next link across sibling sections, got %+v", links.Next)
	}
}

func TestPrevNext_TypePageUsesDocsOrdering(t *testing.T) {
	pages := map[string]*Page{
		"p1": {ID: "p1", Title: "First", URL: "/est/study/mahtra_pohikool/2klass/first/", Language: "est", Type: TypePage,
			Frontmatter: &parser.Frontmatter{Order: 1}},
		"p2": {ID: "p2", Title: "Second", URL: "/est/study/mahtra_pohikool/2klass/second/", Language: "est", Type: TypePage,
			Frontmatter: &parser.Frontmatter{Order: 2}},
		"section": {ID: "section", Title: "2 klass", URL: "/est/study/mahtra_pohikool/2klass/", Language: "est", Type: TypePage,
			Frontmatter: &parser.Frontmatter{Order: 1}},
	}

	tree := NewNavigationBuilder().BuildTree(pages)
	gen := NewPrevNextGenerator(true)

	links := gen.Generate(pages["p1"], pages, tree)
	if links == nil {
		t.Fatal("Expected prev/next links")
	}
	if links.Prev != nil {
		t.Fatalf("Expected no previous link for first page, got %+v", links.Prev)
	}
	if links.Next == nil || links.Next.Title != "Second" {
		t.Fatalf("Expected next=Second, got %+v", links.Next)
	}
}
