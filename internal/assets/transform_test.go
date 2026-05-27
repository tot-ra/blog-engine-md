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

func TestImageTransformer_OriginalVariantWithoutDimensions(t *testing.T) {
	images := []*ProcessedImage{
		{
			RelativePath: "badge.webp",
			Variants: []ImageVariant{
				{Size: "original", Width: 0, Height: 0, FilePath: "/assets/img/badge.webp"},
			},
		},
	}

	transformer := NewImageTransformer(images)
	result := transformer.Transform(`<img src="img/badge.webp" alt="">`)

	if strings.Contains(result, `0w`) {
		t.Fatalf("zero-width srcset should not be emitted:\n%s", result)
	}
	if strings.Contains(result, `width="0"`) || strings.Contains(result, `height="0"`) {
		t.Fatalf("zero dimensions should not be emitted:\n%s", result)
	}
	if !strings.Contains(result, `<img src="/assets/img/badge.webp" alt="" loading="lazy">`) {
		t.Fatalf("expected original fallback image without dimensions:\n%s", result)
	}
}

func TestImageTransformer_NoTransformRewritesToProcessedAsset(t *testing.T) {
	images := []*ProcessedImage{
		{
			RelativePath: "download-badge.png",
			Variants: []ImageVariant{
				{Size: "thumbnail", Width: 300, Height: 103, FilePath: "/assets/img/download-badge-thumbnail.webp"},
				{Size: "full", Width: 383, Height: 132, FilePath: "/assets/img/download-badge-full.webp"},
			},
		},
	}

	transformer := NewImageTransformer(images)
	result := transformer.Transform(`<img class="download-badge no-transform" height="50" src="/assets/img/download-badge.png" alt="Download">`)

	if strings.Contains(result, `<picture>`) {
		t.Fatalf("no-transform image should stay a plain img tag:\n%s", result)
	}
	if !strings.Contains(result, `src="/assets/img/download-badge-full.webp"`) {
		t.Fatalf("expected no-transform src to be rewritten to processed asset:\n%s", result)
	}
	if !strings.Contains(result, `class="download-badge no-transform"`) || !strings.Contains(result, `height="50"`) {
		t.Fatalf("expected original attributes to be preserved:\n%s", result)
	}
}
