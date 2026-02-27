# Phase 4: Features Specification

## Overview

Add core features: full-text search, RSS/Atom feeds, sitemap, tag system, and archive pages.

## Goals

1. Full-text search index with Bleve
2. RSS and Atom feed generation
3. XML sitemap for SEO
4. Tag system with tag pages
5. Archive pages by date
6. Pagination for lists

## Dependencies

- Phase 1: Core (MVP)
- Phase 2: Navigation
- Phase 3: Assets

## Components

### 4.1 Search Index Builder

**Purpose**: Build full-text search index

**Interface**:
```go
type SearchIndexBuilder interface {
    Build(ctx *BuildContext) (SearchIndex, error)
    AddDocument(doc SearchDocument) error
    Save(path string) error
}

type SearchIndex interface {
    Query(q string) ([]SearchResult, error)
}

type SearchDocument struct {
    ID          string
    Title       string
    Content     string
    URL         string
    Tags        []string
    Type        string
    Date        time.Time
}

type SearchResult struct {
    ID          string
    Title       string
    URL         string
    Excerpt     string
    Score       float64
}
```

**Indexed Fields**:
| Field | Type | Boost | Store |
|-------|------|-------|-------|
| title | text | 3.0 | yes |
| content | text | 1.0 | no |
| tags | keyword | 2.0 | yes |
| url | keyword | - | yes |
| type | keyword | - | yes |
| date | datetime | - | yes |

**Search Features**:
- Full-text search with stemming
- Fuzzy matching (typo tolerance)
- Boolean operators (AND, OR, NOT)
- Phrase search with quotes
- Filter by type, tags, date range

**Output**:
```
dist/
└── search/
    └── index.bleve/      # Bleve index directory
```

### 4.2 RSS/Atom Generator

**Purpose**: Generate RSS and Atom feeds

**Interface**:
```go
type FeedGenerator interface {
    GenerateRSS(ctx *BuildContext) (string, error)
    GenerateAtom(ctx *BuildContext) (string, error)
    GenerateCategoryRSS(ctx *BuildContext, category string) (string, error)
}
```

**RSS Structure**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>Site Title</title>
    <link>https://site.com</link>
    <description>Site description</description>
    <language>ru</language>
    <lastBuildDate>Mon, 06 Jan 2025 12:00:00 GMT</lastBuildDate>
    <atom:link href="https://site.com/rss.xml" rel="self" type="application/rss+xml"/>
    <item>
      <title>Post Title</title>
      <link>https://site.com/blog/post/</link>
      <guid isPermaLink="true">https://site.com/blog/post/</guid>
      <pubDate>Mon, 06 Jan 2025 10:00:00 GMT</pubDate>
      <description><![CDATA[HTML content or excerpt]]></description>
      <category>tech</category>
    </item>
  </channel>
</rss>
```

**Feed Types**:
- Main feed (all blog posts)
- Category feeds (per category)
- Tag feeds (per tag)

**Configuration**:
```yaml
feeds:
  rss:
    enabled: true
    path: "rss.xml"
    items: 20
    fullContent: false  # true = full HTML, false = excerpt
  atom:
    enabled: true
    path: "atom.xml"
    items: 20
```

### 4.3 Sitemap Generator

**Purpose**: Generate XML sitemap for SEO

**Interface**:
```go
type SitemapGenerator interface {
    Generate(ctx *BuildContext) (string, error)
    GenerateIndex(ctx *BuildContext) (string, error)
}

type SitemapEntry struct {
    URL        string
    LastMod    time.Time
    ChangeFreq string  // always, hourly, daily, weekly, monthly, yearly, never
    Priority   float64 // 0.0 - 1.0
}
```

**Sitemap Structure**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://site.com/</loc>
    <lastmod>2025-01-06</lastmod>
    <changefreq>weekly</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>https://site.com/blog/post/</loc>
    <lastmod>2025-01-06</lastmod>
    <changefreq>monthly</changefreq>
    <priority>0.8</priority>
  </url>
</urlset>
```

**Priority Rules**:
| Page Type | Priority |
|-----------|----------|
| Homepage | 1.0 |
| Blog index | 0.9 |
| Blog posts | 0.8 |
| Docs pages | 0.7 |
| Tag pages | 0.5 |
| Archive pages | 0.3 |

**Change Frequency**:
| Page Type | ChangeFreq |
|-----------|------------|
| Homepage | weekly |
| Blog posts | never (static) |
| Docs | monthly |

### 4.4 Tag System

**Purpose**: Manage tags and generate tag pages

**Interface**:
```go
type TagSystem interface {
    ExtractTags(ctx *BuildContext) (TagIndex, error)
    GenerateTagPages(ctx *BuildContext, index TagIndex) ([]*Page, error)
    GenerateTagCloud(ctx *BuildContext, index TagIndex) (*Page, error)
}

type TagIndex map[string][]string  // tag -> page IDs

type TagPageData struct {
    Tag       string
    Pages     []PageSummary
    PageCount int
}

type PageSummary struct {
    Title       string
    URL         string
    Date        time.Time
    Description string
    Tags        []string
}
```

**Tag Extraction**:
- From `tags` frontmatter field
- Normalize: lowercase, trim spaces
- Support Cyrillic tags

**Tag Page URL**:
```
/tags/backend/          -> Tag page for "backend"
/tags/                  -> Tag cloud/index
```

**Tag Cloud**:
- Visual representation of all tags
- Size based on frequency
- Alphabetical or frequency sort

### 4.5 Archive Generator

**Purpose**: Generate archive pages by date

**Interface**:
```go
type ArchiveGenerator interface {
    Generate(ctx *BuildContext) (*ArchiveTree, error)
    GeneratePages(ctx *BuildContext, tree *ArchiveTree) ([]*Page, error)
}

type ArchiveTree struct {
    Years map[int]*YearArchive
}

type YearArchive struct {
    Year   int
    Months map[int]*MonthArchive
    Count  int
}

type MonthArchive struct {
    Year  int
    Month int
    Pages []PageSummary
    Count int
}
```

**Archive URL Structure**:
```
/archive/               -> All years
/archive/2025/          -> Year archive
/archive/2025/01/       -> Month archive
```

**Archive Page Content**:
- Group posts by year/month
- Show count per period
- Link to individual posts
- Chronological order (newest first)

### 4.6 Pagination

**Purpose**: Paginate long lists

**Interface**:
```go
type Paginator interface {
    Paginate(items []PageSummary, pageSize int) ([]PageGroup, error)
    GeneratePages(baseURL string, groups []PageGroup) ([]*Page, error)
}

type PageGroup struct {
    Number      int
    Items       []PageSummary
    TotalItems  int
    TotalPages  int
    PrevPage    *int
    NextPage    *int
    FirstURL    string
    LastURL     string
}
```

**Pagination URL Pattern**:
```
/blog/              -> Page 1
/blog/page/2/       -> Page 2
/blog/page/3/       -> Page 3
```

**Pagination Component**:
```html
<nav class="pagination" aria-label="Page navigation">
  <a href="/blog/" class="first">First</a>
  <a href="/blog/" class="prev">← Previous</a>
  
  <span class="pages">
    <a href="/blog/">1</a>
    <span class="current" aria-current="page">2</span>
    <a href="/blog/page/3/">3</a>
    <span class="ellipsis">...</span>
    <a href="/blog/page/10/">10</a>
  </span>
  
  <a href="/blog/page/3/" class="next">Next →</a>
  <a href="/blog/page/10/" class="last">Last</a>
</nav>
```

**Configuration**:
```yaml
pagination:
  pageSize: 10
  maxPages: 5        # Max page numbers to show
  prevNext: true     # Show prev/next links
  firstLast: true    # Show first/last links
```

### 4.7 Robots.txt Generator

**Purpose**: Generate robots.txt

**Interface**:
```go
type RobotsGenerator interface {
    Generate(ctx *BuildContext) (string, error)
}
```

**Output**:
```
User-agent: *
Allow: /
Disallow: /assets/

Sitemap: https://site.com/sitemap.xml
```

## Configuration

```yaml
features:
  search:
    enabled: true
    path: "/search/"
    placeholder: "Search..."
    minQueryLength: 2
    maxResults: 10
    
  feeds:
    rss:
      enabled: true
      path: "rss.xml"
      items: 20
      fullContent: false
    atom:
      enabled: true
      path: "atom.xml"
      items: 20
      
  sitemap:
    enabled: true
    path: "sitemap.xml"
    
  tags:
    enabled: true
    path: "/tags/"
    cloud: true
    
  archive:
    enabled: true
    path: "/archive/"
    
  pagination:
    enabled: true
    pageSize: 10
```

## Output Structure

```
dist/
├── search/
│   └── index.bleve/
├── rss.xml
├── atom.xml
├── sitemap.xml
├── robots.txt
├── tags/
│   ├── index.html
│   ├── backend/
│   │   └── index.html
│   └── frontend/
│       └── index.html
└── archive/
    ├── index.html
    ├── 2025/
    │   ├── index.html
    │   └── 01/
    │       └── index.html
    └── 2024/
        └── index.html
```

## Frontend Search

**Search UI**:
```html
<form class="search-form" action="/search/" method="get">
  <input type="search" name="q" placeholder="Search..." minlength="2">
  <button type="submit">Search</button>
</form>

<div class="search-results">
  <p class="search-stats">Found 5 results for "golang"</p>
  <ul>
    <li>
      <a href="/blog/tech/go-guide/">
        <h3>Go Guide <span class="score">95%</span></h3>
        <p class="excerpt">...golang is a great language...</p>
      </a>
    </li>
  </ul>
</div>
```

**Search JS** (minimal):
```javascript
// Simple fetch to search endpoint
// Display results without page reload
```

## Testing

### Unit Tests
- Search index building
- Query parsing and results
- Feed generation
- Sitemap generation
- Tag extraction
- Pagination logic

### Integration Tests
- Full search flow
- Feed validation (RSS/Atom validators)
- Sitemap validation

## Deliverables

- [ ] Search index builder (Bleve)
- [ ] Search query handler
- [ ] RSS generator
- [ ] Atom generator
- [ ] Sitemap generator
- [ ] Tag system
- [ ] Archive generator
- [ ] Pagination system
- [ ] Robots.txt generator
- [ ] Search UI component
- [ ] Feed auto-discovery in HTML
