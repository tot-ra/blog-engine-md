package config

import "strings"

// LanguageDirection returns the configured writing direction for a language code.
// WHY: templates need a stable dir="rtl" signal for Arabic, Hebrew and future RTL locales.
// WHAT: explicit i18n.languages[].direction wins; known RTL language codes fall back to rtl.
func LanguageDirection(code string, languages []LanguageConfig) string {
	normalized := strings.ToLower(strings.TrimSpace(code))
	for _, lang := range languages {
		if strings.ToLower(strings.TrimSpace(lang.Code)) != normalized {
			continue
		}
		direction := strings.ToLower(strings.TrimSpace(lang.Direction))
		if direction == "rtl" {
			return "rtl"
		}
		if direction == "ltr" {
			return "ltr"
		}
		break
	}
	return defaultLanguageDirection(normalized)
}

func defaultLanguageDirection(code string) string {
	base := strings.Split(strings.ToLower(strings.TrimSpace(code)), "-")[0]
	switch base {
	case "ar", "arc", "dv", "fa", "ha", "he", "khw", "ks", "ku", "ps", "sd", "ur", "yi":
		return "rtl"
	default:
		return "ltr"
	}
}
