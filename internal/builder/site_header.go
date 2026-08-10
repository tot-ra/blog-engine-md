package builder

import (
	"net/url"
	"strings"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/i18n"
	"github.com/tot-ra/blog-engine/internal/renderer"
)

func (b *SiteBuilder) buildHeaderNav(lang, currentURL string) []renderer.NavLink {
	if !b.config.Navigation.Header.Enabled {
		return nil
	}
	items := b.headerNavItemsForLanguage(lang)
	if len(items) == 0 {
		ui := i18n.UI(lang)
		items = []config.HeaderItem{
			{Title: ui.Docs, Path: "docs"},
			{Title: ui.Blog, Path: "blog"},
		}
	}

	nav := make([]renderer.NavLink, 0, len(items))
	activeIndex := -1
	activeMatchLength := -1
	for _, item := range items {
		if !headerItemVisibleForLanguage(item, lang) {
			continue
		}

		target := strings.TrimSpace(item.URL)
		path := strings.TrimSpace(item.Path)
		if target == "" && path != "" {
			langPrefix := ""
			if len(b.config.I18n.Languages) > 1 {
				langPrefix = strings.Trim(languageURLPrefix(lang, b.config.I18n.Default, currentURL), "/")
			}
			// Preserve the URL shape of the page being rendered. Some sites keep
			// explicitly localized default-language content under /<lang>/, while
			// others publish it at the root.
			target = buildLanguageScopedURL(langPrefix, path)
		}
		if target == "" {
			continue
		}
		target = b.localizedHeaderTarget(lang, target)

		title := b.localizedHeaderTitle(item, lang, target)
		if title == "" {
			continue
		}

		nav = append(nav, renderer.NavLink{
			Title: title,
			URL:   target,
			Type:  "header",
			Class: strings.TrimSpace(item.Class),
		})

		if headerNavLinkIsCurrent(currentURL, target) {
			matchLength := len(normalizeHeaderNavPath(target))
			if matchLength > activeMatchLength {
				activeIndex = len(nav) - 1
				activeMatchLength = matchLength
			}
		}
	}
	if activeIndex >= 0 {
		nav[activeIndex].IsCurrent = true
		nav[activeIndex].Class = strings.TrimSpace(nav[activeIndex].Class + " is-active")
	}
	return nav
}

func (b *SiteBuilder) headerNavItemsForLanguage(lang string) []config.HeaderItem {
	if b == nil || b.config == nil {
		return nil
	}

	// Keep navigation.header.items for shared/backward-compatible links, then append
	// navigation.header.languages[lang] for compact per-locale configs. When both
	// formats are present, single-language items that duplicate a language group
	// are treated as legacy fallback for older binaries and skipped by new builds.
	header := b.config.Navigation.Header
	current := strings.ToLower(strings.TrimSpace(lang))
	groupItems, hasCurrentGroup := headerLanguageItemsForLanguage(header.LanguageItems, current)
	items := make([]config.HeaderItem, 0, len(header.Items)+len(groupItems))
	for _, item := range header.Items {
		if hasCurrentGroup && headerItemIsLanguageGroupDuplicate(item, groupItems) {
			continue
		}
		items = append(items, item)
	}
	items = append(items, groupItems...)
	return items
}

func headerLanguageItemsForLanguage(languageItems map[string][]config.HeaderItem, lang string) ([]config.HeaderItem, bool) {
	if len(languageItems) == 0 {
		return nil, false
	}
	if groupItems, ok := languageItems[lang]; ok {
		return groupItems, true
	}
	// Be forgiving when YAML keys use different casing/spacing, while keeping the
	// common lowercase path as a direct deterministic map lookup.
	for groupLang, groupItems := range languageItems {
		if strings.ToLower(strings.TrimSpace(groupLang)) == lang {
			return groupItems, true
		}
	}
	return nil, false
}
func headerItemIsLanguageGroupDuplicate(item config.HeaderItem, groupItems []config.HeaderItem) bool {
	if len(item.Languages) != 1 {
		return false
	}
	for _, groupItem := range groupItems {
		if strings.TrimSpace(item.Title) == strings.TrimSpace(groupItem.Title) &&
			strings.TrimSpace(item.URL) == strings.TrimSpace(groupItem.URL) &&
			strings.TrimSpace(item.Path) == strings.TrimSpace(groupItem.Path) &&
			strings.TrimSpace(item.Class) == strings.TrimSpace(groupItem.Class) {
			return true
		}
	}
	return false
}

func (b *SiteBuilder) localizedHeaderTarget(lang, target string) string {
	candidate := b.localizedHeaderCandidate(lang, target)
	if candidate != "" && b.hasLocalizedHeaderContentRoute(candidate) {
		return candidate
	}
	return strings.TrimSpace(target)
}

func (b *SiteBuilder) localizedHeaderCandidate(lang, target string) string {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" || strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		return ""
	}
	if b == nil || b.config == nil || len(b.config.I18n.Languages) <= 1 {
		return ""
	}
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" || lang == strings.ToLower(strings.TrimSpace(b.config.I18n.Default)) {
		return ""
	}

	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) > 0 {
		if _, alreadyLocalized := b.languages[strings.ToLower(parts[0])]; alreadyLocalized {
			return ""
		}
	}

	candidate := "/" + lang + "/"
	if inner := strings.Trim(trimmed, "/"); inner != "" {
		candidate += inner + "/"
	}
	return candidate
}

func (b *SiteBuilder) localizedHeaderTitle(item config.HeaderItem, lang, target string) string {
	fallback := strings.TrimSpace(item.Title)
	if fallback == "" {
		fallback = strings.Trim(strings.TrimSpace(item.Path), "/")
	}
	// Prefer config titleI18n first. Default-language contentNavTitle intentionally
	// skips page titles, so sites that keep short Russian/Estonian labels in
	// config.yaml (dina.kurapov.ee, kurapov.ee) would otherwise render English
	// fallback titles like "About" / "Pedagogy".
	if title := headerItemTitleI18n(item, lang); title != "" {
		return title
	}
	labelTarget := target
	if candidate := b.localizedHeaderCandidate(lang, target); candidate != "" && b.hasPageOrNavRoute(candidate) {
		labelTarget = candidate
	}
	if title := b.contentNavTitle(labelTarget, lang, fallback); title != "" {
		return title
	}
	if labelTarget != target {
		if title := b.contentNavTitle(target, lang, fallback); title != "" {
			return title
		}
	}
	if title := localizedEngineSegmentTitle(fallback, lang); title != "" {
		return title
	}
	return localizedStaticHeaderTitle(fallback, lang)
}

func headerItemTitleI18n(item config.HeaderItem, lang string) string {
	if len(item.TitleI18n) == 0 {
		return ""
	}
	current := strings.ToLower(strings.TrimSpace(lang))
	if title, ok := item.TitleI18n[current]; ok {
		return strings.TrimSpace(title)
	}
	for code, title := range item.TitleI18n {
		if strings.ToLower(strings.TrimSpace(code)) == current {
			return strings.TrimSpace(title)
		}
	}
	return ""
}

func (b *SiteBuilder) contentNavTitle(target, lang, fallback string) string {
	if b == nil {
		return ""
	}
	normalizedTarget := normalizeNavURL(target)
	if normalizedTarget == "" || strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return ""
	}

	if page := b.pagesByURL[normalizedTarget]; page != nil {
		if page.Frontmatter != nil {
			if title := strings.TrimSpace(page.Frontmatter.NavTitle); title != "" {
				return title
			}
		}
		if !b.isDefaultLanguage(lang) && strings.EqualFold(strings.TrimSpace(page.Language), strings.TrimSpace(lang)) {
			if title := strings.TrimSpace(page.Title); title != "" && !strings.EqualFold(title, fallback) {
				return title
			}
		}
	}

	if b.navTree != nil {
		if node := b.navTree.ByPath[normalizedTarget]; node != nil {
			if title := strings.TrimSpace(node.Title); !b.isDefaultLanguage(lang) && pathHasLanguagePrefix(normalizedTarget, lang) && title != "" && !strings.EqualFold(title, fallback) {
				return title
			}
		}
	}
	return ""
}

func (b *SiteBuilder) isDefaultLanguage(lang string) bool {
	if b == nil || b.config == nil {
		return strings.TrimSpace(lang) == ""
	}
	return strings.EqualFold(strings.TrimSpace(lang), strings.TrimSpace(b.config.I18n.Default))
}

func pathHasLanguagePrefix(path, lang string) bool {
	lang = strings.ToLower(strings.Trim(strings.TrimSpace(lang), "/"))
	if lang == "" {
		return false
	}
	parts := strings.Split(strings.Trim(normalizeNavURL(path), "/"), "/")
	return len(parts) > 0 && strings.EqualFold(parts[0], lang)
}

func (b *SiteBuilder) hasLocalizedHeaderContentRoute(target string) bool {
	target = normalizeNavURL(target)
	if target == "" || b == nil {
		return false
	}
	if page := b.pagesByURL[target]; page != nil {
		return !isRedirectPage(page)
	}
	prefix := ensureTrailingSlash(target)
	for _, page := range b.pagesByURL {
		if page == nil || page.URL == "" {
			continue
		}
		if strings.HasPrefix(page.URL, prefix) && !isRedirectPage(page) {
			return true
		}
	}
	if b.navTree != nil {
		_, ok := b.navTree.ByPath[target]
		return ok
	}
	return false
}

func (b *SiteBuilder) hasPageOrNavRoute(target string) bool {
	target = normalizeNavURL(target)
	if target == "" {
		return false
	}
	if b != nil {
		if _, ok := b.pagesByURL[target]; ok {
			return true
		}
		if b.navTree != nil {
			_, ok := b.navTree.ByPath[target]
			return ok
		}
	}
	return false
}

func isRedirectPage(page *Page) bool {
	return page != nil && page.Frontmatter != nil && strings.TrimSpace(page.Frontmatter.RedirectURL) != ""
}

// isHiddenFromListings skips redirect stubs and hideNav pages from blog
// timelines, archives, and feeds while still rendering their redirect HTML.
func isHiddenFromListings(page *Page) bool {
	if page == nil || page.Frontmatter == nil {
		return false
	}
	return page.Frontmatter.HideNav || strings.TrimSpace(page.Frontmatter.RedirectURL) != ""
}

func localizedEngineSegmentTitle(title, lang string) string {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "blog", "docs", "tags", "archive", "graph":
		return i18n.SegmentLabel(lang, title)
	}
	return ""
}

func localizedStaticHeaderTitle(title, lang string) string {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "log in", "login":
		return i18n.UI(lang).LogIn
	}
	return title
}

func headerNavLinkIsCurrent(currentURL, targetURL string) bool {
	current := normalizeHeaderNavPath(currentURL)
	target := normalizeHeaderNavPath(targetURL)
	if current == "" || target == "" {
		return false
	}
	if current == target {
		return true
	}
	if target == "/" {
		return false
	}
	return strings.HasPrefix(current, strings.TrimSuffix(target, "/")+"/")
}

func normalizeHeaderNavPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" {
		if parsed.Path == "" {
			return "/"
		}
		trimmed = parsed.Path
	}
	if !strings.HasPrefix(trimmed, "/") {
		return ""
	}
	trimmed = "/" + strings.Trim(trimmed, "/")
	if trimmed == "/" {
		return trimmed
	}
	return trimmed + "/"
}

func headerItemVisibleForLanguage(item config.HeaderItem, lang string) bool {
	if len(item.Languages) == 0 {
		return true
	}

	current := strings.ToLower(strings.TrimSpace(lang))
	for _, allowed := range item.Languages {
		if strings.ToLower(strings.TrimSpace(allowed)) == current {
			return true
		}
	}

	return false
}

func buildLanguageScopedURL(lang, path string) string {
	trimmedPath := strings.Trim(path, "/")
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if trimmedPath == "" {
		if lang == "" {
			return "/"
		}
		return "/" + strings.Trim(lang, "/") + "/"
	}
	if lang == "" {
		return "/" + trimmedPath + "/"
	}
	return "/" + strings.Trim(lang, "/") + "/" + trimmedPath + "/"
}

func (b *SiteBuilder) buildLanguageScopedURL(lang, path string) string {
	// Keep default-language routes canonical at the site root; reserve
	// /<lang>/ prefixes for non-default translations.
	if b.isDefaultLanguage(lang) {
		lang = ""
	}
	return buildLanguageScopedURL(lang, path)
}

func (b *SiteBuilder) buildLanguageOptions(page *Page) []renderer.LanguageOption {
	if len(b.config.I18n.Languages) <= 1 {
		return nil
	}
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
		candidate, exists := b.existingLanguageScopedURL(code, targetPath)

		if !exists {
			if section != "" {
				candidate, _ = b.existingLanguageScopedURL(code, section)
			} else {
				candidate, _ = b.existingLanguageScopedURL(code, "")
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

func (b *SiteBuilder) existingLanguageScopedURL(lang, path string) (string, bool) {
	canonical := b.buildLanguageScopedURL(lang, path)
	if _, ok := b.pagesByURL[canonical]; ok {
		return canonical, true
	}

	// Explicitly prefixed default-language content is also supported. Prefer the
	// canonical root form when both routes exist, but use the real prefixed page
	// instead of returning a missing root URL.
	prefixed := buildLanguageScopedURL(lang, path)
	if prefixed != canonical {
		if _, ok := b.pagesByURL[prefixed]; ok {
			return prefixed, true
		}
	}
	return canonical, false
}

// buildHeaderSocial maps author.social entries to header icon links.
// Only networks with built-in SVG icons are included; order is stable.
func (b *SiteBuilder) buildHeaderSocial() []renderer.HeaderSocialLink {
	if b.config == nil || len(b.config.Author.Social) == 0 {
		return nil
	}

	type socialDef struct {
		key   string
		label string
		icon  string
	}
	// Keep GitHub/LinkedIn first - they are the primary professional profiles.
	defs := []socialDef{
		{key: "github", label: "GitHub", icon: "github"},
		{key: "linkedin", label: "LinkedIn", icon: "linkedin"},
	}

	out := make([]renderer.HeaderSocialLink, 0, len(defs))
	for _, def := range defs {
		raw, ok := b.config.Author.Social[def.key]
		if !ok {
			continue
		}
		url := resolveAuthorSocialURL(def.key, raw)
		if url == "" {
			continue
		}
		out = append(out, renderer.HeaderSocialLink{
			Label: def.label,
			URL:   url,
			Icon:  def.icon,
		})
	}
	return out
}

func resolveAuthorSocialURL(network, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return value
	}

	handle := strings.TrimPrefix(value, "@")
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "github":
		return "https://github.com/" + handle
	case "linkedin":
		handle = strings.TrimPrefix(handle, "/")
		if strings.Contains(handle, "/") {
			return "https://www.linkedin.com/" + handle
		}
		return "https://www.linkedin.com/in/" + handle
	default:
		return value
	}
}
