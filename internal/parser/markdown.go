package parser

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"regexp"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

// wikiLinkRegex matches [[Page Title]] or [[Page Title|Display Text]]
var wikiLinkRegex = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)
var mdxStyleAttrRegex = regexp.MustCompile(`style=\{\{([^{}]+)\}\}`)
var markdownImageLineRegex = regexp.MustCompile(`(?m)^\s*!\[([^\]]*)\]\(([^)]+)\)\s*$`)

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
			// WHY: emit syntax token classes at build time so sites can theme code
			// without shipping a client-side highlighter or inline color styles.
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(html.WithClasses(true)),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			ghtml.WithHardWraps(),
			ghtml.WithXHTML(),
			ghtml.WithUnsafe(),
		),
	)

	return &MarkdownParser{md: md}
}

// Render converts markdown content to HTML
func (p *MarkdownParser) Render(content string) (string, error) {
	content = preprocessMDXCompat(content)
	var buf bytes.Buffer
	// WHY: use slug IDs so heading anchors match TOC links for Cyrillic headings.
	context := parser.NewContext(parser.WithIDs(NewSlugIDs()))
	if err := p.md.Convert([]byte(content), &buf, parser.WithContext(context)); err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}
	return buf.String(), nil
}

// RenderWithMeta converts markdown and extracts metadata
func (p *MarkdownParser) RenderWithMeta(content string) (string, map[string]interface{}, error) {
	content = preprocessMDXCompat(content)
	var buf bytes.Buffer
	context := parser.NewContext(parser.WithIDs(NewSlugIDs()))

	if err := p.md.Convert([]byte(content), &buf, parser.WithContext(context)); err != nil {
		return "", nil, fmt.Errorf("failed to render markdown: %w", err)
	}

	metaData := meta.Get(context)
	return buf.String(), metaData, nil
}

func preprocessMDXCompat(content string) string {
	segments := splitMarkdownByCodeFences(content)
	for i := range segments {
		if segments[i].isCode {
			continue
		}
		segments[i].text = mdxStyleAttrRegex.ReplaceAllStringFunc(segments[i].text, func(match string) string {
			submatches := mdxStyleAttrRegex.FindStringSubmatch(match)
			if len(submatches) != 2 {
				return match
			}
			return `style="` + stdhtml.EscapeString(mdxStyleObjectToCSS(submatches[1])) + `"`
		})
		segments[i].text = markdownImageLineRegex.ReplaceAllStringFunc(segments[i].text, func(match string) string {
			if !isLikelyInsideHTMLBlock(segments[i].text, match) {
				return match
			}
			submatches := markdownImageLineRegex.FindStringSubmatch(match)
			if len(submatches) != 3 {
				return match
			}
			alt := stdhtml.EscapeString(submatches[1])
			src, title := splitMarkdownImageDestination(strings.TrimSpace(submatches[2]))
			if src == "" {
				return match
			}

			attrs := []string{
				`src="` + stdhtml.EscapeString(src) + `"`,
				`alt="` + alt + `"`,
			}
			if title != "" {
				attrs = append(attrs, `title="`+stdhtml.EscapeString(title)+`"`)
			}
			return "<img " + strings.Join(attrs, " ") + ">"
		})
	}
	return joinMarkdownSegments(segments)
}

type markdownSegment struct {
	text   string
	isCode bool
}

func splitMarkdownByCodeFences(content string) []markdownSegment {
	lines := strings.SplitAfter(content, "\n")
	segments := []markdownSegment{}
	var current strings.Builder
	inFence := false

	flush := func() {
		if current.Len() == 0 {
			return
		}
		segments = append(segments, markdownSegment{text: current.String(), isCode: inFence})
		current.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		isFence := strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
		if isFence {
			flush()
			current.WriteString(line)
			flush()
			inFence = !inFence
			continue
		}
		current.WriteString(line)
	}
	flush()

	return segments
}

func joinMarkdownSegments(segments []markdownSegment) string {
	var out strings.Builder
	for _, segment := range segments {
		out.WriteString(segment.text)
	}
	return out.String()
}

func mdxStyleObjectToCSS(style string) string {
	parts := strings.Split(style, ",")
	css := make([]string, 0, len(parts))
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		key = strings.Trim(key, `"'`)
		value := normalizeMDXStyleValue(key, strings.TrimSpace(kv[1]))
		value = strings.Trim(value, `"'`)
		if key == "" || value == "" {
			continue
		}
		css = append(css, camelToKebab(key)+":"+value)
	}
	sort.Strings(css)
	return strings.Join(css, ";")
}

func normalizeMDXStyleValue(key, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, `'`) {
		return value
	}
	if !isUnitlessCSSProperty(key) && isIntegerLiteral(value) {
		return value + "px"
	}
	return value
}

func isIntegerLiteral(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isUnitlessCSSProperty(key string) bool {
	switch key {
	case "animationIterationCount", "borderImageOutset", "borderImageSlice", "borderImageWidth", "boxFlex", "boxFlexGroup", "boxOrdinalGroup", "columnCount", "columns", "flex", "flexGrow", "flexPositive", "flexShrink", "flexNegative", "flexOrder", "gridArea", "gridRow", "gridRowEnd", "gridRowSpan", "gridRowStart", "gridColumn", "gridColumnEnd", "gridColumnSpan", "gridColumnStart", "fontWeight", "lineClamp", "lineHeight", "opacity", "order", "orphans", "tabSize", "widows", "zIndex", "zoom":
		return true
	default:
		return false
	}
}

func camelToKebab(s string) string {
	var out strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				out.WriteByte('-')
			}
			out.WriteRune(r + ('a' - 'A'))
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func isLikelyInsideHTMLBlock(content, match string) bool {
	idx := strings.Index(content, match)
	if idx < 0 {
		return false
	}
	before := content[:idx]
	lastOpen := strings.LastIndex(before, "<")
	lastClose := strings.LastIndex(before, ">")
	if lastClose <= lastOpen {
		return false
	}

	lastOpenTagStart := strings.LastIndex(before, "<")
	if lastOpenTagStart < 0 {
		return false
	}
	lastTag := before[lastOpenTagStart:]
	if strings.HasPrefix(lastTag, "</") || strings.HasSuffix(strings.TrimSpace(lastTag), "/>") {
		return false
	}

	after := content[idx+len(match):]
	return strings.Contains(after, "</")
}

func splitMarkdownImageDestination(destination string) (src string, title string) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return "", ""
	}

	if strings.HasPrefix(destination, "<") {
		end := strings.Index(destination, ">")
		if end > 0 {
			src = strings.TrimSpace(destination[1:end])
			title = strings.TrimSpace(destination[end+1:])
			title = strings.Trim(title, `"'`)
			return src, title
		}
	}

	if strings.Count(destination, `"`) >= 2 {
		first := strings.Index(destination, `"`)
		last := strings.LastIndex(destination, `"`)
		if first >= 0 && last > first {
			src = strings.TrimSpace(destination[:first])
			title = destination[first+1 : last]
			return src, title
		}
	}
	if strings.Count(destination, `'`) >= 2 {
		first := strings.Index(destination, `'`)
		last := strings.LastIndex(destination, `'`)
		if first >= 0 && last > first {
			src = strings.TrimSpace(destination[:first])
			title = destination[first+1 : last]
			return src, title
		}
	}
	return destination, ""
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
