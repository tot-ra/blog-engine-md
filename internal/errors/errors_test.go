package errors

import (
	"strings"
	"testing"
)

func TestBuildError_Error(t *testing.T) {
	err := &BuildError{
		Type:    ParseError,
		File:    "blog/post.md",
		Line:    15,
		Column:  23,
		Message: "invalid date format",
	}

	s := err.Error()
	if !strings.Contains(s, "PARSE") {
		t.Error("expected PARSE in error string")
	}
	if !strings.Contains(s, "blog/post.md:15:23") {
		t.Error("expected file:line:col in error string")
	}
}

func TestFormatError_WithContext(t *testing.T) {
	err := &BuildError{
		Type:       ParseError,
		File:       "blog/post.md",
		Line:       3,
		Message:    "invalid date format",
		Suggestion: "Use format YYYY-MM-DD",
		Context:    "---\ntitle: Test\ndate: 2025-13-45\n---",
	}

	output := FormatError(err)

	if !strings.Contains(output, "ERROR") {
		t.Error("expected ERROR header")
	}
	if !strings.Contains(output, "blog/post.md:3") {
		t.Error("expected file location")
	}
	if !strings.Contains(output, "💡") {
		t.Error("expected suggestion")
	}
}

func TestFormatErrors_Multiple(t *testing.T) {
	errs := []*BuildError{
		NewParseError("file1.md", 5, "bad frontmatter", ""),
		NewLinkError("file2.md", 10, "/missing/", "page not found"),
	}

	output := FormatErrors(errs)

	if !strings.Contains(output, "2 build error(s)") {
		t.Error("expected error count")
	}
}

func TestFormatErrors_Empty(t *testing.T) {
	output := FormatErrors(nil)
	if output != "" {
		t.Error("expected empty string for no errors")
	}
}

func TestNewConfigError(t *testing.T) {
	err := NewConfigError("missing site.url", "Add site.url to config.yaml")
	if err.Type != ConfigError {
		t.Error("expected ConfigError type")
	}
	if err.File != "config.yaml" {
		t.Error("expected config.yaml as file")
	}
}
