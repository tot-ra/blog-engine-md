package parser

import (
	"strings"
	"testing"
)

func TestTransformMermaid_Basic(t *testing.T) {
	input := "Some text.\n\n```mermaid\ngraph TD\n    A[Start] --> B[End]\n```\n\nMore text."

	result, hasMermaid := TransformMermaid(input)

	if !hasMermaid {
		t.Error("expected hasMermaid to be true")
	}
	if !strings.Contains(result, `<pre class="mermaid">`) {
		t.Error("expected mermaid pre tag")
	}
	if !strings.Contains(result, "graph TD") {
		t.Error("expected diagram content")
	}
	if !strings.Contains(result, "Some text.") {
		t.Error("expected surrounding text preserved")
	}
	// Should NOT contain the ```mermaid fence
	if strings.Contains(result, "```mermaid") {
		t.Error("mermaid fence should be removed")
	}
}

func TestTransformMermaid_NoMermaid(t *testing.T) {
	input := "Just regular text.\n\n```go\nfmt.Println(\"hello\")\n```"

	result, hasMermaid := TransformMermaid(input)

	if hasMermaid {
		t.Error("expected hasMermaid to be false")
	}
	if result != input {
		t.Error("expected content unchanged")
	}
}

func TestTransformMermaid_Multiple(t *testing.T) {
	input := "```mermaid\ngraph LR\n    A --> B\n```\n\n```mermaid\nsequenceDiagram\n    A->>B: Hello\n```"

	result, hasMermaid := TransformMermaid(input)

	if !hasMermaid {
		t.Error("expected hasMermaid to be true")
	}
	if strings.Count(result, `<pre class="mermaid">`) != 2 {
		t.Errorf("expected 2 mermaid blocks, got %d", strings.Count(result, `<pre class="mermaid">`))
	}
}

func TestTransformMermaid_Unclosed(t *testing.T) {
	input := "```mermaid\ngraph TD\n    A --> B"

	result, _ := TransformMermaid(input)

	// Unclosed should be output raw
	if !strings.Contains(result, "```mermaid") {
		t.Error("expected unclosed block to be output raw")
	}
}
