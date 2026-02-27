package seo

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SEOMeta holds all SEO metadata for a page
type SEOMeta struct {
	Title       string
	Description string
	Canonical   string
	OG          OpenGraph
	Twitter     TwitterCard
	Robots      string
	JSONLD      string // JSON-LD structured data
}

// OpenGraph holds Open Graph metadata
type OpenGraph struct {
	Title       string
	Description string
	Type        string
	URL         string
	Image       string
	SiteName    string
}

// TwitterCard holds Twitter Card metadata
type TwitterCard struct {
	Card        string // summary, summary_large_image
	Title       string
	Description string
	Image       string
	Site        string
	Creator     string
}

// PageSEOInput holds page info needed for SEO generation
type PageSEOInput struct {
	Title        string
	Description  string
	URL          string
	Type         string // "blog", "doc", "page"
	Date         time.Time
	ModifiedDate time.Time
	Tags         []string
	Image        string
	AuthorName   string
	Content      string // raw content for auto-description
}

// Config holds SEO configuration
type Config struct {
	SiteTitle     string
	SiteURL       string
	DefaultImage  string
	DefaultDesc   string
	TitleTemplate string // e.g. "%s | Site Name"
	TwitterSite   string
	TwitterCreator string
	AuthorName    string
}

// GenerateMeta creates SEO metadata for a page
func GenerateMeta(page PageSEOInput, cfg Config) *SEOMeta {
	meta := &SEOMeta{}

	// Title
	if cfg.TitleTemplate != "" && page.Title != "" {
		meta.Title = fmt.Sprintf(cfg.TitleTemplate, page.Title)
	} else if page.Title != "" {
		meta.Title = page.Title
		if cfg.SiteTitle != "" {
			meta.Title = page.Title + " | " + cfg.SiteTitle
		}
	} else {
		meta.Title = cfg.SiteTitle
	}

	// Description — use page description, or auto-generate from content
	meta.Description = page.Description
	if meta.Description == "" && page.Content != "" {
		meta.Description = autoDescription(page.Content, 160)
	}
	if meta.Description == "" {
		meta.Description = cfg.DefaultDesc
	}

	// Canonical URL
	canonical := page.URL
	if cfg.SiteURL != "" && !strings.HasPrefix(canonical, "http") {
		canonical = strings.TrimSuffix(cfg.SiteURL, "/") + canonical
	}
	meta.Canonical = canonical

	// Robots
	meta.Robots = "index, follow"

	// Image
	image := page.Image
	if image == "" {
		image = cfg.DefaultImage
	}
	if image != "" && !strings.HasPrefix(image, "http") {
		image = strings.TrimSuffix(cfg.SiteURL, "/") + image
	}

	// Open Graph
	meta.OG = OpenGraph{
		Title:       page.Title,
		Description: meta.Description,
		Type:        ogType(page.Type),
		URL:         meta.Canonical,
		Image:       image,
		SiteName:    cfg.SiteTitle,
	}

	// Twitter Card
	cardType := "summary"
	if image != "" {
		cardType = "summary_large_image"
	}
	meta.Twitter = TwitterCard{
		Card:        cardType,
		Title:       page.Title,
		Description: meta.Description,
		Image:       image,
		Site:        cfg.TwitterSite,
		Creator:     cfg.TwitterCreator,
	}

	// JSON-LD
	meta.JSONLD = generateJSONLD(page, cfg, meta.Canonical)

	return meta
}

// RenderMetaTags generates HTML meta tags string
func RenderMetaTags(meta *SEOMeta) string {
	var sb strings.Builder

	// Basic tags
	sb.WriteString(fmt.Sprintf("    <title>%s</title>\n", escapeHTML(meta.Title)))
	if meta.Description != "" {
		sb.WriteString(fmt.Sprintf("    <meta name=\"description\" content=\"%s\">\n", escapeHTML(meta.Description)))
	}
	if meta.Robots != "" {
		sb.WriteString(fmt.Sprintf("    <meta name=\"robots\" content=\"%s\">\n", meta.Robots))
	}
	if meta.Canonical != "" {
		sb.WriteString(fmt.Sprintf("    <link rel=\"canonical\" href=\"%s\">\n", meta.Canonical))
	}

	// Open Graph
	sb.WriteString("\n    <!-- Open Graph -->\n")
	sb.WriteString(fmt.Sprintf("    <meta property=\"og:title\" content=\"%s\">\n", escapeHTML(meta.OG.Title)))
	if meta.OG.Description != "" {
		sb.WriteString(fmt.Sprintf("    <meta property=\"og:description\" content=\"%s\">\n", escapeHTML(meta.OG.Description)))
	}
	sb.WriteString(fmt.Sprintf("    <meta property=\"og:type\" content=\"%s\">\n", meta.OG.Type))
	if meta.OG.URL != "" {
		sb.WriteString(fmt.Sprintf("    <meta property=\"og:url\" content=\"%s\">\n", meta.OG.URL))
	}
	if meta.OG.Image != "" {
		sb.WriteString(fmt.Sprintf("    <meta property=\"og:image\" content=\"%s\">\n", meta.OG.Image))
	}
	if meta.OG.SiteName != "" {
		sb.WriteString(fmt.Sprintf("    <meta property=\"og:site_name\" content=\"%s\">\n", escapeHTML(meta.OG.SiteName)))
	}

	// Twitter Card
	sb.WriteString("\n    <!-- Twitter Card -->\n")
	sb.WriteString(fmt.Sprintf("    <meta name=\"twitter:card\" content=\"%s\">\n", meta.Twitter.Card))
	sb.WriteString(fmt.Sprintf("    <meta name=\"twitter:title\" content=\"%s\">\n", escapeHTML(meta.Twitter.Title)))
	if meta.Twitter.Description != "" {
		sb.WriteString(fmt.Sprintf("    <meta name=\"twitter:description\" content=\"%s\">\n", escapeHTML(meta.Twitter.Description)))
	}
	if meta.Twitter.Image != "" {
		sb.WriteString(fmt.Sprintf("    <meta name=\"twitter:image\" content=\"%s\">\n", meta.Twitter.Image))
	}
	if meta.Twitter.Site != "" {
		sb.WriteString(fmt.Sprintf("    <meta name=\"twitter:site\" content=\"%s\">\n", meta.Twitter.Site))
	}
	if meta.Twitter.Creator != "" {
		sb.WriteString(fmt.Sprintf("    <meta name=\"twitter:creator\" content=\"%s\">\n", meta.Twitter.Creator))
	}

	// JSON-LD
	if meta.JSONLD != "" {
		sb.WriteString("\n    <!-- Structured Data -->\n")
		sb.WriteString("    <script type=\"application/ld+json\">\n    ")
		sb.WriteString(meta.JSONLD)
		sb.WriteString("\n    </script>\n")
	}

	return sb.String()
}

func ogType(pageType string) string {
	switch pageType {
	case "blog":
		return "article"
	default:
		return "website"
	}
}

func autoDescription(content string, maxLen int) string {
	// Strip markdown-ish formatting
	desc := content
	// Remove headers
	lines := strings.Split(desc, "\n")
	var textLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		textLines = append(textLines, trimmed)
	}
	desc = strings.Join(textLines, " ")

	// Remove markdown links
	desc = strings.NewReplacer("[", "", "](", " ", ")", "").Replace(desc)

	// Truncate
	if len(desc) > maxLen {
		desc = desc[:maxLen-3] + "..."
	}

	return desc
}

type jsonLDData struct {
	Context      string      `json:"@context"`
	Type         string      `json:"@type"`
	Headline     string      `json:"headline"`
	Description  string      `json:"description,omitempty"`
	Author       *jsonLDPerson `json:"author,omitempty"`
	DatePublished string     `json:"datePublished,omitempty"`
	DateModified string      `json:"dateModified,omitempty"`
	URL          string      `json:"url,omitempty"`
	Keywords     string      `json:"keywords,omitempty"`
	Image        string      `json:"image,omitempty"`
}

type jsonLDPerson struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

func generateJSONLD(page PageSEOInput, cfg Config, canonical string) string {
	ld := jsonLDData{
		Context:  "https://schema.org",
		Headline: page.Title,
		Description: page.Description,
		URL:      canonical,
	}

	switch page.Type {
	case "blog":
		ld.Type = "BlogPosting"
	default:
		ld.Type = "WebPage"
	}

	authorName := page.AuthorName
	if authorName == "" {
		authorName = cfg.AuthorName
	}
	if authorName != "" {
		ld.Author = &jsonLDPerson{
			Type: "Person",
			Name: authorName,
		}
	}

	if !page.Date.IsZero() {
		ld.DatePublished = page.Date.Format("2006-01-02")
	}
	if !page.ModifiedDate.IsZero() {
		ld.DateModified = page.ModifiedDate.Format("2006-01-02")
	}

	if len(page.Tags) > 0 {
		ld.Keywords = strings.Join(page.Tags, ", ")
	}

	if page.Image != "" {
		img := page.Image
		if !strings.HasPrefix(img, "http") {
			img = strings.TrimSuffix(cfg.SiteURL, "/") + img
		}
		ld.Image = img
	}

	data, err := json.MarshalIndent(ld, "    ", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
