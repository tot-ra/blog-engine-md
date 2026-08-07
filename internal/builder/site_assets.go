package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/renderer"
	"github.com/tot-ra/blog-engine/internal/tags"
)

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
		// Redirect/hideNav stubs keep old URLs working but must not duplicate
		// the canonical post in timeline, archive, or feeds.
		if page.Type == TypeBlog && !isHiddenFromListings(page) {
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

// collectTagPages returns all source-backed pages with tags, sorted newest first.
func (b *SiteBuilder) collectTagPages() []*Page {
	var pages []*Page
	for _, page := range b.pages {
		if page == nil || page.Frontmatter == nil || len(page.Frontmatter.Tags) == 0 {
			continue
		}
		// Generated utility pages should not appear in the tag index.
		if page.SourcePath == "" {
			continue
		}
		pages = append(pages, page)
	}
	sort.Slice(pages, func(i, j int) bool {
		di := pages[i].Frontmatter.Date
		dj := pages[j].Frontmatter.Date
		switch {
		case di.Equal(dj):
			return pages[i].URL < pages[j].URL
		case di.IsZero():
			return false
		case dj.IsZero():
			return true
		default:
			return di.After(dj)
		}
	})
	return pages
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
		if page.Type != TypeBlog || isHiddenFromListings(page) {
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

func (b *SiteBuilder) sidebarSectionConfig(section string) config.SidebarSectionConfig {
	cfg := config.SidebarSectionConfig{
		DefaultMode: "categories",
		EnableTime:  true,
		EnableGraph: section == "blog",
	}
	if section == "" || b.config == nil {
		return cfg
	}
	if custom, ok := b.config.Navigation.Sidebar.Sections[section]; ok {
		if strings.TrimSpace(custom.DefaultMode) != "" {
			cfg.DefaultMode = strings.TrimSpace(custom.DefaultMode)
		}
		cfg.EnableTime = custom.EnableTime
		cfg.EnableGraph = custom.EnableGraph
		cfg.GraphPath = custom.GraphPath
		cfg.ShowChildrenList = custom.ShowChildrenList
		cfg.RecentEmbeds = custom.RecentEmbeds
	}
	return cfg
}

func (b *SiteBuilder) matchingSidebarSection(page *Page) (string, config.SidebarSectionConfig) {
	var zero config.SidebarSectionConfig
	if page == nil {
		return "", zero
	}
	if b.config == nil {
		return "", zero
	}
	pageURL := ensureTrailingSlash(page.URL)
	bestKey := ""
	bestCfg := zero
	bestMatchLen := -1
	for key, cfg := range b.config.Navigation.Sidebar.Sections {
		matchPaths := cfg.MatchPaths
		if len(matchPaths) == 0 && strings.TrimSpace(key) != "" {
			matchPaths = []string{key}
		}
		for _, rawPath := range matchPaths {
			path := normalizeSectionPattern(rawPath)
			if path == "" {
				continue
			}
			if !strings.Contains(pageURL, path) {
				continue
			}
			if len(path) > bestMatchLen {
				bestKey = key
				bestCfg = cfg
				bestMatchLen = len(path)
			}
		}
	}
	return bestKey, bestCfg
}

func ensureTrailingSlash(raw string) string {
	if strings.HasSuffix(raw, "/") {
		return raw
	}
	return raw + "/"
}

func normalizeSectionPattern(raw string) string {
	trimmed := strings.Trim(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return ""
	}
	return "/" + trimmed + "/"
}

func localizedSectionURL(language, raw string) string {
	return localizedSectionURLForPage(language, "", "", raw)
}

func (b *SiteBuilder) localizedSectionURL(language, raw string) string {
	defaultLanguage := ""
	if b != nil && b.config != nil {
		defaultLanguage = b.config.I18n.Default
	}
	return localizedSectionURLForPage(language, defaultLanguage, "", raw)
}

func localizedSectionURLForPage(language, defaultLanguage, currentURL, raw string) string {
	trimmedRaw := strings.Trim(strings.TrimSpace(raw), "/")
	if trimmedRaw == "" {
		return ""
	}
	prefix := languageURLPrefix(language, defaultLanguage, currentURL)
	return prefix + "/" + trimmedRaw + "/"
}

func languageURLPrefix(language, defaultLanguage, currentURL string) string {
	trimmedLang := strings.Trim(strings.TrimSpace(language), "/")
	trimmedDefault := strings.Trim(strings.TrimSpace(defaultLanguage), "/")
	if trimmedLang == "" {
		trimmedLang = trimmedDefault
	}
	if trimmedLang == "" {
		return ""
	}

	trimmedURL := strings.Trim(currentURL, "/")
	if trimmedURL != "" {
		parts := strings.Split(trimmedURL, "/")
		if len(parts) > 0 && strings.EqualFold(parts[0], trimmedLang) {
			return "/" + parts[0]
		}
	}

	// Default-language content without an explicit language prefix stays at the
	// root. If currentURL was explicitly prefixed, preserve that shape above.
	if trimmedDefault != "" && strings.EqualFold(trimmedLang, trimmedDefault) {
		return ""
	}

	return "/" + trimmedLang
}

func (b *SiteBuilder) sidebarExcludeURLs(page *Page) []string {
	if page == nil || b.config == nil {
		return nil
	}
	pageURL := ensureTrailingSlash(page.URL)
	var excludes []string
	for _, rule := range b.config.Navigation.Sidebar.ExcludeRules {
		if !sidebarRuleMatches(pageURL, rule.MatchPaths) {
			continue
		}
		for _, raw := range rule.ExcludePaths {
			if localized := localizedSectionURLForPage(page.Language, b.config.I18n.Default, page.URL, raw); localized != "" {
				excludes = append(excludes, localized)
			}
		}
	}
	return excludes
}

func sidebarRuleMatches(pageURL string, matchPaths []string) bool {
	for _, raw := range matchPaths {
		pattern := normalizeSectionPattern(raw)
		if pattern == "" {
			continue
		}
		if strings.Contains(pageURL, pattern) {
			return true
		}
	}
	return false
}

func (b *SiteBuilder) prevNextConfigForPage(page *Page) config.PrevNextConfig {
	cfg := b.config.Navigation.PrevNext
	if len(cfg.Sections) == 0 {
		return cfg
	}

	matches := make([]string, 0, len(cfg.Sections))
	overridesByPath := make(map[string]config.PrevNextSectionConfig, len(cfg.Sections))
	for sectionPath := range cfg.Sections {
		normalized := normalizePrevNextSectionPath(sectionPath)
		if normalized == "" {
			continue
		}
		overridesByPath[normalized] = cfg.Sections[sectionPath]
		if prevNextSectionMatches(normalized, page.URL, b.languages) {
			matches = append(matches, normalized)
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if len(matches[i]) != len(matches[j]) {
			return len(matches[i]) < len(matches[j])
		}
		return matches[i] < matches[j]
	})

	for _, normalized := range matches {
		override := overridesByPath[normalized]
		if override.Enabled != nil {
			cfg.Enabled = *override.Enabled
		}
		if override.SameCategoryOnly != nil {
			cfg.SameCategoryOnly = *override.SameCategoryOnly
		}
	}

	return cfg
}

func normalizePrevNextSectionPath(sectionPath string) string {
	trimmed := strings.Trim(strings.TrimSpace(sectionPath), "/")
	if trimmed == "" {
		return ""
	}
	return "/" + trimmed + "/"
}

func prevNextSectionMatches(sectionPath, pageURL string, languages map[string]struct{}) bool {
	candidates := []string{normalizePrevNextSectionPath(pageURL)}

	trimmed := strings.Trim(strings.TrimSpace(pageURL), "/")
	if trimmed != "" {
		segments := strings.Split(trimmed, "/")
		if len(segments) > 1 {
			if _, ok := languages[segments[0]]; ok {
				candidates = append(candidates, "/"+strings.Join(segments[1:], "/")+"/")
			}
		}
	}

	for _, candidate := range candidates {
		if candidate == sectionPath || strings.HasPrefix(candidate, sectionPath) {
			return true
		}
	}

	return false
}

func (b *SiteBuilder) buildSectionTimeline(root *renderer.NavNode, language string, maxPerYear int) []renderer.TimelineYear {
	if root == nil || root.URL == "" {
		return nil
	}

	var pages []*Page
	for _, page := range b.pages {
		if page == nil || page.Language != language {
			continue
		}
		if page.URL == root.URL || !strings.HasPrefix(page.URL, root.URL) {
			continue
		}
		if page.Frontmatter != nil {
			if page.Frontmatter.HideNav || strings.TrimSpace(page.Frontmatter.RedirectURL) != "" {
				continue
			}
			if page.Frontmatter.Date.IsZero() {
				continue
			}
		} else {
			continue
		}
		pages = append(pages, page)
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Frontmatter.Date.After(pages[j].Frontmatter.Date)
	})

	return buildBlogTimeline(pages, maxPerYear)
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
