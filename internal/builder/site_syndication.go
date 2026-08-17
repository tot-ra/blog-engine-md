package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tot-ra/blog-engine/internal/feed"
	"github.com/tot-ra/blog-engine/internal/graph"
	"github.com/tot-ra/blog-engine/internal/related"
	"github.com/tot-ra/blog-engine/internal/sitemap"
)

// generateFeeds creates RSS and/or Atom feed XML files
func (b *SiteBuilder) generateFeeds(blogPosts []*Page) error {
	gen := feed.NewFeedGenerator(
		b.config.Site.Title,
		b.config.Site.URL,
		b.config.Site.Language,
		b.config.Author.Name,
		b.config.Author.Email,
	)

	// Build feed items from blog posts
	maxItems := b.config.Feeds.RSS.Items
	if b.config.Feeds.Atom.Items > maxItems {
		maxItems = b.config.Feeds.Atom.Items
	}
	if maxItems <= 0 {
		maxItems = 20
	}

	var items []feed.FeedItem
	for i, p := range blogPosts {
		if i >= maxItems {
			break
		}
		desc := p.Description
		if b.config.Feeds.RSS.FullContent {
			desc = p.Content
		}
		if desc == "" {
			desc = p.Title
		}

		absURL := strings.TrimSuffix(b.config.Site.URL, "/") + p.URL

		items = append(items, feed.FeedItem{
			Title:       p.Title,
			URL:         absURL,
			Date:        p.Frontmatter.Date,
			Description: desc,
			Categories:  p.Frontmatter.Tags,
			GUID:        absURL,
		})
	}

	// Generate RSS
	if b.config.Feeds.RSS.Enabled {
		rssPath := b.config.Feeds.RSS.Path
		if rssPath == "" {
			rssPath = "rss.xml"
		}
		rssContent, err := gen.GenerateRSS(items, rssPath)
		if err != nil {
			return fmt.Errorf("failed to generate RSS: %w", err)
		}
		outputPath := filepath.Join(b.config.Build.OutputDir, rssPath)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(outputPath, []byte(rssContent), 0644); err != nil {
			return fmt.Errorf("failed to write RSS: %w", err)
		}
		fmt.Printf("Generated RSS feed: %s (%d items)\n", rssPath, len(items))
	}

	// Generate Atom
	if b.config.Feeds.Atom.Enabled {
		atomPath := b.config.Feeds.Atom.Path
		if atomPath == "" {
			atomPath = "atom.xml"
		}
		atomContent, err := gen.GenerateAtom(items, atomPath)
		if err != nil {
			return fmt.Errorf("failed to generate Atom: %w", err)
		}
		outputPath := filepath.Join(b.config.Build.OutputDir, atomPath)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(outputPath, []byte(atomContent), 0644); err != nil {
			return fmt.Errorf("failed to write Atom: %w", err)
		}
		fmt.Printf("Generated Atom feed: %s (%d items)\n", atomPath, len(items))
	}

	return nil
}

// generateSitemap creates a sitemap.xml from all pages
func (b *SiteBuilder) generateSitemap() error {
	siteURL := strings.TrimSuffix(b.config.Site.URL, "/")

	var entries []sitemap.SitemapEntry

	for _, page := range b.pages {
		absURL := siteURL + page.URL

		var priority float64
		var changeFreq string

		switch {
		case page.URL == "/":
			priority = 1.0
			changeFreq = "weekly"
		case page.URL == "/blog/" || page.URL == "/docs/":
			priority = 0.9
			changeFreq = "weekly"
		case page.Type == TypeBlog:
			priority = 0.8
			changeFreq = "never"
		case page.Type == TypeDoc:
			priority = 0.7
			changeFreq = "monthly"
		case strings.HasPrefix(page.URL, "/tags/"):
			priority = 0.5
			changeFreq = "monthly"
		case strings.HasPrefix(page.URL, "/archive/"):
			priority = 0.3
			changeFreq = "yearly"
		default:
			priority = 0.5
			changeFreq = "monthly"
		}

		entries = append(entries, sitemap.SitemapEntry{
			URL:        absURL,
			LastMod:    page.ModifiedTime,
			ChangeFreq: changeFreq,
			Priority:   priority,
		})
	}

	// Sort entries by priority descending for readability
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Priority > entries[j].Priority
	})

	sitemapContent, err := sitemap.Generate(entries)
	if err != nil {
		return fmt.Errorf("failed to generate sitemap: %w", err)
	}

	outputPath := filepath.Join(b.config.Build.OutputDir, "sitemap.xml")
	if err := os.WriteFile(outputPath, []byte(sitemapContent), 0644); err != nil {
		return fmt.Errorf("failed to write sitemap: %w", err)
	}

	fmt.Printf("Generated sitemap.xml (%d URLs)\n", len(entries))
	return nil
}

// generateGraph creates the graph visualization page and JSON data
func (b *SiteBuilder) generateGraph() error {
	// Convert pages to PageInfo for graph builder
	var pageInfos []graph.PageInfo
	for _, page := range b.pages {
		pageType := string(page.Type)
		if pageType == "" {
			pageType = "page"
		}
		var tags []string
		if page.Frontmatter != nil {
			tags = page.Frontmatter.Tags
		}
		pageInfos = append(pageInfos, graph.PageInfo{
			ID:         page.ID,
			Title:      page.Title,
			URL:        page.URL,
			Type:       pageType,
			Tags:       tags,
			RawContent: page.RawContent,
			// WHY: 3D graph places articles by embedding PCA, not link forces.
			Vector: b.pageEmbeddingVector(page),
		})
	}

	// Build graph data
	graphData := graph.BuildGraph(pageInfos)

	// Write graph.json
	if err := graph.WriteGraphJSON(graphData, b.config.Build.OutputDir); err != nil {
		return fmt.Errorf("failed to write graph JSON: %w", err)
	}

	// Write graph HTML page
	if err := graph.WriteGraphPage(b.config.Build.OutputDir, b.config.Site.Title); err != nil {
		return fmt.Errorf("failed to write graph page: %w", err)
	}

	// Mirror graph endpoints under language prefixes for multilingual sites.
	if len(b.config.I18n.Languages) > 1 {
		for _, lang := range b.config.I18n.Languages {
			code := strings.ToLower(strings.TrimSpace(lang.Code))
			if code == "" || b.isDefaultLanguage(code) {
				continue
			}
			graphDir := filepath.Join(b.config.Build.OutputDir, code, "graph")
			if err := os.MkdirAll(graphDir, 0755); err != nil {
				return fmt.Errorf("failed to create language graph dir: %w", err)
			}

			rootGraphHTML := filepath.Join(b.config.Build.OutputDir, "graph", "index.html")
			htmlData, err := os.ReadFile(rootGraphHTML)
			if err != nil {
				return fmt.Errorf("failed to read root graph HTML: %w", err)
			}
			if err := os.WriteFile(filepath.Join(graphDir, "index.html"), htmlData, 0644); err != nil {
				return fmt.Errorf("failed to write language graph HTML: %w", err)
			}

			rootGraphJSON := filepath.Join(b.config.Build.OutputDir, "graph.json")
			jsonData, err := os.ReadFile(rootGraphJSON)
			if err != nil {
				return fmt.Errorf("failed to read root graph JSON: %w", err)
			}
			if err := os.WriteFile(filepath.Join(b.config.Build.OutputDir, code, "graph.json"), jsonData, 0644); err != nil {
				return fmt.Errorf("failed to write language graph JSON: %w", err)
			}
		}
	}

	fmt.Printf("Generated graph view (%d nodes, %d edges)\n", len(graphData.Nodes), len(graphData.Edges))
	return nil
}

// pageEmbeddingVector returns a decoded frontmatter embedding for graph layout.
func (b *SiteBuilder) pageEmbeddingVector(page *Page) []float32 {
	if page == nil || page.Frontmatter == nil || page.Frontmatter.Embedding == nil {
		return nil
	}
	embedding := page.Frontmatter.Embedding
	if embedding.Version != 1 || embedding.Model != b.config.Related.Model || embedding.Dimensions != b.config.Related.Dimensions {
		return nil
	}
	if embedding.Vector == "" || embedding.Scale <= 0 {
		return nil
	}
	vec, err := related.DecodeVector(related.CacheEntry{
		Hash:  embedding.Hash,
		Vec:   embedding.Vector,
		Scale: embedding.Scale,
	}, embedding.Dimensions)
	if err != nil {
		return nil
	}
	return vec
}

