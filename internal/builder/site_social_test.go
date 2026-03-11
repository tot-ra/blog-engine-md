package builder

import (
	"testing"

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
