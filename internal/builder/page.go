package builder

import (
	"fmt"
	"os"
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

// TocItem represents a table of contents entry
type TocItem struct {
	Level    int
	Text     string
	Anchor   string
	Children []*TocItem
}

// Page represents a single page to be rendered
type Page struct {
	ID          string
	URL         string
	SourcePath  string
	Title       string
	Description string
	Content     string
	RawContent  string
	Frontmatter *parser.Frontmatter
	TOC         []*TocItem
	Type        PageType
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
	urlGen   *URLGenerator
	mdParser *parser.MarkdownParser
}

// NewPageBuilder creates a new page builder
func NewPageBuilder(baseURL string) *PageBuilder {
	return &PageBuilder{
		urlGen:   NewURLGenerator(baseURL),
		mdParser: parser.NewMarkdownParser(),
	}
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

	// Determine page type
	pageType := determinePageType(file.RelativePath)

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

	// Render markdown to HTML
	htmlContent, err := b.mdParser.Render(remaining)
	if err != nil {
		return nil, fmt.Errorf("failed to render markdown in %s: %w", file.Path, err)
	}

	// Extract TOC from content
	toc := extractTOC(remaining)

	page := &Page{
		ID:           id,
		URL:          url,
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

// determinePageType determines the page type from the file path
func determinePageType(relPath string) PageType {
	if strings.HasPrefix(relPath, "blog/") {
		return TypeBlog
	}
	if strings.HasPrefix(relPath, "docs/") {
		return TypeDoc
	}
	return TypePage
}

// generateID creates a unique ID from a URL
func generateID(url string) string {
	id := strings.Trim(url, "/")
	id = strings.ReplaceAll(id, "/", "-")
	id = regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(id, "")
	return id
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

		// Extract text
		text := strings.TrimSpace(line[level:])
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

// readFile reads a file and returns its content
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
