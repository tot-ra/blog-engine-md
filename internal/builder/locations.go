package builder

import (
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
)

var htmlAnchorOpenRegex = regexp.MustCompile(`(?is)<a\b([^>]*)>`)
var locationPreviewScriptRe = regexp.MustCompile(`(?s)<script\b[^>]*\bclass=["'][^"']*\blocation-previews\b[^"']*["'][^>]*>.*?</script>`)
var locationPageMapRe = regexp.MustCompile(`(?s)<div\b[^>]*\bclass=["'][^"']*\blocation-page-map\b[^"']*["'][^>]*>.*?</div>`)

type locationPreview struct {
	Title   string   `json:"title"`
	Address string   `json:"address,omitempty"`
	Lat     *float64 `json:"lat,omitempty"`
	Lng     *float64 `json:"lng,omitempty"`
	HTML    string   `json:"html"`
	URL     string   `json:"url"`
}

type locationPreviewPayload struct {
	Locations map[string]locationPreview `json:"locations"`
}

// enhanceLocationLinks turns plain links into a configured locations folder
// into inline previews. Detection is path-based so authors can keep regular
// Markdown/HTML anchors instead of inventing a second syntax.
func (b *SiteBuilder) enhanceLocationLinks() {
	if b == nil || !locationsEnabled(b.config) {
		return
	}

	previews := make(map[string]locationPreview)
	for _, page := range b.pagesByURL {
		if !b.isLocationPlace(page) {
			continue
		}
		previews[page.URL] = buildLocationPreview(page)
	}
	if len(previews) == 0 {
		return
	}

	for _, page := range b.pages {
		if b.isLocationPlace(page) {
			lang := languageFromURL(page.URL, b)
			if mapHTML := locationMapEmbedHTML(page.Frontmatter, page.Title, lang); mapHTML != "" {
				page.Content = `<div class="location-page-map">` + mapHTML + `</div>` + page.Content
			}
		}
		page.Content = decorateLocationLinks(page.Content, page.URL, previews, b)
	}
}

func locationsEnabled(cfg *config.SiteConfig) bool {
	return cfg != nil && cfg.Locations.Enabled && strings.TrimSpace(cfg.Locations.Section) != ""
}

func (b *SiteBuilder) locationsSection() string {
	if b == nil || b.config == nil {
		return "locations"
	}
	section := strings.Trim(strings.TrimSpace(b.config.Locations.Section), "/")
	if section == "" {
		return "locations"
	}
	return section
}

func (b *SiteBuilder) isLocationPlace(page *Page) bool {
	if page == nil || !locationsEnabled(b.config) {
		return false
	}
	return urlHasLocationPlace(page.URL, b.locationsSection())
}

func urlHasLocationPlace(pageURL, section string) bool {
	section = strings.Trim(strings.ToLower(section), "/")
	if section == "" {
		return false
	}
	parts := strings.Split(strings.Trim(pageURL, "/"), "/")
	for i, part := range parts {
		if !strings.EqualFold(part, section) {
			continue
		}
		// WHY: /ru/locations/ is the directory, not a physical place.
		// WHAT: only treat child URLs as location notes.
		return i < len(parts)-1 && strings.TrimSpace(parts[i+1]) != ""
	}
	return false
}

func buildLocationPreview(page *Page) locationPreview {
	preview := locationPreview{
		Title: page.Title,
		URL:   page.URL,
		HTML:  locationPreviewHTML(page.Content),
	}
	if page.Frontmatter != nil {
		preview.Address = strings.TrimSpace(page.Frontmatter.Address)
		if lat, lng, ok := locationCoords(page.Frontmatter); ok {
			preview.Lat = &lat
			preview.Lng = &lng
		}
	}
	return preview
}

func locationPreviewHTML(content string) string {
	content = locationPreviewScriptRe.ReplaceAllString(content, "")
	content = locationPageMapRe.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

func locationCoords(fm *parser.Frontmatter) (float64, float64, bool) {
	if fm == nil {
		return 0, 0, false
	}
	if fm.Lat == 0 && fm.Lng == 0 {
		return 0, 0, false
	}
	if fm.Lat < -90 || fm.Lat > 90 || fm.Lng < -180 || fm.Lng > 180 {
		return 0, 0, false
	}
	return fm.Lat, fm.Lng, true
}

func decorateLocationLinks(content, currentURL string, previews map[string]locationPreview, b *SiteBuilder) string {
	if content == "" || len(previews) == 0 {
		return content
	}

	used := make(map[string]locationPreview)
	rewritten := htmlAnchorOpenRegex.ReplaceAllStringFunc(content, func(match string) string {
		sub := htmlAnchorOpenRegex.FindStringSubmatch(match)
		if len(sub) != 2 {
			return match
		}
		attrs := sub[1]
		attrMap := parseHTMLAttrs(attrs)
		if shouldSkipLocationAnchor(attrMap) {
			return match
		}
		href := strings.TrimSpace(attrMap["href"])
		target := b.lookupLocationPreview(href, currentURL, previews)
		if target == nil {
			return match
		}
		used[target.URL] = *target
		return rewriteLocationAnchor(attrs, attrMap, target.URL)
	})

	if len(used) == 0 {
		return rewritten
	}
	payload, err := json.Marshal(locationPreviewPayload{Locations: used})
	if err != nil {
		return rewritten
	}
	return rewritten + "\n<script type=\"application/json\" class=\"location-previews\">" + string(payload) + "</script>\n"
}

func (b *SiteBuilder) lookupLocationPreview(href, currentURL string, previews map[string]locationPreview) *locationPreview {
	url := normalizeLocationHref(href)
	if url == "" {
		return nil
	}
	if preview, ok := previews[url]; ok && preview.URL != currentURL {
		return &preview
	}
	if lang := languageFromURL(currentURL, b); lang != "" {
		prefixed := "/" + lang + url
		if !strings.HasPrefix(url, "/"+lang+"/") {
			if preview, ok := previews[prefixed]; ok && preview.URL != currentURL {
				return &preview
			}
		}
	}
	return nil
}

func languageFromURL(pageURL string, b *SiteBuilder) string {
	if b == nil {
		return ""
	}
	trimmed := strings.Trim(pageURL, "/")
	if trimmed == "" {
		return ""
	}
	first := strings.ToLower(strings.Split(trimmed, "/")[0])
	if _, ok := b.languages[first]; ok {
		return first
	}
	return ""
}

func normalizeLocationHref(href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "javascript:") {
		return ""
	}
	if strings.HasPrefix(href, "//") || strings.Contains(href, "://") {
		return ""
	}
	if cut := strings.IndexAny(href, "?#"); cut >= 0 {
		href = href[:cut]
	}
	href = strings.TrimSuffix(href, "index.html")
	href = strings.TrimSuffix(href, "index.md")
	if !strings.HasPrefix(href, "/") {
		return ""
	}
	if !strings.HasSuffix(href, "/") {
		href += "/"
	}
	return href
}

func shouldSkipLocationAnchor(attrMap map[string]string) bool {
	class := attrMap["class"]
	if class == "" {
		return false
	}
	for _, item := range strings.Fields(class) {
		switch {
		case item == "location-link":
			return true
		case strings.HasPrefix(item, "section-"):
			return true
		case item == "tag" || item == "navbar-logo":
			return true
		}
	}
	return false
}

func rewriteLocationAnchor(attrs string, attrMap map[string]string, canonicalURL string) string {
	class := strings.TrimSpace(attrMap["class"])
	if class == "" {
		class = "location-link"
	} else if !containsClass(class, "location-link") {
		class += " location-link"
	}

	next := replaceOrAppendAttr(attrs, "href", canonicalURL)
	next = replaceOrAppendAttr(next, "class", class)
	next = replaceOrAppendAttr(next, "aria-expanded", "false")
	next = replaceOrAppendAttr(next, "aria-haspopup", "true")
	return "<a" + next + ">"
}

func containsClass(class, want string) bool {
	for _, item := range strings.Fields(class) {
		if item == want {
			return true
		}
	}
	return false
}

func replaceOrAppendAttr(attrs, key, value string) string {
	re := regexp.MustCompile(`(?i)(\s` + regexp.QuoteMeta(key) + `\s*=\s*)("[^"]*"|'[^']*')`)
	if re.MatchString(attrs) {
		return re.ReplaceAllString(attrs, `${1}"`+html.EscapeString(value)+`"`)
	}
	return attrs + ` ` + key + `="` + html.EscapeString(value) + `"`
}

func locationMapOpenLabel(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "ru":
		return "Открыть на карте"
	case "et":
		return "Ava kaardil"
	default:
		return "Open map"
	}
}

func googleMapsOpenURL(lat, lng float64, address string) string {
	query := strings.TrimSpace(address)
	if lat != 0 || lng != 0 {
		query = strconv.FormatFloat(lat, 'f', 6, 64) + "," + strconv.FormatFloat(lng, 'f', 6, 64)
	}
	if query == "" {
		return ""
	}
	return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(query)
}

func locationMapOpenButtonHTML(href, label string) string {
	if strings.TrimSpace(href) == "" {
		return ""
	}
	return `<p class="location-panel-map-link"><a class="location-panel-map-btn" href="` + html.EscapeString(href) + `" target="_blank" rel="noopener">` + html.EscapeString(label) + `</a></p>`
}

func locationMapEmbedHTML(fm *parser.Frontmatter, title, lang string) string {
	lat, lng, ok := locationCoords(fm)
	address := ""
	if fm != nil {
		address = strings.TrimSpace(fm.Address)
	}
	openURL := googleMapsOpenURL(lat, lng, address)
	openButton := locationMapOpenButtonHTML(openURL, locationMapOpenLabel(lang))
	if !ok {
		if openButton == "" {
			return ""
		}
		return openButton
	}

	latStr := strconv.FormatFloat(lat, 'f', 6, 64)
	lngStr := strconv.FormatFloat(lng, 'f', 6, 64)
	label := html.EscapeString(strings.TrimSpace(title))
	if label == "" {
		label = "Map"
	}
	// WHY: OSM's export/embed.html now requires WebGL and renders a blank
	// apology page in headless/older clients. Leaflet raster tiles show a map.
	return `<div class="location-map" role="img" aria-label="` + label + `" data-lat="` + latStr + `" data-lng="` + lngStr + `" data-title="` + label + `"></div>` + openButton
}
