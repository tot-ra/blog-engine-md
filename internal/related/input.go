package related

import (
	"crypto/sha256"
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	frontmatterRE = regexp.MustCompile(`(?s)^\s*---\s*\n.*?\n---\s*(?:\n|$)`)
	fencedCodeRE  = regexp.MustCompile("(?s)```[^\\n]*\\n?(.*?)```")
	inlineCodeRE  = regexp.MustCompile("`([^`]*)`")
	imageRE       = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	linkRE        = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	htmlTagRE     = regexp.MustCompile(`(?s)<[^>]*>`)
)

// PrepareInput must stay byte-compatible with the embed command's canonical input.
func PrepareInput(title, description string, tags []string, body string) string {
	body = strings.TrimPrefix(body, "\ufeff")
	body = frontmatterRE.ReplaceAllString(body, "")
	body = fencedCodeRE.ReplaceAllStringFunc(body, func(block string) string {
		matches := fencedCodeRE.FindStringSubmatch(block)
		if len(matches) < 2 {
			return ""
		}
		return truncateRunes(matches[1], 200)
	})
	body = imageRE.ReplaceAllString(body, "$1")
	body = linkRE.ReplaceAllString(body, "$1")
	body = inlineCodeRE.ReplaceAllString(body, "$1")
	body = htmlTagRE.ReplaceAllString(body, " ")
	body = html.UnescapeString(body)
	parts := []string{title, description}
	if len(tags) > 0 {
		parts = append(parts, strings.Join(tags, ", "))
	}
	parts = append(parts, body)
	return strings.Join(strings.Fields(strings.Join(parts, "\n")), " ")
}

func HashInput(text, model string, dims int) string {
	digest := sha256.Sum256([]byte(text + "\nmodel:" + model + "\ndimensions:" + fmt.Sprint(dims)))
	return fmt.Sprintf("sha256:%x", digest)
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
