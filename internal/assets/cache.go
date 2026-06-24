package assets

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CacheEntry stores cached info for a processed image
type CacheEntry struct {
	SourceModTime int64          `json:"sourceModTime"`
	SourceSize    int64          `json:"sourceSize"`
	Variants      []ImageVariant `json:"variants"`
}

// ImageCache provides file-based caching for processed images
type ImageCache struct {
	cacheDir string
	filesDir string
	entries  map[string]*CacheEntry
	mu       sync.RWMutex
}

// NewImageCache creates a new image cache, loading existing entries from disk
func NewImageCache(cacheDir string) *ImageCache {
	imagesDir := filepath.Join(cacheDir, "images")
	c := &ImageCache{
		cacheDir: imagesDir,
		filesDir: filepath.Join(imagesDir, "files"),
		entries:  make(map[string]*CacheEntry),
	}
	c.load()
	return c
}

// Get retrieves a cache entry by source relative path
func (c *ImageCache) Get(key string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	return entry, ok
}

// Set stores a cache entry
func (c *ImageCache) Set(key string, entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry
}

// Save persists the cache to disk
func (c *ImageCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(c.cacheDir, "cache.json"), data, 0644)
}

// CachedFilePath returns the on-disk cache path for an output asset path.
func (c *ImageCache) CachedFilePath(outputRelPath string) (string, error) {
	relPath, err := sanitizedCacheRelPath(outputRelPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(c.filesDir, relPath), nil
}

// StoreVariantFile copies a generated image variant from the build output into
// the persistent cache. This lets later clean builds restore variants without
// doing expensive resize/WebP encoding again.
func (c *ImageCache) StoreVariantFile(outputRelPath, outputDir string) error {
	relPath, err := sanitizedCacheRelPath(outputRelPath)
	if err != nil {
		return err
	}
	srcPath := filepath.Join(outputDir, relPath)
	dstPath := filepath.Join(c.filesDir, relPath)
	return copyFile(srcPath, dstPath)
}

// RestoreVariantFile copies a cached image variant back into the build output.
func (c *ImageCache) RestoreVariantFile(outputRelPath, outputDir string) error {
	relPath, err := sanitizedCacheRelPath(outputRelPath)
	if err != nil {
		return err
	}
	srcPath := filepath.Join(c.filesDir, relPath)
	dstPath := filepath.Join(outputDir, relPath)
	return copyFile(srcPath, dstPath)
}

// load reads existing cache from disk
func (c *ImageCache) load() {
	data, err := os.ReadFile(filepath.Join(c.cacheDir, "cache.json"))
	if err != nil {
		return // No cache file, start fresh
	}

	var entries map[string]*CacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return // Corrupted cache, start fresh
	}

	c.entries = entries
}

// Invalidate removes a cache entry
func (c *ImageCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func sanitizedCacheRelPath(path string) (string, error) {
	rel := strings.TrimPrefix(filepath.ToSlash(path), "/")
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || clean == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe cache path %q", path)
	}
	return clean, nil
}

func copyFile(srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	os.Remove(dstPath)
	if err := os.Link(srcPath, dstPath); err == nil { return nil }

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
