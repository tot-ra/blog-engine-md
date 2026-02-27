package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"blog/post1.md":     "# Post 1",
		"blog/post2.md":     "# Post 2",
		"docs/readme.md":    "# Readme",
		"img/logo.png":      "fake png data",
		"css/style.css":     "body {}",
		".hidden/file.md":   "# Hidden",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	index, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Check markdown files
	if len(index.MarkdownFiles) != 3 {
		t.Errorf("Expected 3 markdown files, got %d", len(index.MarkdownFiles))
	}

	// Check image files
	if len(index.ImageFiles) != 1 {
		t.Errorf("Expected 1 image file, got %d", len(index.ImageFiles))
	}

	// Check asset files
	if len(index.AssetFiles) != 1 {
		t.Errorf("Expected 1 asset file, got %d", len(index.AssetFiles))
	}

	// Verify markdown file paths
	foundPaths := make(map[string]bool)
	for _, f := range index.MarkdownFiles {
		foundPaths[f.RelativePath] = true
	}

	expectedPaths := []string{"blog/post1.md", "blog/post2.md", "docs/readme.md"}
	for _, path := range expectedPaths {
		if !foundPaths[path] {
			t.Errorf("Expected to find %s", path)
		}
	}

	// Verify hidden files are excluded
	if foundPaths[".hidden/file.md"] {
		t.Error("Hidden files should be excluded")
	}
}

func TestDiscoverEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	index, err := Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(index.MarkdownFiles) != 0 {
		t.Errorf("Expected 0 markdown files, got %d", len(index.MarkdownFiles))
	}
}

func TestDiscoverNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "non-existent")

	index, err := Discover(nonExistent)
	if err != nil {
		t.Fatalf("Discover should not fail for non-existent directory: %v", err)
	}

	if len(index.MarkdownFiles) != 0 {
		t.Error("Expected empty index for non-existent directory")
	}
}
