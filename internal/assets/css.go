package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
)

// CSSProcessor handles CSS concatenation and minification
type CSSProcessor struct {
	minifyEnabled bool
}

// NewCSSProcessor creates a new CSS processor
func NewCSSProcessor(minifyEnabled bool) *CSSProcessor {
	return &CSSProcessor{minifyEnabled: minifyEnabled}
}

// CSSBundle represents the processed CSS output
type CSSBundle struct {
	Path    string
	Content string
	Size    int64
}

// Process concatenates and optionally minifies CSS files, writing the result to outputDir
func (p *CSSProcessor) Process(cssFiles []string, outputDir string) (*CSSBundle, error) {
	if len(cssFiles) == 0 {
		return nil, nil
	}

	var sb strings.Builder

	for _, path := range cssFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read CSS file %s: %w", path, err)
		}
		sb.WriteString("/* " + filepath.Base(path) + " */\n")
		sb.Write(data)
		sb.WriteString("\n\n")
	}

	content := sb.String()

	// Minify
	if p.minifyEnabled {
		m := minify.New()
		m.AddFunc("text/css", css.Minify)
		minified, err := m.String("text/css", content)
		if err != nil {
			return nil, fmt.Errorf("failed to minify CSS: %w", err)
		}
		content = minified
	}

	// Write output
	outPath := filepath.Join(outputDir, "assets", "css", "main.css")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create CSS output directory: %w", err)
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write CSS bundle: %w", err)
	}

	return &CSSBundle{
		Path:    "/assets/css/main.css",
		Content: content,
		Size:    int64(len(content)),
	}, nil
}

// MinifyString minifies a CSS string directly (useful for inline styles)
func (p *CSSProcessor) MinifyString(input string) (string, error) {
	if !p.minifyEnabled {
		return input, nil
	}
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	return m.String("text/css", input)
}
