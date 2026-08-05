package parser

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"
)

// slugIDs generates heading IDs with the same transliteration rules as TOC anchors.
// WHY: goldmark's default AutoHeadingID drops non-ASCII chars, so Cyrillic headings
// become "heading" / "---" while TOC links use GenerateSlug ("professiya").
type slugIDs struct {
	values map[string]bool
}

// NewSlugIDs returns a goldmark IDs implementation aligned with GenerateSlug.
func NewSlugIDs() gparser.IDs {
	return &slugIDs{values: map[string]bool{}}
}

func (s *slugIDs) Generate(value []byte, kind ast.NodeKind) []byte {
	base := GenerateSlug(string(value))
	if base == "" {
		if kind == ast.KindHeading {
			base = "heading"
		} else {
			base = "id"
		}
	}

	if !s.values[base] {
		s.values[base] = true
		return []byte(base)
	}

	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !s.values[candidate] {
			s.values[candidate] = true
			return []byte(candidate)
		}
	}
}

func (s *slugIDs) Put(value []byte) {
	s.values[strings.ToLower(string(value))] = true
}
