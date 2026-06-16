package builder

import (
	"strings"

	"github.com/tot-ra/blog-engine/internal/i18n"
)

// segmentLabel returns translated labels for engine-owned URL segments only.
// Website/domain-specific labels should come from localized content frontmatter
// (for example content/<lang>/docs/<section>/<section>.md), not from config.
func segmentLabel(lang, segment string) string {
	seg := strings.ToLower(strings.TrimSpace(segment))
	if seg == "" {
		return ""
	}
	return i18n.SegmentLabel(lang, seg)
}
