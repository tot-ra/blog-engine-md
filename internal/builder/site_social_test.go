package builder

import (
	"testing"

	"github.com/tot-ra/blog-engine/internal/assets"
	"github.com/tot-ra/blog-engine/internal/config"
)

func TestExtractFirstImageSrc(t *testing.T) {
	html := `<p>Intro</p><img alt="a" src="/assets/img/one.webp"><img src="/assets/img/two.webp">`
	got := extractFirstImageSrc(html)
	if got != "/assets/img/one.webp" {
		t.Fatalf("expected first image src, got %q", got)
	}
}

func TestResolveAssetURL_Relative(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Site: config.Site{
				URL: "https://kurapov.ee",
			},
		},
	}

	got := b.resolveAssetURL("img/cover.webp", "/ru/blog/post/")
	want := "https://kurapov.ee/ru/blog/post/img/cover.webp"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveSocialImageURL_UsesBlogFirstImageFallbackDefault(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Site: config.Site{
				URL: "https://kurapov.ee",
			},
			SEO: config.SEOConfig{
				DefaultImage: "/assets/img/fallback.png",
			},
		},
	}

	post := &Page{
		Type:    TypeBlog,
		URL:     "/ru/blog/post/",
		Content: `<p>x</p><img src="img/cover.webp">`,
	}
	if got := b.resolveSocialImageURL(post); got != "https://kurapov.ee/ru/blog/post/img/cover.webp" {
		t.Fatalf("expected blog first image, got %q", got)
	}

	doc := &Page{
		Type:    TypeDoc,
		URL:     "/ru/docs/about/",
		Content: `<img src="img/doc.webp">`,
	}
	if got := b.resolveSocialImageURL(doc); got != "https://kurapov.ee/assets/img/fallback.png" {
		t.Fatalf("expected fallback image, got %q", got)
	}
}

func TestResolveSocialImageURL_UsesOptimizedDefaultImageVariant(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Site: config.Site{
				URL: "https://kurapov.ee",
			},
			SEO: config.SEOConfig{
				DefaultImage: "/assets/img/preview-banner.png",
			},
		},
		processedImages: []*assets.ProcessedImage{
			{
				RelativePath: "preview-banner.png",
				Variants: []assets.ImageVariant{
					{Size: "thumbnail", FilePath: "/assets/img/preview-banner-thumbnail.webp"},
					{Size: "preview", FilePath: "/assets/img/preview-banner-preview.webp"},
					{Size: "full", FilePath: "/assets/img/preview-banner-full.webp"},
				},
			},
		},
	}

	page := &Page{Type: TypePage, URL: "/ru/"}
	got := b.resolveSocialImageURL(page)
	want := "https://kurapov.ee/assets/img/preview-banner-preview.webp"
	if got != want {
		t.Fatalf("expected optimized social image path %q, got %q", want, got)
	}
}

func TestResolveSocialImageURL_BlogRelativeParentPathUsesProcessedVariant(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Site: config.Site{
				URL: "https://kurapov.ee",
			},
			SEO: config.SEOConfig{
				DefaultImage: "/assets/img/preview-banner.png",
			},
		},
		processedImages: []*assets.ProcessedImage{
			{
				RelativePath: "ru/blog/img/plow-and-tractor.jpg",
				Variants: []assets.ImageVariant{
					{Size: "preview", FilePath: "/assets/img/ru/blog/img/plow-and-tractor-preview.webp"},
					{Size: "full", FilePath: "/assets/img/ru/blog/img/plow-and-tractor-full.webp"},
				},
			},
		},
	}

	post := &Page{
		Type:    TypeBlog,
		URL:     "/ru/blog/управление/pochemu-layv-koding-plohoy-sposob-proverki-znaniy/",
		Content: `<p>x</p><img src="../img/plow-and-tractor.jpg">`,
	}
	got := b.resolveSocialImageURL(post)
	want := "https://kurapov.ee/assets/img/ru/blog/img/plow-and-tractor-preview.webp"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestMetaDescriptionForPage_FallsBackToExcerpt(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			SEO: config.SEOConfig{
				DefaultDesc: "Site default description",
			},
		},
	}

	page := &Page{
		Description: "",
		RawContent: `
# Heading

![img](x.png)

This is the first meaningful line that should be used as meta description.
`,
	}

	got := b.metaDescriptionForPage(page)
	want := "This is the first meaningful line that should be used as meta description."
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
