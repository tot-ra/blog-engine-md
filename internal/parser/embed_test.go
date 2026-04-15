package parser

import (
	"strings"
	"testing"
)

func TestTransformEmbeds_YouTube(t *testing.T) {
	input := `Some text.

::youtube[dQw4w9WgXcQ]

More text.`

	result := TransformEmbeds(input)

	if !strings.Contains(result, `class="embed embed-youtube"`) {
		t.Error("expected youtube embed class")
	}
	if !strings.Contains(result, "youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Error("expected privacy-enhanced YouTube embed URL")
	}
	if !strings.Contains(result, `loading="lazy"`) {
		t.Error("expected lazy loading")
	}
	if !strings.Contains(result, "Some text.") {
		t.Error("expected surrounding text preserved")
	}
}

func TestTransformEmbeds_Vimeo(t *testing.T) {
	input := `::vimeo[123456789]`

	result := TransformEmbeds(input)

	if !strings.Contains(result, `class="embed embed-vimeo"`) {
		t.Error("expected vimeo embed class")
	}
	if !strings.Contains(result, "player.vimeo.com/video/123456789") {
		t.Error("expected vimeo embed URL")
	}
}

func TestTransformEmbeds_NoEmbeds(t *testing.T) {
	input := `Regular markdown content.

No embeds here.`

	result := TransformEmbeds(input)
	if result != input {
		t.Error("expected content unchanged when no embeds")
	}
}

func TestTransformEmbeds_MultipleEmbeds(t *testing.T) {
	input := `::youtube[abc123def45]

::vimeo[987654321]`

	result := TransformEmbeds(input)

	if !strings.Contains(result, "embed-youtube") {
		t.Error("expected youtube embed")
	}
	if !strings.Contains(result, "embed-vimeo") {
		t.Error("expected vimeo embed")
	}
}

func TestTransformEmbeds_StandaloneMarkdownYouTubeLink(t *testing.T) {
	input := `[Video](https://www.youtube.com/watch?v=dQw4w9WgXcQ)`

	result := TransformEmbeds(input)

	if !strings.Contains(result, `class="embed embed-youtube"`) {
		t.Error("expected youtube embed class from standalone markdown link")
	}
	if strings.Contains(result, `[Video](`) {
		t.Error("expected standalone markdown link to be replaced")
	}
}

func TestTransformEmbeds_StandaloneMarkdownVimeoLink(t *testing.T) {
	input := `[Video](https://vimeo.com/123456789)`

	result := TransformEmbeds(input)

	if !strings.Contains(result, `class="embed embed-vimeo"`) {
		t.Error("expected vimeo embed class from standalone markdown link")
	}
}

func TestTransformEmbeds_InlineMarkdownLinkStaysLink(t *testing.T) {
	input := `Intro [Video](https://www.youtube.com/watch?v=dQw4w9WgXcQ) outro.`

	result := TransformEmbeds(input)

	if result != input {
		t.Error("expected inline markdown link to stay unchanged")
	}
}
