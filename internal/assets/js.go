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

// builtinScripts contains theme toggle, mobile menu toggle, and optional Mermaid rendering
const builtinScripts = `
// Mermaid diagrams
(function() {
  function renderMermaidDiagrams() {
    var blocks = document.querySelectorAll('pre > code.language-mermaid, pre > code.lang-mermaid');
    if (!blocks.length) return;

    function replaceBlocks(mermaid) {
      try {
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: 'strict',
          theme: document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'default'
        });
      } catch (_) {}

      blocks.forEach(function(code, index) {
        var pre = code.parentElement;
        if (!pre || pre.dataset.mermaidRendered === 'true') return;
        var diagram = document.createElement('div');
        diagram.className = 'mermaid';
        diagram.textContent = code.textContent || '';
        diagram.id = 'mermaid-diagram-' + index;
        pre.dataset.mermaidRendered = 'true';
        pre.replaceWith(diagram);
      });

      try {
        var result = mermaid.run({ querySelector: '.mermaid' });
        if (result && typeof result.catch === 'function') result.catch(function() {});
      } catch (_) {}
    }

    if (window.mermaid) {
      replaceBlocks(window.mermaid);
      return;
    }

    // WHY: Mermaid is only needed on pages that contain mermaid fences.
    // WHAT: load one browser ESM bundle lazily instead of adding a build-time dependency.
    import('https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs')
      .then(function(mod) { replaceBlocks(mod.default || mod); })
      .catch(function() {});
  }

  document.addEventListener('DOMContentLoaded', renderMermaidDiagrams);
})();

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

    function toggleSidebarSection(section) {
      if (!section) return;
      var btn = section.querySelector(':scope > .sidebar-section-head .sidebar-toggle');
      section.classList.toggle('expanded');
      if (btn) {
        btn.setAttribute('aria-expanded', section.classList.contains('expanded') ? 'true' : 'false');
      }
    }

    var sectionToggles = document.querySelectorAll('.sidebar-toggle');
    sectionToggles.forEach(function(btn) {
      btn.addEventListener('click', function(event) {
        event.preventDefault();
        event.stopPropagation();
        toggleSidebarSection(btn.closest('.sidebar-section'));
      });
    });

    var sectionLinks = document.querySelectorAll('.sidebar-section > .sidebar-section-head > a');
    sectionLinks.forEach(function(link) {
      link.addEventListener('click', function(event) {
        if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return;
        event.preventDefault();
        toggleSidebarSection(link.closest('.sidebar-section'));
      });
    });

    var modeSidebars = document.querySelectorAll('[data-sidebar-mode]');
    modeSidebars.forEach(function(sidebar, idx) {
      var layout = document.querySelector('.site-layout');
      var defaultMode = sidebar.getAttribute('data-sidebar-default-mode') || 'categories';
      var storageKey = 'blog-sidebar-mode:' + idx + ':' + defaultMode;
      var panes = sidebar.querySelectorAll('[data-sidebar-mode-pane]');
      var btns = sidebar.querySelectorAll('[data-sidebar-mode-btn]');

      function applyMode(mode) {
        if (!mode) mode = 'categories';
        sidebar.setAttribute('data-sidebar-mode', mode);
        btns.forEach(function(btn) {
          var active = btn.getAttribute('data-sidebar-mode-btn') === mode;
          btn.classList.toggle('is-active', active);
          btn.setAttribute('aria-selected', active ? 'true' : 'false');
        });
        panes.forEach(function(pane) {
          var shown = pane.getAttribute('data-sidebar-mode-pane') === mode;
          pane.hidden = !shown;
        });

        if (layout) {
          layout.classList.toggle('sidebar-graph-mode', mode === 'graph');
        }

        if (mode === 'graph') {
          var frame = sidebar.querySelector('.sidebar-graph-frame');
          if (frame && !frame.getAttribute('src')) {
            var src = frame.getAttribute('data-src');
            if (src) frame.setAttribute('src', src);
          }
        }

        try {
          localStorage.setItem(storageKey, mode);
        } catch (_) {}
      }

      btns.forEach(function(btn) {
        btn.addEventListener('click', function() {
          applyMode(btn.getAttribute('data-sidebar-mode-btn'));
        });
      });

      var savedMode = defaultMode;
      try {
        savedMode = localStorage.getItem(storageKey) || savedMode;
      } catch (_) {}
      applyMode(savedMode);
    });

    window.addEventListener('message', function(event) {
      if (!event || event.origin !== window.location.origin || !event.data) return;
      if (event.data.type !== 'blog-graph-navigate') return;
      if (typeof event.data.url !== 'string' || event.data.url.length === 0) return;
      window.location.href = event.data.url;
    });
  });
})();
`
