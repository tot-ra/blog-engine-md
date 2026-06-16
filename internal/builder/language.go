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
	if trimmed == "" {
		if trimmedLang == "" || strings.EqualFold(trimmedLang, trimmedDefault) {
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
	if trimmedLang == "" || strings.EqualFold(trimmedLang, trimmedDefault) {
		return "/"
	}
	return "/" + trimmedLang + "/"
}

func withLanguagePrefix(lang, defaultLang, path string) string {
	_ = defaultLang
	p := "/" + strings.Trim(path, "/")
	if p == "/" {
		return p
	}
	if lang == "" {
		return p + "/"
	}
	return "/" + strings.Trim(lang, "/") + p + "/"
}
