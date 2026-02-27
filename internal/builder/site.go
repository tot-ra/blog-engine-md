package builder

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/tot-ra/blog-engine/internal/assets"
	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/renderer"
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
