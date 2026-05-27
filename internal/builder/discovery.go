package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContentType represents the type of content file
type ContentType string

const (
	TypeMarkdown ContentType = "markdown"
	TypeImage    ContentType = "image"
	TypeAsset    ContentType = "asset"
)

// ContentFile represents a discovered content file
type ContentFile struct {
	Path         string
	RelativePath string
	ContentType  ContentType
	ModifiedTime int64
	Size         int64
}

// ContentIndex holds all discovered content files
type ContentIndex struct {
	MarkdownFiles []ContentFile
	ImageFiles    []ContentFile
	AssetFiles    []ContentFile
}

// Discover scans the content directory and catalogs all files
func Discover(root string) (*ContentIndex, error) {
	index := &ContentIndex{
		MarkdownFiles: make([]ContentFile, 0),
		ImageFiles:    make([]ContentFile, 0),
		AssetFiles:    make([]ContentFile, 0),
	}

	// Check if root exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return index, nil // Return empty index if directory doesn't exist
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden/private directories
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || strings.HasPrefix(info.Name(), "_") {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		file := ContentFile{
			Path:         path,
			RelativePath: relPath,
			ModifiedTime: info.ModTime().Unix(),
			Size:         info.Size(),
		}

		normalizedRelPath := filepath.ToSlash(relPath)
		if strings.HasPrefix(normalizedRelPath, "files/") {
			file.ContentType = TypeAsset
			index.AssetFiles = append(index.AssetFiles, file)
			return nil
		}

		// Classify file
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".md", ".markdown":
			file.ContentType = TypeMarkdown
			index.MarkdownFiles = append(index.MarkdownFiles, file)
		case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp":
			file.ContentType = TypeImage
			index.ImageFiles = append(index.ImageFiles, file)
		default:
			file.ContentType = TypeAsset
			index.AssetFiles = append(index.AssetFiles, file)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk content directory: %w", err)
	}

	return index, nil
}
