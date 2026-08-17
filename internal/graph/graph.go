package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GraphData holds the complete graph structure
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode represents a page in the graph
type GraphNode struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	Type  string  `json:"type"` // "blog", "doc", "tag"
	URL   string  `json:"url"`
	Size  int     `json:"size"` // Based on link count
	Color string  `json:"color"`
	X     float64 `json:"x"` // Embedding-space layout (PCA)
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
}

// GraphEdge represents a relationship between nodes
type GraphEdge struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Type   string  `json:"type"` // "link", "tag", "related"
	Weight float64 `json:"weight"`
}

// PageInfo holds minimal page information for graph building
type PageInfo struct {
	ID         string
	Title      string
	URL        string
	Type       string // "blog", "doc", "page"
	Tags       []string
	RawContent string
	// Vector is an optional normalized embedding used for 3D layout.
	Vector []float32
}

// internal link pattern: [text](./path) or [text](/path) or [text](../path)
var internalLinkRegex = regexp.MustCompile(`\[([^\]]+)\]\((/[^)]+|\.\.?/[^)]+)\)`)

// nodeColors maps page types to colors
var nodeColors = map[string]string{
	"blog": "#4CAF50",
	"doc":  "#2196F3",
	"page": "#607D8B",
	"tag":  "#FF9800",
}

// BuildGraph creates graph data from a list of pages
func BuildGraph(pages []PageInfo) *GraphData {
	graph := &GraphData{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}
	edgeSet := make(map[string]struct{})

	// Build URL-to-ID lookup for resolving internal links
	urlToID := make(map[string]string)
	for _, p := range pages {
		urlToID[p.URL] = p.ID
		// Also map without trailing slash
		urlToID[strings.TrimSuffix(p.URL, "/")] = p.ID
	}

	// Track link counts for sizing
	linkCount := make(map[string]int)

	// Collect all tags
	tagSet := make(map[string]bool)

	for _, p := range pages {
		// Add page node
		color := nodeColors[p.Type]
		if color == "" {
			color = nodeColors["page"]
		}

		graph.Nodes = append(graph.Nodes, GraphNode{
			ID:    p.ID,
			Label: p.Title,
			Type:  p.Type,
			URL:   p.URL,
			Color: color,
		})

		// Extract internal links
		matches := internalLinkRegex.FindAllStringSubmatch(p.RawContent, -1)
		for _, m := range matches {
			linkURL := m[2]
			// Normalize the link URL
			linkURL = normalizeLinkURL(p.URL, linkURL)

			if targetID, ok := urlToID[linkURL]; ok {
				if !addUniqueEdge(graph, edgeSet, GraphEdge{
					Source: p.ID,
					Target: targetID,
					Type:   "link",
					Weight: 1.0,
				}) {
					continue
				}
				linkCount[p.ID]++
				linkCount[targetID]++
			}
		}

		// Add tag edges (hub-and-spoke). Do not also draw pairwise
		// article-to-article "related" edges for shared tags: broad tags
		// like events/ai create a complete clique that overwhelms natural
		// in-body links in graph view.
		for _, tag := range p.Tags {
			tagID := "tag-" + tag
			tagSet[tag] = true
			if !addUniqueEdge(graph, edgeSet, GraphEdge{
				Source: p.ID,
				Target: tagID,
				Type:   "tag",
				Weight: 0.5,
			}) {
				continue
			}
			linkCount[p.ID]++
			linkCount[tagID]++
		}
	}

	// Add tag nodes
	for tag := range tagSet {
		graph.Nodes = append(graph.Nodes, GraphNode{
			ID:    "tag-" + tag,
			Label: "#" + tag,
			Type:  "tag",
			URL:   "/tags/" + tag + "/",
			Color: nodeColors["tag"],
		})
	}

	// Set node sizes based on link count
	for i := range graph.Nodes {
		count := linkCount[graph.Nodes[i].ID]
		graph.Nodes[i].Size = 3 + count*2
		if graph.Nodes[i].Size > 20 {
			graph.Nodes[i].Size = 20
		}
	}

	// Place nodes from embeddings (PCA → 3D). Links remain as visual edges only.
	assignEmbeddingLayout(graph, pages)

	return graph
}

func addUniqueEdge(graph *GraphData, edgeSet map[string]struct{}, edge GraphEdge) bool {
	key := edgeKey(edge.Source, edge.Target, edge.Type)
	if _, exists := edgeSet[key]; exists {
		return false
	}
	edgeSet[key] = struct{}{}
	graph.Edges = append(graph.Edges, edge)
	return true
}

func edgeKey(source, target, edgeType string) string {
	switch edgeType {
	case "tag", "related":
		if source > target {
			source, target = target, source
		}
	}
	return source + "|" + target + "|" + edgeType
}

// normalizeLinkURL resolves a relative link URL based on the source page URL
func normalizeLinkURL(sourceURL, linkURL string) string {
	// Absolute path
	if strings.HasPrefix(linkURL, "/") {
		url := strings.TrimSuffix(linkURL, ".md")
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		return url
	}

	// Relative path — resolve relative to source
	sourceDir := filepath.Dir(sourceURL)
	resolved := filepath.Join(sourceDir, linkURL)
	resolved = strings.TrimSuffix(resolved, ".md")
	if !strings.HasSuffix(resolved, "/") {
		resolved += "/"
	}
	return "/" + strings.TrimPrefix(resolved, "/")
}

// WriteGraphJSON writes the graph data to a JSON file
func WriteGraphJSON(graph *GraphData, outputDir string) error {
	outPath := filepath.Join(outputDir, "graph.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for graph.json: %w", err)
	}

	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal graph data: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write graph.json: %w", err)
	}

	return nil
}

// WriteGraphPage writes the graph HTML page
func WriteGraphPage(outputDir, siteTitle string) error {
	outPath := filepath.Join(outputDir, "graph", "index.html")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("failed to create graph directory: %w", err)
	}

	html := generateGraphHTML(siteTitle)
	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write graph page: %w", err)
	}

	return nil
}

