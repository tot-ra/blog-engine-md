package builder

import (
	neturl "net/url"
	"regexp"
	"strings"
	"unicode"
)

func splitTextIntoChunks(text string, maxRunes int) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	if maxRunes <= 0 {
		return []string{trimmed}
	}

	parts := splitIntoSentences(trimmed)
	chunks := make([]string, 0, len(parts))
	var current strings.Builder
	currentLen := 0

	flush := func() {
		if currentLen == 0 {
			return
		}
		chunks = append(chunks, strings.TrimSpace(current.String()))
		current.Reset()
		currentLen = 0
	}

	for _, part := range parts {
		segment := strings.TrimSpace(part)
		if segment == "" {
			continue
		}
		segmentRunes := []rune(segment)
		if len(segmentRunes) > maxRunes {
			flush()
			for len(segmentRunes) > maxRunes {
				chunks = append(chunks, strings.TrimSpace(string(segmentRunes[:maxRunes])))
				segmentRunes = segmentRunes[maxRunes:]
			}
			if len(segmentRunes) > 0 {
				current.WriteString(string(segmentRunes))
				currentLen = len(segmentRunes)
			}
			continue
		}

		additional := len(segmentRunes)
		if currentLen > 0 {
			additional++ // space
		}
		if currentLen+additional > maxRunes {
			flush()
		}
		if currentLen > 0 {
			current.WriteByte(' ')
			currentLen++
		}
		current.WriteString(segment)
		currentLen += len(segmentRunes)
	}

	flush()
	return chunks
}

func splitIntoSentences(s string) []string {
	runes := []rune(s)
	parts := make([]string, 0, 64)
	start := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		j := i + 1
		for j < len(runes) && unicode.IsSpace(runes[j]) {
			j++
		}
		if j > i+1 {
			part := strings.TrimSpace(string(runes[start:j]))
			if part != "" {
				parts = append(parts, part)
			}
			start = j
			i = j - 1
		}
	}
	if start < len(runes) {
		last := strings.TrimSpace(string(runes[start:]))
		if last != "" {
			parts = append(parts, last)
		}
	}
	return parts
}

func toSpeechText(markdown string) string {
	s := strings.TrimSpace(markdown)
	if s == "" {
		return ""
	}

	// Skip markdown tables entirely and insert stronger pauses around headings.
	s = stripMarkdownTables(s)
	s = addHeadingPauses(s)

	// Remove fenced code blocks and inline code first to avoid spelling code aloud.
	s = regexp.MustCompile("(?s)```.*?```").ReplaceAllString(s, " ")
	s = regexp.MustCompile("`[^`]*`").ReplaceAllString(s, " ")

	// Images/links/wiki links to readable text labels.
	s = regexp.MustCompile(`!\[([^\]]*)\]\([^\)]*\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\[([^\]]+)\]\([^\)]*\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\[\[([^\]|]+)\|([^\]]+)\]\]`).ReplaceAllString(s, "$2")
	s = regexp.MustCompile(`\[\[([^\]]+)\]\]`).ReplaceAllString(s, "$1")
	s = replaceURLsWithDomainSpeech(s)

	// Remove common markdown syntax tokens.
	s = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s*`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?m)^\s*([-*+]|\d+\.)\s+`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "~~", "")
	s = strings.ReplaceAll(s, "<!--truncate-->", " ")
	s = stripEmojiLikeRunes(s)

	// Remove any remaining HTML tags and collapse whitespace.
	s = regexp.MustCompile(`(?s)<[^>]*>`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func addHeadingPauses(markdown string) string {
	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}

		heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if heading == "" {
			lines[i] = ""
			continue
		}
		// Extra sentence boundaries create audible pauses before and after headings.
		lines[i] = "\n\n" + heading + ".\n\n"
	}
	return strings.Join(lines, "\n")
}

func stripMarkdownTables(markdown string) string {
	lines := strings.Split(markdown, "\n")
	out := make([]string, 0, len(lines))

	for i := 0; i < len(lines); i++ {
		if i+1 < len(lines) && looksLikeTableHeader(lines[i]) && isMarkdownTableSeparatorLine(lines[i+1]) {
			// Skip header + separator.
			i += 2
			// Skip table body rows.
			for i < len(lines) {
				trimmed := strings.TrimSpace(lines[i])
				if trimmed == "" || !strings.Contains(trimmed, "|") {
					i--
					break
				}
				i++
			}
			continue
		}
		out = append(out, lines[i])
	}

	return strings.Join(out, "\n")
}

func looksLikeTableHeader(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	return strings.Contains(trimmed, "|")
}

func isMarkdownTableSeparatorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.Contains(trimmed, "-") || !strings.Contains(trimmed, "|") {
		return false
	}

	parts := strings.Split(trimmed, "|")
	validCols := 0
	sepCell := regexp.MustCompile(`^:?-{3,}:?$`)
	for _, p := range parts {
		cell := strings.TrimSpace(p)
		if cell == "" {
			continue
		}
		if !sepCell.MatchString(cell) {
			return false
		}
		validCols++
	}
	return validCols >= 2
}

func replaceURLsWithDomainSpeech(s string) string {
	urlLike := regexp.MustCompile(`(?i)\b(?:https?|ftp)://[^\s<>()]+|\bwww\.[^\s<>()]+`)
	return urlLike.ReplaceAllStringFunc(s, func(raw string) string {
		trimmed := strings.TrimRight(raw, ".,;:!?)]}\"'")
		if trimmed == "" {
			return " "
		}

		toParse := trimmed
		if strings.HasPrefix(strings.ToLower(toParse), "www.") {
			toParse = "https://" + toParse
		}

		u, err := neturl.Parse(toParse)
		if err != nil || strings.TrimSpace(u.Hostname()) == "" {
			return " link "
		}

		domain := strings.ToLower(strings.TrimSpace(u.Hostname()))
		return " link to " + domain + " "
	})
}

func stripEmojiLikeRunes(s string) string {
	var b strings.Builder
	for _, r := range s {
		if isEmojiLikeRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isEmojiLikeRune(r rune) bool {
	if r == '\u200d' || r == '\ufe0f' {
		return true
	}
	if (r >= 0x1F300 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) {
		return true
	}
	if unicode.Is(unicode.So, r) || unicode.Is(unicode.Sk, r) {
		return true
	}
	return false
}

func trimToRunes(s string, limit int) string {
	if limit <= 0 {
		return strings.TrimSpace(s)
	}
	r := []rune(strings.TrimSpace(s))
	if len(r) <= limit {
		return string(r)
	}
	return strings.TrimSpace(string(r[:limit]))
}
