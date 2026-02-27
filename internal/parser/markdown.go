package parser

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

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
