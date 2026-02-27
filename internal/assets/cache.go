package assets

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	entries  map[string]*CacheEntry
	mu       sync.RWMutex
}

// NewImageCache creates a new image cache, loading existing entries from disk
func NewImageCache(cacheDir string) *ImageCache {
	c := &ImageCache{
		cacheDir: filepath.Join(cacheDir, "images"),
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
