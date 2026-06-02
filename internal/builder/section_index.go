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

var sectionClassTitleRe = regexp.MustCompile(`(?i)^\s*\d+(?:\s*-\s*\d+)?\s*klass\b`)
var sectionClassURLRe = regexp.MustCompile(`(?i)/(?:\d+(?:-\d+)?)klass/$`)

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
	children := sectionChildrenFromNode(node)

	var sb strings.Builder
	if strings.HasSuffix(strings.TrimSuffix(node.URL, "/"), "/blog") {
		sb.WriteString(sectionBlogPostsHTML(node.URL, pages))
	}
	if sb.Len() == 0 {
		sb.WriteString(sectionChildrenHTML(children))
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

func sectionChildrenFromNode(node *NavNode) []SectionChild {
	if node == nil || len(node.Children) == 0 {
		return nil
	}

	children := make([]SectionChild, 0, len(node.Children))
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		children = append(children, SectionChild{
			Title:     child.Title,
			URL:       child.URL,
			IsSection: child.Type == "section",
		})
	}
	return children
}

func sectionChildrenHTML(children []SectionChild) string {
	if len(children) == 0 {
		return ""
	}

	var sb strings.Builder
	if shouldUseSectionMatrix(children) {
		children = onlyClassLikeSectionChildren(children)
		sb.WriteString("<div class=\"section-index-grid section-index-matrix\">\n")
		for _, child := range children {
			sb.WriteString(fmt.Sprintf("  <a class=\"section-index-card\" href=\"%s\">%s</a>\n", template.HTMLEscapeString(child.URL), template.HTMLEscapeString(child.Title)))
		}
		sb.WriteString("</div>\n")
		return sb.String()
	}

	sb.WriteString("<ul class=\"section-index\">\n")
	for _, child := range children {
		sb.WriteString(fmt.Sprintf("  <li><a href=\"%s\">%s</a></li>\n", template.HTMLEscapeString(child.URL), template.HTMLEscapeString(child.Title)))
	}
	sb.WriteString("</ul>\n")
	return sb.String()
}

func sectionBlogPostsHTML(sectionURL string, pages map[string]*Page) string {
	posts := collectBlogPostsForSection(sectionURL, pages)
	if len(posts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<div class=\"section-article-list\">\n")
	for _, post := range posts {
		sb.WriteString("  <article class=\"section-article-preview\">\n")
		if imageHTML := firstPreviewImageHTML(post); imageHTML != "" {
			// Put the first post image into the card before text. The generated <img>
			// is intentionally plain here: the later image transformer upgrades it to
			// responsive <picture> markup during the normal build pipeline.
			sb.WriteString(fmt.Sprintf("    <a class=\"section-article-image\" href=\"%s\">%s</a>\n", template.HTMLEscapeString(post.URL), imageHTML))
		}
		sb.WriteString(fmt.Sprintf("    <h2><a href=\"%s\">%s</a></h2>\n", template.HTMLEscapeString(post.URL), template.HTMLEscapeString(post.Title)))
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
	return sb.String()
}

func firstPreviewImageHTML(post *Page) string {
	if post == nil || strings.TrimSpace(post.Content) == "" {
		return ""
	}
	match := firstImageSrcRe.FindStringSubmatch(post.Content)
	if len(match) < 2 || strings.TrimSpace(match[1]) == "" {
		return ""
	}

	alt := ""
	if altMatch := firstImageAltRe.FindStringSubmatch(match[0]); len(altMatch) >= 2 {
		alt = altMatch[1]
	}
	if strings.TrimSpace(alt) == "" {
		alt = post.Title
	}

	return fmt.Sprintf("<img src=\"%s\" alt=\"%s\" loading=\"lazy\">",
		template.HTMLEscapeString(match[1]),
		template.HTMLEscapeString(alt),
	)
}

func shouldUseSectionMatrix(children []SectionChild) bool {
	if len(children) < 2 {
		return false
	}

	classLike := len(onlyClassLikeSectionChildren(children))
	return classLike >= 2
}

func onlyClassLikeSectionChildren(children []SectionChild) []SectionChild {
	filtered := make([]SectionChild, 0, len(children))
	for _, child := range children {
		if isClassLikeSectionChild(child) {
			filtered = append(filtered, child)
		}
	}
	return filtered
}

func isClassLikeSectionChild(child SectionChild) bool {
	if sectionClassTitleRe.MatchString(strings.TrimSpace(child.Title)) {
		return true
	}
	return sectionClassURLRe.MatchString(strings.TrimSpace(child.URL))
}

func collectBlogPostsForSection(sectionURL string, pages map[string]*Page) []*Page {
	posts := make([]*Page, 0)
	for _, page := range pages {
		if page == nil || page.Type != TypeBlog || page.URL == "" {
			continue
		}
		if page.URL == sectionURL {
			continue
		}
		if page.Frontmatter != nil {
			if page.Frontmatter.HideNav || strings.TrimSpace(page.Frontmatter.RedirectURL) != "" {
				continue
			}
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
	firstImageAltRe   = regexp.MustCompile(`(?is)\balt\s*=\s*['"]([^'"]*)['"]`)
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
