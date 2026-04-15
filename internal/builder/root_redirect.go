package builder

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tot-ra/blog-engine/internal/config"
)

func rootRedirectHTML(cfg *config.SiteConfig) string {
	target := languageRootPath(cfg.I18n.Default)
	if !cfg.I18n.BrowserRedirect.Enabled {
		return staticRootRedirectHTML(target)
	}

	aliases := browserLanguageAliases(cfg.I18n.Languages)
	if len(aliases) == 0 {
		return staticRootRedirectHTML(target)
	}

	return browserAwareRootRedirectHTML(target, aliases)
}

func staticRootRedirectHTML(target string) string {
	return "<!doctype html><html><head><meta charset=\"utf-8\"><meta http-equiv=\"refresh\" content=\"0; url=" + target + "\"><link rel=\"canonical\" href=\"" + target + "\"></head><body><a href=\"" + target + "\">Redirecting...</a></body></html>"
}

func browserAwareRootRedirectHTML(fallback string, aliases map[string]string) string {
	lines := make([]string, 0, len(aliases))
	keys := make([]string, 0, len(aliases))
	for alias := range aliases {
		keys = append(keys, alias)
	}
	sort.Strings(keys)
	for _, alias := range keys {
		lines = append(lines, fmt.Sprintf("      %q: %q,", alias, languageRootPath(aliases[alias])))
	}

	return "<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><link rel=\"canonical\" href=\"" + fallback + "\"><script>(function(){var fallback=" + fmt.Sprintf("%q", fallback) + ";var aliasMap={\n" + strings.Join(lines, "\n") + "\n    };function normalize(value){return String(value||'').toLowerCase().replace(/_/g,'-').trim();}function withSuffix(target){return target+window.location.search+window.location.hash;}function targetFor(locale){var normalized=normalize(locale);if(!normalized){return'';}if(aliasMap[normalized]){return aliasMap[normalized];}var primary=normalized.split('-')[0];if(aliasMap[primary]){return aliasMap[primary];}return'';}var locales=[];if(Array.isArray(navigator.languages)){locales=locales.concat(navigator.languages);}if(navigator.language){locales.push(navigator.language);}for(var i=0;i<locales.length;i++){var target=targetFor(locales[i]);if(target){window.location.replace(withSuffix(target));return;}}window.location.replace(withSuffix(fallback));})();</script><noscript><meta http-equiv=\"refresh\" content=\"0; url=" + fallback + "\"></noscript></head><body><a href=\"" + fallback + "\">Redirecting...</a></body></html>"
}

func browserLanguageAliases(languages []config.LanguageConfig) map[string]string {
	out := make(map[string]string)
	for _, lang := range languages {
		code := strings.ToLower(strings.TrimSpace(lang.Code))
		if code == "" {
			continue
		}
		out[code] = code
		for _, alias := range lang.Aliases {
			normalized := strings.ToLower(strings.TrimSpace(alias))
			if normalized == "" {
				continue
			}
			out[normalized] = code
		}
	}
	return out
}

func languageRootPath(code string) string {
	trimmed := strings.Trim(strings.ToLower(code), "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed + "/"
}
