# Phase 2: Navigation Specification

## Overview

Add navigation capabilities including sidebar menus, table of contents, breadcrumbs, and previous/next post links.

## Goals

1. Generate hierarchical sidebar menu from folder structure
2. Extract and render table of contents from headings
3. Generate breadcrumb navigation
4. Add previous/next navigation for sequential content
5. Support ordering and hiding pages

## Dependencies

- Phase 1: Core (MVP)

## Components

### 2.1 Navigation Tree Builder

**Purpose**: Build hierarchical navigation structure from content

**Interface**:
```go
type NavigationBuilder interface {
    BuildTree(ctx *BuildContext) (*NavTree, error)
    BuildForSection(ctx *BuildContext, section string) (*NavTree, error)
}

type NavTree struct {
    Root     *NavNode
    ByPath   map[string]*NavNode  // Quick lookup by URL path
}

type NavNode struct {
    ID       string
    Title    string
    URL      string
    Children []*NavNode
    Parent   *NavNode
    Order    int
    Hidden   bool
    Type     string  // "section" | "page"
}
```

**Tree Structure Rules**:
- Root nodes = top-level directories (blog/, docs/)
- Branch nodes = subdirectories
- Leaf nodes = markdown files
- Order by: `order` frontmatter → filename (alphabetical)

**Example Tree**:
```
NavTree
├── blog/
│   ├── tech/
│   │   ├── backend/
│   │   │   └── golang-cheatsheet (order: 1)
│   │   └── frontend/
│   └── philosophy/
└── docs/
    ├── about-author (order: 1)
    └── work-experience/
        └── pipedrive (order: 2)
```

**Requirements**:
- Respect `order` frontmatter for sorting
- Skip nodes with `hideNav: true`
- Generate titles from frontmatter or filename
- Support nested levels (unlimited depth)
- Cache tree for reuse

### 2.2 Sidebar Renderer

**Purpose**: Generate sidebar HTML from navigation tree

**Interface**:
```go
type SidebarRenderer interface {
    Render(tree *NavTree, currentPath string) (template.HTML, error)
    RenderForSection(tree *NavTree, section string, currentPath string) (template.HTML, error)
}
```

**Sidebar Structure**:
```html
<nav class="sidebar">
  <ul class="sidebar-menu">
    <li class="sidebar-item sidebar-section">
      <a href="/blog/">Блог</a>
      <ul class="sidebar-submenu">
        <li class="sidebar-item sidebar-section">
          <a href="/blog/tech/">Tech</a>
          <ul class="sidebar-submenu">
            <li class="sidebar-item active">
              <a href="/blog/tech/backend/golang/" aria-current="page">Golang</a>
            </li>
          </ul>
        </li>
      </ul>
    </li>
  </ul>
</nav>
```

**Styling Classes**:
| Class | Description |
|-------|-------------|
| `sidebar` | Root nav element |
| `sidebar-menu` | Top-level list |
| `sidebar-submenu` | Nested list |
| `sidebar-item` | List item |
| `sidebar-section` | Directory node |
| `active` | Current page or ancestor |
| `expanded` | Section is expanded |

**Behavior**:
- Expand all ancestor sections of current page
- Highlight current page with `active` class
- Collapse sibling sections (optional, configurable)
- Support custom section titles via `_category_.md` or frontmatter

### 2.3 TOC Extractor

**Purpose**: Extract table of contents from markdown headings

**Interface**:
```go
type TOCExtractor interface {
    Extract(astNode ast.Node) ([]TocItem, error)
    Render(items []TocItem) (template.HTML, error)
}
```

**Extraction Rules**:
- Extract H2-H4 headings only (skip H1 as it's usually the title)
- Generate anchor IDs from heading text (slugify)
- Maintain hierarchy (H2 → H3 → H4)
- Skip headings inside code blocks

**Anchor Generation**:
```go
// "Variables and Types" → "variables-and-types"
// "Привет мир" → "privet-mir" (transliterate)
func Slugify(text string) string
```

**TOC Output**:
```html
<nav class="toc" aria-label="Table of contents">
  <h2 class="toc-title">On this page</h2>
  <ul class="toc-list">
    <li class="toc-item">
      <a href="#variables-and-types">Variables and Types</a>
      <ul class="toc-sublist">
        <li class="toc-item">
          <a href="#type-inference">Type Inference</a>
        </li>
      </ul>
    </li>
  </ul>
</nav>
```

**Requirements**:
- Sticky positioning via CSS
- Scroll spy (highlight current section) - Phase 5
- Collapsible on mobile
- Respect `hideToc: true` frontmatter

### 2.4 Breadcrumb Generator

**Purpose**: Generate breadcrumb navigation from page path

**Interface**:
```go
type BreadcrumbGenerator interface {
    Generate(ctx *BuildContext, page *Page) ([]BreadcrumbItem, error)
}

type BreadcrumbItem struct {
    Title string
    URL   string
    IsCurrent bool
}
```

**Generation Rules**:
1. Always start with Home → `/`
2. Add section root (Blog/Docs) → `/blog/` or `/docs/`
3. Add intermediate directories
4. Add current page (no URL, marked as current)

**Example**:
| Page URL | Breadcrumbs |
|----------|-------------|
| `/blog/tech/backend/golang/` | Home > Blog > Tech > Backend > Golang |
| `/docs/about/` | Home > Docs > About Author |

**Breadcrumb Schema** (JSON-LD):
```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "BreadcrumbList",
  "itemListElement": [
    {"@type": "ListItem", "position": 1, "name": "Home", "item": "https://site.com/"},
    {"@type": "ListItem", "position": 2, "name": "Blog", "item": "https://site.com/blog/"}
  ]
}
</script>
```

### 2.5 Prev/Next Navigator

**Purpose**: Generate previous/next links for sequential navigation

**Interface**:
```go
type PrevNextNavigator interface {
    Generate(ctx *BuildContext, page *Page) (*PrevNextLinks, error)
}

type PrevNextLinks struct {
    Prev *NavLink
    Next *NavLink
}

type NavLink struct {
    Title string
    URL   string
    Type  string  // "blog" | "doc"
}
```

**Navigation Logic**:

**Blog Posts**:
- Sort by date (newest first)
- Within same category if possible
- Cross-category navigation as fallback

**Docs**:
- Follow navigation tree order
- Depth-first traversal
- Respect `order` frontmatter

**Example**:
```
Blog order (by date):
1. /blog/tech/2025-05-01-post1/
2. /blog/tech/2025-04-15-post2/
3. /blog/philosophy/2025-04-10-post3/

Current: post2
Prev: post1 (same category)
Next: post3 (different category)
```

**Rendering**:
```html
<nav class="prev-next" aria-label="Page navigation">
  <a href="/blog/tech/post1/" class="prev-link">
    <span class="prev-label">← Previous</span>
    <span class="prev-title">Post 1 Title</span>
  </a>
  <a href="/blog/philosophy/post3/" class="next-link">
    <span class="next-label">Next →</span>
    <span class="next-title">Post 3 Title</span>
  </a>
</nav>
```

### 2.6 Section Index Generator

**Purpose**: Generate index pages for sections (blog/, docs/tech/)

**Interface**:
```go
type SectionIndexGenerator interface {
    Generate(ctx *BuildContext, section string) (*Page, error)
}
```

**Index Page Content**:
- List all direct children (pages and subsections)
- Show title, description, date (for blog)
- Link to each child
- Auto-generated if no index.md present

**Data for Template**:
```go
type SectionIndexData struct {
    Title       string
    Description string
    Children    []SectionChild
}

type SectionChild struct {
    Title       string
    URL         string
    Description string
    Date        *time.Time
    IsSection   bool
}
```

## Template Updates

### Updated PageData
```go
type PageData struct {
    Site        SiteConfig
    Page        Page
    Content     template.HTML
    // New fields:
    Sidebar     template.HTML
    TOC         template.HTML
    Breadcrumbs []BreadcrumbItem
    PrevNext    *PrevNextLinks
}
```

### New Template Blocks
```html
<!-- sidebar block -->
{{define "sidebar"}}
  {{.Sidebar}}
{{end}}

<!-- toc block -->
{{define "toc"}}
  {{.TOC}}
{{end}}

<!-- breadcrumbs block -->
{{define "breadcrumbs"}}
  <nav aria-label="Breadcrumb">
    <ol class="breadcrumbs">
      {{range .Breadcrumbs}}
        <li>
          {{if .IsCurrent}}
            <span aria-current="page">{{.Title}}</span>
          {{else}}
            <a href="{{.URL}}">{{.Title}}</a>
          {{end}}
        </li>
      {{end}}
    </ol>
  </nav>
{{end}}

<!-- prev-next block -->
{{define "prev-next"}}
  {{if .PrevNext}}
    <nav class="prev-next">
      {{if .PrevNext.Prev}}
        <a href="{{.PrevNext.Prev.URL}}" class="prev">
          ← {{.PrevNext.Prev.Title}}
        </a>
      {{end}}
      {{if .PrevNext.Next}}
        <a href="{{.PrevNext.Next.URL}}" class="next">
          {{.PrevNext.Next.Title}} →
        </a>
      {{end}}
    </nav>
  {{end}}
{{end}}
```

## Configuration Additions

```yaml
navigation:
  sidebar:
    collapsed: false        # Start with collapsed sections
    maxDepth: 3             # Maximum nesting level to show
    includeIndex: true      # Show index pages in menu
  
  toc:
    minDepth: 2             # Minimum heading level (H2)
    maxDepth: 4             # Maximum heading level (H4)
    sticky: true            # Sticky positioning
  
  breadcrumbs:
    enabled: true
    homeLabel: "Home"
  
  prevNext:
    enabled: true
    sameCategoryOnly: false  # For blog posts
```

## Styling Requirements

### CSS Variables
```css
:root {
  --sidebar-width: 280px;
  --toc-width: 240px;
  --nav-bg: var(--bg-secondary);
  --nav-border: var(--border-color);
  --nav-active: var(--primary-color);
}
```

### Responsive Behavior
| Breakpoint | Layout |
|------------|--------|
| > 1200px | Sidebar + Content + TOC |
| 768-1200px | Sidebar + Content (collapsible) |
| < 768px | Content only (hamburger menu) |

## Testing

### Unit Tests
- Navigation tree building
- TOC extraction from various heading structures
- Breadcrumb generation
- Prev/next link calculation

### Integration Tests
- Full page render with navigation
- Mobile viewport behavior
- Keyboard navigation accessibility

### Accessibility Tests
- ARIA labels present
- Keyboard navigable
- Screen reader friendly
- Focus indicators visible

## Deliverables

- [ ] Navigation tree builder
- [ ] Sidebar renderer
- [ ] TOC extractor and renderer
- [ ] Breadcrumb generator
- [ ] Prev/next navigator
- [ ] Section index generator
- [ ] Updated templates with navigation blocks
- [ ] CSS for navigation components
- [ ] Responsive navigation behavior
- [ ] Accessibility compliance (WCAG 2.1 AA)
