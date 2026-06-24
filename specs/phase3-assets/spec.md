# Phase 3: Assets Specification

## Overview

Image optimization and asset processing pipeline for efficient web delivery.

## Goals

1. Convert images to WebP format with quality settings
2. Generate responsive image sizes (thumbnail, preview, full)
3. Process CSS/JS minification
4. Implement asset caching for incremental builds
5. Handle image references in markdown

## Dependencies

- Phase 1: Core (MVP)
- Phase 2: Navigation

## Components

### 3.1 Image Processor

**Purpose**: Optimize and convert images for web delivery

**Interface**:
```go
type ImageProcessor interface {
    Process(ctx *BuildContext, file ContentFile) (*ProcessedImage, error)
    ProcessBatch(ctx *BuildContext, files []ContentFile) ([]*ProcessedImage, error)
}

type ProcessedImage struct {
    OriginalPath  string
    Variants      []ImageVariant
    Width         int
    Height        int
    DominantColor string  // For placeholder
}

type ImageVariant struct {
    Format   string  // "webp", "avif"
    Size     string  // "thumbnail", "preview", "full"
    Width    int
    Height   int
    FilePath string
    FileSize int64
}
```

**Supported Input Formats**:
- JPEG/JPG
- PNG
- GIF (static only, no animation)
- SVG (pass-through with optimization)
- WebP (re-encode if needed)

**Output Formats**:
- WebP (primary)
- AVIF (optional, future)

**Size Variants**:
| Size | Width | Use Case |
|------|-------|----------|
| thumbnail | 150px | Lists, grids |
| preview | 400px | Article previews |
| full | 1200px | Article content |
| original | - | Click to enlarge |

**Processing Pipeline**:
1. Decode source image
2. Calculate dimensions
3. Generate variants:
   - Resize to target width (maintain aspect ratio)
   - Encode to WebP
   - Apply quality setting
4. Write to output directory

**Quality Settings**:
```yaml
images:
  quality: 85
  lossless: false  # For PNG/SVG
  method: 4        # WebP compression method (0-6)
```

**Filename Convention**:
```
Original:  blog/tech/img/photo.jpg
Output:    assets/img/blog/tech/photo-{size}.webp
           assets/img/blog/tech/photo-thumb.webp
           assets/img/blog/tech/photo-preview.webp
           assets/img/blog/tech/photo-full.webp
```

### 3.2 Image Cache

**Purpose**: Skip reprocessing unchanged images

**Interface**:
```go
type ImageCache interface {
    Get(key string) (*CacheEntry, bool)
    Set(key string, entry *CacheEntry) error
    Invalidate(path string) error
}

type CacheEntry struct {
    SourceHash    string    // SHA256 of source file
    SourceModTime time.Time
    Variants      []ImageVariant
    CreatedAt     time.Time
}
```

**Cache Key Generation**:
```go
func CacheKey(filePath string, config ImageConfig) string {
    // Hash of: file path + mod time + config
    return fmt.Sprintf("%x", sha256.Sum256(...))
}
```

**Cache Storage**:
```
.cache/
└── images/
    └── blog/
        └── tech/
            └── img-photo.jpg.json  # Cache entry
```

**Invalidation Rules**:
- Source file changed (mtime or size)
- Config changed (quality, sizes)
- Cache entry corrupted

### 3.3 Markdown Image Transformer

**Purpose**: Transform markdown image syntax to responsive HTML

**Interface**:
```go
type ImageTransformer interface {
    Transform(node ast.Node, page *Page) (ast.Node, error)
}
```

**Input**:
```markdown
![Alt text](./img/photo.jpg)
![Alt text](./img/photo.jpg "Title")
<img src="./img/photo.jpg" alt="Alt" width="400">
```

**Output**:
```html
<!-- Simple image -->
<figure class="md-image">
  <picture>
    <source srcset="/assets/img/post/photo-full.webp" type="image/webp">
    <img src="/assets/img/post/photo-full.jpg" 
         alt="Alt text" 
         loading="lazy"
         width="1200" height="800">
  </picture>
  <figcaption>Title (if provided)</figcaption>
</figure>

<!-- Linked image (click to enlarge) -->
<figure class="md-image md-image-linked">
  <a href="/assets/img/post/photo-original.jpg" data-lightbox>
    <picture>...</picture>
  </a>
</figure>
```

**Transformation Rules**:
- Wrap images in `<figure>` with optional `<figcaption>`
- Generate `<picture>` with WebP source
- Add `loading="lazy"` for below-fold images
- Preserve original dimensions
- Support explicit width/height attributes

### 3.4 CSS Processor

**Purpose**: Process and optimize CSS files

**Interface**:
```go
type CSSProcessor interface {
    Process(ctx *BuildContext, files []ContentFile) (*CSSBundle, error)
}

type CSSBundle struct {
    Path        string
    Content     string
    SourceMap   string
    Size        int64
    SizeGzipped int64
}
```

**Processing Steps**:
1. Concatenate CSS files (respecting @import order)
2. Process @import statements
3. Minify (remove whitespace, comments)
4. Add vendor prefixes (autoprefixer)
5. Generate source map (optional)

**CSS Variables Support**:
```css
:root {
  --primary-color: #0066cc;
  --text-color: #333;
  --bg-color: #fff;
}

[data-theme="dark"] {
  --text-color: #eee;
  --bg-color: #1a1a1a;
}
```

### 3.5 JS Processor

**Purpose**: Process and optimize JavaScript files

**Interface**:
```go
type JSProcessor interface {
    Process(ctx *BuildContext, files []ContentFile) (*JSBundle, error)
}

type JSBundle struct {
    Path        string
    Content     string
    SourceMap   string
    Size        int64
    SizeGzipped int64
}
```

**Processing Steps**:
1. Concatenate JS files
2. Minify (terser-style)
3. Generate source map (optional)

**Built-in Scripts**:
- Theme toggle (dark/light)
- Mobile menu toggle
- TOC scroll spy (Phase 5)

### 3.6 Asset URL Rewriter

**Purpose**: Rewrite asset URLs in HTML/CSS to hashed/cache-busted versions

**Interface**:
```go
type URLRewriter interface {
    RewriteHTML(html string, assets map[string]string) (string, error)
    RewriteCSS(css string, assets map[string]string) (string, error)
}
```

**Hash Generation**:
```go
// Original: /assets/css/main.css
// Hashed:   /assets/css/main.a3f7b2.css
func HashAsset(path string, content []byte) string
```

**URL Patterns**:
| Original | Rewritten |
|----------|-----------|
| `url('./img/logo.png')` | `url('/assets/img/logo.a1b2c3.webp')` |
| `src="/img/photo.jpg"` | `src="/assets/img/photo.a1b2c3.webp"` |
| `href="/css/style.css"` | `href="/assets/css/style.a1b2c3.css"` |

### 3.7 Responsive Image Generator

**Purpose**: Generate srcset for responsive images

**Interface**:
```go
type ResponsiveGenerator interface {
    GenerateSrcset(variants []ImageVariant) string
    GenerateSizes(layout string) string
}
```

**Output**:
```html
<img src="/assets/img/photo-400.webp"
     srcset="/assets/img/photo-150.webp 150w,
             /assets/img/photo-400.webp 400w,
             /assets/img/photo-1200.webp 1200w"
     sizes="(max-width: 768px) 100vw, 800px"
     alt="Description">
```

## Configuration

```yaml
assets:
  images:
    enabled: true
    formats: ["webp"]
    quality: 85
    sizes:
      thumbnail: 150
      preview: 400
      full: 1200
    lazyLoading: true
    parallelWorkers: 2   # optional image-only worker cap; 0 inherits build.parallelWorkers
    maxSourcePixels: 0   # optional guardrail; 0 disables source-image pixel limit
    maxVariantPixels: 0  # optional guardrail; 0 disables per-variant pixel limit
    placeholder: "blur"  # blur, color, none
    
  css:
    enabled: true
    minify: true
    autoprefixer: true
    sourceMap: false
    
  js:
    enabled: true
    minify: true
    sourceMap: false
    
  cache:
    enabled: true
    directory: ".cache"
    maxAge: "7d"
```

## Output Structure

```
dist/
├── assets/
│   ├── css/
│   │   └── main.a3f7b2.css
│   ├── js/
│   │   └── main.c8d9e1.js
│   └── img/
│       └── blog/
│           └── tech/
│               ├── photo-thumb.webp
│               ├── photo-preview.webp
│               └── photo-full.webp
└── ...
```

## Performance Targets

| Metric | Target |
|--------|--------|
| Image processing (100 images) | < 30s |
| Cache hit skip time | < 10ms per image |
| WebP quality vs size | ~70% smaller than JPEG |
| CSS/JS minification | ~50% size reduction |

## Dependencies

```go
require (
    github.com/disintegration/imaging v1.6.2
    github.com/chromedp/chromedp v0.9.0  // For Mermaid in Phase 5
    github.com/tdewolff/minify/v2 v2.12.0
)
```

## Testing

### Unit Tests
- Image resize with various aspect ratios
- WebP encoding quality
- Cache hit/miss scenarios
- URL rewriting

### Integration Tests
- Full image pipeline
- CSS/JS bundling
- HTML output verification

### Visual Regression
- Compare processed images with originals
- Verify responsive behavior

## Deliverables

- [ ] Image processor with WebP conversion
- [ ] Image cache system
- [ ] Markdown image transformer
- [ ] CSS processor with minification
- [ ] JS processor with minification
- [ ] Asset URL rewriter
- [ ] Responsive image generator
- [ ] Cache invalidation logic
- [ ] Configuration options
- [ ] Performance benchmarks
