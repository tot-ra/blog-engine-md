package parser

import (
	"strings"
	"testing"
)

func TestTransformAdmonitions_Note(t *testing.T) {
	input := `Some text before.

:::note
This is a note.
:::

Some text after.`

	result := TransformAdmonitions(input)

	if !strings.Contains(result, `class="admonition admonition-note"`) {
		t.Error("expected admonition-note class")
	}
	if !strings.Contains(result, "ℹ️") {
		t.Error("expected note icon")
	}
	if !strings.Contains(result, `<span class="admonition-title">Note</span>`) {
		t.Error("expected default title 'Note'")
	}
	if !strings.Contains(result, "<p>This is a note.</p>") {
		t.Error("expected note content")
	}
	if !strings.Contains(result, "Some text before.") {
		t.Error("expected text before to be preserved")
	}
	if !strings.Contains(result, "Some text after.") {
		t.Error("expected text after to be preserved")
	}
}

func TestTransformAdmonitions_CustomTitle(t *testing.T) {
	input := `:::tip My Custom Title
This is a tip.
:::
`
	result := TransformAdmonitions(input)

	if !strings.Contains(result, `class="admonition admonition-tip"`) {
		t.Error("expected admonition-tip class")
	}
	if !strings.Contains(result, `<span class="admonition-title">My Custom Title</span>`) {
		t.Error("expected custom title")
	}
	if !strings.Contains(result, "💡") {
		t.Error("expected tip icon")
	}
}

func TestTransformAdmonitions_Warning(t *testing.T) {
	input := `:::warning
Be careful here!
:::
`
	result := TransformAdmonitions(input)

	if !strings.Contains(result, `admonition-warning`) {
		t.Error("expected warning type")
	}
	if !strings.Contains(result, "⚠️") {
		t.Error("expected warning icon")
	}
}

func TestTransformAdmonitions_Danger(t *testing.T) {
	input := `:::danger STOP!
Critical warning here.
:::
`
	result := TransformAdmonitions(input)

	if !strings.Contains(result, `admonition-danger`) {
		t.Error("expected danger type")
	}
	if !strings.Contains(result, "🛑") {
		t.Error("expected danger icon")
	}
	if !strings.Contains(result, `<span class="admonition-title">STOP!</span>`) {
		t.Error("expected custom title STOP!")
	}
}

func TestTransformAdmonitions_MultiParagraph(t *testing.T) {
	input := `:::note
First paragraph.

Second paragraph.
:::
`
	result := TransformAdmonitions(input)

	if strings.Count(result, "<p>") != 2 {
		t.Errorf("expected 2 paragraphs, got %d", strings.Count(result, "<p>"))
	}
}

func TestTransformAdmonitions_NoAdmonitions(t *testing.T) {
	input := `Just regular markdown content.

No admonitions here.`

	result := TransformAdmonitions(input)
	if result != input {
		t.Error("expected content to be unchanged when no admonitions present")
	}
}

func TestTransformAdmonitions_UnclosedBlock(t *testing.T) {
	input := `:::note
This block is never closed.`

	result := TransformAdmonitions(input)

	// Unclosed block should be output raw
	if !strings.Contains(result, ":::note") {
		t.Error("expected unclosed block to be output raw")
	}
}

func TestTransformAdmonitions_Multiple(t *testing.T) {
	input := `:::note
A note.
:::

:::warning
A warning.
:::
`
	result := TransformAdmonitions(input)

	if strings.Count(result, "admonition-note") != 1 {
		t.Error("expected exactly one note")
	}
	if strings.Count(result, "admonition-warning") != 1 {
		t.Error("expected exactly one warning")
	}
}
