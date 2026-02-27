package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImageCache_SetGet(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewImageCache(tmpDir)

	entry := &CacheEntry{
		SourceModTime: 12345,
		SourceSize:    67890,
		Variants: []ImageVariant{
			{Size: "full", Width: 800, Height: 600, FilePath: "/assets/img/test-full.webp"},
		},
	}

	cache.Set("test.jpg", entry)

	got, ok := cache.Get("test.jpg")
	if !ok {
		t.Fatal("Expected cache hit")
	}
	if got.SourceModTime != 12345 {
		t.Errorf("Expected modtime 12345, got %d", got.SourceModTime)
	}
}

func TestImageCache_Miss(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewImageCache(tmpDir)

	_, ok := cache.Get("nonexistent.jpg")
	if ok {
		t.Error("Expected cache miss")
	}
}

func TestImageCache_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and save
	cache1 := NewImageCache(tmpDir)
	cache1.Set("test.jpg", &CacheEntry{SourceModTime: 111, SourceSize: 222})
	if err := cache1.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	cachePath := filepath.Join(tmpDir, "images", "cache.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file not created")
	}

	// Load in new instance
	cache2 := NewImageCache(tmpDir)
	got, ok := cache2.Get("test.jpg")
	if !ok {
		t.Fatal("Expected cache hit after reload")
	}
	if got.SourceModTime != 111 {
		t.Errorf("Expected modtime 111, got %d", got.SourceModTime)
	}
}

func TestImageCache_Invalidate(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewImageCache(tmpDir)

	cache.Set("test.jpg", &CacheEntry{SourceModTime: 111})
	cache.Invalidate("test.jpg")

	_, ok := cache.Get("test.jpg")
	if ok {
		t.Error("Expected cache miss after invalidation")
	}
}
