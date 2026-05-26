package assets

import (
	"fmt"
	"net/url"
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
		// Keep explicitly opted-out images as plain <img> tags, but still rewrite
		// their src to a processed/copied asset when the original is not emitted to
		// dist (for example static icons/badges).
		if strings.Contains(match, "no-transform") {
			srcMatch := srcAttrRegex.FindStringSubmatch(match)
			if srcMatch == nil {
				return match
			}
			img := t.findImage(srcMatch[1])
			if img == nil {
				return match
			}
			variant := pickNoTransformVariant(img)
			if variant == nil || variant.FilePath == "" {
				return match
			}
			return srcAttrRegex.ReplaceAllString(match, `src="`+variant.FilePath+`"`)
		}

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
	candidates := []string{src}

	cleaned := strings.TrimPrefix(src, "./")
	if cleaned != src {
		candidates = append(candidates, cleaned)
	}

	decoded, err := url.PathUnescape(src)
	if err == nil && decoded != src {
		candidates = append(candidates, decoded)
		decodedCleaned := strings.TrimPrefix(decoded, "./")
		if decodedCleaned != decoded {
			candidates = append(candidates, decodedCleaned)
		}
	}

	for _, candidate := range candidates {
		if img, ok := t.processedImages[candidate]; ok {
			return img
		}
		base := filepath.Base(candidate)
		if img, ok := t.processedImages[base]; ok {
			return img
		}
	}

	return nil
}

func pickNoTransformVariant(img *ProcessedImage) *ImageVariant {
	if img == nil || len(img.Variants) == 0 {
		return nil
	}
	for i := range img.Variants {
		if img.Variants[i].Size == "full" || img.Variants[i].Size == "original" {
			return &img.Variants[i]
		}
	}
	return &img.Variants[0]
}

// buildPictureElement creates a responsive <picture> element
func (t *ImageTransformer) buildPictureElement(img *ProcessedImage, alt string) string {
	// Find the "full" variant for the main display
	var fullVariant *ImageVariant
	var srcsetParts []string
	allWebP := true

	for i := range img.Variants {
		v := &img.Variants[i]
		if v.Width > 0 {
			srcsetParts = append(srcsetParts, fmt.Sprintf("%s %dw", v.FilePath, v.Width))
		}
		if strings.ToLower(filepath.Ext(v.FilePath)) != ".webp" {
			allWebP = false
		}
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

	// Emit <source type="image/webp"> only when variants are actually webp files.
	if len(srcsetParts) > 0 && allWebP {
		sb.WriteString(fmt.Sprintf("    <source srcset=\"%s\" sizes=\"(max-width: 768px) 100vw, 800px\" type=\"image/webp\">\n",
			strings.Join(srcsetParts, ", ")))
	}

	// Fallback img
	dimensions := ""
	if fullVariant.Width > 0 && fullVariant.Height > 0 {
		dimensions = fmt.Sprintf(` width="%d" height="%d"`, fullVariant.Width, fullVariant.Height)
	}
	if len(srcsetParts) > 0 && !allWebP {
		sb.WriteString(fmt.Sprintf("    <img src=\"%s\" srcset=\"%s\" sizes=\"(max-width: 768px) 100vw, 800px\" alt=\"%s\" loading=\"lazy\"%s>\n",
			fullVariant.FilePath, strings.Join(srcsetParts, ", "), alt, dimensions))
	} else {
		sb.WriteString(fmt.Sprintf("    <img src=\"%s\" alt=\"%s\" loading=\"lazy\"%s>\n",
			fullVariant.FilePath, alt, dimensions))
	}
	sb.WriteString("  </picture>\n")
	sb.WriteString("</figure>")

	return sb.String()
}
