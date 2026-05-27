package builder

import (
	"fmt"
	"net/url"
	"os"
	urlpath "path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tot-ra/blog-engine/internal/parser"
)

// PageType represents the type of page
type PageType string

const (
	TypeBlog PageType = "blog"
	TypeDoc  PageType = "doc"
	TypePage PageType = "page"
)

var mdLocalAssetLinkRegex = regexp.MustCompile(`\]\(([^)]+)\)`)
var mdxRequireDefaultRegex = regexp.MustCompile(`\{require\(['"]([^'"]+)['"]\)\.default\}`)
var pdfObjectRegex = regexp.MustCompile(`(?is)<object\s+([^>]*\btype=["']application/pdf["'][^>]*)>\s*</object>|<object\s+([^>]*)>\s*</object>`)
var htmlAttrRegex = regexp.MustCompile(`(?is)([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*("[^"]*"|'[^']*')`)
var localAssetExtensions = map[string]struct{}{
	".pdf":  {},
	".zip":  {},
	".csv":  {},
	".tsv":  {},
	".json": {},
	".xml":  {},
	".txt":  {},
	".doc":  {},
	".docx": {},
	".xls":  {},
	".xlsx": {},
	".ppt":  {},
	".pptx": {},
	".mp3":  {},
	".wav":  {},
	".ogg":  {},
	".mp4":  {},
	".mov":  {},
	".webm": {},
}

// TocItem represents a table of contents entry
type TocItem struct {
	Level    int
	Text     string
	Anchor   string
	Children []*TocItem
}

// Page represents a single page to be rendered
type Page struct {
	ID           string
	URL          string
	Language     string
	SourcePath   string
	Title        string
	Description  string
	Content      string
	RawContent   string
	AudioURL     string
	Frontmatter  *parser.Frontmatter
	TOC          []*TocItem
	Type         PageType
	ModifiedTime time.Time
}

// URLGenerator generates URLs from file paths
type URLGenerator struct {
	baseURL string
}

// NewURLGenerator creates a new URL generator
func NewURLGenerator(baseURL string) *URLGenerator {
	return &URLGenerator{baseURL: strings.TrimSuffix(baseURL, "/")}
}

// Generate creates a URL from a content file path and frontmatter
func (g *URLGenerator) Generate(filePath string, fm *parser.Frontmatter) string {
	// Get directory and filename
	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	// Handle index files
	if name == "index" || name == "README" {
		if dir == "." {
			return "/"
		}
		return "/" + dir + "/"
	}

	// Use custom slug if provided
	slug := fm.Slug
	if slug == "" {
		slug = parser.GenerateSlug(name)
	}

	// Build URL
	url := "/"
	if dir != "." {
		url += dir + "/"
	}
	url += slug + "/"

	// Clean up URL
	url = strings.ReplaceAll(url, "//", "/")

	return url
}

// PageBuilder builds pages from content files
type PageBuilder struct {
	urlGen               *URLGenerator
	mdParser             *parser.MarkdownParser
	pageResolver         func(title string) (url string, exists bool)
	markdownLinkResolver func(destination, pageRelPath string) (url string, exists bool)
	defaultLang          string
	languages            map[string]struct{}
}

// NewPageBuilder creates a new page builder
func NewPageBuilder(baseURL, defaultLang string, languages map[string]struct{}) *PageBuilder {
	return &PageBuilder{
		urlGen:               NewURLGenerator(baseURL),
		mdParser:             parser.NewMarkdownParser(),
		pageResolver:         nil, // Will be set later when all pages are known
		markdownLinkResolver: nil, // Will be set later when all pages are known
		defaultLang:          defaultLang,
		languages:            languages,
	}
}

// SetPageResolver sets the page resolver for wiki links
func (b *PageBuilder) SetPageResolver(resolver func(title string) (url string, exists bool)) {
	b.pageResolver = resolver
}

// SetMarkdownLinkResolver sets the resolver for local markdown links like [Page](other-page.md).
func (b *PageBuilder) SetMarkdownLinkResolver(resolver func(destination, pageRelPath string) (url string, exists bool)) {
	b.markdownLinkResolver = resolver
}

// Build creates a Page from a content file
func (b *PageBuilder) Build(file ContentFile) (*Page, error) {
	// Read file content
	content, err := readFile(file.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", file.Path, err)
	}

	// Parse frontmatter
	fm, remaining, err := parser.ParseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter in %s: %w", file.Path, err)
	}

	// Skip drafts
	if fm.Draft {
		return nil, nil
	}

	lang, contentPath := detectLanguageAndContentPath(file.RelativePath, b.defaultLang, b.languages)

	// Determine page type
	pageType := determinePageType(contentPath)
	if pageType == TypeBlog && fm.Date.IsZero() {
		fm.Date = inferDateFromFilename(contentPath)
	}

	// Generate URL
	url := b.urlGen.Generate(file.RelativePath, fm)

	// Generate ID from URL
	id := generateID(url)

	// Use filename as title if not provided
	title := fm.Title
	if title == "" {
		title = filepath.Base(file.Path)
		ext := filepath.Ext(title)
		title = strings.TrimSuffix(title, ext)
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.ReplaceAll(title, "_", " ")
	}

	// Process wiki links [[Page Title]] -> [Page Title](/page-title/)
	processedContent := remaining
	if b.pageResolver != nil {
		processedContent = parser.ProcessWikiLinks(remaining, b.pageResolver)
	} else {
		processedContent = parser.SimpleWikiLinkProcessor(remaining)
	}
	processedContent = rewriteLocalMarkdownLinks(processedContent, file.RelativePath, b.markdownLinkResolver)
	processedContent = rewriteLocalAssetReferences(processedContent, file.RelativePath)
	processedContent = rewritePDFObjects(processedContent)
	processedContent = parser.TransformEmbeds(processedContent)

	// Render markdown to HTML
	htmlContent, err := b.mdParser.Render(processedContent)
	if err != nil {
		return nil, fmt.Errorf("failed to render markdown in %s: %w", file.Path, err)
	}

	// Extract TOC from content
	toc := extractTOC(remaining)

	page := &Page{
		ID:           id,
		URL:          url,
		Language:     lang,
		SourcePath:   file.Path,
		Title:        title,
		Description:  fm.Description,
		Content:      htmlContent,
		RawContent:   remaining,
		Frontmatter:  fm,
		TOC:          toc,
		Type:         pageType,
		ModifiedTime: time.Unix(file.ModifiedTime, 0),
	}

	return page, nil
}

func inferDateFromFilename(relPath string) time.Time {
	name := filepath.Base(relPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))

	// Common pattern: "YYYY-MM-DD title"
	re := regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})(?:[\s_-].*)?$`)
	m := re.FindStringSubmatch(name)
	if len(m) != 4 {
		return time.Time{}
	}

	t, err := time.Parse("2006-01-02", m[1]+"-"+m[2]+"-"+m[3])
	if err != nil {
		return time.Time{}
	}
	return t
}

// determinePageType determines the page type from the file path
func determinePageType(relPath string) PageType {
	path := strings.Trim(strings.ReplaceAll(relPath, "\\", "/"), "/")
	if strings.HasPrefix(path, "blog/") || path == "blog" {
		return TypeBlog
	}
	if strings.HasPrefix(path, "docs/") || path == "docs" {
		return TypeDoc
	}

	// Be tolerant to language-prefixed or nested paths (e.g. ru/blog/...).
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		switch parts[1] {
		case "blog":
			return TypeBlog
		case "docs":
			return TypeDoc
		}
	}
	return TypePage
}

func rewriteLocalMarkdownLinks(content, pageRelPath string, resolver func(destination, pageRelPath string) (string, bool)) string {
	if resolver == nil {
		return content
	}

	return mdLocalAssetLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := mdLocalAssetLinkRegex.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}
		destination := strings.TrimSpace(submatches[1])
		rewritten, ok := resolver(destination, pageRelPath)
		if !ok || rewritten == "" || rewritten == destination {
			return match
		}
		return `](` + rewritten + `)`
	})
}

// generateID creates a unique ID from a URL
func generateID(url string) string {
	id := strings.Trim(url, "/")
	id = strings.ReplaceAll(id, "/", "-")
	id = regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(id, "")
	return id
}

func rewriteLocalAssetReferences(content, pageRelPath string) string {
	pageDir := filepath.ToSlash(filepath.Dir(pageRelPath))
	if pageDir == "." {
		pageDir = ""
	}

	content = mdLocalAssetLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := mdLocalAssetLinkRegex.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}
		destination := strings.TrimSpace(submatches[1])
		rewritten := rewriteLocalAssetDestination(destination, pageDir)
		if rewritten == destination || rewritten == "" {
			return match
		}
		return `](` + rewritten + `)`
	})

	content = mdxRequireDefaultRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := mdxRequireDefaultRegex.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}
		rewritten := rewriteLocalAssetDestination(submatches[1], pageDir)
		if rewritten == "" || rewritten == submatches[1] {
			return match
		}
		return `"` + rewritten + `"`
	})

	return content
}

func rewritePDFObjects(content string) string {
	return pdfObjectRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := pdfObjectRegex.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		attrs := strings.TrimSpace(submatches[1])
		if attrs == "" {
			attrs = strings.TrimSpace(submatches[2])
		}
		attrMap := parseHTMLAttrs(attrs)
		src := strings.TrimSpace(attrMap["data"])
		if src == "" {
			return match
		}
		typeAttr := strings.TrimSpace(strings.ToLower(attrMap["type"]))
		if typeAttr != "" && typeAttr != "application/pdf" {
			return match
		}
		if strings.ToLower(urlpath.Ext(src)) != ".pdf" {
			return match
		}

		height := strings.TrimSpace(attrMap["height"])
		if height == "" {
			height = "800"
		}
		return `<iframe class="pdf-embed" src="` + src + `" title="PDF preview" loading="lazy" height="` + height + `"></iframe>`
	})
}

func parseHTMLAttrs(attrs string) map[string]string {
	parsed := make(map[string]string)
	for _, match := range htmlAttrRegex.FindAllStringSubmatch(attrs, -1) {
		if len(match) != 3 {
			continue
		}
		key := strings.ToLower(match[1])
		value := strings.Trim(match[2], `"'`)
		parsed[key] = value
	}
	return parsed
}

func rewriteLocalAssetDestination(destination, pageDir string) string {
	if destination == "" || strings.HasPrefix(destination, "#") {
		return destination
	}
	if strings.HasPrefix(destination, "/") || strings.HasPrefix(destination, "//") {
		return destination
	}
	if u, err := url.Parse(destination); err == nil && u.IsAbs() {
		return destination
	}

	pathPart := destination
	suffix := ""
	if idx := strings.IndexAny(pathPart, "?#"); idx >= 0 {
		suffix = pathPart[idx:]
		pathPart = pathPart[:idx]
	}

	ext := strings.ToLower(urlpath.Ext(pathPart))
	if _, ok := localAssetExtensions[ext]; !ok {
		return destination
	}

	joined := urlpath.Clean(urlpath.Join("/assets", pageDir, pathPart))
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined + suffix
}

// extractTOC extracts table of contents from markdown content
func extractTOC(content string) []*TocItem {
	var toc []*TocItem
	var stack []*TocItem

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Match heading lines
		if !strings.HasPrefix(line, "#") {
			continue
		}

		// Count heading level
		level := 0
		for i, ch := range line {
			if ch == '#' && i < 6 {
				level++
			} else {
				break
			}
		}

		// Only include H2-H4
		if level < 2 || level > 4 {
			continue
		}

		// Extract and normalize heading text for cleaner TOC labels.
		text := sanitizeTOCHeading(strings.TrimSpace(line[level:]))
		anchor := parser.GenerateSlug(text)

		item := &TocItem{
			Level:  level,
			Text:   text,
			Anchor: anchor,
		}

		// Build hierarchy
		if len(stack) == 0 {
			toc = append(toc, item)
			stack = []*TocItem{item}
		} else {
			// Find parent
			for len(stack) > 0 && stack[len(stack)-1].Level >= level {
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 {
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, item)
			} else {
				toc = append(toc, item)
			}
			stack = append(stack, item)
		}
	}

	return toc
}

func sanitizeTOCHeading(text string) string {
	// Remove optional trailing ATX markers: "Heading ###"
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+#+\s*$`).ReplaceAllString(text, "")

	// Convert markdown links/images to visible label text.
	text = regexp.MustCompile(`!\[([^\]]*)\]\([^\)]*\)`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`\[([^\]]+)\]\([^\)]*\)`).ReplaceAllString(text, "$1")

	// Remove inline code and emphasis delimiters.
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "~~", "")

	// Collapse whitespace for stable labels/slugs.
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

// readFile reads a file and returns its content
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
