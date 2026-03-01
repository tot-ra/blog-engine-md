package builder

import (
	"fmt"
	"html/template"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tot-ra/blog-engine/internal/parser"
)

// SectionChild represents a child entry on a section index page
type SectionChild struct {
	Title       string
	URL         string
	Description string
	Date        *time.Time
	IsSection   bool
}

// SectionIndexGenerator generates index pages for directories without explicit index.md
type SectionIndexGenerator struct{}

const maxSectionBlogPosts = 30

// NewSectionIndexGenerator creates a new section index generator
func NewSectionIndexGenerator() *SectionIndexGenerator {
	return &SectionIndexGenerator{}
}

// GenerateMissing creates index pages for any section nodes that don't already have a page
func (g *SectionIndexGenerator) GenerateMissing(pages map[string]*Page, tree *NavTree, defaultLang string, languages map[string]struct{}) []*Page {
	var generated []*Page

	var walk func(node *NavNode)
	walk = func(node *NavNode) {
		if node.Type == "section" {
			// Check if an index page already exists for this section
			exists := false
			for _, p := range pages {
				if p.URL == node.URL {
					exists = true
					break
				}
			}

			if !exists && node.URL != "" {
				page := g.generateIndexPage(node, pages, defaultLang, languages)
				if page != nil {
					generated = append(generated, page)
				}
			}
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	for _, child := range tree.Root.Children {
		walk(child)
	}

	return generated
}

// generateIndexPage creates an index page for a section node
func (g *SectionIndexGenerator) generateIndexPage(node *NavNode, pages map[string]*Page, defaultLang string, languages map[string]struct{}) *Page {
	// Collect children info
	var children []SectionChild
	for _, child := range node.Children {
		sc := SectionChild{
			Title:     child.Title,
			URL:       child.URL,
			IsSection: child.Type == "section",
		}
		children = append(children, sc)
	}

	// Sort: sections first, then alphabetical
	sort.SliceStable(children, func(i, j int) bool {
		if children[i].IsSection != children[j].IsSection {
			return children[i].IsSection
		}
		return strings.ToLower(children[i].Title) < strings.ToLower(children[j].Title)
	})

	var sb strings.Builder
	if strings.HasSuffix(strings.TrimSuffix(node.URL, "/"), "/blog") {
		posts := collectBlogPostsForSection(node.URL, pages)
		if len(posts) > 0 {
			sb.WriteString("<div class=\"section-article-list\">\n")
			for _, post := range posts {
				sb.WriteString("  <article class=\"section-article-preview\">\n")
				sb.WriteString(fmt.Sprintf("    <h2><a href=\"%s\">%s</a></h2>\n", template.HTMLEscapeString(post.URL), template.HTMLEscapeString(post.Title)))
				if post.Frontmatter != nil && !post.Frontmatter.Date.IsZero() {
					sb.WriteString(fmt.Sprintf("    <time datetime=\"%s\">%s</time>\n",
						post.Frontmatter.Date.Format(time.RFC3339),
						post.Frontmatter.Date.Format("2006-01-02"),
					))
				}
				excerpt := extractPreviewText(post.RawContent, 2, 320)
				if excerpt == "" {
					excerpt = strings.TrimSpace(post.Description)
				}
				if excerpt != "" {
					sb.WriteString(fmt.Sprintf("    <p>%s</p>\n", template.HTMLEscapeString(excerpt)))
				}
				sb.WriteString("  </article>\n")
			}
			sb.WriteString("</div>\n")
		}
	}
	if sb.Len() == 0 {
		sb.WriteString("<ul class=\"section-index\">\n")
		for _, child := range children {
			sb.WriteString(fmt.Sprintf("  <li><a href=\"%s\">%s</a></li>\n", template.HTMLEscapeString(child.URL), template.HTMLEscapeString(child.Title)))
		}
		sb.WriteString("</ul>\n")
	}

	pageType := determinePageType(strings.TrimPrefix(node.URL, "/"))

	lang := defaultLang
	trimmed := strings.Trim(node.URL, "/")
	if trimmed != "" {
		parts := strings.Split(trimmed, "/")
		if len(parts) > 0 {
			if _, ok := languages[strings.ToLower(parts[0])]; ok {
				lang = strings.ToLower(parts[0])
			}
		}
	}

	return &Page{
		ID:           node.ID + "-index",
		URL:          node.URL,
		Language:     lang,
		Title:        node.Title,
		Description:  fmt.Sprintf("Index of %s", node.Title),
		Content:      sb.String(),
		Frontmatter:  &parser.Frontmatter{},
		TOC:          nil,
		Type:         pageType,
		ModifiedTime: time.Now(),
	}
}

func collectBlogPostsForSection(sectionURL string, pages map[string]*Page) []*Page {
	posts := make([]*Page, 0)
	for _, page := range pages {
		if page == nil || page.Type != TypeBlog || page.URL == "" {
			continue
		}
		// Keep only content pages discovered from source markdown files.
		if strings.TrimSpace(page.SourcePath) == "" {
			continue
		}
		if !strings.HasPrefix(page.URL, sectionURL) {
			continue
		}
		posts = append(posts, page)
	}
	sort.Slice(posts, func(i, j int) bool {
		di := sectionPageSortDate(posts[i])
		dj := sectionPageSortDate(posts[j])
		if !di.Equal(dj) {
			return di.After(dj)
		}
		return strings.ToLower(posts[i].Title) < strings.ToLower(posts[j].Title)
	})
	if len(posts) > maxSectionBlogPosts {
		posts = posts[:maxSectionBlogPosts]
	}
	return posts
}

func sectionPageSortDate(page *Page) time.Time {
	if page == nil {
		return time.Time{}
	}
	if page.Frontmatter != nil && !page.Frontmatter.Date.IsZero() {
		return page.Frontmatter.Date
	}
	return page.ModifiedTime
}

var (
	markdownImageRe   = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	markdownLinkRe    = regexp.MustCompile(`\[(.*?)\]\([^)]+\)`)
	markdownRefLinkRe = regexp.MustCompile(`\[[^\]]+\]:\s+\S+`)
	htmlTagRe         = regexp.MustCompile(`<[^>]+>`)
	spaceRe           = regexp.MustCompile(`\s+`)
	tableDividerRe    = regexp.MustCompile(`^\s*\|?[\s:-]+\|[\s|:-]*$`)
	numberedListRe    = regexp.MustCompile(`^\d+\.\s+`)
)

func extractPreviewText(markdown string, maxSentences, maxChars int) string {
	if maxSentences <= 0 {
		maxSentences = 2
	}
	if maxChars <= 0 {
		maxChars = 320
	}

	lines := strings.Split(markdown, "\n")
	fragments := make([]string, 0, len(lines))
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if markdownRefLinkRe.MatchString(trimmed) {
			continue
		}
		if tableDividerRe.MatchString(trimmed) || strings.HasPrefix(trimmed, "|") {
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "<img") || strings.HasPrefix(lower, "<iframe") || strings.HasPrefix(lower, "<video") ||
			strings.HasPrefix(lower, "<table") || strings.HasPrefix(lower, "</table") || strings.HasPrefix(lower, "<thead") ||
			strings.HasPrefix(lower, "<tbody") || strings.HasPrefix(lower, "<tr") || strings.HasPrefix(lower, "<td") ||
			strings.HasPrefix(lower, "<th") || strings.HasPrefix(lower, "<figure") {
			continue
		}

		cleaned := markdownImageRe.ReplaceAllString(trimmed, "")
		cleaned = markdownLinkRe.ReplaceAllString(cleaned, "$1")
		cleaned = strings.TrimLeft(cleaned, "-*+> ")
		cleaned = numberedListRe.ReplaceAllString(cleaned, "")
		cleaned = htmlTagRe.ReplaceAllString(cleaned, " ")
		cleaned = strings.TrimSpace(spaceRe.ReplaceAllString(cleaned, " "))
		if cleaned == "" {
			continue
		}
		fragments = append(fragments, cleaned)
	}

	if len(fragments) == 0 {
		return ""
	}
	return firstSentences(strings.Join(fragments, " "), maxSentences, maxChars)
}

func firstSentences(text string, maxSentences, maxChars int) string {
	if text == "" {
		return ""
	}

	sentences := 0
	var sb strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		sb.WriteRune(r)
		if i+1 >= maxChars {
			break
		}
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		if i < len(runes)-1 && runes[i+1] != ' ' {
			continue
		}
		sentences++
		if sentences >= maxSentences {
			break
		}
	}

	out := strings.TrimSpace(sb.String())
	if out == "" {
		return out
	}
	if len([]rune(out)) < len(runes) && !strings.HasSuffix(out, ".") && !strings.HasSuffix(out, "!") && !strings.HasSuffix(out, "?") {
		out += "..."
	}
	return out
}
