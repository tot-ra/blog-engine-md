package assets

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func createTestJPEG(t *testing.T, path string, width, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 100, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
}

func TestImageProcessor_ProcessFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	outDir := filepath.Join(tmpDir, "dist")

	// Create test image
	imgPath := filepath.Join(srcDir, "test.jpg")
	createTestJPEG(t, imgPath, 800, 600)

	config := ImageConfig{
		Quality: 85,
		Sizes: map[string]int{
			"thumbnail": 150,
			"full":      1200,
		},
		Enabled: true,
	}

	processor := NewImageProcessor(config, outDir, nil)
	result, err := processor.ProcessFile(imgPath, "test.jpg", 0, 0)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Variants) != 2 {
		t.Fatalf("Expected 2 variants (thumbnail, full), got %d", len(result.Variants))
	}

	// Verify files exist
	for _, v := range result.Variants {
		fullPath := filepath.Join(outDir, v.FilePath[1:]) // strip leading /
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Variant file does not exist: %s", fullPath)
		}
	}
}

func TestImageProcessor_SVGPassthrough(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	outDir := filepath.Join(tmpDir, "dist")

	// Create test SVG
	svgPath := filepath.Join(srcDir, "icon.svg")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"><circle cx="50" cy="50" r="40"/></svg>`
	if err := os.WriteFile(svgPath, []byte(svgContent), 0644); err != nil {
		t.Fatal(err)
	}

	processor := NewImageProcessor(DefaultImageConfig(), outDir, nil)
	result, err := processor.ProcessFile(svgPath, "icon.svg", 0, 0)
	if err != nil {
		t.Fatalf("SVG passthrough failed: %v", err)
	}

	if len(result.Variants) != 1 {
		t.Fatalf("Expected 1 variant (original), got %d", len(result.Variants))
	}
	if result.Variants[0].Size != "original" {
		t.Errorf("Expected 'original' size, got '%s'", result.Variants[0].Size)
	}
}

func TestImageProcessor_SmallImage(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	outDir := filepath.Join(tmpDir, "dist")

	// Create a small 100x80 image — smaller than thumbnail (150px)
	imgPath := filepath.Join(srcDir, "small.jpg")
	createTestJPEG(t, imgPath, 100, 80)

	config := ImageConfig{
		Quality: 85,
		Sizes:   map[string]int{"thumbnail": 150, "full": 1200},
		Enabled: true,
	}

	processor := NewImageProcessor(config, outDir, nil)
	result, err := processor.ProcessFile(imgPath, "small.jpg", 0, 0)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Both variants should be generated but capped at original width (100)
	for _, v := range result.Variants {
		if v.Width > 100 {
			t.Errorf("Variant %s width %d exceeds original 100", v.Size, v.Width)
		}
	}
}

func TestImageProcessor_RestoresVariantsFromPersistentCache(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	outDir := filepath.Join(tmpDir, "dist")
	cacheDir := filepath.Join(tmpDir, ".cache")

	imgPath := filepath.Join(srcDir, "cached.jpg")
	createTestJPEG(t, imgPath, 320, 240)
	info, err := os.Stat(imgPath)
	if err != nil {
		t.Fatal(err)
	}

	config := ImageConfig{
		Quality: 85,
		Sizes:   map[string]int{"thumbnail": 150},
		Enabled: true,
	}

	processor := NewImageProcessor(config, outDir, NewImageCache(cacheDir))
	first, err := processor.ProcessFile(imgPath, "cached.jpg", info.ModTime().Unix(), info.Size())
	if err != nil {
		t.Fatalf("initial ProcessFile failed: %v", err)
	}
	if len(first.Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(first.Variants))
	}
	if err := processor.cache.Save(); err != nil {
		t.Fatalf("cache save failed: %v", err)
	}

	variantPath := filepath.Join(outDir, first.Variants[0].FilePath[1:])
	if err := os.RemoveAll(outDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(variantPath); !os.IsNotExist(err) {
		t.Fatalf("expected clean build output to be removed, stat err: %v", err)
	}

	restoredProcessor := NewImageProcessor(config, outDir, NewImageCache(cacheDir))
	restored, err := restoredProcessor.ProcessFile(imgPath, "cached.jpg", info.ModTime().Unix(), info.Size())
	if err != nil {
		t.Fatalf("cached ProcessFile failed: %v", err)
	}
	if len(restored.Variants) != 1 || restored.Variants[0].FilePath != first.Variants[0].FilePath {
		t.Fatalf("expected cached variant metadata to be reused, got %#v", restored.Variants)
	}
	if _, err := os.Stat(variantPath); err != nil {
		t.Fatalf("expected cached variant to be restored into dist: %v", err)
	}
}
