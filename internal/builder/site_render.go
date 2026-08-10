package builder

import (
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/i18n"
	"github.com/tot-ra/blog-engine/internal/parser"
	"github.com/tot-ra/blog-engine/internal/renderer"
)

func findSidebarNodeByURL(root *renderer.NavNode, targetURL string) *renderer.NavNode {
	if root == nil {
		return nil
	}
	if root.URL == targetURL {
		return root
	}
	for _, child := range root.Children {
		if found := findSidebarNodeByURL(child, targetURL); found != nil {
			return found
		}
	}
	return nil
}

func cloneSidebarNodeWithoutURL(root *renderer.NavNode, excludeURL string) *renderer.NavNode {
	if root == nil {
		return nil
	}
	cloned := *root
	cloned.Children = make([]*renderer.NavNode, 0, len(root.Children))
	for _, child := range root.Children {
		if child == nil || child.URL == excludeURL {
			continue
		}
		cloned.Children = append(cloned.Children, cloneSidebarNodeWithoutURL(child, excludeURL))
	}
	return &cloned
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
	renderedContent := page.Content
	if strings.TrimSpace(page.SourcePath) != "" {
		renderedContent += b.sectionChildrenContent(page)
	}

	// Prepare base data
	data := renderer.PageData{
		Site: *b.config,
		Page: renderer.Page{
			ID:           page.ID,
			URL:          page.URL,
			Language:     page.Language,
			Direction:    config.LanguageDirection(page.Language, b.config.I18n.Languages),
			Title:        page.Title,
			Description:  page.Description,
			Content:      renderedContent,
			AudioURL:     page.AudioURL,
			Type:         string(page.Type),
			ModifiedTime: page.ModifiedTime,
			Layout:       page.Frontmatter.Layout,
		},
		Frontmatter: renderer.Frontmatter{
			Date:         page.Frontmatter.Date,
			Tags:         page.Frontmatter.Tags,
			TemplateHero: page.Frontmatter.TemplateHero,
			Params:       page.Frontmatter.Params,
		},
		UI:      ui,
		Content: template.HTML(renderedContent),
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
	if b.config.Build.PublishMarkdown && strings.TrimSpace(page.SourcePath) != "" && (page.Frontmatter == nil || strings.TrimSpace(page.Frontmatter.RedirectURL) == "") {
		data.MarkdownURL = pageMarkdownURL(page.URL)
	}
	data.TagURL = func(tag string) string {
		return b.buildLanguageScopedURL(page.Language, "tags/"+parser.GenerateSlug(tag))
	}
	data.Site.Site.Language = page.Language
	data.Homepage = b.homepageForLanguage(page.Language)
	data.HomeURL, _ = b.existingLanguageScopedURL(page.Language, "")
	if page.Frontmatter.Layout == "homepage" && data.Homepage.BlogShowcase.Enabled {
		data.BlogShowcase = b.homepageBlogShowcase(page.Language, data.Homepage.BlogShowcase.Limit)
	}
	if b.cssBundle != nil {
		data.CSSPath = b.cssBundle.Path
	}
	if b.jsBundle != nil {
		data.JSPath = b.jsBundle.Path
	}

	data.HeaderNav = b.buildHeaderNav(page.Language, page.URL)
	data.HeaderSocial = b.buildHeaderSocial()
	data.Languages = b.buildLanguageOptions(page)

	// Generate sidebar
	rendererRoot := convertNavNode(b.navTree.Root)
	sidebarRoot := selectSidebarRoot(rendererRoot, page.URL, b.languages)
	hideSidebar := false
	if page.Language != "" {
		normalizedURL := strings.Trim(page.URL, "/")
		switch normalizedURL {
		case fmt.Sprintf("%s/music", page.Language), fmt.Sprintf("%s/projects", page.Language):
			hideSidebar = true
		}
	}
	sidebarSectionKey, matchedSectionCfg := b.matchingSidebarSection(page)
	if targetURL := localizedSectionURLForPage(page.Language, b.config.I18n.Default, page.URL, matchedSectionCfg.SidebarRoot); targetURL != "" {
		if found := findSidebarNodeByURL(rendererRoot, targetURL); found != nil {
			sidebarRoot = found
		}
	}
	for _, excludeURL := range b.sidebarExcludeURLs(page) {
		sidebarRoot = cloneSidebarNodeWithoutURL(sidebarRoot, excludeURL)
	}
	if hideSidebar {
		data.Sidebar = ""
	} else if page.Type == TypeBlog {
		timeline := b.blogTimeline[page.Language]
		graphURL := b.buildLanguageScopedURL(page.Language, "graph")
		sectionCfg := b.sidebarSectionConfig("blog")
		if !sectionCfg.EnableTime {
			timeline = nil
		}
		defaultMode := sectionCfg.DefaultMode
		if strings.TrimSpace(defaultMode) == "" {
			defaultMode = "categories"
		}
		data.Sidebar = renderer.RenderModeSidebar(sidebarRoot, page.URL, b.config.Navigation.Sidebar.MaxDepth, b.config.Navigation.Sidebar.Collapsed, timeline, ui, graphURL, defaultMode, sectionCfg.EnableGraph)
	} else if sidebarSectionKey != "" {
		sectionCfg := b.sidebarSectionConfig(sidebarSectionKey)
		var timeline []renderer.TimelineYear
		if sectionCfg.EnableTime {
			timeline = b.buildSectionTimeline(sidebarRoot, page.Language, 20)
		}
		if len(timeline) == 0 {
			// Category-only sections must avoid mode-sidebar markup so a persisted Blog
			// time/graph choice cannot hide their only navigation pane.
			data.Sidebar = renderer.RenderSidebar(sidebarRoot, page.URL, b.config.Navigation.Sidebar.MaxDepth, b.config.Navigation.Sidebar.Collapsed, ui)
		} else {
			defaultMode := sectionCfg.DefaultMode
			if strings.TrimSpace(defaultMode) == "" {
				defaultMode = "categories"
			}
			data.Sidebar = renderer.RenderModeSidebar(sidebarRoot, page.URL, b.config.Navigation.Sidebar.MaxDepth, b.config.Navigation.Sidebar.Collapsed, timeline, ui, "", defaultMode, false)
		}
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
		bcGen := NewDefaultAwareBreadcrumbGenerator(b.languages, b.config.I18n.Default)
		builderCrumbs := bcGen.Generate(page, b.navTree)
		data.Breadcrumbs = convertBreadcrumbs(builderCrumbs)
	}

	// Generate prev/next links
	prevNextCfg := b.prevNextConfigForPage(page)
	if prevNextCfg.Enabled {
		pnGen := NewPrevNextGenerator(prevNextCfg.SameCategoryOnly)
		links := pnGen.Generate(page, b.pages, b.navTree)
		if links != nil {
			data.PrevNext = convertPrevNext(links)
		}
	}

	var html string
	if page.Frontmatter != nil && strings.TrimSpace(page.Frontmatter.RedirectURL) != "" {
		target := strings.TrimSpace(page.Frontmatter.RedirectURL)
		html = "<!doctype html><html><head><meta charset=\"utf-8\"><meta http-equiv=\"refresh\" content=\"0; url=" + target + "\"><link rel=\"canonical\" href=\"" + target + "\"></head><body><a href=\"" + target + "\">Redirecting...</a></body></html>"
	} else {
		// Render template
		rendered, err := b.templates.RenderPage(data)
		if err != nil {
			return err
		}
		html = rendered
	}
	html = b.localizeInternalLinks(html, page.Language, page.URL)
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

	if data.MarkdownURL != "" {
		if err := b.writeMarkdownAlternative(page, outputPath); err != nil {
			return err
		}
	}

	return nil
}

func pageMarkdownURL(pageURL string) string {
	if strings.HasSuffix(pageURL, "/") {
		return pageURL + "index.md"
	}
	return strings.TrimSuffix(pageURL, ".html") + ".md"
}

func (b *SiteBuilder) writeMarkdownAlternative(page *Page, htmlOutputPath string) error {
	source, err := os.ReadFile(page.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to read markdown alternative %s: %w", page.SourcePath, err)
	}

	markdownOutputPath := filepath.Join(filepath.Dir(htmlOutputPath), "index.md")
	if strings.HasSuffix(page.URL, ".html") {
		markdownOutputPath = strings.TrimSuffix(htmlOutputPath, ".html") + ".md"
	}
	if err := os.WriteFile(markdownOutputPath, source, 0644); err != nil {
		return fmt.Errorf("failed to write markdown alternative: %w", err)
	}
	return nil
}

func (b *SiteBuilder) sectionChildrenContent(page *Page) string {
	if b.navTree == nil || page == nil || page.URL == "" {
		return ""
	}
	if page.Frontmatter != nil && page.Frontmatter.HideChildren {
		return ""
	}
	if strings.HasSuffix(strings.TrimSuffix(page.URL, "/"), "/blog") {
		return sectionBlogPostsHTML(page.URL, b.pagesByURL)
	}
	if page.Frontmatter == nil || !page.Frontmatter.ShowChildren {
		return ""
	}

	node, ok := b.navTree.ByPath[page.URL]
	if !ok || node == nil || len(node.Children) == 0 {
		return ""
	}

	sectionKey, _ := b.matchingSidebarSection(page)
	sectionCfg := b.sidebarSectionConfig(sectionKey)
	var sb strings.Builder
	sb.WriteString(b.sectionRecentEmbedsHTML(page, sectionCfg.RecentEmbeds))
	showChildrenList := true
	if sectionCfg.ShowChildrenList != nil {
		showChildrenList = *sectionCfg.ShowChildrenList
	}
	if showChildrenList {
		sb.WriteString(sectionChildrenHTML(sectionChildrenFromNode(node)))
	}
	return sb.String()
}

var (
	youtubeShortcodeRe = regexp.MustCompile(`::youtube\[([A-Za-z0-9_-]{11})\]`)
	vimeoShortcodeRe   = regexp.MustCompile(`::vimeo\[([0-9]+)\]`)
)

func (b *SiteBuilder) sectionRecentEmbedsHTML(page *Page, cfg config.RecentEmbedsConfig) string {
	if page == nil || page.URL == "" || page.Language == "" || !cfg.Enabled {
		return ""
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = "youtube"
	}
	limit := cfg.Limit
	if limit <= 0 {
		limit = 4
	}
	sortBy := strings.ToLower(strings.TrimSpace(cfg.SortBy))
	if sortBy == "" {
		sortBy = "date"
	}

	type videoEntry struct {
		Title     string
		URL       string
		EmbedHTML string
		SortTime  time.Time
	}

	prefix := ensureTrailingSlash(page.URL)
	entries := make([]videoEntry, 0, 8)
	for _, candidate := range b.pagesByURL {
		if candidate == nil || candidate.URL == "" || candidate.URL == page.URL {
			continue
		}
		if !strings.HasPrefix(candidate.URL, prefix) {
			continue
		}
		if candidate.Frontmatter != nil && (candidate.Frontmatter.HideNav || strings.TrimSpace(candidate.Frontmatter.RedirectURL) != "") {
			continue
		}
		embedHTML := renderRecentEmbedHTML(provider, candidate.RawContent)
		if embedHTML == "" {
			continue
		}
		entries = append(entries, videoEntry{
			Title:     candidate.Title,
			URL:       candidate.URL,
			EmbedHTML: embedHTML,
			SortTime:  candidate.ModifiedTime,
		})
		if sortBy == "date" && candidate.Frontmatter != nil && !candidate.Frontmatter.Date.IsZero() {
			entries[len(entries)-1].SortTime = candidate.Frontmatter.Date
		}
	}

	if len(entries) == 0 {
		return ""
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].SortTime.Equal(entries[j].SortTime) {
			return entries[i].URL > entries[j].URL
		}
		return entries[i].SortTime.After(entries[j].SortTime)
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}

	title := strings.TrimSpace(cfg.Title)
	if page != nil && len(cfg.TitleI18n) > 0 {
		lang := strings.ToLower(strings.TrimSpace(page.Language))
		if localized, ok := cfg.TitleI18n[lang]; ok && strings.TrimSpace(localized) != "" {
			title = strings.TrimSpace(localized)
		} else {
			for code, localized := range cfg.TitleI18n {
				if strings.ToLower(strings.TrimSpace(code)) == lang && strings.TrimSpace(localized) != "" {
					title = strings.TrimSpace(localized)
					break
				}
			}
		}
	}
	if title == "" {
		title = "Latest embeds"
	}

	var sb strings.Builder
	sb.WriteString(`
<section class="section-recent-embeds">
  <style>
    .section-recent-embeds {
      margin: 2rem 0 2.5rem;
    }
    .section-recent-embeds-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 1.25rem;
      margin-top: 1rem;
    }
    .section-recent-embeds-card {
      padding: 1rem;
      border: 1px solid var(--nav-border);
      border-radius: 16px;
      background: linear-gradient(180deg, rgba(0, 102, 204, 0.05), rgba(0, 102, 204, 0.02));
    }
    .section-recent-embeds-card h3 {
      margin: 0 0 0.8rem;
      font-size: 1rem;
      line-height: 1.35;
    }
    .section-recent-embeds-card h3 a {
      color: inherit;
      text-decoration: none;
    }
    .section-recent-embeds-card h3 a:hover {
      color: var(--nav-active);
      text-decoration: underline;
    }
  </style>
`)
	sb.WriteString(fmt.Sprintf("  <h2>%s</h2>\n", html.EscapeString(title)))
	sb.WriteString(`  <div class="section-recent-embeds-grid">
`)
	for _, entry := range entries {
		sb.WriteString(fmt.Sprintf(
			`    <article class="section-recent-embeds-card">
      <h3><a href="%s">%s</a></h3>
      %s
    </article>
`,
			template.HTMLEscapeString(entry.URL),
			template.HTMLEscapeString(entry.Title),
			entry.EmbedHTML,
		))
	}
	sb.WriteString("  </div>\n</section>\n")
	return sb.String()
}

func renderRecentEmbedHTML(provider, rawContent string) string {
	switch provider {
	case "youtube":
		match := youtubeShortcodeRe.FindStringSubmatch(rawContent)
		if len(match) != 2 {
			return ""
		}
		return fmt.Sprintf(`<div class="embed embed-youtube">
        <iframe
          src="https://www.youtube-nocookie.com/embed/%s"
          frameborder="0"
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
          allowfullscreen
          loading="lazy">
        </iframe>
      </div>`, template.HTMLEscapeString(match[1]))
	case "vimeo":
		match := vimeoShortcodeRe.FindStringSubmatch(rawContent)
		if len(match) != 2 {
			return ""
		}
		return fmt.Sprintf(`<div class="embed embed-vimeo">
        <iframe
          src="https://player.vimeo.com/video/%s"
          frameborder="0"
          allow="autoplay; fullscreen; picture-in-picture"
          allowfullscreen
          loading="lazy">
        </iframe>
      </div>`, template.HTMLEscapeString(match[1]))
	default:
		return ""
	}
}

func (b *SiteBuilder) homepageBlogShowcase(language string, limit int) []renderer.BlogShowcasePost {
	if limit <= 0 {
		limit = 4
	}

	posts := make([]*Page, 0, limit)
	for _, page := range b.pages {
		if page == nil || page.Type != TypeBlog || page.Language != language || strings.TrimSpace(page.SourcePath) == "" {
			continue
		}
		if page.Frontmatter != nil && (page.Frontmatter.HideNav || strings.TrimSpace(page.Frontmatter.RedirectURL) != "") {
			continue
		}
		posts = append(posts, page)
	}
	sort.SliceStable(posts, func(i, j int) bool {
		di := sectionPageSortDate(posts[i])
		dj := sectionPageSortDate(posts[j])
		if !di.Equal(dj) {
			return di.After(dj)
		}
		return strings.ToLower(posts[i].Title) < strings.ToLower(posts[j].Title)
	})
	if len(posts) > limit {
		posts = posts[:limit]
	}

	showcase := make([]renderer.BlogShowcasePost, 0, len(posts))
	for _, post := range posts {
		description := extractPreviewText(post.RawContent, 2, 220)
		if description == "" {
			description = strings.TrimSpace(post.Description)
		}
		showcase = append(showcase, renderer.BlogShowcasePost{
			Title:       post.Title,
			URL:         post.URL,
			Description: description,
			ImageHTML:   template.HTML(firstPreviewImageHTML(post)),
			Date:        sectionPageSortDate(post),
		})
	}
	return showcase
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
	if override.BlogShowcase.Enabled {
		out.BlogShowcase.Enabled = true
	}
	if override.BlogShowcase.Limit > 0 {
		out.BlogShowcase.Limit = override.BlogShowcase.Limit
	}
	if override.BlogShowcase.Title != "" {
		out.BlogShowcase.Title = override.BlogShowcase.Title
	}
	if override.HideProjects {
		out.HideProjects = true
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

func (b *SiteBuilder) localizeInternalLinks(html, lang, currentURL string) string {
	if len(b.config.I18n.Languages) <= 1 {
		return html
	}
	// languageURLPrefix returns an empty prefix for the default language, keeping
	// canonical default-language links at root while prefixing translated pages.
	prefix := languageURLPrefix(lang, b.config.I18n.Default, currentURL)
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
