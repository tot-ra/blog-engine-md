# Blog Engine MD Specifications

This directory contains detailed specifications for each development phase of the blog engine.

## Structure

```
specs/
├── README.md                    # This file
├── shared/
│   └── schemas.md              # Common data structures
├── phase1-core/
│   └── spec.md                 # MVP: Markdown → HTML
├── phase2-navigation/
│   └── spec.md                 # Sidebar, TOC, breadcrumbs
├── phase3-assets/
│   └── spec.md                 # Images, CSS, JS
├── phase4-features/
│   └── spec.md                 # Search, RSS, tags, archive
├── phase5-advanced/
│   └── spec.md                 # Mermaid, graph, dark mode
└── phase6-polish/
    └── spec.md                 # Incremental builds, validation
```

## Quick Reference

| Phase | Focus | Key Components |
|-------|-------|----------------|
| Phase 1 | Core | Config, Markdown parser, URL generator, Templates |
| Phase 2 | Navigation | Sidebar, TOC, Breadcrumbs, Prev/Next |
| Phase 3 | Assets | Image optimization, CSS/JS minification |
| Phase 4 | Features | Search, RSS, Sitemap, Tags, Archive |
| Phase 5 | Advanced | Mermaid, Graph view, Dark mode, Embeds |
| Phase 6 | Polish | Incremental builds, Validation, Dev server |

## Dependency Graph

```
Phase 1 (Core)
    ↓
Phase 2 (Navigation)
    ↓
Phase 3 (Assets)
    ↓
Phase 4 (Features)
    ↓
Phase 5 (Advanced)
    ↓
Phase 6 (Polish)
```

## Shared Schemas

All phases use common data structures defined in `shared/schemas.md`:

- `ContentFile` - File metadata
- `Frontmatter` - YAML frontmatter
- `Page` - Internal page representation
- `SiteConfig` - Site configuration
- `BuildContext` - Runtime build state

## Specification Format

Each spec follows this structure:

1. **Overview** - What this phase accomplishes
2. **Goals** - Specific objectives
3. **Non-Goals** - Out of scope items
4. **Components** - Detailed interface definitions
5. **Configuration** - YAML options
6. **Testing** - Test strategy
7. **Deliverables** - Checklist of items to implement

## Reading Order

1. Start with `shared/schemas.md` to understand data structures
2. Read phases in order (1 → 6)
3. Each phase builds on previous ones

## Implementation Notes

- Each phase can be developed and tested independently
- Use feature flags to enable/disable incomplete features
- Maintain backward compatibility within phases
- Document breaking changes between phases
