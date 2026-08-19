# Blog Engine MD

A high-performance static site generator written in Go, designed as a memory-efficient alternative to [Docusaurus](https://docusaurus.io/) or [Hugo](https://gohugo.io/) for markdown-based blogs and documentation in [Obsidian](https://obsidian.md/) style.

<img width="1200" height="800" alt="blog-engine-rewrite-preview-full" src="https://github.com/user-attachments/assets/e153d483-68aa-4cb4-8e35-79b6409d52fa" />

## Features

- **[Related articles via vector search](#related-articles-vector-search)** - OpenAI embeddings are generated once and stored in article frontmatter; every build ranks related cards offline with cosine similarity, tag bonuses, and MMR diversity re-ranking. No API key or network access at build time.
- **[Obsidian-style markdown](#markdown-processing)** - CommonMark/GFM, `[[wiki links]]`, task lists, tables, strikethrough, autolinks, YAML frontmatter, `<!--truncate-->` previews.
- **[Graph view](#graph-view)** - 3D embedding-space graph of articles (PCA layout from frontmatter vectors), with link/tag edges as visual connections.
- **[Multi-language sites](#internationalization)** - built-in UI strings for `en`/`ru`/`et`, per-language navigation, localized dates, optional browser-language redirect.
- **[Audio narration (TTS)](#audio-narration-tts)** - generate and cache MP3 narration per post via `edge-tts` or ElevenLabs, with an inline waveform player.
- **[Image pipeline](#image-processing)** - WebP conversion, responsive size variants, automatic HDR gain-map preservation, lazy loading, mtime/hash-based caching, parallel workers.
- **[Rich content blocks](#extended-syntax)** - Mermaid diagrams (lazy-loaded), admonitions (`:::info`, `:::warning`, `:::tip`), YouTube/Vimeo embeds, syntax highlighting with copy buttons.
- **[Auto navigation](#navigation--layout)** - sidebar tree derived from folders, breadcrumbs, sticky table of contents with scrollspy, prev/next links.
- **[Interactive HTML articles](#interactive-html-partials)** - drop in a `.html` file with its own `<style>`/`<script>` and it becomes a first-class article.
- **[SEO & syndication](#seo--syndication)** - meta/OpenGraph/Twitter tags, JSON-LD, canonical URLs, `sitemap.xml`, RSS and Atom feeds.
- **[Agent-friendly output](#markdown-publishing)** - optionally publish `index.md` next to `index.html` so LLMs and agents can read source markdown directly.
- **[Theming](#theming)** - light/dark with auto-detection and manual toggle, driven by CSS custom properties.
- **[Tags, archive, pagination](#special-pages)** - tag index and per-tag pages, chronological archive by year/month, paginated listings.
- **[Dev server with live reload](#main-workflows)** - `just serve` rebuilds and refreshes the browser on content, template, or config changes.

Everything ships as a single Go binary. No Node.js toolchain, no runtime dependencies.

## Quick Start

Requirements:

- Go 1.24 or newer
- [`just`](https://github.com/casey/just) for the project recipes

The repository includes a `justfile` so the main development flows do not depend on a globally installed `blog-engine` binary:

```bash
# compile bin/blog-engine
just build

# build the static site into dist/
just generate

# start the dev server with live reload at http://127.0.0.1:3000
just serve

# use another port when needed
just serve 8080
```

The direct CLI equivalent is also available after `just build`:

```bash
./bin/blog-engine build
./bin/blog-engine serve --port 3000
```

Run `just` or `just --list` to see all available recipes. The CLI loads `.env` from the working directory (and from the binary's directory) before reading `config.yaml`.

## Main Workflows

### Build and preview the site

Use `just generate` for a one-off production-style build. It compiles the binary first and writes the generated site to `build.outputDir`, which is `dist/` in the example configuration.

Use `just serve` during authoring. It performs an initial build, serves the generated output at `http://127.0.0.1:3000`, and rebuilds the site when content, templates, assets, configuration, or `.env` files change. Stop it with `Ctrl-C`.

```bash
just generate
just serve              # default port 3000
just serve 8080         # custom port
```

### Generate and verify embeddings

Embeddings are stored in article frontmatter and are used by related-article cards and the graph view. Generate them before committing content changes that should participate in semantic search:

```bash
just embed-dry-run      # offline estimate; no API key or network access
just embed              # generate missing or stale vectors; requires OPENAI_API_KEY
just embed-check        # offline CI/pre-commit check; fails when vectors are stale
just embed-force        # regenerate every eligible article; requires OPENAI_API_KEY
```

`just embed` and `just embed-force` load `OPENAI_API_KEY` from the environment or an untracked `.env` file. Review `just embed-dry-run` before a paid run. Production builds only read the committed frontmatter vectors, so `just generate` does not need an OpenAI key.

### Test and clean local artifacts

```bash
just test               # run go test -v ./...
just clean              # remove bin/, dist/, and .cache/
```

## CLI Commands

```bash
./bin/blog-engine build                         # build the site from ./config.yaml
./bin/blog-engine serve [--port 3000]           # dev server with file watching and live reload
./bin/blog-engine embed [--dry-run|--check|--force] # manage article embeddings
./bin/blog-engine help                           # show CLI help
```

The corresponding just recipes are `build`, `generate`, `serve`, `embed-dry-run`, `embed`, `embed-check`, `embed-force`, `test`, and `clean`.

## Content Structure

The engine supports two primary content types:

1. **Blog posts** (`blog/`) - date-dependent entries organized by category/subfolder, with `<!--truncate-->` previews, prev/next navigation, and archive pages.
2. **Documentation/pages** (`docs/`) - permanent content with a hierarchical menu derived from the folder structure and no date dependency.

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

### URL patterns

```
/blog/<category>/<slug>/              # Blog posts
/docs/<path>/<slug>/                  # Documentation
/tags/<tag>/                          # Tag pages
/archive/<year>/<month>/              # Archive pages
/page/<n>/                            # Pagination
/graph/                               # Graph view
```

### Output structure

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
│               └── index.md          # when build.publishMarkdown is enabled
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
├── graph.json
├── sitemap.xml
├── rss.xml
└── atom.xml
```

---

## Markdown Processing

### Supported syntax

Standard CommonMark plus the GFM extension set: tables, task lists, strikethrough, and autolinks. Code blocks are highlighted server-side with [chroma](https://github.com/alecthomas/chroma) and get a copy button.

### Extended syntax

- **YAML frontmatter**: `title`, `date`, `tags`, `description`, `draft`, `order` (Docusaurus `sidebar_position` is accepted as an alias), `navTitle`, `redirectUrl`, `related`, `hideRelated`.
- **Wiki links**: `[[Page Title]]` and `[[Page Title|Display Text]]` are resolved against the content tree and feed the [graph view](#graph-view).
- **`<!--truncate-->`**: cutoff point for article previews in listings and feeds.
- **Mermaid diagrams**: ` ```mermaid ` fences render client-side; the Mermaid ESM bundle is loaded lazily only on pages that contain a diagram.
- **Admonitions**: `:::info`, `:::warning`, `:::tip`.
- **Embeds**: `::youtube[ID]`, `::vimeo[ID]`, or a bare YouTube/Vimeo link on its own line.

### Cross-reference resolution

Internal markdown links `[text](./other-file.md)` are converted to HTML paths, with relative path resolution across folder boundaries.

### Interactive HTML partials

`.html` content files are embedded without Markdown conversion, including per-article inline `<style>` and `<script>`. Put the same YAML frontmatter inside a leading HTML comment:

```html
<!--
---
title: Interactive article
date: 2026-08-13
tags: [demo]
---
-->
<section>...</section>
```

### Markdown publishing

With `build.publishMarkdown: true`, every markdown-backed page also emits `index.md` next to `index.html`. This makes source content directly readable by LLMs, agents, and scripts without HTML parsing. Redirect stubs (`redirectUrl`) are excluded.

---

## Related Articles (Vector Search)

Blog Engine generates semantic article embeddings once, stores each compact vector in the article's Markdown frontmatter, and uses those vectors to rank related article cards during every build. **Production builds are offline and never call OpenAI.**

Because path, URL, and language are derived at build time rather than stored in frontmatter, embedded articles can be moved or renamed without updating a separate index.

### How ranking works

1. `just embed` requests vectors from OpenAI only for articles whose content hash is missing, stale, or forced.
2. `just generate` collects every valid frontmatter embedding into an ephemeral JSON at `related.cachePath`.
3. Candidates are scored by cosine similarity, boosted by shared tags, filtered by `minScore`, then re-ranked with MMR so the resulting cards are relevant *and* varied.
4. Rendered cards are precomputed before HTML generation, so the ranking cost is paid once per build.

### Configuration

```yaml
related:
  enabled: true
  provider: "openai"
  model: "text-embedding-3-small"
  dimensions: 512
  apiKeyEnv: "OPENAI_API_KEY"
  cachePath: "content/embeddings.json" # generated during build; do not commit
  sections: ["blog"]
  count: 4
  minScore: 0.3
  diversity: 0.7
  crossLanguage: false
```

- `sections` limits embedding and matching to content sections such as `blog`.
- `count` is the maximum number of cards per article.
- `minScore` is the minimum cosine similarity accepted before tag bonuses and MMR re-ranking.
- `diversity` is the MMR relevance weight from `0` to `1`: lower values favor variety, while `1` favors query similarity only.
- `crossLanguage: false` keeps candidates in the article's language. Translation counterparts are excluded from one another.
- Frontmatter `related` can explicitly select articles, while `hideRelated: true` suppresses the block.
- Drafts, section index files, and `redirectUrl` compatibility stubs are not embedded.

### Embedding workflow

```bash
# Show changed articles, estimated tokens, and estimated cost. No key or network is used.
just embed-dry-run

# Generate or incrementally update missing/stale article frontmatter. Reads OPENAI_API_KEY from the environment or .env.
just embed

# Verify frontmatter hashes and metadata without a key or network access.
just embed-check

# Re-embed every eligible article after intentionally changing model/input settings.
just embed-force
```

Review the dry-run estimate before a paid run. The normal workflow is:

1. Run `just embed-dry-run` locally and review the estimate.
2. Run `just embed` locally with the API key available only in the environment or an untracked `.env` file. It only requests vectors for missing, stale, or forced articles.
3. Review and commit the changed Markdown articles together with relevant `related` configuration changes. A pre-commit hook may run `just embed-check` to reject commits with missing or stale vectors; generating paid embeddings automatically in a hook is intentionally optional because it requires a local API key.
4. Deploy and run `just generate` without `OPENAI_API_KEY`.
5. Do not commit the generated `cachePath` JSON. Add it to the site repository's `.gitignore`; it is recreated on every build and contains path, URL, and language metadata derived from the current article location.

The article frontmatter representation is:

```yaml
embedding:
  version: 1
  model: "text-embedding-3-small"
  dimensions: 512
  hash: "sha256:..."
  vector: "..." # normalized signed-int8 vector encoded as Base64
  scale: 0.001234
```

The hash includes the prepared article text, model, and dimensions, so editing meaningful content makes the vector stale. Moving or renaming the file does not. The generated central JSON retains the versioned `version`, `model`, `dims`, and path-keyed `entries` schema only as an ephemeral build-stage interchange format.

Do not commit API keys, `.env` files, or the generated embeddings JSON.

---

## Graph View

An interactive 3D graph of articles placed by semantic embeddings (PCA of frontmatter vectors into x/y/z), with link and tag edges drawn only as visual connections:

- `graph.json` with nodes (including precomputed `x`,`y`,`z`), and deduplicated edges
- an interactive `/graph/` page (Three.js + orbit controls) for browsing the embedding space
- node relationships usable as a secondary sidebar mode (`navigation.sidebar.sections.*.enableGraph`)

Enabled via `advanced.graph.enabled`. Articles without embeddings fall back to neighbor centroids; tag hubs sit at the centroid of connected articles.

---

## Image Processing

- Source formats: JPG, PNG, GIF, SVG
- Target format: WebP via the `cwebp` binary, with automatic JPEG fallback when `cwebp` is unavailable
- HDR JPEG preservation: JPEGs carrying an Adobe/Google/Apple XMP gain map or ISO 21496-1 gain-map metadata bypass resizing and WebP conversion. The original file is copied byte-for-byte so its SDR rendition, embedded gain map, and HDR rendering intent remain intact.
- Responsive variants generated per configured size (for example `small`, `medium`, `large`) for non-HDR raster images
- Lazy loading attributes and alt text extracted from markdown
- Parallel worker pool with a dedicated `parallelWorkers` cap
- Unchanged files are skipped via the asset cache
- Optional `maxSourcePixels` / `maxVariantPixels` guardrails against decompression bombs

```yaml
assets:
  images:
    enabled: true
    quality: 85
    sizes:
      small: 400
      medium: 800
      large: 1200
    lazyLoading: true
    parallelWorkers: 2   # 0 inherits build.parallelWorkers
    maxSourcePixels: 0   # 0 disables the source-image pixel limit
    maxVariantPixels: 0  # 0 disables the per-variant pixel limit
  css:
    enabled: true
    minify: true
  js:
    enabled: true
    minify: true
```

---

## Navigation & Layout

```
┌─────────────────────────────────────────────────────────────┐
│  Logo    [Home] [Blog] [Docs] [Tags]              [Theme]   │  Header
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
│          │  │  Related articles                          │  │
│          │  ├────────────────────────────────────────────┤  │
│          │  │  Previous Post    |    Next Post           │  │
│          │  └────────────────────────────────────────────┘  │
├──────────┴──────────────────────────────────────────────────┤
│                        Footer                               │
└─────────────────────────────────────────────────────────────┘
```

- **Top navigation**: main sections, theme toggle, per-language header items
- **Left sidebar**: hierarchical menu auto-generated from the folder structure, with per-section category/time/graph modes and exclude rules
- **Right sidebar**: table of contents extracted from H2-H4 headings, with scrollspy highlighting
- **Breadcrumbs**: full path navigation
- **Prev/next**: within the same category or across a whole section

```yaml
navigation:
  sidebar:
    enabled: true
    maxDepth: 3
  breadcrumbs:
    enabled: true
    homeLabel: "Home"
  prevNext:
    enabled: true
    sameCategoryOnly: false
```

---

## Internationalization

The engine hardcodes labels only for engine-owned UI and route segments such as `blog`, `docs`, `tags`, `archive`, and `graph` (built-in strings for `en`, `ru`, `et`, including localized month names and date formats). Website- and domain-specific labels should live with content, not in `config.yaml` or `internal/i18n`.

```yaml
i18n:
  default: "en"
  browserRedirect:
    enabled: false
  languages:
    - code: "en"
      label: "English"
      aliases: ["en", "en-us", "en-gb"]
    - code: "ru"
      label: "Русский"
      aliases: ["ru", "ru-ru"]
```

### Localized navigation labels

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

---

## Audio Narration (TTS)

Blog Engine can generate and cache narration audio for blog posts and show a built-in player in article pages.

- Generates audio for the latest `N` blog posts during build
- Stores generated files under `content/audio/posts` (or a custom `audio.outputDir`)
- Reuses cached files - no re-generation if the MP3 already exists
- Renders a play/stop + waveform player near the article date when `AudioURL` exists

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
- Set the API key in the environment or `.env`:

```env
ELEVENLABS_API_KEY=your_api_key_here
```

### Speech text normalization

Before synthesis, blog markdown is normalized: emoji/symbol-like runes, markdown tables, code blocks, and inline code are stripped; URLs become domain speech (`link to example.com`); headings get extra pauses.

### Regeneration

Audio generation is cache-based, so existing MP3 files are reused:

```bash
just generate
```

---

## SEO & Syndication

- Semantic HTML5 structure
- Meta tags (description, keywords, `og:*`, Twitter cards)
- Structured data (JSON-LD)
- Canonical URLs
- `sitemap.xml`
- RSS and Atom feeds
- Automatic descriptions derived from content when frontmatter has none

```yaml
seo:
  enabled: true
  defaultDescription: "A high-performance static site generator"
  defaultImage: "/assets/images/sample.svg"
  twitter:
    site: ""
    creator: ""

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
```

---

## Theming

- Light mode (default), dark mode (auto-detected plus manual toggle)
- CSS custom properties for easy customization
- System font stack for UI, configurable content font
- Monospaced code blocks with chroma-based highlighting

```yaml
advanced:
  theme:
    default: "auto"   # light, dark, auto
    allowToggle: true
  mermaid:
    enabled: true
    theme: "default"
  graph:
    enabled: true
    path: "graph"
```

---

## Special Pages

### Homepage

Configurable hero (title, subtitle, description, background image, optional video embed, CTA buttons), project cards, blog and events showcases, social link groups, and custom HTML. Per-language overrides live under `homepageI18n`.

### Tag pages

Tag index with cloud visualization plus a page per tag.

### Archive

Chronological listing grouped by year and month, with pagination.

---

## Configuration

Full `config.yaml` example:

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
  publishMarkdown: true   # expose source markdown as index.md alternatives
  profile: false          # print build stage timings

tags:
  enabled: true

archive:
  enabled: true

pagination:
  enabled: true
  pageSize: 10

homepage:
  enabled: true
  hero:
    title: "Artjom Kurapov"
    subtitle: "Full-Stack Product Engineer"
    background: "static/img/bg.webp"
  projects:
    - title: "Gratheon"
      url: "https://gratheon.com"
      description: "..."
      image: "/img/projects/gratheon.png"
```

See the sections above for `assets`, `audio`, `related`, `navigation`, `i18n`, `seo`, `feeds`, and `advanced`.

---

## Technical Architecture

### Module structure

```
cmd/
└── blog-engine/           # CLI entry point (build, serve, embed)

internal/
├── config/                # Configuration parsing and defaults
├── parser/                # Frontmatter, markdown, wiki links, admonitions, embeds, mermaid
├── renderer/              # Templates, sidebar, TOC
├── builder/               # Site assembly: pages, navigation, sections, assets, audio, related
├── embeddings/            # OpenAI embedding generation and frontmatter persistence
├── related/               # Offline cosine similarity + MMR ranking
├── assets/                # Image/CSS/JS processing and cache
├── graph/                 # Link graph data and page
├── i18n/                  # Built-in UI strings and date formatting
├── seo/                   # Meta tags and JSON-LD
├── feed/                  # RSS and Atom
├── sitemap/               # sitemap.xml
├── tags/                  # Tag taxonomy
├── archive/               # Date archives
├── pagination/            # Paginated listings
├── validator/             # Internal link validation
├── profiler/              # Build stage timings
├── errors/                # Shared error types
└── server/                # Dev server with live reload and file watching
```

### Key dependencies

- `github.com/yuin/goldmark` - markdown parser (extensible, GFM support)
- `github.com/yuin/goldmark-meta` - YAML frontmatter
- `github.com/yuin/goldmark-highlighting/v2` + `github.com/alecthomas/chroma/v2` - syntax highlighting
- `github.com/disintegration/imaging` + `golang.org/x/image` - image processing and WebP output
- `github.com/tdewolff/minify/v2` - HTML/CSS/JS minification
- `gopkg.in/yaml.v3` - configuration

The dev server, file watching, and CLI parsing use the standard library only.

### Data flow

```
1. Discovery
   └── Walk content/ and collect .md / .html files

2. Parsing
   └── For each file:
       ├── Extract frontmatter (including embedding vectors)
       ├── Parse markdown → AST
       ├── Resolve links and wiki links
       └── Build content graph

3. Processing
   └── Parallel workers:
       ├── Render markdown → HTML
       ├── Process images
       └── Generate TOC

4. Relations
   ├── Collect embeddings into the build-stage cache
   └── Precompute related cards (cosine + tag bonus + MMR)

5. Generation
   └── For each page: apply template, inject navigation, write HTML (+ index.md)

6. Post-processing
   ├── sitemap.xml
   ├── RSS / Atom
   ├── graph.json and /graph/ page
   └── Copy static assets
```

---

## Deployment

### GitHub Actions

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
          go-version: '1.24'
      - name: Build
        run: |
          go install github.com/tot-ra/blog-engine/cmd/blog-engine@latest
          blog-engine build
      - name: Deploy
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./dist
```

`OPENAI_API_KEY` is intentionally not needed in CI: embeddings are committed with the articles.

## Migration from Docusaurus

1. Copy `blog/` and `docs/` content
2. Convert `docusaurus.config.ts` → `config.yaml`
3. Move `static/` assets
4. Frontmatter: `sidebar_position` is accepted as an alias for `order` (no rename required)
5. Update internal links (Docusaurus uses `.md` extensions)
6. Test build and fix issues

## License

MIT
