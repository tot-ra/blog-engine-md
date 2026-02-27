package assets

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

func normalizeImageRelativePath(relativePath string) string {
	rel := filepath.ToSlash(relativePath)
	return strings.TrimPrefix(rel, "img/")
}

// ImageConfig holds image processing configuration
type ImageConfig struct {
	Quality int
	Sizes   map[string]int // name → max width
	Enabled bool
}

// DefaultImageConfig returns sensible defaults
func DefaultImageConfig() ImageConfig {
	return ImageConfig{
		Quality: 85,
		Sizes: map[string]int{
			"thumbnail": 150,
			"preview":   400,
			"full":      1200,
		},
		Enabled: true,
	}
}

// ImageVariant represents one processed size variant of an image
type ImageVariant struct {
	Size     string // "thumbnail", "preview", "full"
	Width    int
	Height   int
	FilePath string // output path relative to dist/
	FileSize int64
}

// ProcessedImage holds all variants of a processed image
type ProcessedImage struct {
	OriginalPath string
	RelativePath string
	Variants     []ImageVariant
	Width        int
	Height       int
}

// ImageProcessor handles image optimization and conversion
type ImageProcessor struct {
	config    ImageConfig
	outputDir string
	cache     *ImageCache
	mu        sync.Mutex
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(config ImageConfig, outputDir string, cache *ImageCache) *ImageProcessor {
	return &ImageProcessor{
		config:    config,
		outputDir: outputDir,
		cache:     cache,
	}
}

// ProcessFile processes a single image file, generating size variants as WebP.
// If WebP encoding is unavailable, it falls back to JPEG.
func (p *ImageProcessor) ProcessFile(srcPath, relativePath string, modTime int64, size int64) (*ProcessedImage, error) {
	normalizedRelPath := normalizeImageRelativePath(relativePath)
	ext := strings.ToLower(filepath.Ext(srcPath))

	// SVG pass-through: just copy
	if ext == ".svg" {
		return p.copySVG(srcPath, normalizedRelPath)
	}

	// Check cache
	if p.cache != nil {
		if entry, ok := p.cache.Get(normalizedRelPath); ok {
			if entry.SourceModTime == modTime && entry.SourceSize == size && p.variantsExist(entry.Variants) {
				return &ProcessedImage{
					OriginalPath: srcPath,
					RelativePath: normalizedRelPath,
					Variants:     entry.Variants,
				}, nil
			}
		}
	}

	// Decode image
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	srcImg, _, err := image.Decode(srcFile)
	if err != nil {
		// Fall back to passthrough copy for formats/variants not supported by the decoder.
		return p.copyOriginalImage(srcPath, normalizedRelPath)
	}

	bounds := srcImg.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	result := &ProcessedImage{
		OriginalPath: srcPath,
		RelativePath: normalizedRelPath,
		Width:        origWidth,
		Height:       origHeight,
	}

	// Generate each size variant
	baseName := strings.TrimSuffix(filepath.Base(normalizedRelPath), filepath.Ext(normalizedRelPath))
	relDir := filepath.Dir(normalizedRelPath)

	for sizeName, maxWidth := range p.config.Sizes {
		// Skip if original is smaller than target
		targetWidth := maxWidth
		if targetWidth > origWidth {
			targetWidth = origWidth
		}

		// Resize maintaining aspect ratio
		resized := imaging.Resize(srcImg, targetWidth, 0, imaging.Lanczos)
		resizedBounds := resized.Bounds()

		// Output path
		outRelPath := filepath.Join("assets", "img", relDir, fmt.Sprintf("%s-%s.webp", baseName, sizeName))
		outFullPath := filepath.Join(p.outputDir, outRelPath)

		// Ensure directory
		if err := os.MkdirAll(filepath.Dir(outFullPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", outFullPath, err)
		}

		// Encode as WebP with cwebp.
		if err := saveAsWebP(resized, outFullPath, p.config.Quality); err != nil {
			// If WebP encoding is unavailable, fall back to JPEG.
			jpegPath := filepath.Join("assets", "img", relDir, fmt.Sprintf("%s-%s.jpg", baseName, sizeName))
			jpegFullPath := filepath.Join(p.outputDir, jpegPath)
			if err2 := imaging.Save(resized, jpegFullPath, imaging.JPEGQuality(p.config.Quality)); err2 != nil {
				return nil, fmt.Errorf("failed to save image %s: %w", jpegFullPath, err2)
			}
			outRelPath = jpegPath
			outFullPath = jpegFullPath
		}

		// Get file size
		info, _ := os.Stat(outFullPath)
		var fileSize int64
		if info != nil {
			fileSize = info.Size()
		}

		variant := ImageVariant{
			Size:     sizeName,
			Width:    resizedBounds.Dx(),
			Height:   resizedBounds.Dy(),
			FilePath: "/" + outRelPath,
			FileSize: fileSize,
		}
		result.Variants = append(result.Variants, variant)
	}

	// Update cache
	if p.cache != nil {
		p.cache.Set(normalizedRelPath, &CacheEntry{
			SourceModTime: modTime,
			SourceSize:    size,
			Variants:      result.Variants,
		})
	}

	return result, nil
}

func saveAsWebP(img image.Image, outPath string, quality int) error {
	tmpInput := outPath + ".tmp.png"
	if err := imaging.Save(img, tmpInput); err != nil {
		return fmt.Errorf("failed to create temporary input for webp: %w", err)
	}
	defer os.Remove(tmpInput)

	cmd := exec.Command("cwebp", "-quiet", "-q", strconv.Itoa(quality), tmpInput, "-o", outPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to encode webp: %w", err)
	}
	return nil
}

func (p *ImageProcessor) variantsExist(variants []ImageVariant) bool {
	if len(variants) == 0 {
		return false
	}
	for _, v := range variants {
		rel := strings.TrimPrefix(v.FilePath, "/")
		fullPath := filepath.Join(p.outputDir, rel)
		if _, err := os.Stat(fullPath); err != nil {
			return false
		}
	}
	return true
}

// ProcessBatch processes multiple images concurrently
func (p *ImageProcessor) ProcessBatch(files []FileInfo, workers int) ([]*ProcessedImage, []error) {
	if workers <= 0 {
		workers = 4
	}

	type result struct {
		img *ProcessedImage
		err error
	}

	results := make([]result, len(files))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i, file := range files {
		wg.Add(1)
		go func(idx int, f FileInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			img, err := p.ProcessFile(f.Path, f.RelativePath, f.ModTime, f.Size)
			results[idx] = result{img: img, err: err}
		}(i, file)
	}

	wg.Wait()

	var images []*ProcessedImage
	var errors []error
	for _, r := range results {
		if r.err != nil {
			errors = append(errors, r.err)
		} else if r.img != nil {
			images = append(images, r.img)
		}
	}
	return images, errors
}

// FileInfo holds basic file info for batch processing
type FileInfo struct {
	Path         string
	RelativePath string
	ModTime      int64
	Size         int64
}

// copySVG copies an SVG file as-is to the output directory
func (p *ImageProcessor) copySVG(srcPath, relativePath string) (*ProcessedImage, error) {
	outRelPath := filepath.Join("assets", "img", relativePath)
	outFullPath := filepath.Join(p.outputDir, outRelPath)

	if err := os.MkdirAll(filepath.Dir(outFullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SVG: %w", err)
	}

	if err := os.WriteFile(outFullPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write SVG: %w", err)
	}

	return &ProcessedImage{
		OriginalPath: srcPath,
		RelativePath: relativePath,
		Variants: []ImageVariant{
			{Size: "original", FilePath: "/" + outRelPath, FileSize: int64(len(data))},
		},
	}, nil
}

// copyOriginalImage copies an image as-is when transformation is not possible.
func (p *ImageProcessor) copyOriginalImage(srcPath, relativePath string) (*ProcessedImage, error) {
	outRelPath := filepath.Join("assets", "img", relativePath)
	outFullPath := filepath.Join(p.outputDir, outRelPath)

	if err := os.MkdirAll(filepath.Dir(outFullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}
	if err := os.WriteFile(outFullPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write image: %w", err)
	}

	return &ProcessedImage{
		OriginalPath: srcPath,
		RelativePath: relativePath,
		Variants: []ImageVariant{
			{Size: "original", FilePath: "/" + outRelPath, FileSize: int64(len(data))},
		},
	}, nil
}
