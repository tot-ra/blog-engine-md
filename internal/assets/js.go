package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/js"
)

// JSProcessor handles JS concatenation and minification
type JSProcessor struct {
	minifyEnabled bool
}

// NewJSProcessor creates a new JS processor
func NewJSProcessor(minifyEnabled bool) *JSProcessor {
	return &JSProcessor{minifyEnabled: minifyEnabled}
}

// JSBundle represents the processed JS output
type JSBundle struct {
	Path    string
	Content string
	Size    int64
}

// Process concatenates and optionally minifies JS files, writing the result to outputDir.
// Built-in scripts (theme toggle, mobile menu) are always included.
func (p *JSProcessor) Process(jsFiles []string, outputDir string) (*JSBundle, error) {
	var sb strings.Builder

	// Built-in scripts
	sb.WriteString(builtinScripts)
	sb.WriteString("\n\n")

	// User JS files
	for _, path := range jsFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read JS file %s: %w", path, err)
		}
		sb.WriteString("/* " + filepath.Base(path) + " */\n")
		sb.Write(data)
		sb.WriteString("\n\n")
	}

	content := sb.String()

	// Minify
	if p.minifyEnabled {
		m := minify.New()
		m.AddFunc("application/javascript", js.Minify)
		minified, err := m.String("application/javascript", content)
		if err != nil {
			return nil, fmt.Errorf("failed to minify JS: %w", err)
		}
		content = minified
	}

	// Write output
	outPath := filepath.Join(outputDir, "assets", "js", "main.js")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create JS output directory: %w", err)
	}

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write JS bundle: %w", err)
	}

	return &JSBundle{
		Path:    "/assets/js/main.js",
		Content: content,
		Size:    int64(len(content)),
	}, nil
}

// builtinScripts contains theme toggle and mobile menu toggle
const builtinScripts = `
// Theme toggle
(function() {
  var theme = localStorage.getItem('theme') || 'light';
  document.documentElement.setAttribute('data-theme', theme);

  document.addEventListener('DOMContentLoaded', function() {
    var btn = document.getElementById('theme-toggle');
    if (btn) {
      btn.addEventListener('click', function() {
        var current = document.documentElement.getAttribute('data-theme');
        var next = current === 'dark' ? 'light' : 'dark';
        document.documentElement.setAttribute('data-theme', next);
        localStorage.setItem('theme', next);
      });
    }
  });
})();

// Mobile menu toggle
(function() {
  document.addEventListener('DOMContentLoaded', function() {
    var toggle = document.getElementById('menu-toggle');
    var sidebar = document.querySelector('.sidebar');
    if (toggle && sidebar) {
      toggle.addEventListener('click', function() {
        sidebar.classList.toggle('sidebar-open');
      });
    }
  });
})();
`
