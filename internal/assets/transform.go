package assets

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ImageTransformer replaces <img> tags in rendered HTML with responsive <picture> elements
type ImageTransformer struct {
	processedImages map[string]*ProcessedImage // keyed by original relative path
}

// NewImageTransformer creates a new image transformer
func NewImageTransformer(images []*ProcessedImage) *ImageTransformer {
	m := make(map[string]*ProcessedImage)
	for _, img := range images {
		m[img.RelativePath] = img
		// Also index by filename for simpler lookups
		m[filepath.Base(img.RelativePath)] = img
	}
	return &ImageTransformer{processedImages: m}
}

// imgTagRegex matches <img> tags in HTML
var imgTagRegex = regexp.MustCompile(`<img\s+([^>]*?)>`)
var srcAttrRegex = regexp.MustCompile(`src="([^"]*)"`)
var altAttrRegex = regexp.MustCompile(`alt="([^"]*)"`)

// Transform replaces <img> tags with responsive <picture> elements
// where processed images are available
func (t *ImageTransformer) Transform(html string) string {
	return imgTagRegex.ReplaceAllStringFunc(html, func(match string) string {
		// Extract src
		srcMatch := srcAttrRegex.FindStringSubmatch(match)
		if srcMatch == nil {
			return match
		}
		src := srcMatch[1]

		// Extract alt
		alt := ""
		altMatch := altAttrRegex.FindStringSubmatch(match)
		if altMatch != nil {
			alt = altMatch[1]
		}

		// Look up processed image
		img := t.findImage(src)
		if img == nil {
			// No processed version, just add lazy loading
			if !strings.Contains(match, "loading=") {
				return strings.Replace(match, "<img ", "<img loading=\"lazy\" ", 1)
			}
			return match
		}

		return t.buildPictureElement(img, alt)
	})
}

// findImage looks up a processed image by various path forms
func (t *ImageTransformer) findImage(src string) *ProcessedImage {
	// Try exact match
	if img, ok := t.processedImages[src]; ok {
		return img
	}
	// Try without leading ./
	cleaned := strings.TrimPrefix(src, "./")
	if img, ok := t.processedImages[cleaned]; ok {
		return img
	}
	// Try just filename
	base := filepath.Base(src)
	if img, ok := t.processedImages[base]; ok {
		return img
	}
	return nil
}

// buildPictureElement creates a responsive <picture> element
func (t *ImageTransformer) buildPictureElement(img *ProcessedImage, alt string) string {
	// Find the "full" variant for the main display
	var fullVariant *ImageVariant
	var srcsetParts []string

	for i := range img.Variants {
		v := &img.Variants[i]
		srcsetParts = append(srcsetParts, fmt.Sprintf("%s %dw", v.FilePath, v.Width))
		if v.Size == "full" || (fullVariant == nil && v.Size == "original") {
			fullVariant = v
		}
	}

	if fullVariant == nil && len(img.Variants) > 0 {
		fullVariant = &img.Variants[0]
	}
	if fullVariant == nil {
		return fmt.Sprintf(`<img src="" alt="%s" loading="lazy">`, alt)
	}

	var sb strings.Builder
	sb.WriteString("<figure class=\"md-image\">\n")
	sb.WriteString("  <picture>\n")

	// WebP sources with srcset
	if len(srcsetParts) > 0 {
		sb.WriteString(fmt.Sprintf("    <source srcset=\"%s\" sizes=\"(max-width: 768px) 100vw, 800px\" type=\"image/webp\">\n",
			strings.Join(srcsetParts, ", ")))
	}

	// Fallback img
	sb.WriteString(fmt.Sprintf("    <img src=\"%s\" alt=\"%s\" loading=\"lazy\" width=\"%d\" height=\"%d\">\n",
		fullVariant.FilePath, alt, fullVariant.Width, fullVariant.Height))
	sb.WriteString("  </picture>\n")
	sb.WriteString("</figure>")

	return sb.String()
}
