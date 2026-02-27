# Shared Schemas

Common data structures used across all phases.

## ContentFile

Represents a single content file (markdown or asset).

```yaml
ContentFile:
  type: object
  required:
    - path
    - relativePath
    - contentType
    - modifiedTime
  properties:
    path:
      type: string
      description: Absolute filesystem path
      example: "/content/blog/tech/article.md"
    relativePath:
      type: string
      description: Path relative to content directory
      example: "blog/tech/article.md"
    contentType:
      type: string
      enum: [markdown, image, asset]
      description: Type of content
    modifiedTime:
      type: string
      format: date-time
      description: File modification timestamp
    size:
      type: integer
      description: File size in bytes
```

## Frontmatter

YAML frontmatter extracted from markdown files.

```yaml
Frontmatter:
  type: object
  properties:
    title:
      type: string
      description: Page/article title
      example: "Шпаргалка по golang"
    date:
      type: string
      format: date-time
      description: Publication date (for blog posts)
      example: "2018-04-18T10:00:00Z"
    draft:
      type: boolean
      default: false
      description: If true, skip during build
    tags:
      type: array
      items:
        type: string
      description: Article tags
      example: ["backend", "go"]
    description:
      type: string
      description: Meta description for SEO
    slug:
      type: string
      description: Custom URL slug (optional)
      example: "golang-cheatsheet"
    order:
      type: integer
      description: Sort order in navigation (docs only)
      example: 1
    hideToc:
      type: boolean
      default: false
      description: Hide table of contents
    hideNav:
      type: boolean
      default: false
      description: Hide from navigation
```

## Page

Internal representation of a page to be rendered.

```yaml
Page:
  type: object
  required:
    - id
    - url
    - title
    - content
  properties:
    id:
      type: string
      description: Unique identifier (derived from path)
      example: "blog-tech-backend-golang"
    url:
      type: string
      description: Output URL path
      example: "/blog/tech/backend/golang-cheatsheet/"
    sourcePath:
      type: string
      description: Source markdown file path
    title:
      type: string
    description:
      type: string
    content:
      type: string
      description: Rendered HTML content
    rawContent:
      type: string
      description: Original markdown content
    frontmatter:
      $ref: "#/Frontmatter"
    toc:
      type: array
      items:
        $ref: "#/TocItem"
    links:
      type: array
      items:
        $ref: "#/Link"
    assets:
      type: array
      items:
        type: string
      description: Referenced asset paths
    modifiedTime:
      type: string
      format: date-time
    type:
      type: string
      enum: [blog, doc, page, tag, archive]
```

## TocItem

Table of contents entry.

```yaml
TocItem:
  type: object
  required:
    - level
    - text
    - anchor
  properties:
    level:
      type: integer
      minimum: 1
      maximum: 6
      description: Heading level (H1-H6)
    text:
      type: string
      description: Heading text
    anchor:
      type: string
      description: URL anchor (slugified)
      example: "variables-and-types"
    children:
      type: array
      items:
        $ref: "#/TocItem"
```

## Link

Internal or external link.

```yaml
Link:
  type: object
  required:
    - text
    - href
  properties:
    text:
      type: string
    href:
      type: string
      example: "./other-article.md" or "https://example.com"
    type:
      type: string
      enum: [internal, external, anchor]
    resolved:
      type: boolean
      description: Whether internal link was resolved
    targetPageId:
      type: string
      description: ID of target page (if resolved)
```

## SiteConfig

Site-wide configuration.

```yaml
SiteConfig:
  type: object
  required:
    - site
    - build
  properties:
    site:
      type: object
      required:
        - title
        - url
      properties:
        title:
          type: string
        tagline:
          type: string
        url:
          type: string
          format: uri
        baseUrl:
          type: string
          default: "/"
        language:
          type: string
          default: "en"
        favicon:
          type: string
    author:
      type: object
      properties:
        name:
          type: string
        email:
          type: string
        social:
          type: object
          additionalProperties:
            type: string
    build:
      type: object
      properties:
        contentDir:
          type: string
          default: "content"
        outputDir:
          type: string
          default: "dist"
        cacheDir:
          type: string
          default: ".cache"
        parallelWorkers:
          type: integer
          default: 4
    features:
      type: object
      properties:
        search:
          type: boolean
          default: true
        darkMode:
          type: boolean
          default: true
        rss:
          type: boolean
          default: true
        graphView:
          type: boolean
          default: false
    images:
      type: object
      properties:
        formats:
          type: array
          items:
            type: string
            enum: [webp, avif]
          default: ["webp"]
        quality:
          type: integer
          minimum: 1
          maximum: 100
          default: 85
        sizes:
          type: object
          properties:
            thumbnail:
              type: integer
              default: 150
            preview:
              type: integer
              default: 400
            full:
              type: integer
              default: 1200
    menu:
      type: array
      items:
        type: object
        properties:
          label:
            type: string
          href:
            type: string
          external:
            type: boolean
            default: false
```

## BuildContext

Runtime build state.

```yaml
BuildContext:
  type: object
  properties:
    config:
      $ref: "#/SiteConfig"
    pages:
      type: object
      additionalProperties:
        $ref: "#/Page"
      description: Map of page ID to Page
    contentGraph:
      type: object
      description: Graph of page relationships
      properties:
        nodes:
          type: array
          items:
            type: object
            properties:
              id:
                type: string
              type:
                type: string
        edges:
          type: array
          items:
            type: object
            properties:
              from:
                type: string
              to:
                type: string
              type:
                type: string
                enum: [link, tag, category]
    tags:
      type: object
      additionalProperties:
        type: array
        items:
          type: string
      description: Map of tag to page IDs
    categories:
      type: object
      additionalProperties:
        type: array
        items:
          type: string
      description: Map of category to page IDs
```

## Error

Standard error response.

```yaml
Error:
  type: object
  required:
    - code
    - message
  properties:
    code:
      type: string
      example: "PARSE_ERROR"
    message:
      type: string
      example: "Failed to parse frontmatter"
    file:
      type: string
      description: File path where error occurred
    line:
      type: integer
    details:
      type: string
```
