package assets

import (
	"strings"
	"testing"
)

func TestImageTransformer_BasicTransform(t *testing.T) {
	images := []*ProcessedImage{
		{
			RelativePath: "photo.jpg",
			Variants: []ImageVariant{
				{Size: "thumbnail", Width: 150, Height: 100, FilePath: "/assets/img/photo-thumbnail.webp"},
				{Size: "full", Width: 1200, Height: 800, FilePath: "/assets/img/photo-full.webp"},
			},
		},
	}

	transformer := NewImageTransformer(images)
	input := `<p>Some text</p><img src="photo.jpg" alt="A photo"><p>More text</p>`
	result := transformer.Transform(input)

	if !strings.Contains(result, "<picture>") {
		t.Error("Expected <picture> element in output")
	}
	if !strings.Contains(result, `loading="lazy"`) {
		t.Error("Expected lazy loading")
	}
	if !strings.Contains(result, "photo-full.webp") {
		t.Error("Expected full variant in output")
	}
	if !strings.Contains(result, `alt="A photo"`) {
		t.Error("Expected alt text preserved")
	}
}

func TestImageTransformer_NoMatch(t *testing.T) {
	transformer := NewImageTransformer(nil)
	input := `<img src="unknown.jpg" alt="test">`
	result := transformer.Transform(input)

	// Should add lazy loading even without a processed match
	if !strings.Contains(result, `loading="lazy"`) {
		t.Error("Expected lazy loading added to unmatched images")
	}
	if strings.Contains(result, "<picture>") {
		t.Error("Should not have <picture> for unmatched images")
	}
}

func TestImageTransformer_AlreadyHasLoading(t *testing.T) {
	transformer := NewImageTransformer(nil)
	input := `<img src="test.jpg" alt="test" loading="eager">`
	result := transformer.Transform(input)

	// Should not double-add loading attribute
	if strings.Count(result, "loading=") != 1 {
		t.Error("Should not add duplicate loading attribute")
	}
}
