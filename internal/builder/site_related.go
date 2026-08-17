package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tot-ra/blog-engine/internal/related"
	"github.com/tot-ra/blog-engine/internal/renderer"
)

func (b *SiteBuilder) prepareRelatedArticles() {
	b.relatedByPageID = make(map[string][]related.RelatedMatch)
	b.relatedEntries = nil
	if !b.config.Related.Enabled {
		return
	}

	sections := make(map[string]struct{}, len(b.config.Related.Sections))
	for _, section := range b.config.Related.Sections {
		sections[strings.ToLower(strings.Trim(section, "/"))] = struct{}{}
	}
	pageByPath := make(map[string]*Page)
	for _, page := range b.pages {
		relPath, ok := b.relatedPagePath(page, sections)
		if !ok {
			continue
		}
		_, translationPath := detectLanguageAndContentPath(relPath, b.config.I18n.Default, b.languages)
		entry := related.Entry{Path: relPath, URL: page.URL, Title: page.Title, Language: page.Language, TranslationPath: translationPath}
		if page.Frontmatter != nil {
			entry.Tags = page.Frontmatter.Tags
		}
		b.relatedEntries = append(b.relatedEntries, entry)
		pageByPath[relPath] = page
	}
	related.SortedEntries(b.relatedEntries)

	cache := &related.Cache{
		Version: 1,
		Model:   b.config.Related.Model,
		Dims:    b.config.Related.Dimensions,
		Entries: make(map[string]related.CacheEntry),
	}
	for _, entry := range b.relatedEntries {
		page := pageByPath[entry.Path]
		if page == nil || page.Frontmatter == nil || page.Frontmatter.Embedding == nil {
			continue
		}
		embedding := page.Frontmatter.Embedding
		if embedding.Version != cache.Version || embedding.Model != cache.Model || embedding.Dimensions != cache.Dims {
			continue
		}
		cache.Entries[entry.Path] = related.CacheEntry{
			Hash: embedding.Hash, Vec: embedding.Vector, Scale: embedding.Scale,
			Lang: entry.Language, URL: entry.URL,
		}
	}
	// Keep the central cache as a build artifact for the existing ranking pipeline.
	// It is regenerated from portable article frontmatter on every production build.
	if err := cache.Save(b.config.Related.CachePath); err != nil {
		fmt.Printf("Related articles: cannot write generated cache: %v\n", err)
	}

	vectorEntries := make([]related.Entry, 0, len(b.relatedEntries))
	stale := 0
	for _, entry := range b.relatedEntries {
		cached, ok := cache.Entries[entry.Path]
		if !ok {
			continue
		}
		vector, err := related.DecodeVector(cached, cache.Dims)
		if err != nil {
			continue
		}
		entry.Vector = vector
		if page := pageByPath[entry.Path]; page != nil && b.relatedPageHash(page) != cached.Hash {
			stale++
		}
		vectorEntries = append(vectorEntries, entry)
	}

	computed := related.ComputeRelated(vectorEntries, related.Config{
		Count: b.config.Related.Count, MinScore: b.config.Related.MinScore,
		Diversity: b.config.Related.Diversity, CrossLanguage: b.config.Related.CrossLanguage,
	})
	for path, matches := range computed {
		if page := pageByPath[path]; page != nil {
			b.relatedByPageID[page.ID] = matches
		}
	}
	fmt.Printf("Related articles: covered %d of %d articles\n", len(vectorEntries), len(b.relatedEntries))
	if stale > 0 {
		fmt.Printf("%d articles have stale embeddings, run `blog-engine embed`\n", stale)
	}
}

func (b *SiteBuilder) relatedPagePath(page *Page, sections map[string]struct{}) (string, bool) {
	if page == nil || strings.TrimSpace(page.SourcePath) == "" || (page.Frontmatter != nil && strings.TrimSpace(page.Frontmatter.RedirectURL) != "") {
		return "", false
	}
	rel, err := filepath.Rel(b.config.Build.ContentDir, page.SourcePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	_, contentPath := detectLanguageAndContentPath(rel, b.config.I18n.Default, b.languages)
	parts := strings.Split(strings.Trim(contentPath, "/"), "/")
	if len(parts) < 2 {
		return "", false
	}
	if _, ok := sections[strings.ToLower(parts[0])]; !ok {
		return "", false
	}
	name := strings.TrimSuffix(filepath.Base(contentPath), filepath.Ext(contentPath))
	if strings.EqualFold(name, "index") || strings.EqualFold(name, "README") || isSelfNamedSectionFile(filepath.Dir(contentPath), name, page.Frontmatter) {
		return "", false
	}
	return rel, true
}

func (b *SiteBuilder) relatedPageHash(page *Page) string {
	data, err := os.ReadFile(page.SourcePath)
	if err != nil {
		return ""
	}
	_, body, err := parseContentFrontmatter(ContentFile{ContentType: TypeMarkdown}, string(data))
	if err != nil {
		return ""
	}
	text := related.PrepareInput(page.Title, page.Description, page.Frontmatter.Tags, body)
	return related.HashInput(text, b.config.Related.Model, b.config.Related.Dimensions)
}

func (b *SiteBuilder) relatedForPage(page *Page) []related.RelatedMatch {
	if page == nil || page.Frontmatter == nil || page.Frontmatter.HideRelated {
		return nil
	}
	if page.Frontmatter.Related != nil {
		return related.ResolveManual(page.Frontmatter.Related, b.relatedEntries)
	}
	return b.relatedByPageID[page.ID]
}

func (b *SiteBuilder) relatedArticlesForPage(page *Page) []renderer.RelatedArticle {
	matches := b.relatedForPage(page)
	if len(matches) == 0 {
		return nil
	}

	articles := make([]renderer.RelatedArticle, 0, len(matches))
	for _, match := range matches {
		relatedPage := b.pagesByURL[match.URL]
		if relatedPage == nil {
			continue
		}

		excerpt := extractArticlePreviewTextWithLimits(relatedPage, 2, 220)
		if excerpt == "" {
			excerpt = strings.TrimSpace(relatedPage.Description)
		}
		article := renderer.RelatedArticle{
			Title:   relatedPage.Title,
			URL:     relatedPage.URL,
			Excerpt: excerpt,
			Score:   match.Score,
		}
		if relatedPage.Frontmatter != nil {
			article.Date = relatedPage.Frontmatter.Date
		}
		article.ImageURL = firstPreviewImageURL(relatedPage)
		articles = append(articles, article)
	}
	return articles
}
