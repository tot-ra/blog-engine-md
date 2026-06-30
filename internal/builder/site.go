package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/tot-ra/blog-engine/internal/assets"
	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
	"github.com/tot-ra/blog-engine/internal/renderer"
)

var siteFooterRe = regexp.MustCompile(`(?s)<footer class="site-footer">.*?</footer>`)
var firstImageSrcRe = regexp.MustCompile(`(?is)<img[^>]+src\s*=\s*['"]([^'"]+)['"]`)
var excerptLinkRe = regexp.MustCompile(`\[(.*?)\]\([^)]+\)`)
var markdownExtensions = map[string]struct{}{
	".md":       {},
	".markdown": {},
}

// SiteBuilder orchestrates the site building process
type SiteBuilder struct {
	config          *config.SiteConfig
	templates       *renderer.TemplateEngine
	pages           map[string]*Page
	pagesByURL      map[string]*Page
	navTree         *NavTree
	blogTimeline    map[string][]renderer.TimelineYear
	languages       map[string]struct{}
	processedImages []*assets.ProcessedImage
	cssBundle       *assets.CSSBundle
	jsBundle        *assets.JSBundle
}

// NewSiteBuilder creates a new site builder
func NewSiteBuilder(cfg *config.SiteConfig) *SiteBuilder {
	return &SiteBuilder{
		config:     cfg,
		pages:      make(map[string]*Page),
		pagesByURL: make(map[string]*Page),
		languages:  buildLanguageSet(cfg),
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

	// First pass: collect page info for wiki and local markdown link resolution
	pageBuilder := NewPageBuilder(b.config.Site.URL, b.config.I18n.Default, b.languages)
	explicitIndexDirs := collectExplicitIndexDirs(index.MarkdownFiles)
	pageBuilder.urlGen.SetExplicitIndexDirs(explicitIndexDirs)
	titleToURL := make(map[string]map[string]string)
	pathToURL := make(map[string]map[string]string)

	for _, file := range index.MarkdownFiles {
		// Quick parse to get title and URL without full rendering
		data, err := os.ReadFile(file.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file.Path, err)
			continue
		}
		fm, _, err := parser.ParseFrontmatter(string(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing frontmatter %s: %v\n", file.Path, err)
			continue
		}
		if fm.Draft {
			continue
		}

		lang, _ := detectLanguageAndContentPath(file.RelativePath, b.config.I18n.Default, b.languages)
		url := pageBuilder.urlGen.Generate(file.RelativePath, fm)
		title := fm.Title
		if title == "" {
			title = filepath.Base(file.Path)
			ext := filepath.Ext(title)
			title = strings.TrimSuffix(title, ext)
			title = strings.ReplaceAll(title, "-", " ")
			title = strings.ReplaceAll(title, "_", " ")
		}

		if _, ok := titleToURL[lang]; !ok {
			titleToURL[lang] = make(map[string]string)
		}
		if _, ok := pathToURL[lang]; !ok {
			pathToURL[lang] = make(map[string]string)
		}
		// Map both exact title and slugified title
		titleToURL[lang][title] = url
		titleToURL[lang][parser.GenerateSlug(title)] = url
		addMarkdownLinkPathAliases(pathToURL[lang], file.RelativePath, url)
	}

	// Second pass: build pages with wiki and local markdown link resolution
	pages, buildErrs := b.buildPages(index.MarkdownFiles, titleToURL, pathToURL, explicitIndexDirs)
	for _, err := range buildErrs {
		fmt.Fprintf(os.Stderr, "Error building page: %v\n", err)
	}
	for _, page := range pages {
		b.pages[page.ID] = page
		b.pagesByURL[page.URL] = page
	}

	fmt.Printf("Built %d pages\n", len(b.pages))

	// Build navigation tree
	navBuilder := NewNavigationBuilder()
	b.navTree = navBuilder.BuildTree(b.pages)

	// Generate section index pages for sections without explicit index
	sectionGen := NewSectionIndexGenerator()
	indexPages := sectionGen.GenerateMissing(b.pages, b.navTree, b.config.I18n.Default, b.languages)
	for _, page := range indexPages {
		b.pages[page.ID] = page
		b.pagesByURL[page.URL] = page
	}
	if len(indexPages) > 0 {
		fmt.Printf("Generated %d section index pages\n", len(indexPages))
		// Rebuild nav tree with generated index pages. This is especially important
		// for /blog/: once the section index exists, blog posts become children of
		// that Blog node, so individual post pages can keep the left blog menu.
		b.navTree = navBuilder.BuildTree(b.pages)
	}

	// Generate and cache audio narration for recent blog posts.
	if err := b.prepareBlogAudio(index); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating blog audio: %v\n", err)
	}

	// Reset output directory to avoid stale routes from previous builds.
	if err := b.prepareOutputDir(); err != nil {
		return err
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
			rel := filepath.ToSlash(f.RelativePath)
			// Keep ES module files (e.g. content/triangle/*.js) as standalone assets.
			if strings.HasSuffix(f.Path, ".js") && !strings.HasPrefix(rel, "triangle/") {
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

	// Precompute blog timeline data used by blog sidebar "time" mode.
	b.blogTimeline = make(map[string][]renderer.TimelineYear)
	for lang, posts := range b.collectBlogPostsByLanguage() {
		b.blogTimeline[lang] = buildBlogTimeline(posts, 20)
	}

	// Render pages (after CSS/JS processing so templates can reference bundles)
	renderPages := make([]*Page, 0, len(b.pages))
	for _, page := range b.pages {
		renderPages = append(renderPages, page)
	}
	renderErrs := b.renderPages(renderPages)
	for _, err := range renderErrs {
		fmt.Fprintf(os.Stderr, "Error rendering page: %v\n", err)
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
	if err := b.copyConfiguredLogo(); err != nil {
		return fmt.Errorf("failed to copy configured logo: %w", err)
	}
	if err := b.copyTriangleModules(); err != nil {
		return fmt.Errorf("failed to copy triangle modules: %w", err)
	}

	// Remove legacy footer blocks from generated pages.
	if err := b.stripSiteFooters(); err != nil {
		fmt.Fprintf(os.Stderr, "Error stripping site footers: %v\n", err)
	}

	// Collect blog posts sorted by date descending (used by archive, feeds)
	blogPosts := b.collectBlogPosts()

	// Generate tag pages
	if b.config.Tags.Enabled {
		if err := b.generateTagPages(b.collectTagPages()); err != nil {
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

	// Generate graph visualization
	if b.config.Advanced.Graph.Enabled {
		if err := b.generateGraph(); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating graph: %v\n", err)
		}
	}

	if err := b.ensureDefaultLanguageEntry(); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating default language entry: %v\n", err)
	}

	return nil
}

func (b *SiteBuilder) ensureDefaultLanguageEntry() error {
	if _, exists := b.pagesByURL["/"]; exists {
		return nil
	}
	html := rootRedirectHTML(b.config)
	out := filepath.Join(b.config.Build.OutputDir, "index.html")
	return os.WriteFile(out, []byte(html), 0644)
}

func (b *SiteBuilder) prepareOutputDir() error {
	out := strings.TrimSpace(b.config.Build.OutputDir)
	if out == "" || out == "." || out == "/" {
		return fmt.Errorf("unsafe output directory: %q", out)
	}
	if err := os.RemoveAll(out); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to fully clean output directory %s: %v\n", out, err)
		if clearErr := clearGeneratedOutput(out); clearErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clear generated output in %s: %v\n", out, clearErr)
		}
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

// clearGeneratedOutput removes generated site output while preserving assets if full cleanup fails.
func clearGeneratedOutput(out string) error {
	entries, err := os.ReadDir(out)
	if err != nil {
		// Directory may already be gone.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "assets" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(out, name)); err != nil {
			return err
		}
	}
	return nil
}

func (b *SiteBuilder) stripSiteFooters() error {
	return filepath.Walk(b.config.Build.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".html") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		updated := siteFooterRe.ReplaceAll(data, []byte(""))
		if string(updated) == string(data) {
			return nil
		}
		return os.WriteFile(path, updated, 0644)
	})
}

func (b *SiteBuilder) workerCount() int {
	if b.config.Build.ParallelWorkers <= 0 {
		return 1
	}
	return b.config.Build.ParallelWorkers
}

func (b *SiteBuilder) parallelForEach(total int, fn func(i int) error) []error {
	if total == 0 {
		return nil
	}

	workers := b.workerCount()
	if workers > total {
		workers = total
	}
	if workers <= 1 {
		var errs []error
		for i := 0; i < total; i++ {
			if err := fn(i); err != nil {
				errs = append(errs, err)
			}
		}
		return errs
	}

	jobs := make(chan int, total)
	errCh := make(chan error, total)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if err := fn(i); err != nil {
					errCh <- err
				}
			}
		}()
	}

	for i := 0; i < total; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errs
}

func (b *SiteBuilder) renderPages(pages []*Page) []error {
	return b.parallelForEach(len(pages), func(i int) error {
		page := pages[i]
		if err := b.renderPage(page); err != nil {
			return fmt.Errorf("%s: %w", page.URL, err)
		}
		return nil
	})
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
		Quality:          b.config.Assets.Images.Quality,
		Sizes:            b.config.Assets.Images.Sizes,
		Enabled:          true,
		MaxSourcePixels:  b.config.Assets.Images.MaxSourcePixels,
		MaxVariantPixels: b.config.Assets.Images.MaxVariantPixels,
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

	workers := b.config.Assets.Images.ParallelWorkers
	if workers <= 0 {
		workers = b.config.Build.ParallelWorkers
	}
	if workers <= 0 {
		workers = 4
	}

	fmt.Printf("Processing %d images with %d worker(s)\n", len(files), workers)
	images, errs := processor.ProcessBatch(files, assets.BatchOptions{
		Workers: workers,
		Logf: func(format string, args ...any) {
			fmt.Printf(format+"\n", args...)
		},
		LogEvery:   25,
		ProgressID: "images",
	})
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
	pages := make([]*Page, 0, len(b.pages))
	for _, page := range b.pages {
		pages = append(pages, page)
	}

	errs := b.parallelForEach(len(pages), func(i int) error {
		page := pages[i]
		outputPath := filepath.Join(b.config.Build.OutputDir, page.URL)
		if !strings.HasSuffix(outputPath, ".html") {
			outputPath = filepath.Join(outputPath, "index.html")
		}

		data, err := os.ReadFile(outputPath)
		if err != nil {
			return nil
		}

		transformed := transformer.Transform(string(data))
		if transformed != string(data) {
			if err := os.WriteFile(outputPath, []byte(transformed), 0644); err != nil {
				return fmt.Errorf("failed to write transformed page %s: %w", outputPath, err)
			}
		}
		return nil
	})
	if len(errs) > 0 {
		return errs[0]
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
func selectSidebarRoot(root *renderer.NavNode, currentPath string, languages map[string]struct{}) *renderer.NavNode {
	if root == nil {
		return nil
	}
	trimmed := strings.Trim(currentPath, "/")
	if trimmed == "" {
		return root
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		return root
	}

	sectionIndex := 0
	langRootURL := ""
	// In multilingual sites URLs look like /en/blog/post/. In single-language
	// sites they look like /blog/post/. Distinguishing those shapes keeps blog
	// posts attached to the Blog sidebar instead of selecting the post leaf.
	if _, ok := languages[strings.ToLower(parts[0])]; ok && len(parts) > 1 {
		sectionIndex = 1
		langRootURL = "/" + parts[0] + "/"
	}

	sectionURL := "/" + strings.Join(parts[:sectionIndex+1], "/") + "/"

	for _, child := range root.Children {
		if child.URL == sectionURL {
			return child
		}
		if langRootURL != "" && child.URL == langRootURL {
			for _, nested := range child.Children {
				if nested.URL == sectionURL {
					return nested
				}
			}
		}
	}
	return root
}
