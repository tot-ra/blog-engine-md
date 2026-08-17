package graph

import (
	"encoding/json"
	"fmt"
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
			if n.Color != "#F97316" {
				t.Errorf("expected tag color #F97316, got %s", n.Color)
			}
		}
	}
	if !tagNodeFound {
		t.Error("expected tag node 'tag-go' to exist")
	}
}

func TestBuildGraphUsesDocsColorForGenericPages(t *testing.T) {
	graph := BuildGraph([]PageInfo{{
		ID:   "about",
		URL:  "/about/",
		Type: "page",
	}})

	if len(graph.Nodes) != 1 || graph.Nodes[0].Color != "#3B82F6" {
		t.Fatalf("expected generic content page to use docs blue, got %#v", graph.Nodes)
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

	// Hub has 3 tag edges, so linkCount = 3 and size = 3 + 3 = 6.
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
	if hubNode.Size != 6 {
		t.Errorf("expected hub size 6, got %d", hubNode.Size)
	}
}

func TestBuildGraph_DoesNotAddPairwiseRelatedEdgesForSharedTags(t *testing.T) {
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
	tagEdges := 0
	for _, e := range graph.Edges {
		switch e.Type {
		case "related":
			relatedEdges++
		case "tag":
			tagEdges++
		}
	}

	// Shared tags should connect via tag hubs only, not article cliques.
	if relatedEdges != 0 {
		t.Fatalf("expected 0 related edges, got %d", relatedEdges)
	}
	// a:{go,api} + b:{go,api} + c:{go} => 5 page→tag edges
	if tagEdges != 5 {
		t.Fatalf("expected 5 tag edges, got %d", tagEdges)
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
	if !strings.Contains(html, `three@0.170.0`) {
		t.Fatal("expected Three.js CDN import for 3D graph view")
	}
	if !strings.Contains(html, `OrbitControls`) {
		t.Fatal("expected OrbitControls for 3D camera navigation")
	}
	if strings.Contains(html, `d3.forceSimulation`) {
		t.Fatal("force-directed D3 layout should be removed from graph page")
	}
	if !strings.Contains(html, `background:#22C55E`) || !strings.Contains(html, `background:#3B82F6`) || !strings.Contains(html, `background:#F97316`) {
		t.Fatal("expected bright graph palette in legend")
	}
	if !strings.Contains(html, `const radius = Math.max(0.55, (n.size || 3) * 0.22)`) {
		t.Fatal("expected reduced Three.js node radius")
	}
	if !strings.Contains(html, `emissiveIntensity: dark ? 0.2 : 0.12`) {
		t.Fatal("expected graph colors to remain vivid under scene lighting")
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
	if hubNode.Size != 12 {
		t.Fatalf("expected capped hub size 12, got %d", hubNode.Size)
	}
}

func TestBuildGraphCapsTagNodeSizeBelowPageHubs(t *testing.T) {
	pages := make([]PageInfo, 20)
	for i := range pages {
		pages[i] = PageInfo{
			ID:   fmt.Sprintf("page-%d", i),
			URL:  fmt.Sprintf("/page-%d/", i),
			Type: "blog",
			Tags: []string{"shared"},
		}
	}

	graph := BuildGraph(pages)
	for _, node := range graph.Nodes {
		if node.ID == "tag-shared" {
			if node.Size != 9 {
				t.Fatalf("expected capped tag size 9, got %d", node.Size)
			}
			return
		}
	}
	t.Fatal("shared tag node not found")
}
