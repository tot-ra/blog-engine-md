# Phase 6: Polish Specification

## Overview

Final optimizations: incremental builds, link validation, SEO enhancements, performance tuning, and developer experience improvements.

## Goals

1. Incremental builds with change detection
2. Link validation (internal and external)
3. SEO optimization and meta tags
4. Performance profiling and tuning
5. Hot reload development server
6. Build caching optimization

## Dependencies

- Phase 1-5: All previous phases

## Components

### 6.1 Incremental Build System

**Purpose**: Only rebuild changed content

**Interface**:
```go
type IncrementalBuilder interface {
    Build(ctx *BuildContext) error
    GetChanges() (*ChangeSet, error)
    Invalidate(path string) error
}

type ChangeSet struct {
    Added    []ContentFile
    Modified []ContentFile
    Deleted  []string
    Unchanged []ContentFile
}
```

**Change Detection**:
```go
type FileState struct {
    Path         string
    Size         int64
    ModTime      time.Time
    ContentHash  string  // SHA256 of content
}

type BuildState struct {
    Version   string
    Files     map[string]FileState
    Timestamp time.Time
}
```

**State Storage**:
```
.cache/
└── build-state.json
```

**Invalidation Rules**:
| Change Type | Invalidated |
|-------------|-------------|
| Markdown file | File + related pages (prev/next) + index pages + tags |
| Template | All pages |
| Config | Full rebuild |
| CSS/JS | Assets only |
| Image | Image + pages using it |

**Dependency Graph**:
```go
type DependencyGraph struct {
    Files map[string]*FileNode
}

type FileNode struct {
    Path      string
    DependsOn []string  // This file depends on...
    DependedBy []string // Files that depend on this...
}
```

### 6.2 Link Validator

**Purpose**: Validate internal and external links

**Interface**:
```go
type LinkValidator interface {
    Validate(ctx *BuildContext) (*ValidationReport, error)
    ValidateExternal(url string) (bool, error)
}

type ValidationReport struct {
    Internal   []LinkCheck
    External   []LinkCheck
    Summary    ValidationSummary
}

type LinkCheck struct {
    SourceFile string
    LinkText   string
    LinkURL    string
    Status     LinkStatus
    Message    string
}

type LinkStatus int
const (
    LinkOK LinkStatus = iota
    LinkBroken
    LinkWarning
    LinkSkipped
)

type ValidationSummary struct {
    Total      int
    OK         int
    Broken     int
    Warnings   int
    Skipped    int
}
```

**Internal Link Validation**:
- Check all `[text](./path.md)` links
- Verify target file exists
- Check anchor targets (`#section`)
- Report missing files

**External Link Validation**:
```go
// Optional, can be slow
// HTTP HEAD request to check status
// Respect robots.txt
// Rate limiting
```

**CLI Command**:
```bash
blog-engine validate [flags]
  --external    # Check external links (slow)
  --fix         # Auto-fix relative paths
```

### 6.3 SEO Optimizer

**Purpose**: Generate comprehensive SEO metadata

**Interface**:
```go
type SEOOptimizer interface {
    GenerateMeta(page *Page) (*SEOMeta, error)
    GenerateStructuredData(page *Page) (map[string]interface{}, error)
}

type SEOMeta struct {
    Title       string
    Description string
    Canonical   string
    OG          OpenGraph
    Twitter     TwitterCard
    Robots      string
}

type OpenGraph struct {
    Title       string
    Description string
    Type        string
    URL         string
    Image       string
    SiteName    string
}

type TwitterCard struct {
    Card        string  // summary, summary_large_image
    Title       string
    Description string
    Image       string
}
```

**Generated Meta Tags**:
```html
<!-- Basic -->
<title>Page Title | Site Title</title>
<meta name="description" content="Page description...">
<meta name="robots" content="index, follow">
<link rel="canonical" href="https://site.com/page/">

<!-- Open Graph -->
<meta property="og:title" content="Page Title">
<meta property="og:description" content="Page description...">
<meta property="og:type" content="article">
<meta property="og:url" content="https://site.com/page/">
<meta property="og:image" content="https://site.com/img/cover.jpg">
<meta property="og:site_name" content="Site Name">

<!-- Twitter -->
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="Page Title">
<meta name="twitter:description" content="Page description...">
<meta name="twitter:image" content="https://site.com/img/cover.jpg">
```

**Structured Data (JSON-LD)**:
```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "BlogPosting",
  "headline": "Post Title",
  "description": "Post description",
  "author": {
    "@type": "Person",
    "name": "Author Name"
  },
  "datePublished": "2025-01-06",
  "dateModified": "2025-01-06",
  "url": "https://site.com/blog/post/"
}
</script>
```

### 6.4 Performance Profiler

**Purpose**: Profile and report build performance

**Interface**:
```go
type Profiler interface {
    StartPhase(name string)
    EndPhase(name string)
    Report() *PerformanceReport
}

type PerformanceReport struct {
    TotalTime    time.Duration
    Phases       []PhaseReport
    MemoryStats  MemoryStats
    FileStats    FileStats
}

type PhaseReport struct {
    Name     string
    Duration time.Duration
    Percent  float64
}

type MemoryStats struct {
    PeakMB      float64
    CurrentMB   float64
}

type FileStats struct {
    TotalFiles  int
    TotalSizeMB float64
}
```

**CLI Flag**:
```bash
blog-engine build --profile
```

**Output**:
```
Build Performance Report
========================
Total time: 4.23s
Peak memory: 87 MB

Phases:
  Content discovery:  0.12s (3%)
  Markdown parsing:   1.45s (34%)
  Image processing:   1.89s (45%)
  HTML rendering:     0.56s (13%)
  Asset copying:      0.21s (5%)

Files processed:
  Markdown: 245 files (12 MB)
  Images:   156 files (45 MB)
  Assets:   23 files (2 MB)
```

### 6.5 Development Server

**Purpose**: Hot reload development server

**Interface**:
```go
type DevServer interface {
    Start(port int) error
    Stop() error
    Watch(paths []string) error
}
```

**Features**:
- HTTP server on configurable port
- File watching with fsnotify
- Incremental rebuild on change
- Live reload via WebSocket
- Error overlay in browser

**CLI Command**:
```bash
blog-engine serve [flags]
  -p, --port int       Port to serve on (default 3000)
  -h, --host string    Host to bind to (default "localhost")
      --no-reload      Disable live reload
```

**Live Reload**:
```html
<!-- Injected into HTML during dev -->
<script>
  const ws = new WebSocket('ws://localhost:3000/__livereload');
  ws.onmessage = (event) => {
    if (event.data === 'reload') location.reload();
  };
</script>
```

**Watch Patterns**:
```go
watchPaths := []string{
    "content/**",
    "templates/**",
    "static/**",
    "config.yaml",
}
ignorePaths := []string{
    "**/.git/**",
    "**/node_modules/**",
    "**/.cache/**",
}
```

### 6.6 Build Cache Optimizer

**Purpose**: Optimize cache storage and retrieval

**Interface**:
```go
type CacheOptimizer interface {
    Cleanup(maxAge time.Duration) error
    Compact() error
    Stats() (*CacheStats, error)
}

type CacheStats struct {
    Entries      int
    SizeMB       float64
    HitRate      float64
    MissRate     float64
}
```

**Cache Strategies**:

**Image Cache**:
- Key: `sha256(file_path + file_mtime + config)`
- Value: Processed image variants
- TTL: 30 days

**HTML Cache**:
- Key: `sha256(source_hash + template_hash + config_hash)`
- Value: Rendered HTML
- TTL: 7 days

**Cache Cleanup**:
```bash
blog-engine cache clean      # Remove expired entries
blog-engine cache clear      # Clear all cache
blog-engine cache stats      # Show cache statistics
```

### 6.7 Error Reporter

**Purpose**: Better error messages and diagnostics

**Interface**:
```go
type ErrorReporter interface {
    Report(err BuildError) string
    ReportSummary(errors []BuildError) string
}

type BuildError struct {
    Type     ErrorType
    File     string
    Line     int
    Column   int
    Message  string
    Suggestion string
    Context  string  // Surrounding lines
}
```

**Error Types**:
- `ParseError` - Markdown/frontmatter parse failure
- `TemplateError` - Template execution error
- `LinkError` - Broken internal link
- `AssetError` - Image/asset processing error
- `ConfigError` - Configuration error

**Error Output**:
```
ERROR: Parse error in blog/post.md:15:23

  14 | ---
  15 | date: 2025-13-45
     |       ^^^^^^^^^^
     |       Invalid date format

Expected format: YYYY-MM-DD or ISO8601
Suggestion: Change to: 2025-01-15
```

## Configuration

```yaml
build:
  incremental: true
  cache:
    enabled: true
    directory: ".cache"
    maxSizeMB: 500
    maxAge: "30d"
    
  validation:
    enabled: true
    checkExternal: false
    failOnBroken: false
    
  performance:
    profile: false
    parallelWorkers: 4
    
  devServer:
    port: 3000
    host: "localhost"
    liveReload: true
    openBrowser: false

seo:
  defaults:
    titleTemplate: "%s | Site Name"
    description: "Default site description"
    image: "/img/og-default.jpg"
  twitter:
    site: "@username"
    creator: "@username"
```

## Performance Targets

| Metric | Cold Build | Incremental |
|--------|------------|-------------|
| 100 pages | < 5s | < 1s |
| 1000 pages | < 30s | < 3s |
| Memory | < 200MB | < 100MB |
| Cache hit | - | > 90% |

## Deliverables

- [ ] Incremental build system
- [ ] Change detection with dependency graph
- [ ] Link validator (internal + external)
- [ ] SEO optimizer with structured data
- [ ] Performance profiler
- [ ] Development server with hot reload
- [ ] Cache optimizer and cleanup
- [ ] Enhanced error reporting
- [ ] Build state persistence
- [ ] CLI commands: `serve`, `validate`, `cache`

## Testing

### Performance Tests
- Benchmark cold build
- Benchmark incremental build
- Memory profiling
- Cache hit rate measurement

### Integration Tests
- Full workflow: edit → rebuild → verify
- Link validation accuracy
- SEO output validation

## Future Considerations

- Distributed builds (for very large sites)
- Cloud storage for cache
- Build analytics dashboard
- Plugin system for extensions
