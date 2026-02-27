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
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`  // "blog", "doc", "tag"
	URL   string `json:"url"`
	Size  int    `json:"size"`  // Based on link count
	Color string `json:"color"`
}

// GraphEdge represents a relationship between nodes
type GraphEdge struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Type   string  `json:"type"` // "link", "tag"
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
				graph.Edges = append(graph.Edges, GraphEdge{
					Source: p.ID,
					Target: targetID,
					Type:   "link",
					Weight: 1.0,
				})
				linkCount[p.ID]++
				linkCount[targetID]++
			}
		}

		// Add tag edges
		for _, tag := range p.Tags {
			tagID := "tag-" + tag
			tagSet[tag] = true
			graph.Edges = append(graph.Edges, GraphEdge{
				Source: p.ID,
				Target: tagID,
				Type:   "tag",
				Weight: 0.5,
			})
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

	return graph
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

func generateGraphHTML(siteTitle string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Graph View | %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; overflow: hidden; }
        #graph-container { width: 100vw; height: 100vh; }
        .controls {
            position: fixed; top: 16px; left: 16px; z-index: 10;
            background: rgba(255,255,255,0.95); padding: 12px 16px;
            border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.15);
            font-size: 14px;
        }
        .controls h3 { margin-bottom: 8px; font-size: 16px; }
        .controls label { display: block; margin: 4px 0; cursor: pointer; }
        .controls input[type="checkbox"] { margin-right: 6px; }
        .legend { margin-top: 12px; }
        .legend-item { display: flex; align-items: center; gap: 6px; margin: 3px 0; }
        .legend-dot { width: 10px; height: 10px; border-radius: 50%%; }
        .tooltip {
            position: fixed; padding: 6px 10px; background: rgba(0,0,0,0.85);
            color: white; border-radius: 4px; font-size: 13px;
            pointer-events: none; display: none; z-index: 20;
        }
        .back-link {
            position: fixed; top: 16px; right: 16px; z-index: 10;
            background: rgba(255,255,255,0.95); padding: 8px 16px;
            border-radius: 6px; box-shadow: 0 2px 8px rgba(0,0,0,0.15);
            text-decoration: none; color: #333; font-size: 14px;
        }
        .back-link:hover { background: #f0f0f0; }
    </style>
</head>
<body>
    <div class="controls">
        <h3>🔗 Graph View</h3>
        <label><input type="checkbox" id="filter-blog" checked> Blog posts</label>
        <label><input type="checkbox" id="filter-doc" checked> Documentation</label>
        <label><input type="checkbox" id="filter-tag" checked> Tags</label>
        <div class="legend">
            <div class="legend-item"><div class="legend-dot" style="background:#4CAF50"></div> Blog</div>
            <div class="legend-item"><div class="legend-dot" style="background:#2196F3"></div> Docs</div>
            <div class="legend-item"><div class="legend-dot" style="background:#FF9800"></div> Tags</div>
        </div>
    </div>
    <a href="/" class="back-link">← Back to site</a>
    <div id="tooltip" class="tooltip"></div>
    <canvas id="graph-container"></canvas>

    <script src="https://d3js.org/d3.v7.min.js"></script>
    <script>
    (function() {
        fetch('/graph.json')
            .then(r => r.json())
            .then(data => initGraph(data));

        function initGraph(data) {
            const canvas = document.getElementById('graph-container');
            const ctx = canvas.getContext('2d');
            const tooltip = document.getElementById('tooltip');
            let width = canvas.width = window.innerWidth;
            let height = canvas.height = window.innerHeight;

            const simulation = d3.forceSimulation(data.nodes)
                .force('link', d3.forceLink(data.edges).id(d => d.id).distance(80))
                .force('charge', d3.forceManyBody().strength(-200))
                .force('center', d3.forceCenter(width / 2, height / 2))
                .force('collision', d3.forceCollide().radius(d => d.size + 5));

            // Filter state
            const filters = { blog: true, doc: true, tag: true, page: true };

            document.querySelectorAll('.controls input').forEach(cb => {
                cb.addEventListener('change', () => {
                    const type = cb.id.replace('filter-', '');
                    filters[type] = cb.checked;
                    simulation.alpha(0.3).restart();
                });
            });

            function isVisible(node) {
                return filters[node.type] !== false;
            }

            simulation.on('tick', () => {
                ctx.clearRect(0, 0, width, height);

                // Draw edges
                ctx.strokeStyle = 'rgba(0,0,0,0.1)';
                ctx.lineWidth = 1;
                data.edges.forEach(e => {
                    const s = e.source, t = e.target;
                    if (!isVisible(s) || !isVisible(t)) return;
                    ctx.beginPath();
                    ctx.moveTo(s.x, s.y);
                    ctx.lineTo(t.x, t.y);
                    ctx.stroke();
                });

                // Draw nodes
                data.nodes.forEach(n => {
                    if (!isVisible(n)) return;
                    ctx.beginPath();
                    ctx.arc(n.x, n.y, n.size, 0, 2 * Math.PI);
                    ctx.fillStyle = n.color;
                    ctx.fill();
                    ctx.strokeStyle = 'white';
                    ctx.lineWidth = 1.5;
                    ctx.stroke();

                    // Label
                    if (n.size >= 5) {
                        ctx.fillStyle = '#333';
                        ctx.font = '11px sans-serif';
                        ctx.textAlign = 'center';
                        ctx.fillText(n.label, n.x, n.y + n.size + 14);
                    }
                });
            });

            // Drag
            let dragNode = null;
            canvas.addEventListener('mousedown', e => {
                const [mx, my] = [e.offsetX, e.offsetY];
                dragNode = data.nodes.find(n =>
                    isVisible(n) && Math.hypot(n.x - mx, n.y - my) < n.size + 3
                );
                if (dragNode) {
                    simulation.alphaTarget(0.3).restart();
                    dragNode.fx = dragNode.x;
                    dragNode.fy = dragNode.y;
                }
            });
            canvas.addEventListener('mousemove', e => {
                const [mx, my] = [e.offsetX, e.offsetY];
                if (dragNode) {
                    dragNode.fx = mx;
                    dragNode.fy = my;
                } else {
                    const hover = data.nodes.find(n =>
                        isVisible(n) && Math.hypot(n.x - mx, n.y - my) < n.size + 3
                    );
                    if (hover) {
                        tooltip.style.display = 'block';
                        tooltip.style.left = (e.clientX + 12) + 'px';
                        tooltip.style.top = (e.clientY + 12) + 'px';
                        tooltip.textContent = hover.label + ' (' + hover.type + ')';
                        canvas.style.cursor = 'pointer';
                    } else {
                        tooltip.style.display = 'none';
                        canvas.style.cursor = 'default';
                    }
                }
            });
            canvas.addEventListener('mouseup', () => {
                if (dragNode) {
                    simulation.alphaTarget(0);
                    dragNode.fx = null;
                    dragNode.fy = null;
                    dragNode = null;
                }
            });
            canvas.addEventListener('click', e => {
                if (dragNode) return;
                const [mx, my] = [e.offsetX, e.offsetY];
                const clicked = data.nodes.find(n =>
                    isVisible(n) && Math.hypot(n.x - mx, n.y - my) < n.size + 3
                );
                if (clicked && clicked.url) {
                    window.location.href = clicked.url;
                }
            });

            window.addEventListener('resize', () => {
                width = canvas.width = window.innerWidth;
                height = canvas.height = window.innerHeight;
                simulation.force('center', d3.forceCenter(width / 2, height / 2));
                simulation.alpha(0.3).restart();
            });
        }
    })();
    </script>
</body>
</html>`, siteTitle)
}
