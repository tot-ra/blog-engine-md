package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// wikiLinkRegex matches [[Page Title]] or [[Page Title|Display Text]]
var wikiLinkRegex = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// MarkdownParser handles markdown to HTML conversion
type MarkdownParser struct {
	md goldmark.Markdown
}

// NewMarkdownParser creates a new markdown parser with GFM support
func NewMarkdownParser() *MarkdownParser {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // GitHub Flavored Markdown
			extension.Table,
			extension.Strikethrough,
			extension.Linkify,
			extension.TaskList,
			meta.Meta,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)

	return &MarkdownParser{md: md}
}

// Render converts markdown content to HTML
func (p *MarkdownParser) Render(content string) (string, error) {
	var buf bytes.Buffer
	if err := p.md.Convert([]byte(content), &buf); err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}
	return buf.String(), nil
}

// RenderWithMeta converts markdown and extracts metadata
func (p *MarkdownParser) RenderWithMeta(content string) (string, map[string]interface{}, error) {
	var buf bytes.Buffer
	context := parser.NewContext()
	
	if err := p.md.Convert([]byte(content), &buf, parser.WithContext(context)); err != nil {
		return "", nil, fmt.Errorf("failed to render markdown: %w", err)
	}

	metaData := meta.Get(context)
	return buf.String(), metaData, nil
}

// ProcessWikiLinks converts [[Page Title]] wiki-style links to markdown links
// It uses the provided page resolver to map page titles to URLs
func ProcessWikiLinks(content string, pageResolver func(title string) (url string, exists bool)) string {
	return wikiLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := wikiLinkRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		pageTitle := strings.TrimSpace(submatches[1])
		displayText := pageTitle
		if len(submatches) > 2 && submatches[2] != "" {
			displayText = strings.TrimSpace(submatches[2])
		}

		if url, exists := pageResolver(pageTitle); exists {
			return fmt.Sprintf("[%s](%s)", displayText, url)
		}
		// If page doesn't exist, render as plain text with styling
		return fmt.Sprintf("<span class=\"wiki-link-missing\">%s</span>", displayText)
	})
}

// SimpleWikiLinkProcessor converts wiki links to markdown links using a simple slug generator
// Use this when you don't have access to the full page index
func SimpleWikiLinkProcessor(content string) string {
	return wikiLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := wikiLinkRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		pageTitle := strings.TrimSpace(submatches[1])
		displayText := pageTitle
		if len(submatches) > 2 && submatches[2] != "" {
			displayText = strings.TrimSpace(submatches[2])
		}

		// Generate slug from title
		slug := GenerateSlug(pageTitle)
		return fmt.Sprintf("[%s](/%s/)", displayText, slug)
	})
}
