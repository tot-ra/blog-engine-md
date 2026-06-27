package builder

import (
	"html"
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/tot-ra/blog-engine/internal/assets"
)

func (b *SiteBuilder) resolveSocialImageURL(page *Page) string {
	if page != nil && page.Type == TypeBlog {
		if src := extractFirstImageSrc(page.Content); src != "" {
			if resolved := b.resolveAssetURL(src, page.URL); resolved != "" {
				if optimized := b.resolveProcessedSocialImageURL(src); optimized != "" {
					return optimized
				}
				return resolved
			}
		}
	}
	defaultSrc := strings.TrimSpace(b.config.SEO.DefaultImage)
	if optimized := b.resolveProcessedSocialImageURL(defaultSrc); optimized != "" {
		return optimized
	}
	return b.resolveAssetURL(defaultSrc, page.URL)
}

func (b *SiteBuilder) resolveAssetURL(src, pageURL string) string {
	src = html.UnescapeString(strings.TrimSpace(src))
	if src == "" {
		return ""
	}
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return src
	}

	baseURL := strings.TrimSpace(b.config.Site.URL)
	if strings.HasPrefix(src, "//") {
		siteURL, err := url.Parse(baseURL)
		if err != nil || siteURL.Scheme == "" {
			return "https:" + src
		}
		return siteURL.Scheme + ":" + src
	}

	if strings.HasPrefix(src, "/") {
		if baseURL == "" {
			return src
		}
		return strings.TrimSuffix(baseURL, "/") + src
	}

	pageAbs := b.absolutePageURL(pageURL)
	base, err := url.Parse(pageAbs)
	if err != nil {
		return src
	}
	ref, err := url.Parse(src)
	if err != nil {
		return src
	}
	return base.ResolveReference(ref).String()
}

func extractFirstImageSrc(htmlContent string) string {
	matches := firstImageSrcRe.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		return html.UnescapeString(strings.TrimSpace(matches[1]))
	}
	return ""
}

func (b *SiteBuilder) resolveProcessedSocialImageURL(src string) string {
	if len(b.processedImages) == 0 {
		return ""
	}
	candidates := processedImageLookupCandidates(src)
	if len(candidates) == 0 {
		return ""
	}

	for _, img := range b.processedImages {
		if img == nil {
			continue
		}
		rel := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(img.RelativePath)), "/")
		if rel == "" {
			continue
		}
		base := filepath.Base(rel)
		for _, candidate := range candidates {
			if candidate == rel || candidate == base {
				v := pickSocialImageVariant(img)
				if v == nil || v.FilePath == "" {
					continue
				}
				return b.resolveAssetURL(v.FilePath, "/")
			}
		}
	}
	return ""
}

func pickSocialImageVariant(img *assets.ProcessedImage) *assets.ImageVariant {
	if img == nil || len(img.Variants) == 0 {
		return nil
	}
	for i := range img.Variants {
		if img.Variants[i].Size == "preview" {
			return &img.Variants[i]
		}
	}
	for i := range img.Variants {
		if img.Variants[i].Size == "full" || img.Variants[i].Size == "original" {
			return &img.Variants[i]
		}
	}
	return &img.Variants[0]
}

func processedImageLookupCandidates(src string) []string {
	src = html.UnescapeString(strings.TrimSpace(src))
	if src == "" {
		return nil
	}

	seen := make(map[string]struct{})
	add := func(v string) {
		v = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(v)), "/")
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
	}
	addVariants := func(v string) {
		cleaned := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(v)), "/")
		add(cleaned)
		if strings.HasPrefix(cleaned, "assets/img/") {
			add(strings.TrimPrefix(cleaned, "assets/img/"))
		}
	}

	addVariants(src)
	if u, err := url.Parse(src); err == nil {
		addVariants(u.Path)
	}
	if decoded, err := url.PathUnescape(src); err == nil {
		addVariants(decoded)
	}

	out := make([]string, 0, len(seen)*2)
	for candidate := range seen {
		out = append(out, candidate)
		out = append(out, filepath.Base(candidate))
	}
	return out
}

func (b *SiteBuilder) metaDescriptionForPage(page *Page) string {
	if page == nil {
		return strings.TrimSpace(b.config.SEO.DefaultDesc)
	}
	if desc := strings.TrimSpace(page.Description); desc != "" {
		return desc
	}
	if excerpt := markdownExcerpt(page.RawContent, 200); excerpt != "" {
		return excerpt
	}
	return strings.TrimSpace(b.config.SEO.DefaultDesc)
}

func markdownExcerpt(raw string, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = 200
	}

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "![") || strings.HasPrefix(s, "<!--") {
			continue
		}

		s = excerptLinkRe.ReplaceAllString(s, "$1")
		s = strings.ReplaceAll(s, "`", "")
		s = strings.Join(strings.Fields(s), " ")
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if utf8.RuneCountInString(s) <= maxRunes {
			return s
		}
		runes := []rune(s)
		return strings.TrimSpace(string(runes[:maxRunes])) + "..."
	}

	return ""
}
