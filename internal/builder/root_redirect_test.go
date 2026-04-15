package builder

import (
	"strings"
	"testing"

	"github.com/tot-ra/blog-engine/internal/config"
)

func TestRootRedirectHTMLUsesStaticFallbackWhenBrowserRedirectDisabled(t *testing.T) {
	cfg := &config.SiteConfig{
		I18n: config.I18nConfig{
			Default: "rus",
			Languages: []config.LanguageConfig{
				{Code: "rus", Label: "Русский"},
				{Code: "est", Label: "Eesti", Aliases: []string{"et", "et-ee"}},
			},
		},
	}

	html := rootRedirectHTML(cfg)

	if !strings.Contains(html, `http-equiv="refresh" content="0; url=/rus/"`) {
		t.Fatalf("expected fallback redirect to /rus/, got %q", html)
	}
	if strings.Contains(html, "navigator.languages") {
		t.Fatalf("expected no browser detection script when disabled")
	}
}

func TestRootRedirectHTMLUsesBrowserAliasesAndFallback(t *testing.T) {
	cfg := &config.SiteConfig{
		I18n: config.I18nConfig{
			Default: "rus",
			BrowserRedirect: config.BrowserRedirectConfig{
				Enabled: true,
			},
			Languages: []config.LanguageConfig{
				{Code: "rus", Label: "Русский"},
				{Code: "est", Label: "Eesti", Aliases: []string{"et", "et-ee"}},
			},
		},
	}

	html := rootRedirectHTML(cfg)

	for _, expected := range []string{
		`var fallback="/rus/"`,
		`"et": "/est/"`,
		`"et-ee": "/est/"`,
		`"rus": "/rus/"`,
		`navigator.languages`,
		`window.location.replace(withSuffix(fallback))`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected generated redirect html to contain %q, got %q", expected, html)
		}
	}
}

func TestBrowserLanguageAliasesIncludesCodesAndAliases(t *testing.T) {
	aliases := browserLanguageAliases([]config.LanguageConfig{
		{Code: "rus", Aliases: []string{"ru", "ru-RU"}},
		{Code: "est", Aliases: []string{"et", "et-EE"}},
	})

	if got := aliases["rus"]; got != "rus" {
		t.Fatalf("expected code alias for rus, got %q", got)
	}
	if got := aliases["ru"]; got != "rus" {
		t.Fatalf("expected ru to map to rus, got %q", got)
	}
	if got := aliases["et-ee"]; got != "est" {
		t.Fatalf("expected et-ee to map to est, got %q", got)
	}
}
