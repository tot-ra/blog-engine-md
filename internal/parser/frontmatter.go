package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Frontmatter represents YAML frontmatter from markdown files
type Frontmatter struct {
	Title       string    `yaml:"title"`
	Date        time.Time `yaml:"date"`
	Draft       bool      `yaml:"draft"`
	Tags        []string  `yaml:"tags"`
	Description string    `yaml:"description"`
	Slug        string    `yaml:"slug"`
	Order       int       `yaml:"order"`
	HideToc     bool      `yaml:"hideToc"`
	HideNav     bool      `yaml:"hideNav"`
	Layout      string    `yaml:"layout"` // Custom layout template name (e.g., "homepage")
}

// ParseFrontmatter extracts YAML frontmatter from markdown content
// Returns the parsed frontmatter, remaining content, and any error
func ParseFrontmatter(content string) (*Frontmatter, string, error) {
	fm := &Frontmatter{}

	// Check for frontmatter delimiters
	if !strings.HasPrefix(content, "---") {
		return fm, content, nil
	}

	// Find the end of frontmatter
	endIdx := strings.Index(content[3:], "---")
	if endIdx == -1 {
		return fm, content, nil
	}

	// Extract frontmatter content
	fmContent := content[3 : endIdx+3]
	remaining := strings.TrimSpace(content[endIdx+6:])

	type rawFrontmatter struct {
		Title       string      `yaml:"title"`
		Date        interface{} `yaml:"date"`
		Draft       bool        `yaml:"draft"`
		Tags        []string    `yaml:"tags"`
		Description string      `yaml:"description"`
		Slug        string      `yaml:"slug"`
		Order       int         `yaml:"order"`
		HideToc     bool        `yaml:"hideToc"`
		HideNav     bool        `yaml:"hideNav"`
		Layout      string      `yaml:"layout"`
	}

	var raw rawFrontmatter
	if err := yaml.Unmarshal([]byte(fmContent), &raw); err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}
	fm.Title = raw.Title
	fm.Draft = raw.Draft
	fm.Tags = raw.Tags
	fm.Description = raw.Description
	fm.Slug = raw.Slug
	fm.Order = raw.Order
	fm.HideToc = raw.HideToc
	fm.HideNav = raw.HideNav
	fm.Layout = raw.Layout

	parsedDate, err := parseFlexibleTime(raw.Date)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}
	fm.Date = parsedDate

	// Normalize tags
	for i, tag := range fm.Tags {
		fm.Tags[i] = strings.TrimSpace(strings.ToLower(tag))
	}

	return fm, remaining, nil
}

func parseFlexibleTime(v interface{}) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}
	if t, ok := v.(time.Time); ok {
		return t, nil
	}

	s, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("unsupported date type %T", v)
	}

	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, nil
}

// GenerateSlug creates a URL-friendly slug from text
func GenerateSlug(text string) string {
	if text == "" {
		return ""
	}

	// Transliterate Cyrillic to Latin
	text = transliterate(text)

	// Convert to lowercase
	text = strings.ToLower(text)

	// Replace spaces and underscores with hyphens
	text = regexp.MustCompile(`[\s_]+`).ReplaceAllString(text, "-")

	// Remove special characters except hyphens
	text = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(text, "")

	// Remove multiple consecutive hyphens
	text = regexp.MustCompile(`-+`).ReplaceAllString(text, "-")

	// Trim hyphens from ends
	text = strings.Trim(text, "-")

	return text
}

// transliterate converts Cyrillic characters to Latin
func transliterate(text string) string {
	cyrillicToLatin := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
		'е': "e", 'ё': "yo", 'ж': "zh", 'з': "z", 'и': "i",
		'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
		'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
		'у': "u", 'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch",
		'ш': "sh", 'щ': "sch", 'ъ': "", 'ы': "y", 'ь': "",
		'э': "e", 'ю': "yu", 'я': "ya",
		'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D",
		'Е': "E", 'Ё': "Yo", 'Ж': "Zh", 'З': "Z", 'И': "I",
		'Й': "Y", 'К': "K", 'Л': "L", 'М': "M", 'Н': "N",
		'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T",
		'У': "U", 'Ф': "F", 'Х': "H", 'Ц': "Ts", 'Ч': "Ch",
		'Ш': "Sh", 'Щ': "Sch", 'Ъ': "", 'Ы': "Y", 'Ь': "",
		'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	}

	var result strings.Builder
	for _, r := range text {
		if latin, ok := cyrillicToLatin[r]; ok {
			result.WriteString(latin)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
