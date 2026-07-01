package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestBuildGraph_AddsRelatedEdgesForSharedTags(t *testing.T) {
	pages := []PageInfo{
		{
			ID:    "a",
			Title: "A",
			URL:   "/a/",
			Type:  "blog",
			Tags:  []string{"go", "api"},
		},
		{
			ID:    "b",
			Title: "B",
			URL:   "/b/",
			Type:  "blog",
			Tags:  []string{"go", "api"},
		},
		{
			ID:    "c",
			Title: "C",
			URL:   "/c/",
			Type:  "doc",
			Tags:  []string{"go"},
		},
	}

	graph := BuildGraph(pages)

	relatedEdges := 0
	weightedAB := false
	for _, e := range graph.Edges {
		if e.Type != "related" {
			continue
		}
		relatedEdges++
		if (e.Source == "a" && e.Target == "b") || (e.Source == "b" && e.Target == "a") {
			// 2 shared tags => 0.35 + 0.15 = 0.5
			if e.Weight == 0.5 {
				weightedAB = true
			}
		}
	}

	// Pairs sharing at least one tag: a-b, a-c, b-c
	if relatedEdges != 3 {
		t.Fatalf("expected 3 related edges, got %d", relatedEdges)
	}
	if !weightedAB {
		t.Fatal("expected a-b related edge with weight 0.5")
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

func TestWriteGraphJSONWritesIndentedGraphData(t *testing.T) {
	graph := &GraphData{
		Nodes: []GraphNode{{
			ID:    "page-1",
			Label: "Page One",
			Type:  "page",
			URL:   "/page-1/",
			Size:  3,
			Color: "#607D8B",
		}},
		Edges: []GraphEdge{{
			Source: "page-1",
			Target: "tag-go",
			Type:   "tag",
			Weight: 0.5,
		}},
	}
	outputDir := t.TempDir()

	if err := WriteGraphJSON(graph, outputDir); err != nil {
		t.Fatalf("WriteGraphJSON returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "graph.json"))
	if err != nil {
		t.Fatalf("failed to read graph.json: %v", err)
	}

	var got GraphData
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("graph.json does not contain valid JSON: %v", err)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].ID != "page-1" {
		t.Fatalf("unexpected nodes in graph.json: %#v", got.Nodes)
	}
	if len(got.Edges) != 1 || got.Edges[0].Target != "tag-go" || got.Edges[0].Weight != 0.5 {
		t.Fatalf("unexpected edges in graph.json: %#v", got.Edges)
	}
	if !strings.Contains(string(data), "\n  \"nodes\"") {
		t.Fatalf("expected indented JSON, got: %s", string(data))
	}
}

func TestWriteGraphJSONReturnsDirectoryError(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	err := WriteGraphJSON(&GraphData{}, filePath)
	if err == nil {
		t.Fatal("expected error when output path parent is a file")
	}
	if !strings.Contains(err.Error(), "failed to create directory for graph.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteGraphPageWritesEscapedHTML(t *testing.T) {
	outputDir := t.TempDir()
	siteTitle := `My <Site> & "Graph"`

	if err := WriteGraphPage(outputDir, siteTitle); err != nil {
		t.Fatalf("WriteGraphPage returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "graph", "index.html"))
	if err != nil {
		t.Fatalf("failed to read graph page: %v", err)
	}
	html := string(data)

	if !strings.Contains(html, "Graph View | My &lt;Site&gt; &amp; &#34;Graph&#34;") {
		t.Fatalf("expected escaped site title in graph title, got: %s", html)
	}
	if strings.Contains(html, "Graph View | "+siteTitle) {
		t.Fatalf("site title was written without escaping: %s", html)
	}
	if !strings.Contains(html, `fetch((langPrefix || '') + '/graph.json')`) {
		t.Fatal("expected graph page to fetch graph.json")
	}
	if !strings.Contains(html, `postMessage({ type: 'blog-graph-navigate'`) {
		t.Fatal("expected embedded graph navigation support")
	}
}

func TestWriteGraphPageReturnsDirectoryError(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	err := WriteGraphPage(filePath, "Site")
	if err == nil {
		t.Fatal("expected error when output path parent is a file")
	}
	if !strings.Contains(err.Error(), "failed to create graph directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildGraphCapsNodeSize(t *testing.T) {
	pages := []PageInfo{{
		ID:    "hub",
		Title: "Hub",
		URL:   "/hub/",
		Type:  "doc",
	}}
	for i := 0; i < 20; i++ {
		pages[0].Tags = append(pages[0].Tags, string(rune('a'+i)))
	}

	graph := BuildGraph(pages)

	var hubNode *GraphNode
	for i := range graph.Nodes {
		if graph.Nodes[i].ID == "hub" {
			hubNode = &graph.Nodes[i]
			break
		}
	}
	if hubNode == nil {
		t.Fatal("hub node not found")
	}
	if hubNode.Size != 20 {
		t.Fatalf("expected capped hub size 20, got %d", hubNode.Size)
	}
}
