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
	// The default language is served from root URLs (/...), not /<lang>/.
	if trimmedDefault != "" && strings.EqualFold(trimmedLang, trimmedDefault) {
		return "/"
	}

	trimmed := strings.Trim(pageURL, "/")
	if trimmed == "" {
		if trimmedLang == "" {
			return "/"
		}
		return "/" + trimmedLang + "/"
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return "/"
	}
	if trimmedLang != "" && strings.EqualFold(parts[0], trimmedLang) {
		return "/" + parts[0] + "/"
	}
	if trimmedLang == "" {
		return "/"
	}
	return "/" + trimmedLang + "/"
}
