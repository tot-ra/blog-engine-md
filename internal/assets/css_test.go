package assets

import (
	"strings"
	"testing"
)

func TestCSSProcessor_Minify(t *testing.T) {
	proc := NewCSSProcessor(true)
	result, err := proc.MinifyString(`
		body {
			color: red;
			margin: 0;
		}
		/* comment */
		h1 {
			font-size: 2em;
		}
	`)
	if err != nil {
		t.Fatalf("Minify failed: %v", err)
	}
	// Should not contain comments or extra whitespace
	if strings.Contains(result, "/* comment */") {
		t.Error("Expected comments to be stripped")
	}
	if strings.Contains(result, "\n\t\t") {
		t.Error("Expected whitespace to be minimized")
	}
	if !strings.Contains(result, "color:red") || !strings.Contains(result, "font-size:2em") {
		t.Errorf("Expected minified output to contain CSS rules, got: %s", result)
	}
}

func TestCSSProcessor_NoMinify(t *testing.T) {
	proc := NewCSSProcessor(false)
	input := "body { color: red; }"
	result, err := proc.MinifyString(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("Expected unchanged output when minify disabled, got: %s", result)
	}
}
