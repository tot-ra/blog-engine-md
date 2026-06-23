package builder

import (
	"fmt"
	"os"
	"strings"

	"github.com/tot-ra/blog-engine/internal/archive"
	"github.com/tot-ra/blog-engine/internal/i18n"
	"github.com/tot-ra/blog-engine/internal/parser"
	"github.com/tot-ra/blog-engine/internal/tags"
)

// generateTagPages builds tag index and creates tag list pages.
func (b *SiteBuilder) generateTagPages(taggedPages []*Page) error {
	postsByLang := make(map[string][]*Page)
	for _, p := range taggedPages {
		postsByLang[p.Language] = append(postsByLang[p.Language], p)
	}
	total := 0
	for lang, posts := range postsByLang {
		summaries := pageSummariesFromPosts(posts)
		tagIdx := tags.BuildTagIndex(summaries)

		allTags := tagIdx.Tags()
		if len(allTags) == 0 {
			continue
		}

		ui := i18n.UI(lang)
		tagCloudHTML := b.buildTagCloudHTML(tagIdx, allTags, lang)
		tagsURL := b.buildLanguageScopedURL(lang, "tags")
		tagCloudPage := &Page{
			ID:          lang + "-tags",
			URL:         tagsURL,
			Language:    lang,
			Title:       ui.Tags,
			Description: "All tags",
			Content:     tagCloudHTML,
			RawContent:  "",
			Frontmatter: &parser.Frontmatter{},
			Type:        TypePage,
		}
		b.pages[tagCloudPage.ID] = tagCloudPage
		b.pagesByURL[tagCloudPage.URL] = tagCloudPage
		if err := b.renderPage(tagCloudPage); err != nil {
			return fmt.Errorf("failed to render tag cloud page: %w", err)
		}

		for _, tag := range allTags {
			tagPages := tagIdx[tag]
			tagSlug := parser.GenerateSlug(tag)
			tagPageHTML := b.buildTagPageHTML(tag, tagPages, lang)
			tagURL := b.buildLanguageScopedURL(lang, "tags/"+tagSlug)

			tagPage := &Page{
				ID:          lang + "-tags-" + tagSlug,
				URL:         tagURL,
				Language:    lang,
				Title:       "Tag: " + tag,
				Description: fmt.Sprintf("Pages tagged with \"%s\"", tag),
				Content:     tagPageHTML,
				RawContent:  "",
				Frontmatter: &parser.Frontmatter{Tags: []string{tag}},
				Type:        TypePage,
			}
			b.pages[tagPage.ID] = tagPage
			b.pagesByURL[tagPage.URL] = tagPage
			if err := b.renderPage(tagPage); err != nil {
				fmt.Fprintf(os.Stderr, "Error rendering tag page %s: %v\n", tag, err)
				continue
			}
			total++
		}
	}
	fmt.Printf("Generated %d tag pages\n", total)
	return nil
}

// buildTagCloudHTML generates HTML for the tag cloud page
func (b *SiteBuilder) buildTagCloudHTML(idx tags.TagIndex, allTags []string, lang string) string {
	var sb strings.Builder
	sb.WriteString("<div class=\"tag-cloud\">\n")
	sb.WriteString("<ul class=\"tag-list\">\n")
	for _, tag := range allTags {
		slug := parser.GenerateSlug(tag)
		count := idx.Count(tag)
		sb.WriteString(fmt.Sprintf("  <li><a href=\"%s\" class=\"tag\">%s</a> <span class=\"tag-count\">(%d)</span></li>\n", b.buildLanguageScopedURL(lang, "tags/"+slug), tag, count))
	}
	sb.WriteString("</ul>\n")
	sb.WriteString("</div>\n")
	return sb.String()
}

// buildTagPageHTML generates HTML for a single tag page
func (b *SiteBuilder) buildTagPageHTML(tag string, pages []tags.PageSummary, lang string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h2>Pages tagged \"%s\"</h2>\n", tag))
	sb.WriteString(fmt.Sprintf("<p>%d page(s)</p>\n", len(pages)))
	sb.WriteString("<ul class=\"post-list\">\n")
	for _, p := range pages {
		dateStr := ""
		if !p.Date.IsZero() {
			dateStr = fmt.Sprintf(" <time>%s</time>", i18n.FormatDateLong(p.Date, lang))
		}
		sb.WriteString(fmt.Sprintf("  <li><a href=\"%s\">%s</a>%s</li>\n", p.URL, p.Title, dateStr))
	}
	sb.WriteString("</ul>\n")
	return sb.String()
}

// generateArchivePages builds archive structure and creates archive pages
func (b *SiteBuilder) generateArchivePages(blogPosts []*Page) error {
	byLang := make(map[string][]archive.PageSummary)
	for _, p := range blogPosts {
		byLang[p.Language] = append(byLang[p.Language], archive.PageSummary{
			Title:       p.Title,
			URL:         p.URL,
			Date:        p.Frontmatter.Date,
			Description: p.Description,
			Tags:        p.Frontmatter.Tags,
			Type:        string(p.Type),
		})
	}

	totalYears := 0
	for lang, summaries := range byLang {
		archiveData := archive.BuildArchive(summaries)
		if len(archiveData) == 0 {
			continue
		}

		archiveHTML := b.buildArchiveIndexHTML(archiveData, lang)
		archiveURL := b.buildLanguageScopedURL(lang, "archive")
		archivePage := &Page{
			ID:          lang + "-archive",
			URL:         archiveURL,
			Language:    lang,
			Title:       i18n.SegmentLabel(lang, "archive"),
			Description: "Post archive by date",
			Content:     archiveHTML,
			RawContent:  "",
			Frontmatter: &parser.Frontmatter{},
			Type:        TypePage,
		}
		b.pages[archivePage.ID] = archivePage
		b.pagesByURL[archivePage.URL] = archivePage
		if err := b.renderPage(archivePage); err != nil {
			return fmt.Errorf("failed to render archive page: %w", err)
		}

		for _, year := range archiveData {
			yearHTML := b.buildArchiveYearHTML(year, lang)
			yearURL := b.buildLanguageScopedURL(lang, fmt.Sprintf("archive/%d", year.Year))
			yearPage := &Page{
				ID:          fmt.Sprintf("%s-archive-%d", lang, year.Year),
				URL:         yearURL,
				Language:    lang,
				Title:       fmt.Sprintf("%s: %d", i18n.SegmentLabel(lang, "archive"), year.Year),
				Description: fmt.Sprintf("Posts from %d", year.Year),
				Content:     yearHTML,
				RawContent:  "",
				Frontmatter: &parser.Frontmatter{},
				Type:        TypePage,
			}
			b.pages[yearPage.ID] = yearPage
			b.pagesByURL[yearPage.URL] = yearPage
			if err := b.renderPage(yearPage); err != nil {
				fmt.Fprintf(os.Stderr, "Error rendering archive year page %d: %v\n", year.Year, err)
			}
		}
		totalYears += len(archiveData)
	}

	fmt.Printf("Generated archive pages (%d years)\n", totalYears)
	return nil
}

// buildArchiveIndexHTML generates HTML for the main archive page
func (b *SiteBuilder) buildArchiveIndexHTML(years []archive.ArchiveYear, lang string) string {
	var sb strings.Builder
	sb.WriteString("<div class=\"archive\">\n")
	for _, year := range years {
		yearURL := b.buildLanguageScopedURL(lang, fmt.Sprintf("archive/%d", year.Year))
		sb.WriteString(fmt.Sprintf("<h2><a href=\"%s\">%d</a> <span class=\"count\">(%d)</span></h2>\n", yearURL, year.Year, year.Count))
		for _, month := range year.Months {
			sb.WriteString(fmt.Sprintf("<h3>%s %d</h3>\n", i18n.MonthName(lang, month.Month), month.Year))
			sb.WriteString("<ul class=\"post-list\">\n")
			for _, p := range month.Pages {
				dateStr := i18n.FormatDateShort(p.Date, lang)
				sb.WriteString(fmt.Sprintf("  <li><time>%s</time> <a href=\"%s\">%s</a></li>\n", dateStr, p.URL, p.Title))
			}
			sb.WriteString("</ul>\n")
		}
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// buildArchiveYearHTML generates HTML for a single year archive page
func (b *SiteBuilder) buildArchiveYearHTML(year archive.ArchiveYear, lang string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<div class=\"archive-year\">\n"))
	for _, month := range year.Months {
		sb.WriteString(fmt.Sprintf("<h2>%s</h2>\n", i18n.MonthName(lang, month.Month)))
		sb.WriteString("<ul class=\"post-list\">\n")
		for _, p := range month.Pages {
			dateStr := i18n.FormatDateShort(p.Date, lang)
			sb.WriteString(fmt.Sprintf("  <li><time>%s</time> <a href=\"%s\">%s</a></li>\n", dateStr, p.URL, p.Title))
		}
		sb.WriteString("</ul>\n")
	}
	sb.WriteString("</div>\n")
	return sb.String()
}
