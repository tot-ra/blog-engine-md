package graph

import (
	"testing"
)

func TestBuildGraph_BasicPages(t *testing.T) {
	pages := []PageInfo{
		{
			ID:         "blog-hello",
			Title:      "Hello World",
			URL:        "/blog/hello/",
			Type:       "blog",
			Tags:       []string{"go", "tutorial"},
			RawContent: "Check out [about](/docs/about/).",
		},
		{
			ID:         "docs-about",
			Title:      "About",
			URL:        "/docs/about/",
			Type:       "doc",
			RawContent: "See [hello](/blog/hello/).",
		},
	}

	graph := BuildGraph(pages)

	// 2 pages + 2 tags = 4 nodes
	if len(graph.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(graph.Nodes))
	}

	// 2 internal links + 2 tag edges = 4 edges
	if len(graph.Edges) != 4 {
		t.Errorf("expected 4 edges, got %d", len(graph.Edges))
	}

	// Check tag nodes exist
	tagNodeFound := false
	for _, n := range graph.Nodes {
		if n.ID == "tag-go" {
			tagNodeFound = true
			if n.Color != "#FF9800" {
				t.Errorf("expected tag color #FF9800, got %s", n.Color)
			}
		}
	}
	if !tagNodeFound {
		t.Error("expected tag node 'tag-go' to exist")
	}
}

func TestBuildGraph_NoLinks(t *testing.T) {
	pages := []PageInfo{
		{
			ID:         "page-1",
			Title:      "Page 1",
			URL:        "/page-1/",
			Type:       "page",
			RawContent: "No links here.",
		},
	}

	graph := BuildGraph(pages)

	if len(graph.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(graph.Edges))
	}
}

func TestBuildGraph_NodeSizes(t *testing.T) {
	pages := []PageInfo{
		{
			ID:         "hub",
			Title:      "Hub",
			URL:        "/hub/",
			Type:       "doc",
			Tags:       []string{"a", "b", "c"},
			RawContent: "",
		},
	}

	graph := BuildGraph(pages)

	// Hub has 3 tag edges, so linkCount = 3, size = 3 + 3*2 = 9
	var hubNode *GraphNode
	for _, n := range graph.Nodes {
		if n.ID == "hub" {
			hubNode = &n
			break
		}
	}
	if hubNode == nil {
		t.Fatal("hub node not found")
	}
	if hubNode.Size != 9 {
		t.Errorf("expected hub size 9, got %d", hubNode.Size)
	}
}

func TestNormalizeLinkURL(t *testing.T) {
	tests := []struct {
		source string
		link   string
		expect string
	}{
		{"/blog/hello/", "/docs/about/", "/docs/about/"},
		{"/blog/hello/", "/docs/about.md", "/docs/about/"},
	}

	for _, tt := range tests {
		got := normalizeLinkURL(tt.source, tt.link)
		if got != tt.expect {
			t.Errorf("normalizeLinkURL(%q, %q) = %q, want %q", tt.source, tt.link, got, tt.expect)
		}
	}
}
