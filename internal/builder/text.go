package builder

import (
	"unicode"
	"unicode/utf8"
)

func capitalizeFirst(text string) string {
	if text == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(text)
	if r == utf8.RuneError && size == 1 {
		return text
	}
	return string(unicode.ToUpper(r)) + text[size:]
}

