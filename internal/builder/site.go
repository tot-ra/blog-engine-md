package builder

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/tot-ra/blog-engine/internal/archive"
	"github.com/tot-ra/blog-engine/internal/assets"
	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/feed"
	"github.com/tot-ra/blog-engine/internal/graph"
	"github.com/tot-ra/blog-engine/internal/i18n"
	"github.com/tot-ra/blog-engine/internal/parser"
	"github.com/tot-ra/blog-engine/internal/renderer"
	"github.com/tot-ra/blog-engine/internal/sitemap"
	"github.com/tot-ra/blog-engine/internal/tags"
)

var siteFooterRe = regexp.MustCompile(`(?s)<footer class="site-footer">.*?</footer>`)
var firstImageSrcRe = regexp.MustCompile(`(?is)<img[^>]+src\s*=\s*['"]([^'"]+)['"]`)
var excerptLinkRe = regexp.MustCompile(`\[(.*?)\]\([^)]+\)`)

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

	// First pass: collect page info for wiki link resolution
	pageBuilder := NewPageBuilder(b.config.Site.URL, b.config.I18n.Default, b.languages)
	titleToURL := make(map[string]map[string]string)

	for _, file := range index.MarkdownFiles {
		// Quick parse to get title and URL without full rendering
		content, err := readFile(file.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file.Path, err)
			continue
		}
		fm, _, err := parser.ParseFrontmatter(content)
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
		// Map both exact title and slugified title
		titleToURL[lang][title] = url
		titleToURL[lang][parser.GenerateSlug(title)] = url
	}

	// Set up wiki link resolver
	defaultLangMap := titleToURL[b.config.I18n.Default]
	pageBuilder.SetPageResolver(func(title string) (string, bool) {
		// Try exact match first
		if url, ok := defaultLangMap[title]; ok {
			return url, true
		}
		// Try slugified version
		slug := parser.GenerateSlug(title)
		if url, ok := defaultLangMap[slug]; ok {
			return url, true
		}
		return "", false
	})

	// Second pass: build pages with wiki link resolution
	pages, buildErrs := b.buildPages(index.MarkdownFiles, titleToURL)
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
		// Rebuild nav tree with new index pages
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
	target := "/" + strings.Trim(b.config.I18n.Default, "/") + "/"
	html := "<!doctype html><html><head><meta charset=\"utf-8\"><meta http-equiv=\"refresh\" content=\"0; url=" + target + "\"><link rel=\"canonical\" href=\"" + target + "\"></head><body><a href=\"" + target + "\">Redirecting...</a></body></html>"
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

func (b *SiteBuilder) buildPages(files []ContentFile, titleToURL map[string]map[string]string) ([]*Page, []error) {
	if len(files) == 0 {
		return nil, nil
	}

	pages := make([]*Page, len(files))
	workers := b.workerCount()
	if workers > len(files) {
		workers = len(files)
	}

	type buildJob struct {
		idx  int
		file ContentFile
	}

	jobs := make(chan buildJob, len(files))
	errCh := make(chan error, len(files))
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		builder := NewPageBuilder(b.config.Site.URL, b.config.I18n.Default, b.languages)

		wg.Add(1)
		go func(pb *PageBuilder) {
			defer wg.Done()
			for job := range jobs {
				lang, _ := detectLanguageAndContentPath(job.file.RelativePath, b.config.I18n.Default, b.languages)
				pageMap := titleToURL[lang]
				pb.SetPageResolver(func(title string) (string, bool) {
					if url, ok := pageMap[title]; ok {
						return url, true
					}
					slug := parser.GenerateSlug(title)
					if url, ok := pageMap[slug]; ok {
						return url, true
					}
					return "", false
				})
				page, err := pb.Build(job.file)
				if err != nil {
					errCh <- fmt.Errorf("%s: %w", job.file.Path, err)
					continue
				}
				pages[job.idx] = page
			}
		}(builder)
	}

	for i, file := range files {
		jobs <- buildJob{idx: i, file: file}
	}
	close(jobs)
	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	result := make([]*Page, 0, len(files))
	for _, page := range pages {
		if page != nil {
			result = append(result, page)
		}
	}
	return result, errs
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

func selectSidebarRoot(root *renderer.NavNode, currentPath string) *renderer.NavNode {
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
	sectionURL := "/" + parts[0] + "/"
	if len(parts) > 1 && (parts[1] == "blog" || parts[1] == "docs" || parts[1] == "tags" || parts[1] == "archive") {
		sectionURL = "/" + parts[0] + "/" + parts[1] + "/"
	}
	for _, child := range root.Children {
		if child.URL == sectionURL {
			return child
		}
		if len(parts) > 1 && child.URL == "/"+parts[0]+"/" {
			for _, nested := range child.Children {
				if nested.URL == sectionURL {
					return nested
				}
			}
		}
	}
	return root
}

// renderPage renders a single page to HTML
func (b *SiteBuilder) renderPage(page *Page) error {
	if page.Language == "" {
		page.Language = b.detectLanguageFromURL(page.URL)
	}
	if b.config.Audio.Enabled && page.Type == TypeBlog && page.AudioURL == "" {
		absAudioPath, audioURL, err := b.audioFilePathAndURL(page)
		if err == nil && fileExistsNonEmpty(absAudioPath) {
			page.AudioURL = audioURL
		}
	}
	ui := i18n.UI(page.Language)

	// Prepare base data
	data := renderer.PageData{
		Site: *b.config,
		Page: renderer.Page{
			ID:           page.ID,
			URL:          page.URL,
			Language:     page.Language,
			Title:        page.Title,
			Description:  page.Description,
			Content:      page.Content,
			AudioURL:     page.AudioURL,
			Type:         string(page.Type),
			ModifiedTime: page.ModifiedTime,
			Layout:       page.Frontmatter.Layout,
		},
		Frontmatter: renderer.Frontmatter{
			Date: page.Frontmatter.Date,
			Tags: page.Frontmatter.Tags,
		},
		UI:      ui,
		Content: template.HTML(page.Content),
	}
	data.CanonicalURL = b.absolutePageURL(page.URL)
	data.OpenGraphType = "website"
	if page.Type == TypeBlog {
		data.OpenGraphType = "article"
	}
	data.SocialImageURL = b.resolveSocialImageURL(page)
	data.SocialCard = "summary"
	if data.SocialImageURL != "" {
		data.SocialCard = "summary_large_image"
	}
	data.MetaDescription = b.metaDescriptionForPage(page)
	data.Site.Site.Language = page.Language
	data.Homepage = b.homepageForLanguage(page.Language)
	if b.cssBundle != nil {
		data.CSSPath = b.cssBundle.Path
	}
	if b.jsBundle != nil {
		data.JSPath = b.jsBundle.Path
	}

	data.HeaderNav = b.buildHeaderNav(page.Language)
	data.Languages = b.buildLanguageOptions(page)

	// Generate sidebar
	rendererRoot := convertNavNode(b.navTree.Root)
	sidebarRoot := selectSidebarRoot(rendererRoot, page.URL)
	if strings.Contains(page.URL, "/blog/") {
		timeline := b.blogTimeline[page.Language]
		graphURL := fmt.Sprintf("/%s/graph/", page.Language)
		data.Sidebar = renderer.RenderBlogSidebar(sidebarRoot, page.URL, b.config.Navigation.Sidebar.MaxDepth, b.config.Navigation.Sidebar.Collapsed, timeline, ui, graphURL)
	} else {
		data.Sidebar = renderer.RenderSidebar(sidebarRoot, page.URL, b.config.Navigation.Sidebar.MaxDepth, b.config.Navigation.Sidebar.Collapsed, ui)
	}

	// Generate TOC (if page has headings and hideToc is not set)
	if len(page.TOC) > 0 && (page.Frontmatter == nil || !page.Frontmatter.HideToc) {
		tocItems := convertTocItems(page.TOC)
		data.TOC = renderer.RenderTOC(tocItems, ui)
	}

	// Generate breadcrumbs
	if b.config.Navigation.Breadcrumbs.Enabled {
		bcGen := NewBreadcrumbGenerator(b.languages)
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
	html = b.localizeInternalLinks(html, page.Language)
	html = siteFooterRe.ReplaceAllString(html, "")

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

func (b *SiteBuilder) homepageForLanguage(lang string) config.HomepageConfig {
	base := b.config.Homepage
	code := strings.ToLower(strings.TrimSpace(lang))
	if code != "" && b.config.HomepageI18n != nil {
		if hp, ok := b.config.HomepageI18n[code]; ok {
			return mergeHomepageConfig(base, hp)
		}
	}
	return base
}

func mergeHomepageConfig(base, override config.HomepageConfig) config.HomepageConfig {
	out := base

	if override.Enabled {
		out.Enabled = true
	}

	if override.Hero.Enabled {
		out.Hero.Enabled = true
	}
	if override.Hero.Background != "" {
		out.Hero.Background = override.Hero.Background
	}
	if override.Hero.Title != "" {
		out.Hero.Title = override.Hero.Title
	}
	if override.Hero.Subtitle != "" {
		out.Hero.Subtitle = override.Hero.Subtitle
	}
	if override.Hero.Description != "" {
		out.Hero.Description = override.Hero.Description
	}
	if override.Hero.VideoEmbed != "" {
		out.Hero.VideoEmbed = override.Hero.VideoEmbed
	}
	if len(override.Hero.CTAButtons) > 0 {
		out.Hero.CTAButtons = override.Hero.CTAButtons
	}

	if override.Chat.Enabled {
		out.Chat.Enabled = true
	}
	if override.Chat.BaseURL != "" {
		out.Chat.BaseURL = override.Chat.BaseURL
	}
	if override.Chat.RecipientAgentID != "" {
		out.Chat.RecipientAgentID = override.Chat.RecipientAgentID
	}
	if override.Chat.Title != "" {
		out.Chat.Title = override.Chat.Title
	}

	if len(override.Projects) > 0 {
		out.Projects = override.Projects
	}
	if len(override.SocialLinks) > 0 {
		out.SocialLinks = override.SocialLinks
	}
	if override.CustomHTML != "" {
		out.CustomHTML = override.CustomHTML
	}

	return out
}

func (b *SiteBuilder) detectLanguageFromURL(pageURL string) string {
	trimmed := strings.Trim(pageURL, "/")
	if trimmed == "" {
		return b.config.I18n.Default
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) > 0 {
		first := strings.ToLower(parts[0])
		if _, ok := b.languages[first]; ok {
			return first
		}
	}
	return b.config.I18n.Default
}

func (b *SiteBuilder) absolutePageURL(pagePath string) string {
	if pagePath == "" {
		pagePath = "/"
	}
	if strings.HasPrefix(pagePath, "http://") || strings.HasPrefix(pagePath, "https://") {
		return pagePath
	}
	base := strings.TrimSuffix(strings.TrimSpace(b.config.Site.URL), "/")
	if base == "" {
		return pagePath
	}
	if strings.HasPrefix(pagePath, "/") {
		return base + pagePath
	}
	return base + "/" + pagePath
}

func (b *SiteBuilder) resolveSocialImageURL(page *Page) string {
	if page != nil && page.Type == TypeBlog {
		if src := extractFirstImageSrc(page.Content); src != "" {
			if resolved := b.resolveAssetURL(src, page.URL); resolved != "" {
				if optimized := b.resolveProcessedSocialImageURL(src); optimized != "" {
					return optimized
				}
				return resolved
			}
		}
	}
	defaultSrc := strings.TrimSpace(b.config.SEO.DefaultImage)
	if optimized := b.resolveProcessedSocialImageURL(defaultSrc); optimized != "" {
		return optimized
	}
	return b.resolveAssetURL(defaultSrc, page.URL)
}

func (b *SiteBuilder) resolveAssetURL(src, pageURL string) string {
	src = html.UnescapeString(strings.TrimSpace(src))
	if src == "" {
		return ""
	}
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return src
	}

	baseURL := strings.TrimSpace(b.config.Site.URL)
	if strings.HasPrefix(src, "//") {
		siteURL, err := url.Parse(baseURL)
		if err != nil || siteURL.Scheme == "" {
			return "https:" + src
		}
		return siteURL.Scheme + ":" + src
	}

	if strings.HasPrefix(src, "/") {
		if baseURL == "" {
			return src
		}
		return strings.TrimSuffix(baseURL, "/") + src
	}

	pageAbs := b.absolutePageURL(pageURL)
	base, err := url.Parse(pageAbs)
	if err != nil {
		return src
	}
	ref, err := url.Parse(src)
	if err != nil {
		return src
	}
	return base.ResolveReference(ref).String()
}

func extractFirstImageSrc(htmlContent string) string {
	matches := firstImageSrcRe.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		return html.UnescapeString(strings.TrimSpace(matches[1]))
	}
	return ""
}

func (b *SiteBuilder) resolveProcessedSocialImageURL(src string) string {
	if len(b.processedImages) == 0 {
		return ""
	}
	candidates := processedImageLookupCandidates(src, strings.TrimSpace(b.config.Site.URL))
	if len(candidates) == 0 {
		return ""
	}

	for _, img := range b.processedImages {
		if img == nil {
			continue
		}
		rel := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(img.RelativePath)), "/")
		if rel == "" {
			continue
		}
		base := filepath.Base(rel)
		for _, candidate := range candidates {
			if candidate == rel || candidate == base {
				v := pickSocialImageVariant(img)
				if v == nil || v.FilePath == "" {
					continue
				}
				return b.resolveAssetURL(v.FilePath, "/")
			}
		}
	}
	return ""
}

func pickSocialImageVariant(img *assets.ProcessedImage) *assets.ImageVariant {
	if img == nil || len(img.Variants) == 0 {
		return nil
	}
	for i := range img.Variants {
		if img.Variants[i].Size == "preview" {
			return &img.Variants[i]
		}
	}
	for i := range img.Variants {
		if img.Variants[i].Size == "full" || img.Variants[i].Size == "original" {
			return &img.Variants[i]
		}
	}
	return &img.Variants[0]
}

func processedImageLookupCandidates(src, siteURL string) []string {
	src = html.UnescapeString(strings.TrimSpace(src))
	if src == "" {
		return nil
	}

	seen := make(map[string]struct{})
	add := func(v string) {
		v = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(v)), "/")
		if v == "" {
			return
		}
		if strings.HasPrefix(v, "assets/img/") {
			v = strings.TrimPrefix(v, "assets/img/")
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
	}

	add(src)
	if u, err := url.Parse(src); err == nil {
		add(u.Path)
	}
	if siteURL != "" {
		if base, err := url.Parse(siteURL); err == nil {
			if u, err := url.Parse(src); err == nil {
				if u.IsAbs() && base.Host != "" && strings.EqualFold(u.Host, base.Host) {
					add(u.Path)
				}
			}
		}
	}
	if decoded, err := url.PathUnescape(src); err == nil {
		add(decoded)
	}

	out := make([]string, 0, len(seen)*2)
	for candidate := range seen {
		out = append(out, candidate)
		out = append(out, filepath.Base(candidate))
	}
	return out
}

func (b *SiteBuilder) metaDescriptionForPage(page *Page) string {
	if page == nil {
		return strings.TrimSpace(b.config.SEO.DefaultDesc)
	}
	if desc := strings.TrimSpace(page.Description); desc != "" {
		return desc
	}
	if excerpt := markdownExcerpt(page.RawContent, 200); excerpt != "" {
		return excerpt
	}
	return strings.TrimSpace(b.config.SEO.DefaultDesc)
}

func markdownExcerpt(raw string, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 200
	}

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "![") || strings.HasPrefix(s, "<!--") {
			continue
		}

		s = excerptLinkRe.ReplaceAllString(s, "$1")
		s = strings.ReplaceAll(s, "`", "")
		s = strings.Join(strings.Fields(s), " ")
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if utf8.RuneCountInString(s) <= maxRunes {
			return s
		}
		runes := []rune(s)
		return strings.TrimSpace(string(runes[:maxRunes])) + "..."
	}

	return ""
}

func (b *SiteBuilder) localizeInternalLinks(html, lang string) string {
	prefix := "/" + strings.Trim(lang, "/")
	if prefix == "/" {
		prefix = "/" + b.config.I18n.Default
	}
	for _, seg := range []string{"blog", "docs", "tags", "archive", "graph"} {
		html = strings.ReplaceAll(html, "href=\"/"+seg+"/", "href=\""+prefix+"/"+seg+"/")
		html = strings.ReplaceAll(html, "src=\"/"+seg+"/", "src=\""+prefix+"/"+seg+"/")
	}
	return html
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

func (b *SiteBuilder) buildHeaderNav(lang string) []renderer.NavLink {
	ui := i18n.UI(lang)
	base := "/" + strings.Trim(lang, "/") + "/"
	if lang == "" {
		base = "/"
	}
	return []renderer.NavLink{
		{Title: ui.Docs, URL: base + "docs/", Type: "header"},
		{Title: ui.Blog, URL: base + "blog/", Type: "header"},
	}
}

func (b *SiteBuilder) buildLanguageOptions(page *Page) []renderer.LanguageOption {
	options := make([]renderer.LanguageOption, 0, len(b.config.I18n.Languages))
	trimmed := strings.Trim(page.URL, "/")
	parts := []string{}
	if trimmed != "" {
		parts = strings.Split(trimmed, "/")
	}
	relative := strings.Join(parts, "/")
	if len(parts) > 0 {
		if _, ok := b.languages[strings.ToLower(parts[0])]; ok {
			relative = strings.Join(parts[1:], "/")
		}
	}
	relative = strings.Trim(relative, "/")
	section := ""
	if relative != "" {
		rs := strings.Split(relative, "/")
		section = rs[0]
	}

	for _, lang := range b.config.I18n.Languages {
		code := strings.ToLower(lang.Code)
		targetPath := "/" + strings.Trim(relative, "/")
		if targetPath == "/" {
			targetPath = ""
		}
		candidate := "/" + code + "/"
		if targetPath != "" {
			candidate = "/" + code + "/" + strings.Trim(targetPath, "/") + "/"
		}

		if _, ok := b.pagesByURL[candidate]; !ok {
			if section != "" {
				candidate = "/" + code + "/" + section + "/"
			} else {
				candidate = "/" + code + "/"
			}
		}

		options = append(options, renderer.LanguageOption{
			Code:   code,
			Label:  lang.Label,
			URL:    candidate,
			Active: code == page.Language,
		})
	}
	return options
}

// copyAssets copies static assets to output directory (non-image, non-CSS/JS files)
func (b *SiteBuilder) copyAssets(index *ContentIndex) error {
	errs := b.parallelForEach(len(index.AssetFiles), func(i int) error {
		file := index.AssetFiles[i]
		// Skip CSS/JS files (already processed)
		ext := strings.ToLower(filepath.Ext(file.Path))
		rel := filepath.ToSlash(file.RelativePath)
		if ext == ".css" || (ext == ".js" && !strings.HasPrefix(rel, "triangle/")) {
			return nil
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
			return nil
		}

		// Write to output
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write asset: %w", err)
		}
		return nil
	})
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (b *SiteBuilder) copyConfiguredLogo() error {
	logoPath := strings.TrimSpace(b.config.Site.Logo)
	if logoPath == "" || !strings.HasPrefix(logoPath, "/assets/") {
		return nil
	}

	relLogoPath := strings.TrimPrefix(logoPath, "/")
	srcPath := filepath.Join(b.config.Build.ContentDir, relLogoPath)
	if _, err := os.Stat(srcPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	outputPath := filepath.Join(b.config.Build.OutputDir, relLogoPath)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create logo directory: %w", err)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read logo asset: %w", err)
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write logo asset: %w", err)
	}
	return nil
}

// copyTriangleModules ensures chat widget module files are available as standalone assets.
func (b *SiteBuilder) copyTriangleModules() error {
	srcDir := filepath.Join(b.config.Build.ContentDir, "triangle")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	dstDir := filepath.Join(b.config.Build.OutputDir, "assets", "triangle")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".js" && ext != ".mjs" {
			continue
		}

		srcPath := filepath.Join(srcDir, name)
		dstPath := filepath.Join(dstDir, name)

		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return err
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

func buildBlogTimeline(posts []*Page, maxPerYear int) []renderer.TimelineYear {
	if maxPerYear <= 0 {
		maxPerYear = 20
	}

	byYear := make(map[int][]*Page)
	years := make([]int, 0)
	for _, p := range posts {
		if p == nil || p.Frontmatter == nil || p.Frontmatter.Date.IsZero() {
			continue
		}
		year := p.Frontmatter.Date.Year()
		if _, exists := byYear[year]; !exists {
			years = append(years, year)
		}
		byYear[year] = append(byYear[year], p)
	}
	sort.Slice(years, func(i, j int) bool { return years[i] > years[j] })

	result := make([]renderer.TimelineYear, 0, len(years))
	for _, year := range years {
		pages := byYear[year]
		sort.Slice(pages, func(i, j int) bool {
			return pages[i].Frontmatter.Date.After(pages[j].Frontmatter.Date)
		})
		if len(pages) > maxPerYear {
			pages = pages[:maxPerYear]
		}

		items := make([]renderer.TimelineItem, 0, len(pages))
		for _, p := range pages {
			items = append(items, renderer.TimelineItem{
				Title: p.Title,
				URL:   p.URL,
				Date:  p.Frontmatter.Date,
			})
		}

		result = append(result, renderer.TimelineYear{
			Year:  year,
			Items: items,
		})
	}
	return result
}

func (b *SiteBuilder) collectBlogPostsByLanguage() map[string][]*Page {
	out := make(map[string][]*Page)
	for _, page := range b.pages {
		if page.Type != TypeBlog {
			continue
		}
		out[page.Language] = append(out[page.Language], page)
	}
	for lang := range out {
		sort.Slice(out[lang], func(i, j int) bool {
			return out[lang][i].Frontmatter.Date.After(out[lang][j].Frontmatter.Date)
		})
	}
	return out
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
	postsByLang := make(map[string][]*Page)
	for _, p := range blogPosts {
		postsByLang[p.Language] = append(postsByLang[p.Language], p)
	}
	total := 0
	for lang, posts := range postsByLang {
		summaries := pageSummariesFromPosts(posts)
		tagIdx := tags.BuildTagIndex(summaries)

		allTags := tagIdx.Tags()
		if len(allTags) == 0 {
			continue
		}

		ui := i18n.UI(lang)
		tagCloudHTML := b.buildTagCloudHTML(tagIdx, allTags, lang)
		tagCloudPage := &Page{
			ID:          lang + "-tags",
			URL:         "/" + lang + "/tags/",
			Language:    lang,
			Title:       ui.Tags,
			Description: "All tags",
			Content:     tagCloudHTML,
			RawContent:  "",
			Frontmatter: &parser.Frontmatter{},
			Type:        TypePage,
		}
		b.pages[tagCloudPage.ID] = tagCloudPage
		b.pagesByURL[tagCloudPage.URL] = tagCloudPage
		if err := b.renderPage(tagCloudPage); err != nil {
			return fmt.Errorf("failed to render tag cloud page: %w", err)
		}

		for _, tag := range allTags {
			tagPages := tagIdx[tag]
			tagSlug := parser.GenerateSlug(tag)
			tagPageHTML := b.buildTagPageHTML(tag, tagPages, lang)

			tagPage := &Page{
				ID:          lang + "-tags-" + tagSlug,
				URL:         "/" + lang + "/tags/" + tagSlug + "/",
				Language:    lang,
				Title:       "Tag: " + tag,
				Description: fmt.Sprintf("Pages tagged with \"%s\"", tag),
				Content:     tagPageHTML,
				RawContent:  "",
				Frontmatter: &parser.Frontmatter{Tags: []string{tag}},
				Type:        TypePage,
			}
			b.pages[tagPage.ID] = tagPage
			b.pagesByURL[tagPage.URL] = tagPage
			if err := b.renderPage(tagPage); err != nil {
				fmt.Fprintf(os.Stderr, "Error rendering tag page %s: %v\n", tag, err)
				continue
			}
			total++
		}
	}
	fmt.Printf("Generated %d tag pages\n", total)
	return nil
}

// buildTagCloudHTML generates HTML for the tag cloud page
func (b *SiteBuilder) buildTagCloudHTML(idx tags.TagIndex, allTags []string, lang string) string {
	var sb strings.Builder
	sb.WriteString("<div class=\"tag-cloud\">\n")
	sb.WriteString("<ul class=\"tag-list\">\n")
	for _, tag := range allTags {
		slug := parser.GenerateSlug(tag)
		count := idx.Count(tag)
		sb.WriteString(fmt.Sprintf("  <li><a href=\"/%s/tags/%s/\" class=\"tag\">%s</a> <span class=\"tag-count\">(%d)</span></li>\n", lang, slug, tag, count))
	}
	sb.WriteString("</ul>\n")
	sb.WriteString("</div>\n")
	return sb.String()
}

// buildTagPageHTML generates HTML for a single tag page
func (b *SiteBuilder) buildTagPageHTML(tag string, pages []tags.PageSummary, lang string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h2>Posts tagged \"%s\"</h2>\n", tag))
	sb.WriteString(fmt.Sprintf("<p>%d post(s)</p>\n", len(pages)))
	sb.WriteString("<ul class=\"post-list\">\n")
	for _, p := range pages {
		dateStr := ""
		if !p.Date.IsZero() {
			dateStr = fmt.Sprintf(" <time>%s</time>", i18n.FormatDateLong(p.Date, lang))
		}
		sb.WriteString(fmt.Sprintf("  <li><a href=\"%s\">%s</a>%s</li>\n", p.URL, p.Title, dateStr))
	}
	sb.WriteString("</ul>\n")
	return sb.String()
}

// generateArchivePages builds archive structure and creates archive pages
func (b *SiteBuilder) generateArchivePages(blogPosts []*Page) error {
	byLang := make(map[string][]archive.PageSummary)
	for _, p := range blogPosts {
		byLang[p.Language] = append(byLang[p.Language], archive.PageSummary{
			Title:       p.Title,
			URL:         p.URL,
			Date:        p.Frontmatter.Date,
			Description: p.Description,
			Tags:        p.Frontmatter.Tags,
			Type:        string(p.Type),
		})
	}

	totalYears := 0
	for lang, summaries := range byLang {
		archiveData := archive.BuildArchive(summaries)
		if len(archiveData) == 0 {
			continue
		}

		archiveHTML := b.buildArchiveIndexHTML(archiveData, lang)
		archivePage := &Page{
			ID:          lang + "-archive",
			URL:         "/" + lang + "/archive/",
			Language:    lang,
			Title:       i18n.SegmentLabel(lang, "archive"),
			Description: "Post archive by date",
			Content:     archiveHTML,
			RawContent:  "",
			Frontmatter: &parser.Frontmatter{},
			Type:        TypePage,
		}
		b.pages[archivePage.ID] = archivePage
		b.pagesByURL[archivePage.URL] = archivePage
		if err := b.renderPage(archivePage); err != nil {
			return fmt.Errorf("failed to render archive page: %w", err)
		}

		for _, year := range archiveData {
			yearHTML := b.buildArchiveYearHTML(year, lang)
			yearPage := &Page{
				ID:          fmt.Sprintf("%s-archive-%d", lang, year.Year),
				URL:         fmt.Sprintf("/%s/archive/%d/", lang, year.Year),
				Language:    lang,
				Title:       fmt.Sprintf("%s: %d", i18n.SegmentLabel(lang, "archive"), year.Year),
				Description: fmt.Sprintf("Posts from %d", year.Year),
				Content:     yearHTML,
				RawContent:  "",
				Frontmatter: &parser.Frontmatter{},
				Type:        TypePage,
			}
			b.pages[yearPage.ID] = yearPage
			b.pagesByURL[yearPage.URL] = yearPage
			if err := b.renderPage(yearPage); err != nil {
				fmt.Fprintf(os.Stderr, "Error rendering archive year page %d: %v\n", year.Year, err)
			}
		}
		totalYears += len(archiveData)
	}

	fmt.Printf("Generated archive pages (%d years)\n", totalYears)
	return nil
}

// buildArchiveIndexHTML generates HTML for the main archive page
func (b *SiteBuilder) buildArchiveIndexHTML(years []archive.ArchiveYear, lang string) string {
	var sb strings.Builder
	sb.WriteString("<div class=\"archive\">\n")
	for _, year := range years {
		sb.WriteString(fmt.Sprintf("<h2><a href=\"/%s/archive/%d/\">%d</a> <span class=\"count\">(%d)</span></h2>\n", lang, year.Year, year.Year, year.Count))
		for _, month := range year.Months {
			sb.WriteString(fmt.Sprintf("<h3>%s %d</h3>\n", i18n.MonthName(lang, month.Month), month.Year))
			sb.WriteString("<ul class=\"post-list\">\n")
			for _, p := range month.Pages {
				dateStr := i18n.FormatDateShort(p.Date, lang)
				sb.WriteString(fmt.Sprintf("  <li><time>%s</time> <a href=\"%s\">%s</a></li>\n", dateStr, p.URL, p.Title))
			}
			sb.WriteString("</ul>\n")
		}
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// buildArchiveYearHTML generates HTML for a single year archive page
func (b *SiteBuilder) buildArchiveYearHTML(year archive.ArchiveYear, lang string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<div class=\"archive-year\">\n"))
	for _, month := range year.Months {
		sb.WriteString(fmt.Sprintf("<h2>%s</h2>\n", i18n.MonthName(lang, month.Month)))
		sb.WriteString("<ul class=\"post-list\">\n")
		for _, p := range month.Pages {
			dateStr := i18n.FormatDateShort(p.Date, lang)
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

// generateGraph creates the graph visualization page and JSON data
func (b *SiteBuilder) generateGraph() error {
	// Convert pages to PageInfo for graph builder
	var pageInfos []graph.PageInfo
	for _, page := range b.pages {
		pageType := string(page.Type)
		if pageType == "" {
			pageType = "page"
		}
		pageInfos = append(pageInfos, graph.PageInfo{
			ID:         page.ID,
			Title:      page.Title,
			URL:        page.URL,
			Type:       pageType,
			Tags:       page.Frontmatter.Tags,
			RawContent: page.RawContent,
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

	// Mirror graph endpoints under language prefixes.
	for _, lang := range b.config.I18n.Languages {
		code := strings.ToLower(strings.TrimSpace(lang.Code))
		if code == "" {
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

	fmt.Printf("Generated graph view (%d nodes, %d edges)\n", len(graphData.Nodes), len(graphData.Edges))
	return nil
}
