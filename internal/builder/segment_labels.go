package builder

import (
	"strings"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/i18n"
)

// segmentLabel returns the display label for a URL segment.
// Generic UI segments (blog/docs/tags/etc.) live in i18n, while site/domain
// specific labels must come from config so the shared engine stays reusable.
func segmentLabel(lang, segment string, labels map[string]map[string]string) string {
	seg := strings.ToLower(strings.TrimSpace(segment))
	if seg == "" {
		return ""
	}

	if label := configuredSegmentLabel(lang, seg, labels); label != "" {
		return label
	}
	return i18n.SegmentLabel(lang, seg)
}

func configuredSegmentLabel(lang, segment string, labels map[string]map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	lang = strings.ToLower(strings.TrimSpace(lang))
	segment = strings.ToLower(strings.TrimSpace(segment))
	if lang == "" || segment == "" {
		return ""
	}

	// Prefer exact per-language labels, then fall back to "default" / "*".
	for _, langKey := range []string{lang, "default", "*"} {
		if label := labelForSegment(labels, langKey, segment); label != "" {
			return label
		}
	}
	return ""
}

func labelForSegment(labels map[string]map[string]string, lang, segment string) string {
	if len(labels) == 0 {
		return ""
	}
	entries := labels[lang]
	if entries == nil {
		entries = labels[strings.ToLower(lang)]
	}
	if len(entries) == 0 {
		return ""
	}
	for key, value := range entries {
		// Config keys are normalized during lookup to keep YAML ergonomic.
		if strings.ToLower(strings.TrimSpace(key)) == segment {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func segmentLabelsFromConfig(cfg *config.SiteConfig) map[string]map[string]string {
	if cfg == nil {
		return nil
	}
	return cfg.Navigation.SegmentLabels
}
