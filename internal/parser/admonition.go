package parser

import (
	"regexp"
	"strings"
)

// AdmonitionType represents the type of admonition block
type AdmonitionType string

const (
	AdmonitionNote    AdmonitionType = "note"
	AdmonitionTip     AdmonitionType = "tip"
	AdmonitionInfo    AdmonitionType = "info"
	AdmonitionWarning AdmonitionType = "warning"
	AdmonitionDanger  AdmonitionType = "danger"
)

var admonitionIcons = map[AdmonitionType]string{
	AdmonitionNote:    "ℹ️",
	AdmonitionTip:     "💡",
	AdmonitionInfo:    "ℹ️",
	AdmonitionWarning: "⚠️",
	AdmonitionDanger:  "🛑",
}

var admonitionDefaults = map[AdmonitionType]string{
	AdmonitionNote:    "Note",
	AdmonitionTip:     "Tip",
	AdmonitionInfo:    "Info",
	AdmonitionWarning: "Warning",
	AdmonitionDanger:  "Danger",
}

// admonitionOpenRegex matches :::type or :::type Title
var admonitionOpenRegex = regexp.MustCompile(`^:::(note|tip|info|warning|danger)\s*(.*)$`)

// TransformAdmonitions processes :::type blocks in markdown content
// and converts them to HTML admonition divs before markdown rendering.
func TransformAdmonitions(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	var inBlock bool
	var blockType AdmonitionType
	var blockTitle string
	var blockContent []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inBlock {
			if match := admonitionOpenRegex.FindStringSubmatch(trimmed); match != nil {
				inBlock = true
				blockType = AdmonitionType(match[1])
				blockTitle = strings.TrimSpace(match[2])
				if blockTitle == "" {
					blockTitle = admonitionDefaults[blockType]
				}
				blockContent = nil
				continue
			}
			result = append(result, line)
		} else {
			if trimmed == ":::" {
				// Close block — emit HTML
				icon := admonitionIcons[blockType]
				html := renderAdmonitionHTML(blockType, blockTitle, icon, strings.Join(blockContent, "\n"))
				result = append(result, html)
				inBlock = false
				continue
			}
			blockContent = append(blockContent, line)
		}
	}

	// If block was never closed, output raw content
	if inBlock {
		result = append(result, ":::"+string(blockType)+" "+blockTitle)
		result = append(result, blockContent...)
	}

	return strings.Join(result, "\n")
}

func renderAdmonitionHTML(adType AdmonitionType, title, icon, content string) string {
	// Trim empty leading/trailing lines
	content = strings.TrimSpace(content)

	var sb strings.Builder
	sb.WriteString(`<div class="admonition admonition-`)
	sb.WriteString(string(adType))
	sb.WriteString("\">\n")
	sb.WriteString(`  <div class="admonition-header">`)
	sb.WriteString("\n")
	sb.WriteString(`    <span class="admonition-icon">`)
	sb.WriteString(icon)
	sb.WriteString("</span>\n")
	sb.WriteString(`    <span class="admonition-title">`)
	sb.WriteString(title)
	sb.WriteString("</span>\n")
	sb.WriteString("  </div>\n")
	sb.WriteString(`  <div class="admonition-content">`)
	sb.WriteString("\n")

	// Wrap content lines in <p> tags (simple paragraph splitting)
	paragraphs := splitParagraphs(content)
	for _, p := range paragraphs {
		sb.WriteString("    <p>")
		sb.WriteString(strings.TrimSpace(p))
		sb.WriteString("</p>\n")
	}

	sb.WriteString("  </div>\n")
	sb.WriteString("</div>\n")
	return sb.String()
}

// splitParagraphs splits content by blank lines into paragraphs
func splitParagraphs(content string) []string {
	if content == "" {
		return nil
	}

	var paragraphs []string
	var current []string

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if len(current) > 0 {
				paragraphs = append(paragraphs, strings.Join(current, "\n"))
				current = nil
			}
		} else {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, strings.Join(current, "\n"))
	}

	return paragraphs
}
