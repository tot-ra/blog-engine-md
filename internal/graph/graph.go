package graph

import (
	"encoding/json"
	"fmt"
	"html"
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
	Type  string `json:"type"` // "blog", "doc", "tag"
	URL   string `json:"url"`
	Size  int    `json:"size"` // Based on link count
	Color string `json:"color"`
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

func generateGraphHTML(siteTitle string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Graph View | %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; overflow: hidden; background: #ffffff; color: #111; }
        #graph-container { width: 100vw; height: 100vh; display: block; }
        .legend {
            position: fixed;
            left: 50%%;
            bottom: 16px;
            transform: translateX(-50%%);
            z-index: 10;
            display: flex;
            align-items: center;
            gap: 14px;
            background: rgba(255,255,255,0.95);
            padding: 8px 12px;
            border-radius: 999px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.15);
        }
        .legend-item { display: flex; align-items: center; gap: 6px; }
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
        body.embed .back-link { display: none; }
        body.embed .legend {
            bottom: 8px;
            gap: 10px;
            padding: 6px 10px;
            font-size: 12px;
        }
        body.dark {
            background: #111315;
            color: #e8edf6;
        }
        body.dark .legend,
        body.dark .back-link {
            background: rgba(33, 37, 45, 0.94);
            color: #e8edf6;
            border: 1px solid rgba(255,255,255,0.08);
        }
        body.dark .back-link:hover {
            background: rgba(45, 50, 60, 0.98);
        }
    </style>
</head>
<body>
    <div class="legend">
        <div class="legend-item"><div class="legend-dot" style="background:#4CAF50"></div> Blog</div>
        <div class="legend-item"><div class="legend-dot" style="background:#2196F3"></div> Docs</div>
        <div class="legend-item"><div class="legend-dot" style="background:#FF9800"></div> Tags</div>
    </div>
    <a href="/" class="back-link">← Back to site</a>
    <div id="tooltip" class="tooltip"></div>
    <canvas id="graph-container"></canvas>

    <script src="https://d3js.org/d3.v7.min.js"></script>
    <script>
    (function() {
        const savedTheme = localStorage.getItem('theme');
        const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
        const isDarkTheme = savedTheme === 'dark' || (savedTheme !== 'light' && prefersDark);
        if (isDarkTheme) {
            document.body.classList.add('dark');
        }

        const embed = new URLSearchParams(window.location.search).get('embed') === '1';
        if (embed) {
            document.body.classList.add('embed');
        }

        const pathParts = window.location.pathname.split('/').filter(Boolean);
        const langPrefix = (pathParts.length > 0 && (pathParts[0] === 'ru' || pathParts[0] === 'en')) ? '/' + pathParts[0] : '';
        const backLink = document.querySelector('.back-link');
        if (backLink) {
            backLink.setAttribute('href', (langPrefix || '') + '/');
        }

        fetch((langPrefix || '') + '/graph.json')
            .then(r => r.json())
            .then(data => initGraph(data));

        function initGraph(data) {
            const canvas = document.getElementById('graph-container');
            const ctx = canvas.getContext('2d');
            const tooltip = document.getElementById('tooltip');
            let width = canvas.width = window.innerWidth;
            let height = canvas.height = window.innerHeight;
            let scale = 1;
            let offsetX = 0;
            let offsetY = 0;
            let didMove = false;

            const simulation = d3.forceSimulation(data.nodes)
                .force('link', d3.forceLink(data.edges).id(d => d.id).distance(58))
                .force('charge', d3.forceManyBody().strength(-120))
                .force('center', d3.forceCenter(width / 2, height / 2))
                .force('collision', d3.forceCollide().radius(d => d.size + 2));

            function isVisible(node) {
                return true;
            }

            function screenToWorld(x, y) {
                return {
                    x: (x - offsetX) / scale,
                    y: (y - offsetY) / scale
                };
            }

            function worldToScreen(x, y) {
                return {
                    x: x * scale + offsetX,
                    y: y * scale + offsetY
                };
            }

            function hitNode(screenX, screenY) {
                const p = screenToWorld(screenX, screenY);
                return data.nodes.find(n =>
                    isVisible(n) && Math.hypot(n.x - p.x, n.y - p.y) < n.size + 3
                );
            }

            function draw() {
                ctx.clearRect(0, 0, width, height);
                const dark = document.body.classList.contains('dark');
                const edgeColor = dark ? 'rgba(230, 237, 246, 0.34)' : 'rgba(0,0,0,0.1)';
                const labelColor = dark ? '#e8edf6' : '#333';
                const nodeStroke = dark ? 'rgba(255,255,255,0.85)' : 'white';

                // Draw edges
                ctx.strokeStyle = edgeColor;
                ctx.lineWidth = 1;
                data.edges.forEach(e => {
                    const s = e.source, t = e.target;
                    if (!isVisible(s) || !isVisible(t)) return;
                    const sp = worldToScreen(s.x, s.y);
                    const tp = worldToScreen(t.x, t.y);
                    ctx.beginPath();
                    ctx.moveTo(sp.x, sp.y);
                    ctx.lineTo(tp.x, tp.y);
                    ctx.stroke();
                });

                // Draw nodes
                data.nodes.forEach(n => {
                    if (!isVisible(n)) return;
                    const p = worldToScreen(n.x, n.y);
                    const r = Math.max(1.8, n.size * scale * 0.7);
                    ctx.beginPath();
                    ctx.arc(p.x, p.y, r, 0, 2 * Math.PI);
                    ctx.fillStyle = n.color;
                    ctx.fill();
                    ctx.strokeStyle = nodeStroke;
                    ctx.lineWidth = 1.5;
                    ctx.stroke();

                    // Label
                    if (n.size >= 5 && scale >= 0.7) {
                        ctx.fillStyle = labelColor;
                        ctx.font = '11px sans-serif';
                        ctx.textAlign = 'center';
                        ctx.fillText(n.label, p.x, p.y + r + 14);
                    }
                });
            }

            simulation.on('tick', draw);
            draw();

            // Drag nodes + pan view
            let dragNode = null;
            let panStart = null;
            let startOffsetX = 0;
            let startOffsetY = 0;
            canvas.addEventListener('mousedown', e => {
                didMove = false;
                const [mx, my] = [e.offsetX, e.offsetY];
                dragNode = hitNode(mx, my);
                if (dragNode) {
                    simulation.alphaTarget(0.3).restart();
                    dragNode.fx = dragNode.x;
                    dragNode.fy = dragNode.y;
                } else {
                    panStart = { x: e.clientX, y: e.clientY };
                    startOffsetX = offsetX;
                    startOffsetY = offsetY;
                }
            });
            canvas.addEventListener('mousemove', e => {
                const [mx, my] = [e.offsetX, e.offsetY];
                if (dragNode) {
                    const p = screenToWorld(mx, my);
                    dragNode.fx = p.x;
                    dragNode.fy = p.y;
                    didMove = true;
                } else if (panStart) {
                    const dx = e.clientX - panStart.x;
                    const dy = e.clientY - panStart.y;
                    offsetX = startOffsetX + dx;
                    offsetY = startOffsetY + dy;
                    didMove = didMove || Math.abs(dx) > 2 || Math.abs(dy) > 2;
                    canvas.style.cursor = 'grabbing';
                    draw();
                } else {
                    const hover = hitNode(mx, my);
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
                panStart = null;
                if (canvas.style.cursor === 'grabbing') {
                    canvas.style.cursor = 'default';
                }
            });
            canvas.addEventListener('mouseleave', () => {
                panStart = null;
                tooltip.style.display = 'none';
                if (dragNode) {
                    simulation.alphaTarget(0);
                    dragNode.fx = null;
                    dragNode.fy = null;
                    dragNode = null;
                }
            });
            canvas.addEventListener('wheel', e => {
                e.preventDefault();
                const factor = e.deltaY < 0 ? 1.1 : 0.9;
                const nextScale = Math.max(0.12, Math.min(3.2, scale * factor));
                const world = screenToWorld(e.offsetX, e.offsetY);
                scale = nextScale;
                offsetX = e.offsetX - world.x * scale;
                offsetY = e.offsetY - world.y * scale;
                draw();
            }, { passive: false });
            canvas.addEventListener('click', e => {
                if (dragNode || didMove) return;
                const clicked = hitNode(e.offsetX, e.offsetY);
                if (clicked && clicked.url) {
                    if (embed && window.parent && window.parent !== window) {
                        window.parent.postMessage({ type: 'blog-graph-navigate', url: clicked.url }, window.location.origin);
                        return;
                    }
                    window.location.href = clicked.url;
                }
            });

            window.addEventListener('resize', () => {
                width = canvas.width = window.innerWidth;
                height = canvas.height = window.innerHeight;
                simulation.force('center', d3.forceCenter(width / 2, height / 2));
                simulation.alpha(0.3).restart();
                draw();
            });
        }
    })();
    </script>
</body>
</html>`, html.EscapeString(siteTitle))
}
