# Phase 1: Core (MVP) Specification

## Overview

Minimum viable product providing basic markdown-to-HTML conversion with folder-based URL structure.

## Goals

1. Parse markdown files with YAML frontmatter
2. Convert markdown to HTML
3. Generate clean URL structure from folder hierarchy
4. Apply basic HTML templates
5. Copy static assets

## Non-Goals

- Navigation menus (Phase 2)
- Image optimization (Phase 3)
- Search, RSS, tags (Phase 4)
- Diagrams, graph view (Phase 5)
- Incremental builds (Phase 6)

## Components

### 1.1 Config Parser

**Purpose**: Load and validate site configuration

**Interface**:
```go
type ConfigParser interface {
    Load(path string) (*SiteConfig, error)
    Validate(cfg *SiteConfig) error
}
```

**Requirements**:
- Support YAML format
- Validate required fields (site.title, site.url)
- Apply defaults for optional fields
- Return descriptive errors for invalid config

**Test Cases**:
- Valid config loads successfully
- Missing required field returns error
- Invalid YAML returns parse error
- Defaults applied correctly

### 1.2 Content Discovery

**Purpose**: Find and catalog all content files

**Interface**:
```go
type ContentDiscovery interface {
    Scan(root string) (*ContentIndex, error)
}

type ContentIndex struct {
    MarkdownFiles []ContentFile
    AssetFiles    []ContentFile
    ImageFiles    []ContentFile
}
```

**Requirements**:
- Recursively scan content directory
- Classify files by type (.md, images, other assets)
- Skip hidden files and directories (starting with .)
- Respect .gitignore patterns if present
- Collect file metadata (size, mtime)

**File Type Detection**:
| Extension | Type |
|-----------|------|
| .md, .markdown | markdown |
| .jpg, .jpeg, .png, .gif, .svg, .webp | image |
| * | asset |

### 1.3 Frontmatter Parser

**Purpose**: Extract YAML frontmatter from markdown files

**Interface**:
```go
type FrontmatterParser interface {
    Parse(content string) (*Frontmatter, string, error)
}
```

**Format**:
```markdown
---
title: Article Title
date: 2024-01-15T10:00:00Z
tags: [tag1, tag2]
---

Content here...
```

**Requirements**:
- Extract frontmatter between `---` delimiters
- Parse YAML into Frontmatter struct
- Return remaining content (after frontmatter)
- Support all Frontmatter fields from shared schema
- Generate slug from title if not provided
- Parse date in multiple formats (ISO8601, YYYY-MM-DD)

**Slug Generation Rules**:
1. Use `slug` frontmatter if provided
2. Use filename without extension (transliterate Cyrillic)
3. Lowercase, replace spaces with hyphens
4. Remove special characters

### 1.4 Markdown Parser

**Purpose**: Convert markdown to HTML AST

**Interface**:
```go
type MarkdownParser interface {
    Parse(content string) (ast.Node, error)
    RenderToHTML(node ast.Node) (string, error)
}
```

**Requirements**:
- Support CommonMark specification
- Support GitHub Flavored Markdown (GFM):
  - Tables
  - Task lists
  - Strikethrough
  - Autolinks
- Syntax highlighting for code blocks (generate class names)
- Extract headings for TOC (store separately, don't render yet)

**Supported Syntax**:
```markdown
# Heading 1
## Heading 2

**bold**, *italic*, ~~strikethrough~~

- list item
- [ ] task unchecked
- [x] task checked

| Table | Col |
|-------|-----|
| data  | val |

```go
// code block with language
func main() {}
```

https://autolinks.com
```

### 1.5 URL Generator

**Purpose**: Generate clean URLs from file paths

**Interface**:
```go
type URLGenerator interface {
    Generate(filePath string, frontmatter *Frontmatter) (string, error)
}
```

**URL Patterns**:
| Source Path | Output URL |
|-------------|------------|
| `blog/tech/article.md` | `/blog/tech/article/` |
| `docs/about.md` | `/docs/about/` |
| `blog/2025-05-07 Event.md` | `/blog/2025/05/07/event/` |

**Requirements**:
- Remove file extension
- Add trailing slash
- Handle date prefixes in filenames (extract to URL)
- Support custom slug from frontmatter
- Handle index.md as directory root

**Special Cases**:
- `index.md` → `/path/to/dir/`
- `README.md` → `/path/to/dir/` (treated as index)

### 1.6 Template Engine

**Purpose**: Render HTML pages using templates

**Interface**:
```go
type TemplateEngine interface {
    LoadTemplates(dir string) error
    Render(template string, data PageData) (string, error)
}

type PageData struct {
    Site     SiteConfig
    Page     Page
    Content  template.HTML
}
```

**Required Templates**:
- `base.html` - Main layout with HTML skeleton
- `page.html` - Single page content
- `list.html` - List of pages (for index pages)

**Template Structure**:
```html
<!-- base.html -->
<!DOCTYPE html>
<html lang="{{.Site.Language}}">
<head>
    <meta charset="UTF-8">
    <title>{{.Page.Title}} | {{.Site.Title}}</title>
    <meta name="description" content="{{.Page.Description}}">
</head>
<body>
    <header>{{.Site.Title}}</header>
    <main>{{template "content" .}}</main>
    <footer>© {{.Site.Author.Name}}</footer>
</body>
</html>
```

**Requirements**:
- Use Go html/template
- Support template inheritance (define blocks)
- Auto-escape HTML for security
- Provide helper functions (formatDate, slugify)

### 1.7 Page Builder

**Purpose**: Orchestrate page generation

**Interface**:
```go
type PageBuilder interface {
    Build(ctx *BuildContext, file ContentFile) (*Page, error)
}
```

**Build Process**:
1. Read file content
2. Parse frontmatter
3. Parse markdown to AST
4. Render markdown to HTML
5. Extract headings for TOC
6. Generate URL
7. Create Page struct

**Error Handling**:
- Log error and skip file on parse failure
- Continue building other pages
- Return list of errors at end

### 1.8 Static Asset Copier

**Purpose**: Copy static files to output directory

**Interface**:
```go
type AssetCopier interface {
    Copy(ctx *BuildContext, files []ContentFile) error
}
```

**Requirements**:
- Preserve directory structure
- Skip unchanged files (compare mtime)
- Create directories as needed
- Handle symlinks (copy target)

### 1.9 Site Builder

**Purpose**: Main orchestrator for the build process

**Interface**:
```go
type SiteBuilder interface {
    Build(config *SiteConfig) error
}
```

**Build Pipeline**:
```
1. Load configuration
2. Discover content files
3. For each markdown file:
   a. Parse frontmatter
   b. Parse markdown
   c. Render HTML
   d. Generate page
4. For each page:
   a. Apply template
   b. Write to output directory
5. Copy static assets
6. Report results
```

**Output Structure**:
```
dist/
├── index.html                    # Root index (if exists)
├── blog/
│   ├── index.html               # Blog index
│   └── tech/
│       └── article/
│           └── index.html
├── docs/
│   └── about/
│       └── index.html
└── assets/                      # Copied static files
    └── img/
        └── logo.png
```

## CLI Commands

### build

```
blog-engine build [flags]

Flags:
  -c, --config string    Config file path (default "config.yaml")
  -o, --output string    Output directory (overrides config)
      --clean           Clean output before build
```

**Exit Codes**:
- 0: Success
- 1: Config error
- 2: Build error
- 3: Write error

## Performance Targets

- Parse 1000 markdown files: < 2 seconds
- Total build time (cold): < 10 seconds
- Memory usage: < 100MB for 1000 files

## Dependencies

```go
require (
    github.com/yuin/goldmark v1.7.0
    github.com/yuin/goldmark-meta v1.1.0
    gopkg.in/yaml.v3 v3.0.1
)
```

## Testing Strategy

### Unit Tests
- Config parser with various inputs
- Frontmatter extraction edge cases
- Markdown rendering output
- URL generation rules
- Template rendering

### Integration Tests
- Full build with sample content
- Verify output file structure
- Check HTML validity
- Compare with expected output

### Test Data
```
testdata/
├── simple-blog/
│   ├── config.yaml
│   ├── blog/
│   │   └── post.md
│   └── static/
│       └── style.css
└── complex-structure/
    └── ...
```

## Deliverables

- [ ] Config parser implementation
- [ ] Content discovery module
- [ ] Frontmatter parser
- [ ] Markdown parser with GFM
- [ ] URL generator
- [ ] Template engine
- [ ] Page builder
- [ ] Asset copier
- [ ] Site builder (orchestrator)
- [ ] `build` CLI command
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests with testdata
