package assets

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

func normalizeImageRelativePath(relativePath string) string {
	rel := filepath.ToSlash(relativePath)
	return strings.TrimPrefix(rel, "img/")
}

// ImageConfig holds image processing configuration
type ImageConfig struct {
	Quality          int
	Sizes            map[string]int // name → max width
	Enabled          bool
	MaxSourcePixels  int64
	MaxVariantPixels int64
}

// BatchOptions holds concurrency and logging controls for image batches.
type BatchOptions struct {
	Workers    int
	Logf       func(format string, args ...any)
	LogEvery   int
	ProgressID string
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
		Enabled:          true,
		MaxSourcePixels:  0,
		MaxVariantPixels: 0,
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

	// Check cache before doing expensive decode/resize/encode work. The cache
	// stores generated files outside dist, so clean builds can restore variants.
	if p.cache != nil {
		if entry, ok := p.cache.Get(normalizedRelPath); ok {
			if entry.SourceModTime == modTime && entry.SourceSize == size && p.restoreCachedVariants(entry.Variants) {
				return &ProcessedImage{
					OriginalPath: srcPath,
					RelativePath: normalizedRelPath,
					Variants:     entry.Variants,
				}, nil
			}
		}
	}

	ext := strings.ToLower(filepath.Ext(srcPath))

	// SVG pass-through: just copy
	if ext == ".svg" {
		return p.copySVG(srcPath, normalizedRelPath, modTime, size)
	}

	// Decode image config first so source pixel guardrails can reject oversized
	// images before allocating the full decoded pixel buffer.
	cfg, _, err := image.DecodeConfig(srcFile)
	if err != nil {
		if _, seekErr := srcFile.Seek(0, 0); seekErr != nil {
			return nil, fmt.Errorf("failed to rewind image %s after decode config error: %w", srcPath, seekErr)
		}
		srcImg, _, err := image.Decode(srcFile)
		if err != nil {
			// Fall back to passthrough copy for formats/variants not supported by the decoder.
			return p.copyOriginalImage(srcPath, normalizedRelPath, modTime, size)
		}
		return p.processDecodedImage(srcImg, srcPath, normalizedRelPath, modTime, size)
	}
	if err := p.validateSourceImage(cfg.Width, cfg.Height, normalizedRelPath); err != nil {
		return nil, err
	}
	if _, err := srcFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to rewind image %s: %w", srcPath, err)
	}

	srcImg, _, err := image.Decode(srcFile)
	if err != nil {
		// Fall back to passthrough copy for formats/variants not supported by the decoder.
		return p.copyOriginalImage(srcPath, normalizedRelPath, modTime, size)
	}
	return p.processDecodedImage(srcImg, srcPath, normalizedRelPath, modTime, size)
}

func (p *ImageProcessor) processDecodedImage(srcImg image.Image, srcPath, normalizedRelPath string, modTime int64, size int64) (*ProcessedImage, error) {
		OriginalPath: srcPath,
		RelativePath: normalizedRelPath,
		Width:        origWidth,
		Height:       origHeight,
	}

	// Generate each size variant in a stable order so output and test expectations
	// stay deterministic regardless of Go map iteration order.
	baseName := strings.TrimSuffix(filepath.Base(normalizedRelPath), filepath.Ext(normalizedRelPath))
	relDir := filepath.Dir(normalizedRelPath)
	for _, variantSpec := range sortedVariantSpecs(p.config.Sizes) {
		targetWidth := variantSpec.Width
		if targetWidth > origWidth {
			targetWidth = origWidth
		}
		if targetWidth <= 0 {
			continue
		}

		targetHeight := scaledHeight(origWidth, origHeight, targetWidth)
		if err := p.validateVariantImage(targetWidth, targetHeight, normalizedRelPath, variantSpec.Name); err != nil {
			return nil, err
		}

		// Resize maintaining aspect ratio. Each worker only keeps one decoded source
		// image and one resized variant live at a time to cap peak memory usage.
		resized := imaging.Resize(srcImg, targetWidth, 0, imaging.Lanczos)
		resizedBounds := resized.Bounds()

		// Output path
		outRelPath := filepath.Join("assets", "img", relDir, fmt.Sprintf("%s-%s.webp", baseName, variantSpec.Name))
		outFullPath := filepath.Join(p.outputDir, outRelPath)

		// Ensure directory
		if err := os.MkdirAll(filepath.Dir(outFullPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", outFullPath, err)
		}

		// Encode as WebP with cwebp.
		if err := saveAsWebP(resized, outFullPath, p.config.Quality); err != nil {
			// If WebP encoding is unavailable, fall back to JPEG.
			jpegPath := filepath.Join("assets", "img", relDir, fmt.Sprintf("%s-%s.jpg", baseName, variantSpec.Name))
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
			Size:     variantSpec.Name,
			Width:    resizedBounds.Dx(),
			Height:   resizedBounds.Dy(),
			FilePath: "/" + outRelPath,
			FileSize: fileSize,
		}
		result.Variants = append(result.Variants, variant)
	}

	if err := p.cacheProcessedImage(normalizedRelPath, modTime, size, result); err != nil {
		return nil, err
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

func (p *ImageProcessor) restoreCachedVariants(variants []ImageVariant) bool {
	if p.cache == nil || len(variants) == 0 {
		return false
	}
	for _, v := range variants {
		cachedPath, err := p.cache.CachedFilePath(v.FilePath)
		if err != nil {
			return false
		}
		if _, err := os.Stat(cachedPath); err != nil {
			return false
		}
	}
	for _, v := range variants {
		if err := p.cache.RestoreVariantFile(v.FilePath, p.outputDir); err != nil {
			return false
		}
	}
	return true
}

func (p *ImageProcessor) cacheProcessedImage(relativePath string, modTime int64, size int64, result *ProcessedImage) error {
	if p.cache == nil || result == nil {
		return nil
	}
	for _, variant := range result.Variants {
		if err := p.cache.StoreVariantFile(variant.FilePath, p.outputDir); err != nil {
			return fmt.Errorf("failed to store cached image variant %s: %w", variant.FilePath, err)
		}
	}
	p.cache.Set(relativePath, &CacheEntry{
		SourceModTime: modTime,
		SourceSize:    size,
		Variants:      result.Variants,
	})
	return nil
}

// ProcessBatch processes multiple images concurrently with a bounded worker pool.
func (p *ImageProcessor) ProcessBatch(files []FileInfo, opts BatchOptions) ([]*ProcessedImage, []error) {
	if len(files) == 0 {
		return nil, nil
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = 4
	}
	if workers > len(files) {
		workers = len(files)
	}

	type job struct {
		index int
		file  FileInfo
	}
	type result struct {
		index int
		img   *ProcessedImage
		err   error
	}

	results := make([]result, len(files))
	jobs := make(chan job, workers)
	completed := atomic.Int64{}
	logEvery := opts.LogEvery
	if logEvery <= 0 {
		logEvery = 25
	}

	worker := func(resultCh chan<- result) {
		for job := range jobs {
			img, err := p.ProcessFile(job.file.Path, job.file.RelativePath, job.file.ModTime, job.file.Size)
			resultCh <- result{index: job.index, img: img, err: err}

			if opts.Logf != nil {
				done := int(completed.Add(1))
				if done == len(files) || done == 1 || done%logEvery == 0 {
					label := opts.ProgressID
					if label == "" {
						label = "image"
					}
					opts.Logf("Processed %s %d/%d", label, done, len(files))
				}
			}
		}
	}

	resultCh := make(chan result, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(resultCh)
		}()
	}

	go func() {
		for i, file := range files {
			jobs <- job{index: i, file: file}
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		results[res.index] = res
	}

	images := make([]*ProcessedImage, 0, len(files))
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

type imageVariantSpec struct {
	Name  string
	Width int
}

func sortedVariantSpecs(sizes map[string]int) []imageVariantSpec {
	variants := make([]imageVariantSpec, 0, len(sizes))
	for name, width := range sizes {
		variants = append(variants, imageVariantSpec{Name: name, Width: width})
	}
	sort.Slice(variants, func(i, j int) bool {
		if variants[i].Width == variants[j].Width {
			return variants[i].Name < variants[j].Name
		}
		return variants[i].Width < variants[j].Width
	})
	return variants
}

func scaledHeight(origWidth, origHeight, targetWidth int) int {
	if origWidth <= 0 || origHeight <= 0 || targetWidth <= 0 {
		return 0
	}
	targetHeight := int(float64(origHeight) * (float64(targetWidth) / float64(origWidth)))
	if targetHeight < 1 {
		return 1
	}
	return targetHeight
}

func pixelCount(width, height int) int64 {
	if width <= 0 || height <= 0 {
		return 0
	}
	return int64(width) * int64(height)
}

func (p *ImageProcessor) validateSourceImage(width, height int, relativePath string) error {
	if p.config.MaxSourcePixels <= 0 {
		return nil
	}
	pixels := pixelCount(width, height)
	if pixels <= p.config.MaxSourcePixels {
		return nil

func copyFile(srcPath, dstPath string) (int64, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return 0, err
	}
	if err := dst.Close(); err != nil {
		return 0, err
	}
	return n, nil
}
	}
	return fmt.Errorf("image %s exceeds maxSourcePixels: %d > %d", relativePath, pixels, p.config.MaxSourcePixels)
}

func (p *ImageProcessor) validateVariantImage(width, height int, relativePath, sizeName string) error {
	if p.config.MaxVariantPixels <= 0 {
		return nil
	}
	pixels := pixelCount(width, height)
	if pixels <= p.config.MaxVariantPixels {
		return nil
	}
	return fmt.Errorf("image %s variant %s exceeds maxVariantPixels: %d > %d", relativePath, sizeName, pixels, p.config.MaxVariantPixels)
}

// copySVG copies an SVG file as-is to the output directory
func (p *ImageProcessor) copySVG(srcPath, relativePath string, modTime int64, size int64) (*ProcessedImage, error) {
	outRelPath := filepath.Join("assets", "img", relativePath)
	outFullPath := filepath.Join(p.outputDir, outRelPath)

	if err := os.MkdirAll(filepath.Dir(outFullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	fileSize, err := copyFile(srcPath, outFullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy SVG: %w", err)
	}

	result := &ProcessedImage{
		OriginalPath: srcPath,
		RelativePath: relativePath,
		Variants: []ImageVariant{
			{Size: "original", FilePath: "/" + outRelPath, FileSize: fileSize},
		},
	}
	if err := p.cacheProcessedImage(relativePath, modTime, size, result); err != nil {
		return nil, err
	}
	return result, nil
}

// copyOriginalImage copies an image as-is when transformation is not possible.
func (p *ImageProcessor) copyOriginalImage(srcPath, relativePath string, modTime int64, size int64) (*ProcessedImage, error) {
	outRelPath := filepath.Join("assets", "img", relativePath)
	outFullPath := filepath.Join(p.outputDir, outRelPath)

	if err := os.MkdirAll(filepath.Dir(outFullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	fileSize, err := copyFile(srcPath, outFullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy image: %w", err)
	}

	result := &ProcessedImage{
		OriginalPath: srcPath,
		RelativePath: relativePath,
		Variants: []ImageVariant{
			{Size: "original", FilePath: "/" + outRelPath, FileSize: fileSize},
		},
	}
	if err := p.cacheProcessedImage(relativePath, modTime, size, result); err != nil {
		return nil, err
	}
	return result, nil
}
