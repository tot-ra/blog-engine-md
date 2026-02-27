# Phase 5: Advanced Specification

## Overview

Advanced features: Mermaid diagrams, Obsidian-style graph view, dark mode, HTML embeds, and custom blocks.

## Goals

1. Mermaid diagram rendering
2. Obsidian-style graph visualization
3. Dark/light theme toggle
4. HTML embed support (YouTube, etc.)
5. Custom admonition blocks
6. Scroll spy for TOC

## Dependencies

- Phase 1-4: All previous phases

## Components

### 5.1 Mermaid Renderer

**Purpose**: Render Mermaid diagrams to SVG

**Interface**:
```go
type MermaidRenderer interface {
    Render(diagram string, theme string) (string, error)
    RenderFile(inputPath string, outputPath string) error
}
```

**Diagram Types**:
- Flowchart
- Sequence diagram
- Class diagram
- State diagram
- Entity Relationship
- Gantt chart
- Pie chart
- Git graph

**Rendering Strategy**:

**Option A: Server-side (Puppeteer/ChromeDP)**
```go
// Use headless Chrome to render
// Pros: Accurate, no JS on client
// Cons: Slower build, requires Chrome
```

**Option B: Client-side (preferred)**
```go
// Output <pre class="mermaid"> with diagram code
// Let Mermaid.js render in browser
// Pros: Fast build, interactive
// Cons: Requires JS, flash of unstyled content
```

**Output (Client-side)**:
```html
<pre class="mermaid">
graph TD
    A[Start] --> B{Is it?}
    B -->|Yes| C[OK]
    B -->|No| D[End]
</pre>

<script type="module">
  import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs';
  mermaid.initialize({ startOnLoad: true, theme: 'default' });
</script>
```

**Configuration**:
```yaml
mermaid:
  enabled: true
  theme: "default"  # default, dark, forest, neutral
  clientSide: true  # true = browser render, false = build-time
```

### 5.2 Graph View

**Purpose**: Visualize content relationships as interactive graph

**Interface**:
```go
type GraphBuilder interface {
    Build(ctx *BuildContext) (*GraphData, error)
}

type GraphData struct {
    Nodes []GraphNode `json:"nodes"`
    Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
    ID       string  `json:"id"`
    Label    string  `json:"label"`
    Type     string  `json:"type"`      // "blog", "doc", "tag"
    URL      string  `json:"url"`
    Size     int     `json:"size"`      // Based on link count
    X, Y     float64 `json:"x,y"`       // Layout position
    Color    string  `json:"color"`
}

type GraphEdge struct {
    Source string  `json:"source"`
    Target string  `json:"target"`
    Type   string  `json:"type"`        // "link", "tag", "category"
    Weight float64 `json:"weight"`
}
```

**Node Types**:
| Type | Color | Description |
|------|-------|-------------|
| blog | #4CAF50 | Blog posts |
| doc | #2196F3 | Documentation pages |
| tag | #FF9800 | Tag nodes |
| category | #9C27B0 | Category nodes |

**Edge Types**:
- `link` - Internal markdown links
- `tag` - Page connected to tag
- `category` - Page in category

**Graph Page**:
```
/graph/         -> Interactive graph visualization
```

**Frontend (D3.js or ForceGraph)**:
```html
<div id="graph-container"></div>
<script src="/assets/js/graph.js"></script>
<script>
  initGraph('/api/graph.json', '#graph-container');
</script>
```

**Features**:
- Force-directed layout
- Zoom and pan
- Node click to navigate
- Filter by type
- Search nodes
- Cluster by tag/category

### 5.3 Theme System

**Purpose**: Dark/light mode toggle

**Interface**:
```go
type ThemeSystem interface {
    GetCSSVariables(theme string) map[string]string
    GenerateCSS() (string, error)
}
```

**CSS Variables**:
```css
:root {
  /* Light theme (default) */
  --bg-primary: #ffffff;
  --bg-secondary: #f5f5f5;
  --text-primary: #333333;
  --text-secondary: #666666;
  --border-color: #e0e0e0;
  --link-color: #0066cc;
  --link-hover: #0052a3;
  --code-bg: #f4f4f4;
  --accent-color: #0066cc;
}

[data-theme="dark"] {
  --bg-primary: #1a1a1a;
  --bg-secondary: #2d2d2d;
  --text-primary: #e0e0e0;
  --text-secondary: #a0a0a0;
  --border-color: #404040;
  --link-color: #4da3ff;
  --link-hover: #80bdff;
  --code-bg: #2d2d2d;
  --accent-color: #4da3ff;
}
```

**Theme Toggle**:
```html
<button class="theme-toggle" aria-label="Toggle dark mode">
  <span class="theme-icon-light">☀️</span>
  <span class="theme-icon-dark">🌙</span>
</button>
```

**Theme JS**:
```javascript
(function() {
  const theme = localStorage.getItem('theme') || 
                (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
  document.documentElement.setAttribute('data-theme', theme);
  
  document.querySelector('.theme-toggle').addEventListener('click', () => {
    const current = document.documentElement.getAttribute('data-theme');
    const next = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);
  });
})();
```

**Configuration**:
```yaml
theme:
  default: "light"  # light, dark, auto
  allowToggle: true
```

### 5.4 HTML Embed Support

**Purpose**: Support embedded content (YouTube, CodePen, etc.)

**Interface**:
```go
type EmbedTransformer interface {
    Transform(node ast.Node) (ast.Node, error)
}
```

**Supported Embeds**:

**YouTube**:
```markdown
<!-- Markdown link syntax -->
[video](https://www.youtube.com/watch?v=VIDEO_ID)

<!-- Or explicit embed -->
::youtube[VIDEO_ID]
```

**Output**:
```html
<div class="embed embed-youtube">
  <iframe 
    src="https://www.youtube.com/embed/VIDEO_ID" 
    frameborder="0" 
    allowfullscreen
    loading="lazy">
  </iframe>
</div>
```

**Generic oEmbed**:
```markdown
[embed](https://codepen.io/user/pen/ID)
[embed](https://gist.github.com/user/ID)
```

**Requirements**:
- Lazy loading for iframes
- Responsive wrapper (16:9 aspect ratio)
- Privacy-enhanced mode where available

### 5.5 Custom Blocks (Admonitions)

**Purpose**: Special callout blocks for notes, warnings, tips

**Interface**:
```go
type AdmonitionTransformer interface {
    Transform(node ast.Node) (ast.Node, error)
}
```

**Syntax**:
```markdown
:::note
This is a note.
:::

:::warning
This is a warning!
:::

:::tip Title
This is a tip with custom title.
:::

:::danger STOP!
Critical warning here.
:::
```

**Output**:
```html
<div class="admonition admonition-note">
  <div class="admonition-header">
    <span class="admonition-icon">ℹ️</span>
    <span class="admonition-title">Note</span>
  </div>
  <div class="admonition-content">
    <p>This is a note.</p>
  </div>
</div>
```

**Types**:
| Type | Icon | Default Title | Color |
|------|------|---------------|-------|
| note | ℹ️ | Note | blue |
| tip | 💡 | Tip | green |
| info | ℹ️ | Info | blue |
| warning | ⚠️ | Warning | yellow |
| danger | 🛑 | Danger | red |

**Styling**:
```css
.admonition {
  border-left: 4px solid var(--admonition-color);
  background: var(--admonition-bg);
  padding: 1rem;
  margin: 1rem 0;
  border-radius: 4px;
}

.admonition-note { --admonition-color: #0066cc; --admonition-bg: #e6f2ff; }
.admonition-tip { --admonition-color: #00aa44; --admonition-bg: #e6f9ed; }
.admonition-warning { --admonition-color: #ff8800; --admonition-bg: #fff4e6; }
.admonition-danger { --admonition-color: #dd0000; --admonition-bg: #ffe6e6; }
```

### 5.6 Scroll Spy

**Purpose**: Highlight current TOC section while scrolling

**Interface**:
```go
// No Go code - pure client-side JS
```

**JS Implementation**:
```javascript
(function() {
  const tocLinks = document.querySelectorAll('.toc a');
  const headings = document.querySelectorAll('h2[id], h3[id], h4[id]');
  
  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        tocLinks.forEach(link => link.classList.remove('active'));
        const activeLink = document.querySelector(`.toc a[href="#${entry.target.id}"]`);
        if (activeLink) activeLink.classList.add('active');
      }
    });
  }, { rootMargin: '-20% 0px -80% 0px' });
  
  headings.forEach(h => observer.observe(h));
})();
```

**CSS**:
```css
.toc a.active {
  color: var(--link-color);
  font-weight: bold;
  border-left: 2px solid var(--link-color);
}
```

### 5.7 Code Copy Button

**Purpose**: Add copy button to code blocks

**Output**:
```html
<div class="code-block">
  <div class="code-header">
    <span class="code-lang">go</span>
    <button class="code-copy" aria-label="Copy code">
      <svg>...</svg>
    </button>
  </div>
  <pre><code class="language-go">...</code></pre>
</div>
```

## Configuration

```yaml
advanced:
  mermaid:
    enabled: true
    theme: "default"
    
  graph:
    enabled: true
    path: "/graph/"
    
  theme:
    default: "light"
    allowToggle: true
    
  embeds:
    enabled: true
    providers:
      - youtube
      - vimeo
      - codepen
      - gist
      
  admonitions:
    enabled: true
    types:
      - note
      - tip
      - warning
      - danger
      
  scrollSpy:
    enabled: true
    
  codeCopy:
    enabled: true
```

## Dependencies

```html
<!-- Mermaid -->
<script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>

<!-- D3 for graph -->
<script src="https://d3js.org/d3.v7.min.js"></script>
<!-- or -->
<script src="https://unpkg.com/force-graph"></script>
```

## Deliverables

- [ ] Mermaid diagram support
- [ ] Graph view builder
- [ ] Graph visualization frontend
- [ ] Theme system with toggle
- [ ] Embed transformer
- [ ] Admonition blocks
- [ ] Scroll spy implementation
- [ ] Code copy buttons
- [ ] Dark mode CSS variables
- [ ] Theme persistence (localStorage)
