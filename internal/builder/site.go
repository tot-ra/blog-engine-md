package builder

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tot-ra/blog-engine/internal/archive"
	"github.com/tot-ra/blog-engine/internal/assets"
	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/feed"
	"github.com/tot-ra/blog-engine/internal/parser"
	"github.com/tot-ra/blog-engine/internal/renderer"
	"github.com/tot-ra/blog-engine/internal/sitemap"
	"github.com/tot-ra/blog-engine/internal/tags"
)

// SiteBuilder orchestrates the site building process
type SiteBuilder struct {
	config          *config.SiteConfig
	templates       *renderer.TemplateEngine
	pages           map[string]*Page
	navTree         *NavTree
	processedImages []*assets.ProcessedImage
	cssBundle       *assets.CSSBundle
	jsBundle        *assets.JSBundle
}

// NewSiteBuilder creates a new site builder
func NewSiteBuilder(cfg *config.SiteConfig) *SiteBuilder {
	return &SiteBuilder{
		config: cfg,
		pages:  make(map[string]*Page),
	}
}

// Build performs the complete site build
func (b *SiteBuilder) Build() error {
	// Load templates
	b.templates = renderer.NewTemplateEngine()
	if err := b.templates.LoadTemplates("templates"); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Discover content
	index, err := Discover(b.config.Build.ContentDir)
	if err != nil {
		return fmt.Errorf("failed to discover content: %w", err)
	}

	fmt.Printf("Found %d markdown files, %d images, %d assets\n",
		len(index.MarkdownFiles), len(index.ImageFiles), len(index.AssetFiles))

	// Build pages
	pageBuilder := NewPageBuilder(b.config.Site.URL)
	for _, file := range index.MarkdownFiles {
		page, err := pageBuilder.Build(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building page %s: %v\n", file.Path, err)
			continue
		}
		if page == nil {
			continue // Draft or skipped
		}
		b.pages[page.ID] = page
	}

	fmt.Printf("Built %d pages\n", len(b.pages))

	// Build navigation tree
	navBuilder := NewNavigationBuilder()
	b.navTree = navBuilder.BuildTree(b.pages)

	// Generate section index pages for sections without explicit index
	sectionGen := NewSectionIndexGenerator()
	indexPages := sectionGen.GenerateMissing(b.pages, b.navTree)
	for _, page := range indexPages {
		b.pages[page.ID] = page
	}
	if len(indexPages) > 0 {
		fmt.Printf("Generated %d section index pages\n", len(indexPages))
		// Rebuild nav tree with new index pages
		b.navTree = navBuilder.BuildTree(b.pages)
	}

	// Create output directory
	if err := os.MkdirAll(b.config.Build.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Render pages
	for _, page := range b.pages {
		if err := b.renderPage(page); err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering page %s: %v\n", page.URL, err)
			continue
		}
	}

	// Process images
	if b.config.Assets.Images.Enabled && len(index.ImageFiles) > 0 {
		if err := b.processImages(index); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing images: %v\n", err)
		}
	}

	// Process CSS
	if b.config.Assets.CSS.Enabled {
		var cssFiles []string
		for _, f := range index.AssetFiles {
			if strings.HasSuffix(f.Path, ".css") {
				cssFiles = append(cssFiles, f.Path)
			}
		}
		if len(cssFiles) > 0 {
			proc := assets.NewCSSProcessor(b.config.Assets.CSS.Minify)
			bundle, err := proc.Process(cssFiles, b.config.Build.OutputDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error processing CSS: %v\n", err)
			} else {
				b.cssBundle = bundle
				fmt.Printf("CSS bundle: %d bytes\n", bundle.Size)
			}
		}
	}

	// Process JS
	if b.config.Assets.JS.Enabled {
		var jsFiles []string
		for _, f := range index.AssetFiles {
			if strings.HasSuffix(f.Path, ".js") {
				jsFiles = append(jsFiles, f.Path)
			}
		}
		proc := assets.NewJSProcessor(b.config.Assets.JS.Minify)
		bundle, err := proc.Process(jsFiles, b.config.Build.OutputDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing JS: %v\n", err)
		} else {
			b.jsBundle = bundle
			fmt.Printf("JS bundle: %d bytes\n", bundle.Size)
		}
	}

	// Transform image references in rendered HTML
	if len(b.processedImages) > 0 {
		transformer := assets.NewImageTransformer(b.processedImages)
		if err := b.transformRenderedPages(transformer); err != nil {
			fmt.Fprintf(os.Stderr, "Error transforming images: %v\n", err)
		}
	}

	// Copy remaining static assets (non-image, non-CSS/JS)
	if err := b.copyAssets(index); err != nil {
		return fmt.Errorf("failed to copy assets: %w", err)
	}

	// Collect blog posts sorted by date descending (used by tags, archive, feeds)
	blogPosts := b.collectBlogPosts()

	// Generate tag pages
	if b.config.Tags.Enabled {
		if err := b.generateTagPages(blogPosts); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating tag pages: %v\n", err)
		}
	}

	// Generate archive pages
	if b.config.Archive.Enabled {
		if err := b.generateArchivePages(blogPosts); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating archive pages: %v\n", err)
		}
	}

	// Generate RSS/Atom feeds
	if b.config.Feeds.RSS.Enabled || b.config.Feeds.Atom.Enabled {
		if err := b.generateFeeds(blogPosts); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating feeds: %v\n", err)
		}
	}

	// Generate sitemap
	if b.config.Sitemap.Enabled {
		if err := b.generateSitemap(); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating sitemap: %v\n", err)
		}
	}

	return nil
}

// processImages processes all discovered image files
func (b *SiteBuilder) processImages(index *ContentIndex) error {
	cacheDir := b.config.Assets.Cache.Directory
	if cacheDir == "" {
		cacheDir = ".cache"
	}

	var cache *assets.ImageCache
	if b.config.Assets.Cache.Enabled {
		cache = assets.NewImageCache(cacheDir)
	}

	imgConfig := assets.ImageConfig{
		Quality: b.config.Assets.Images.Quality,
		Sizes:   b.config.Assets.Images.Sizes,
		Enabled: true,
	}

	processor := assets.NewImageProcessor(imgConfig, b.config.Build.OutputDir, cache)

	// Convert to FileInfo slice
	files := make([]assets.FileInfo, 0, len(index.ImageFiles))
	for _, f := range index.ImageFiles {
		files = append(files, assets.FileInfo{
			Path:         f.Path,
			RelativePath: f.RelativePath,
			ModTime:      f.ModifiedTime,
			Size:         f.Size,
		})
	}

	images, errs := processor.ProcessBatch(files, b.config.Build.ParallelWorkers)
	for _, err := range errs {
		fmt.Fprintf(os.Stderr, "Image processing error: %v\n", err)
	}

	b.processedImages = images
	fmt.Printf("Processed %d images\n", len(images))

	// Save cache
	if cache != nil {
		if err := cache.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving image cache: %v\n", err)
		}
	}

	return nil
}

// transformRenderedPages re-reads rendered HTML files and transforms image tags
func (b *SiteBuilder) transformRenderedPages(transformer *assets.ImageTransformer) error {
	for _, page := range b.pages {
		outputPath := filepath.Join(b.config.Build.OutputDir, page.URL)
		if !strings.HasSuffix(outputPath, ".html") {
			outputPath = filepath.Join(outputPath, "index.html")
		}

		data, err := os.ReadFile(outputPath)
		if err != nil {
			continue
		}

		transformed := transformer.Transform(string(data))
		if transformed != string(data) {
			if err := os.WriteFile(outputPath, []byte(transformed), 0644); err != nil {
				return fmt.Errorf("failed to write transformed page %s: %w", outputPath, err)
			}
		}
	}
	return nil
}

// convertNavNode converts a builder NavNode to a renderer NavNode (avoids circular deps)
func convertNavNode(node *NavNode) *renderer.NavNode {
	if node == nil {
		return nil
	}
	rn := &renderer.NavNode{
		ID:       node.ID,
		Title:    node.Title,
		URL:      node.URL,
		Order:    node.Order,
		Hidden:   node.Hidden,
		Type:     node.Type,
		Children: make([]*renderer.NavNode, 0, len(node.Children)),
	}
	for _, child := range node.Children {
		rn.Children = append(rn.Children, convertNavNode(child))
	}
	return rn
}

// renderPage renders a single page to HTML
func (b *SiteBuilder) renderPage(page *Page) error {
	// Prepare base data
	data := renderer.PageData{
		Site: *b.config,
		Page: renderer.Page{
			ID:           page.ID,
			URL:          page.URL,
			Title:        page.Title,
			Description:  page.Description,
			Content:      page.Content,
			Type:         string(page.Type),
			ModifiedTime: page.ModifiedTime,
		},
		Frontmatter: renderer.Frontmatter{
			Date: page.Frontmatter.Date,
			Tags: page.Frontmatter.Tags,
		},
		Content: template.HTML(page.Content),
	}

	// Generate sidebar
	rendererRoot := convertNavNode(b.navTree.Root)
	data.Sidebar = renderer.RenderSidebar(rendererRoot, page.URL, b.config.Navigation.Sidebar.MaxDepth)

	// Generate TOC (if page has headings and hideToc is not set)
	if len(page.TOC) > 0 && (page.Frontmatter == nil || !page.Frontmatter.HideToc) {
		tocItems := convertTocItems(page.TOC)
		data.TOC = renderer.RenderTOC(tocItems)
	}

	// Generate breadcrumbs
	if b.config.Navigation.Breadcrumbs.Enabled {
		bcGen := NewBreadcrumbGenerator(b.config.Navigation.Breadcrumbs.HomeLabel)
		builderCrumbs := bcGen.Generate(page, b.navTree)
		data.Breadcrumbs = convertBreadcrumbs(builderCrumbs)
	}

	// Generate prev/next links
	if b.config.Navigation.PrevNext.Enabled {
		pnGen := NewPrevNextGenerator()
		links := pnGen.Generate(page, b.pages, b.navTree)
		if links != nil {
			data.PrevNext = convertPrevNext(links)
		}
	}

	// Render template
	html, err := b.templates.RenderPage(data)
	if err != nil {
		return err
	}

	// Determine output path
	outputPath := filepath.Join(b.config.Build.OutputDir, page.URL)
	if !strings.HasSuffix(outputPath, ".html") {
		outputPath = filepath.Join(outputPath, "index.html")
	}

	// Create directory
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// convertTocItems converts builder TocItems to renderer TocItems
func convertTocItems(items []*TocItem) []*renderer.TocItem {
	result := make([]*renderer.TocItem, 0, len(items))
	for _, item := range items {
		ri := &renderer.TocItem{
			Level:    item.Level,
			Text:     item.Text,
			Anchor:   item.Anchor,
			Children: convertTocItems(item.Children),
		}
		result = append(result, ri)
	}
	return result
}

// convertBreadcrumbs converts builder BreadcrumbItems to renderer BreadcrumbItems
func convertBreadcrumbs(items []BreadcrumbItem) []renderer.BreadcrumbItem {
	result := make([]renderer.BreadcrumbItem, 0, len(items))
	for _, item := range items {
		result = append(result, renderer.BreadcrumbItem{
			Title:     item.Title,
			URL:       item.URL,
			IsCurrent: item.IsCurrent,
		})
	}
	return result
}

// convertPrevNext converts builder PrevNextLinks to renderer PrevNextLinks
func convertPrevNext(links *PrevNextLinks) *renderer.PrevNextLinks {
	result := &renderer.PrevNextLinks{}
	if links.Prev != nil {
		result.Prev = &renderer.NavLink{
			Title: links.Prev.Title,
			URL:   links.Prev.URL,
			Type:  links.Prev.Type,
		}
	}
	if links.Next != nil {
		result.Next = &renderer.NavLink{
			Title: links.Next.Title,
			URL:   links.Next.URL,
			Type:  links.Next.Type,
		}
	}
	return result
}

// copyAssets copies static assets to output directory (non-image, non-CSS/JS files)
func (b *SiteBuilder) copyAssets(index *ContentIndex) error {
	for _, file := range index.AssetFiles {
		// Skip CSS/JS files (already processed)
		ext := strings.ToLower(filepath.Ext(file.Path))
		if ext == ".css" || ext == ".js" {
			continue
		}

		// Determine output path
		outputPath := filepath.Join(b.config.Build.OutputDir, "assets", file.RelativePath)

		// Create directory
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Read source
		data, err := os.ReadFile(file.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading asset %s: %v\n", file.Path, err)
			continue
		}

		// Write to output
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write asset: %w", err)
		}
	}

	return nil
}

// collectBlogPosts returns all blog pages sorted by date descending
func (b *SiteBuilder) collectBlogPosts() []*Page {
	var posts []*Page
	for _, page := range b.pages {
		if page.Type == TypeBlog {
			posts = append(posts, page)
		}
	}
	sort.Slice(posts, func(i, j int) bool {
		di := posts[i].Frontmatter.Date
		dj := posts[j].Frontmatter.Date
		return di.After(dj)
	})
	return posts
}

// pageSummariesFromPosts converts builder Pages to tags.PageSummary slice
func pageSummariesFromPosts(posts []*Page) []tags.PageSummary {
	summaries := make([]tags.PageSummary, 0, len(posts))
	for _, p := range posts {
		summaries = append(summaries, tags.PageSummary{
			Title:       p.Title,
			URL:         p.URL,
			Date:        p.Frontmatter.Date,
			Description: p.Description,
			Tags:        p.Frontmatter.Tags,
			Type:        string(p.Type),
		})
	}
	return summaries
}

// generateTagPages builds tag index and creates tag list pages
func (b *SiteBuilder) generateTagPages(blogPosts []*Page) error {
	summaries := pageSummariesFromPosts(blogPosts)
	tagIdx := tags.BuildTagIndex(summaries)

	allTags := tagIdx.Tags()
	if len(allTags) == 0 {
		return nil
	}

	// Generate tag cloud / index page at /tags/
	tagCloudHTML := b.buildTagCloudHTML(tagIdx, allTags)
	tagCloudPage := &Page{
		ID:          "tags",
		URL:         "/tags/",
		Title:       "Tags",
		Description: "All tags",
		Content:     tagCloudHTML,
		RawContent:  "",
		Frontmatter: &parser.Frontmatter{},
		Type:        TypePage,
	}
	b.pages[tagCloudPage.ID] = tagCloudPage
	if err := b.renderPage(tagCloudPage); err != nil {
		return fmt.Errorf("failed to render tag cloud page: %w", err)
	}
	fmt.Printf("Generated tag cloud page with %d tags\n", len(allTags))

	// Generate individual tag pages
	for _, tag := range allTags {
		pages := tagIdx[tag]
		tagSlug := parser.GenerateSlug(tag)
		tagPageHTML := b.buildTagPageHTML(tag, pages)

		tagPage := &Page{
			ID:          "tags-" + tagSlug,
			URL:         "/tags/" + tagSlug + "/",
			Title:       "Tag: " + tag,
			Description: fmt.Sprintf("Pages tagged with \"%s\"", tag),
			Content:     tagPageHTML,
			RawContent:  "",
			Frontmatter: &parser.Frontmatter{Tags: []string{tag}},
			Type:        TypePage,
		}
		b.pages[tagPage.ID] = tagPage
		if err := b.renderPage(tagPage); err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering tag page %s: %v\n", tag, err)
			continue
		}
	}
	fmt.Printf("Generated %d tag pages\n", len(allTags))

	return nil
}

// buildTagCloudHTML generates HTML for the tag cloud page
func (b *SiteBuilder) buildTagCloudHTML(idx tags.TagIndex, allTags []string) string {
	var sb strings.Builder
	sb.WriteString("<div class=\"tag-cloud\">\n")
	sb.WriteString("<ul class=\"tag-list\">\n")
	for _, tag := range allTags {
		slug := parser.GenerateSlug(tag)
		count := idx.Count(tag)
		sb.WriteString(fmt.Sprintf("  <li><a href=\"/tags/%s/\" class=\"tag\">%s</a> <span class=\"tag-count\">(%d)</span></li>\n", slug, tag, count))
	}
	sb.WriteString("</ul>\n")
	sb.WriteString("</div>\n")
	return sb.String()
}

// buildTagPageHTML generates HTML for a single tag page
func (b *SiteBuilder) buildTagPageHTML(tag string, pages []tags.PageSummary) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h2>Posts tagged \"%s\"</h2>\n", tag))
	sb.WriteString(fmt.Sprintf("<p>%d post(s)</p>\n", len(pages)))
	sb.WriteString("<ul class=\"post-list\">\n")
	for _, p := range pages {
		dateStr := ""
		if !p.Date.IsZero() {
			dateStr = fmt.Sprintf(" <time>%s</time>", p.Date.Format("2006-01-02"))
		}
		sb.WriteString(fmt.Sprintf("  <li><a href=\"%s\">%s</a>%s</li>\n", p.URL, p.Title, dateStr))
	}
	sb.WriteString("</ul>\n")
	return sb.String()
}

// generateArchivePages builds archive structure and creates archive pages
func (b *SiteBuilder) generateArchivePages(blogPosts []*Page) error {
	// Convert to archive.PageSummary
	var summaries []archive.PageSummary
	for _, p := range blogPosts {
		summaries = append(summaries, archive.PageSummary{
			Title:       p.Title,
			URL:         p.URL,
			Date:        p.Frontmatter.Date,
			Description: p.Description,
			Tags:        p.Frontmatter.Tags,
			Type:        string(p.Type),
		})
	}

	archiveData := archive.BuildArchive(summaries)
	if len(archiveData) == 0 {
		return nil
	}

	// Generate main archive page at /archive/
	archiveHTML := b.buildArchiveIndexHTML(archiveData)
	archivePage := &Page{
		ID:          "archive",
		URL:         "/archive/",
		Title:       "Archive",
		Description: "Post archive by date",
		Content:     archiveHTML,
		RawContent:  "",
		Frontmatter: &parser.Frontmatter{},
		Type:        TypePage,
	}
	b.pages[archivePage.ID] = archivePage
	if err := b.renderPage(archivePage); err != nil {
		return fmt.Errorf("failed to render archive page: %w", err)
	}

	// Generate per-year pages
	for _, year := range archiveData {
		yearHTML := b.buildArchiveYearHTML(year)
		yearPage := &Page{
			ID:          fmt.Sprintf("archive-%d", year.Year),
			URL:         fmt.Sprintf("/archive/%d/", year.Year),
			Title:       fmt.Sprintf("Archive: %d", year.Year),
			Description: fmt.Sprintf("Posts from %d", year.Year),
			Content:     yearHTML,
			RawContent:  "",
			Frontmatter: &parser.Frontmatter{},
			Type:        TypePage,
		}
		b.pages[yearPage.ID] = yearPage
		if err := b.renderPage(yearPage); err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering archive year page %d: %v\n", year.Year, err)
		}
	}

	fmt.Printf("Generated archive pages (%d years)\n", len(archiveData))
	return nil
}

// buildArchiveIndexHTML generates HTML for the main archive page
func (b *SiteBuilder) buildArchiveIndexHTML(years []archive.ArchiveYear) string {
	var sb strings.Builder
	sb.WriteString("<div class=\"archive\">\n")
	for _, year := range years {
		sb.WriteString(fmt.Sprintf("<h2><a href=\"/archive/%d/\">%d</a> <span class=\"count\">(%d)</span></h2>\n", year.Year, year.Year, year.Count))
		for _, month := range year.Months {
			sb.WriteString(fmt.Sprintf("<h3>%s %d</h3>\n", month.Month.String(), month.Year))
			sb.WriteString("<ul class=\"post-list\">\n")
			for _, p := range month.Pages {
				dateStr := p.Date.Format("Jan 02")
				sb.WriteString(fmt.Sprintf("  <li><time>%s</time> <a href=\"%s\">%s</a></li>\n", dateStr, p.URL, p.Title))
			}
			sb.WriteString("</ul>\n")
		}
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// buildArchiveYearHTML generates HTML for a single year archive page
func (b *SiteBuilder) buildArchiveYearHTML(year archive.ArchiveYear) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<div class=\"archive-year\">\n"))
	for _, month := range year.Months {
		sb.WriteString(fmt.Sprintf("<h2>%s</h2>\n", month.Month.String()))
		sb.WriteString("<ul class=\"post-list\">\n")
		for _, p := range month.Pages {
			dateStr := p.Date.Format("Jan 02")
			sb.WriteString(fmt.Sprintf("  <li><time>%s</time> <a href=\"%s\">%s</a></li>\n", dateStr, p.URL, p.Title))
		}
		sb.WriteString("</ul>\n")
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

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
