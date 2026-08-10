package builder

import (
	"strings"

	"github.com/tot-ra/blog-engine/internal/config"
)

func buildLanguageSet(cfg *config.SiteConfig) map[string]struct{} {
	out := make(map[string]struct{}, len(cfg.I18n.Languages))
	for _, lang := range cfg.I18n.Languages {
		code := strings.ToLower(strings.TrimSpace(lang.Code))
		if code == "" {
			continue
		}
		out[code] = struct{}{}
	}
	return out
}

func detectLanguageAndContentPath(relPath, defaultLang string, languages map[string]struct{}) (string, string) {
	clean := strings.Trim(strings.ReplaceAll(relPath, "\\", "/"), "/")
	if clean == "" {
		return defaultLang, clean
	}

	parts := strings.Split(clean, "/")
	first := strings.ToLower(parts[0])
	if _, ok := languages[first]; ok && len(parts) > 1 {
		return first, strings.Join(parts[1:], "/")
	}
	if _, ok := languages[first]; ok {
		return first, ""
	}
	return defaultLang, clean
}

func languageBasePath(pageURL, lang, defaultLang string) string {
	trimmedLang := strings.Trim(strings.TrimSpace(lang), "/")
	trimmedDefault := strings.Trim(strings.TrimSpace(defaultLang), "/")

	trimmed := strings.Trim(pageURL, "/")
	if trimmed != "" {
		parts := strings.Split(trimmed, "/")
		// Preserve an explicit /<lang>/ prefix even when lang is the site default.
		// Sites like dina.kurapov.ee keep default-language content under /rus/, and
		// breadcrumbs must stay in that shape or home links fall back to "/" and
		// trigger the browser language redirect to the front page.
		if len(parts) > 0 && trimmedLang != "" && strings.EqualFold(parts[0], trimmedLang) {
			return "/" + parts[0] + "/"
		}
	}

	// Default-language content without an explicit language prefix stays at root.
	if trimmedDefault != "" && strings.EqualFold(trimmedLang, trimmedDefault) {
		return "/"
	}

	if trimmedLang == "" {
		return "/"
	}
	return "/" + trimmedLang + "/"
}
