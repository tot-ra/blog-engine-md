# Blog Engine MD

A high-performance static site generator written in Go, designed as a memory-efficient alternative to [Docusaurus](https://docusaurus.io/) or [Hugo](https://gohugo.io/) for markdown-based blogs and documentation in [Obsidian](https://obsidian.md/) style

<img width="1500" height="843" alt="blog" src="https://github.com/user-attachments/assets/45049191-d90b-491f-828a-c8d9cfd1b310" />

## Overview

This project converts a nested folder structure of Markdown files into a static website, similar to Docusaurus but with significantly lower memory footprint and faster build times.

## Core Requirements

### Content Structure

The engine supports two primary content types:

1. **Blog Posts** (`blog/`)
   - Date-dependent entries organized by category/subfolder
   - Support for `<!--truncate-->` tag for article previews
   - Previous/next post navigation within categories
   - Archive pages by date

2. **Documentation/Pages** (`docs/`)
   - Permanent content (about, projects, etc.)
   - Hierarchical navigation menu based on folder structure
   - No date dependency

### File Organization

```
content/
├── blog/                          # Blog posts
│   ├── tech/
│   │   ├── backend/
│   │   │   └── Шпаргалка по golang.md
│   │   └── frontend/
│   ├── события/
│   │   └── 2025-05-07 AWS Gen AI meetup.md
│   └── философия/
│       └── Эволюция разума.md
├── docs/                          # Permanent pages
│   ├── Об авторе.md
│   ├── Опыт работы/
│   │   └── Pipedrive.md
│   └── Музыка/
│       └── Легкие пустыни.md
├── static/                        # Static assets
│   └── img/
└── config.yaml                    # Site configuration
```

## Features Specification

### 1. Markdown Processing

#### Supported Syntax
- Standard CommonMark/GFM
- Tables (GFM style)
- Task lists
- Strikethrough
- Autolinks
- Footnotes

#### Extended Features
- **YAML Frontmatter**: Metadata support (title, date, tags, description, draft, `order` / Docusaurus `sidebar_position`)
- **Custom Directives**:
  - `<!--truncate-->`: Article preview cutoff point
  - Mermaid diagrams (```mermaid blocks)
  - Custom blocks (:::info, :::warning, :::tip)

#### Cross-Reference Resolution
- Internal markdown links `[text](./other-file.md)` → converted to HTML paths
- Automatic validation of internal links
- Support for relative path resolution across folder boundaries

### 2. Image Processing

#### Optimization Pipeline
- Source formats: JPG, PNG, GIF, SVG
- Target format: WebP (with fallback)
- Responsive images: Generate multiple sizes (thumbnail, preview, full)
- Lazy loading attributes
- Alt text extraction from markdown

#### Processing Strategy
- Parallel processing with worker pool
- Skip unchanged files (mtime comparison)
- Configurable quality settings
- AVIF support (optional, future)

### 3. URL Structure & SEO

#### URL Patterns
```
/blog/<category>/<slug>/              # Blog posts
/docs/<path>/<slug>/                  # Documentation
/tags/<tag>/                          # Tag pages
/archive/<year>/<month>/              # Archive pages
/page/<n>/                            # Pagination
```

#### SEO Features
- Semantic HTML5 structure
- Meta tags (description, keywords, og:*)
- Structured data (JSON-LD)
- Canonical URLs
- XML Sitemap generation
- RSS/Atom feeds
- robots.txt

### 4. Navigation & Layout

#### Page Structure
```
┌─────────────────────────────────────────────────────────────┐
│  Logo    [Home] [Blog] [Docs] [Tags]    [Search]  [Theme]   │  Header
├──────────┬──────────────────────────────────────────────────┤
│          │  Breadcrumbs: Home > Blog > Tech > Backend       │
│  Sidebar │                                                  │
│  Menu    │  ┌────────────────────────────────────────────┐  │
│  (Tree)  │  │                                            │  │
│          │  │           Article Content                  │  │
│          │  │                                            │  │
│          │  └────────────────────────────────────────────┘  │
│          │              ┌───────────────────┐               │
│          │              │   Table of        │               │
│          │              │   Contents        │               │
│          │              │   (Sticky)        │               │
│          │              └───────────────────┘               │
│          │  ┌────────────────────────────────────────────┐  │
│          │  │  Previous Post    |    Next Post           │  │
│          │  └────────────────────────────────────────────┘  │
├──────────┴──────────────────────────────────────────────────┤
│                        Footer                               │
└─────────────────────────────────────────────────────────────┘
```

#### Navigation Components
- **Top Navigation**: Main sections, search, theme toggle
- **Left Sidebar**: Hierarchical menu (auto-generated from folder structure)
- **Right Sidebar**: Table of contents (extracted from H2-H4 headings)
- **Breadcrumbs**: Full path navigation
- **Footer**: Custom content via template

#### Localized navigation labels

The engine only hardcodes labels for engine-owned UI and route segments such as `blog`, `docs`, `tags`, `archive`, and `graph`. Website/domain-specific labels should live with content, not in `config.yaml` or `internal/i18n`.

Use localized page frontmatter for generated sidebar section titles, breadcrumbs, and header labels:

```yaml
---
title: "Hinnaplaanid"
navTitle: "Hinnad"
---
```

When a section has no `index.md`, a self-named page such as `content/et/docs/beehive-sensors/beehive-sensors.md` can provide the parent section label. `navTitle` is optional and is intended for shorter menu labels when the visible page title should stay longer.

Header navigation supports a hybrid configuration. Use `navigation.header.languages` when most items belong to exactly one locale, and keep `navigation.header.items` for shared or multilingual links that should remain visible for several languages:

```yaml
navigation:
  header:
    enabled: true
    items:
      - title: "Status"
        url: "https://status.example.com/"
        languages: ["en", "et"]
    languages:
      en:
        - title: "About"
          url: "/about/"
      et:
        - title: "Meist"
          url: "/et/about/"
```

### 5. Theming

#### Color Schemes
- Light mode (default)
- Dark mode (auto-detected + manual toggle)
- CSS custom properties for easy customization

#### Typography
- System font stack for UI elements
- Configurable content font (serif/sans-serif)
- Code blocks: JetBrains Mono or Fira Code

### 6. Special Pages

#### Homepage
- Customizable layout via template/config
- Featured projects section
- Recent blog posts
- Hero section with background image

#### Tag Pages
- Cloud visualization
- List of posts per tag
- Tag index page

#### Archive
- Chronological listing
- Grouped by year/month
- Pagination support

#### Graph View (Obsidian-style)
- Force-directed graph of article connections
- Node size based on link count
- Clustering by tags/categories
- Interactive navigation

### 7. Build System

#### Performance Goals
- Incremental builds (only changed files)
- Parallel processing (CPU-bound tasks)
- Streaming writes (minimize memory usage)
- Target: <5s for 1000+ articles on modest hardware

#### Caching Strategy
- File hash-based caching
- Separate cache directory (.cache/)
- Skip unchanged content
- Image optimization cache

#### Output Structure
```
dist/
├── index.html
├── blog/
│   ├── index.html
│   ├── page/
│   │   └── 2/index.html
│   └── tech/
│       └── backend/
│           └── shpargalka-po-golang/
│               ├── index.html
│               └── index.md # when build.publishMarkdown is enabled
├── docs/
│   └── ob-avtore/index.html
├── tags/
│   └── backend/index.html
├── archive/
│   └── 2025/
│       └── 05/index.html
├── assets/
│   ├── css/
│   ├── js/
│   └── img/
├── sitemap.xml
├── rss.xml
└── atom.xml
```

## Configuration

### config.yaml

```yaml
site:
  title: "Артём Курапов"
  tagline: "Наблюдения и размышления..."
  url: "https://kurapov.ee"
  baseUrl: "/"
  language: "ru"
  favicon: "static/img/favicon.ico"

author:
  name: "Артём Курапов"
  email: "..."
  social:
    github: "tot-ra"
    linkedin: "..."

build:
  contentDir: "content"
  outputDir: "dist"
  cacheDir: ".cache"
  parallelWorkers: 4
  publishMarkdown: true # optional: expose source-backed routes as discoverable index.md alternatives
  
features:
  search: true
  darkMode: true
  rss: true
  graphView: true
  comments: false
  
assets:
  images:
    formats: ["webp"]
    sizes:
      thumbnail: 150
      preview: 400
      full: 1200
    quality: 85
    parallelWorkers: 2   # optional image-only worker cap; 0 inherits build.parallelWorkers
    maxSourcePixels: 0   # optional guardrail; 0 disables source-image pixel limit
    maxVariantPixels: 0  # optional guardrail; 0 disables per-variant pixel limit

menu:
  - label: "Блог"
    href: "/blog/"
  - label: "Об авторе"
    href: "/docs/ob-avtore/"
  - label: "Проекты"
    href: "/#projects"

homepage:
  layout: "custom"  # or "blog", "docs"
  hero:
    title: "Artjom Kurapov"
    subtitle: "Full-Stack Product Engineer"
    background: "static/img/bg.webp"
  projects:
    - title: "Gratheon"
      link: "https://gratheon.com"
      description: "..."
      image: "/img/projects/gratheon.png"
```

## Audio Narration (TTS)

Blog Engine can generate and cache narration audio for blog posts and show a built-in player in article pages.

### What it does

- Generates audio for latest `N` blog posts during build
- Stores generated files under `content/audio/posts` (or custom `audio.outputDir`)
- Reuses cached files (no re-generation if MP3 already exists)
- Renders a play/stop + waveform player near article date when `AudioURL` exists

### Audio configuration

```yaml
audio:
  enabled: true
  provider: "edge" # "edge" or "elevenlabs"
  outputDir: "content/audio/posts"
  recentPosts: 10      # latest N posts, <=0 means all
  maxChars: 12000      # input text cap before synthesis

  edge:
    binary: "edge-tts"
    rate: "+0%"
    pitch: "+0Hz"
    voice: "ru-RU-SvetlanaNeural"

  elevenlabs:
    apiKeyEnv: "ELEVENLABS_API_KEY"
    baseUrl: "https://api.elevenlabs.io"
    modelId: "eleven_multilingual_v2"
    outputFormat: "mp3_44100_128"
    defaultVoiceId: "EXAVITQu4vr4xnSDxMaL"
    stability: 0.45
    similarityBoost: 0.75
    style: 0.2
    speakerBoost: true

  voices:
    ru: "ru-RU-SvetlanaNeural"
    en: "en-US-EmmaMultilingualNeural"
```

### Provider setup

`edge`:
- Install `edge-tts` (for example via `pipx install edge-tts`)
- Ensure `edge-tts` is in `PATH`

`elevenlabs`:
- Set API key in environment or `.env`
- Example `.env`:

```env
ELEVENLABS_API_KEY=your_api_key_here
```

### Speech text normalization

Before synthesis, blog markdown is normalized:
- strips emoji/symbol-like runes
- strips markdown tables
- strips code blocks/inline code
- converts URLs to domain speech (`link to example.com`)
- adds extra pauses around headings

### Regeneration behavior

Audio generation is cache-based. Existing MP3 files are reused.

To regenerate:

```bash
# remove cached audio
rm -rf content/audio/posts

# build again
blog-engine build
```

## Technical Architecture

### Module Structure

```
cmd/
└── blog-engine/           # Main CLI entry point
    └── main.go

internal/
├── config/                # Configuration parsing
├── parser/                # Markdown parsing
│   ├── frontmatter.go     # YAML frontmatter extraction
│   ├── markdown.go        # Markdown → AST
│   └── links.go           # Link resolution
├── renderer/              # HTML generation
│   ├── html.go            # Markdown → HTML
│   ├── templates.go       # Template engine
│   └── components.go      # UI components
├── builder/               # Site building
│   ├── site.go            # Site structure
│   ├── page.go            # Page generation
│   └── navigation.go      # Menu/TOC generation
├── assets/                # Asset processing
│   ├── images.go          # Image optimization
│   ├── css.go             # CSS bundling
│   └── js.go              # JS bundling
├── search/                # Search index
│   └── index.go           # Full-text search
├── graph/                 # Graph visualization
│   └── graph.go           # Link graph generation
└── server/                # Dev server
    └── server.go          # Hot reload server
```

### Key Dependencies

- `github.com/yuin/goldmark` - Markdown parser (extensible, GFM support)
- `github.com/yuin/goldmark-meta` - YAML frontmatter
- `github.com/chromedp/chromedp` - Mermaid diagram rendering (optional)
- `github.com/disintegration/imaging` - Image processing
- `github.com/tdewolff/minify` - HTML/CSS/JS minification
- `github.com/blevesearch/bleve` - Full-text search index
- `github.com/fsnotify/fsnotify` - File watching (dev mode)
- `github.com/spf13/cobra` - CLI framework

### Data Flow

```
1. Discovery
   └── Walk content/ directory
       └── Collect .md files

2. Parsing
   └── For each file:
       ├── Extract frontmatter
       ├── Parse markdown → AST
       ├── Extract links
       └── Build content graph

3. Processing
   └── Parallel workers:
       ├── Render markdown → HTML
       ├── Process images
       └── Generate TOC

4. Generation
   └── For each page:
       ├── Apply template
       ├── Inject navigation
       ├── Write HTML
       └── Update search index

5. Post-processing
   ├── Generate sitemap
   ├── Generate RSS
   ├── Generate graph data
   └── Copy static assets
```

## CLI Commands

```bash
# Build site
blog-engine build

# Build with specific config
blog-engine build --config ./custom-config.yaml

# Development server with hot reload
blog-engine serve --port 3000

# Clean build (ignore cache)
blog-engine build --clean

# Validate links
blog-engine validate

# Generate graph data only
blog-engine graph
```

## Deployment

### GitHub Actions Workflow

```yaml
name: Deploy
on:
  push:
    branches: [main]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Build
        run: |
          go install github.com/tot-ra/blog-engine@latest
          blog-engine build
      - name: Deploy
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./dist
```

## Migration from Docusaurus

1. Copy `blog/` and `docs/` content
2. Convert `docusaurus.config.ts` → `config.yaml`
3. Move `static/` assets
4. Frontmatter: `sidebar_position` is accepted as an alias for `order` (no rename required)
5. Update internal links (Docusaurus uses `.md` extensions)
6. Test build and fix issues

## License

MIT
