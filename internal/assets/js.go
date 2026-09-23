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

    function normalizeMermaidSource(source) {
      source = (source || '').replace(/\r\n?/g, '\n');
      if (!/^\s*classDiagram\b/m.test(source)) return source;

      var sqlTypes = {
        bigint: true, binary: true, bit: true, blob: true, bool: true, boolean: true,
        char: true, date: true, datetime: true, decimal: true, double: true, enum: true,
        float: true, int: true, integer: true, json: true, longblob: true, longtext: true,
        mediumblob: true, mediumint: true, mediumtext: true, numeric: true, real: true,
        set: true, smallint: true, text: true, time: true, timestamp: true, tinyblob: true,
        tinyint: true, tinytext: true, varbinary: true, varchar: true, year: true
      };

      function indentOf(line) {
        var match = line.match(/^\s*/);
        return match ? match[0] : '';
      }

      function safeType(type) {
        type = (type || '').trim();
        if (/^enum\s*\(/i.test(type)) return 'enum';
        type = type.replace(/\(([^)]*)\)/g, '_$1');
        type = type.replace(/[^A-Za-z0-9_]+/g, '_').replace(/^_+|_+$/g, '').replace(/_+/g, '_');
        return type || 'field';
      }

      function normalizeClassMember(line) {
        var trimmed = line.trim();
        if (!trimmed || trimmed === '}' || /^%%/.test(trimmed) || /^[+\-#~]/.test(trimmed)) return line;

        var match = trimmed.match(/^(.+?)\s+([A-Za-z_][A-Za-z0-9_]*)\s*$/);
        if (!match) return line;

        var type = match[1].trim();
        var name = match[2].trim();
        var firstTypeToken = type.split(/\s|\(/)[0].toLowerCase();
        var needsNormalization = sqlTypes[firstTypeToken] || /\s|\(|\)|'|,/.test(type);
        if (!needsNormalization) return line;

        // WHY: Mermaid 11 rejects SQL-like members such as "int unsigned user_id".
        // WHAT: keep the visible field name and collapse SQL type metadata to one token.
        return indentOf(line) + '+' + safeType(type) + ' ' + name;
      }

      var inClass = false;
      return source.split('\n').map(function(line) {
        var trimmed = line.trim();
        if (/^class\s+[^\s{]+\s*\{\s*$/.test(trimmed)) {
          inClass = true;
          return line;
        }
        if (inClass && trimmed === '}') {
          inClass = false;
          return line;
        }
        return inClass ? normalizeClassMember(line) : line;
      }).join('\n');
    }

    function renderBlock(mermaid, code, index) {
      var pre = code.parentElement;
      if (!pre || pre.dataset.mermaidRendered === 'true') return;

      var source = normalizeMermaidSource(code.textContent || '');
      var diagram = document.createElement('div');
      diagram.className = 'mermaid';
      diagram.id = 'mermaid-diagram-' + index;
      pre.dataset.mermaidRendered = 'true';
      pre.replaceWith(diagram);

      try {
        var renderResult = mermaid.render('mermaid-svg-' + index, source, diagram);
        if (renderResult && typeof renderResult.then === 'function') {
          renderResult.then(function(result) {
            diagram.innerHTML = result.svg;
            if (result.bindFunctions) result.bindFunctions(diagram);
          }).catch(function() {
            diagram.replaceWith(pre);
            pre.dataset.mermaidRendered = 'false';
          });
          return;
        }
        if (renderResult && renderResult.svg) {
          diagram.innerHTML = renderResult.svg;
          if (renderResult.bindFunctions) renderResult.bindFunctions(diagram);
        }
      } catch (_) {
        diagram.replaceWith(pre);
        pre.dataset.mermaidRendered = 'false';
      }
    }

    function replaceBlocks(mermaid) {
      try {
        mermaid.initialize({
          startOnLoad: false,
          securityLevel: 'strict',
          theme: document.documentElement.getAttribute('data-theme') === 'dark' ? 'dark' : 'default',
          suppressErrorRendering: true
        });
      } catch (_) {}

      blocks.forEach(function(code, index) {
        renderBlock(mermaid, code, index);
      });
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

// TOC scrollspy: highlight the nav item for the section currently in view.
// WHY: native JS only (no deps), same idea as Twitter's section spy.
(function() {
  document.addEventListener('DOMContentLoaded', function() {
    var toc = document.querySelector('nav.toc');
    if (!toc) return;

    var links = Array.prototype.slice.call(toc.querySelectorAll('a[href^="#"]'));
    if (!links.length) return;

    var linkById = {};
    var sections = [];
    links.forEach(function(link) {
      var id = decodeURIComponent((link.getAttribute('href') || '').slice(1));
      if (!id) return;
      var el = document.getElementById(id);
      if (!el) return;
      linkById[id] = link;
      sections.push(el);
    });
    if (!sections.length) return;

    var activeId = '';

    function setActive(id) {
      if (id === activeId) return;
      activeId = id;
      links.forEach(function(link) {
        var on = linkById[id] === link;
        link.classList.toggle('is-active', on);
        if (on) {
          link.setAttribute('aria-current', 'location');
        } else {
          link.removeAttribute('aria-current');
        }
      });
    }

    // Prefer the last section whose top has crossed ~25% of the viewport.
    function syncFromScroll() {
      var marker = window.scrollY + Math.min(180, window.innerHeight * 0.25);
      var current = sections[0] ? sections[0].id : '';
      for (var i = 0; i < sections.length; i++) {
        var top = sections[i].getBoundingClientRect().top + window.scrollY;
        if (top <= marker) {
          current = sections[i].id;
        } else {
          break;
        }
      }
      if (current) setActive(current);
    }

    var ticking = false;
    function onScroll() {
      if (ticking) return;
      ticking = true;
      requestAnimationFrame(function() {
        ticking = false;
        syncFromScroll();
      });
    }

    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('resize', onScroll);
    syncFromScroll();
  });
})();

// Location links: folder-based place notes open inline with a map.
(function() {
  var LABELS = {
    ru: { close: 'Закрыть', map: 'Карта', open: 'Открыть на карте' },
    et: { close: 'Sulge', map: 'Kaart', open: 'Ava kaardil' },
    en: { close: 'Close', map: 'Map', open: 'Open map' }
  };

  function labels() {
    var lang = (document.documentElement.lang || 'en').toLowerCase().split('-')[0];
    return LABELS[lang] || LABELS.en;
  }

  function loadPreviews() {
    var node = document.querySelector('script.location-previews');
    if (!node) return {};
    try {
      var data = JSON.parse(node.textContent || '{}');
      return data.locations || {};
    } catch (_) {
      return {};
    }
  }

  function closestLocationLink(el) {
    while (el && el !== document) {
      if (el.classList && el.classList.contains('location-link')) return el;
      el = el.parentNode;
    }
    return null;
  }

  var leafletPromise = null;

  function loadLeaflet() {
    if (window.L) return Promise.resolve(window.L);
    if (leafletPromise) return leafletPromise;
    leafletPromise = new Promise(function(resolve, reject) {
      if (!document.querySelector('link[data-location-leaflet]')) {
        var css = document.createElement('link');
        css.rel = 'stylesheet';
        css.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
        css.setAttribute('data-location-leaflet', 'true');
        document.head.appendChild(css);
      }
      var script = document.createElement('script');
      script.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js';
      script.onload = function() { resolve(window.L); };
      script.onerror = reject;
      document.head.appendChild(script);
    });
    return leafletPromise;
  }

  function mountLocationMap(el, L) {
    if (!el || el.getAttribute('data-map-ready') === 'true') return;
    var lat = parseFloat(el.getAttribute('data-lat'));
    var lng = parseFloat(el.getAttribute('data-lng'));
    if (isNaN(lat) || isNaN(lng)) return;
    el.setAttribute('data-map-ready', 'true');
    var map = L.map(el, { scrollWheelZoom: false }).setView([lat, lng], 16);
    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; OpenStreetMap',
      maxZoom: 19
    }).addTo(map);
    L.marker([lat, lng]).addTo(map);
    setTimeout(function() { map.invalidateSize(); }, 0);
  }

  function hydrateLocationMaps(root) {
    var nodes = (root || document).querySelectorAll('.location-map[data-lat][data-lng]');
    if (!nodes.length) return;
    loadLeaflet().then(function(L) {
      nodes.forEach(function(el) { mountLocationMap(el, L); });
    }).catch(function() {});
  }

  function mapBlock(loc, copy) {
    var wrap = document.createElement('div');
    wrap.className = 'location-panel-map-wrap';
    if (typeof loc.lat === 'number' && typeof loc.lng === 'number') {
      var map = document.createElement('div');
      map.className = 'location-map';
      map.setAttribute('role', 'img');
      map.setAttribute('aria-label', copy.map);
      map.setAttribute('data-lat', String(loc.lat));
      map.setAttribute('data-lng', String(loc.lng));
      map.setAttribute('data-title', loc.title || '');
      wrap.appendChild(map);
      var open = document.createElement('p');
      open.className = 'location-panel-map-link';
      var a = document.createElement('a');
      a.href = 'https://www.openstreetmap.org/?mlat=' + encodeURIComponent(loc.lat) +
        '&mlon=' + encodeURIComponent(loc.lng) + '#map=17/' + loc.lat + '/' + loc.lng;
      a.target = '_blank';
      a.rel = 'noopener';
      a.textContent = copy.open;
      open.appendChild(a);
      wrap.appendChild(open);
      return wrap;
    }
    if (loc.address) {
      var search = document.createElement('p');
      search.className = 'location-panel-map-link';
      var link = document.createElement('a');
      link.href = 'https://www.openstreetmap.org/search?query=' + encodeURIComponent(loc.address);
      link.target = '_blank';
      link.rel = 'noopener';
      link.textContent = copy.open;
      search.appendChild(link);
      wrap.appendChild(search);
      return wrap;
    }
    return null;
  }

  function renderPanel(loc, copy) {
    var panel = document.createElement('div');
    panel.className = 'location-panel';
    panel.setAttribute('role', 'region');
    if (loc.title) panel.setAttribute('aria-label', loc.title);

    var head = document.createElement('div');
    head.className = 'location-panel-head';
    var title = document.createElement('strong');
    title.className = 'location-panel-title';
    title.textContent = loc.title || '';
    var close = document.createElement('button');
    close.type = 'button';
    close.className = 'location-panel-close';
    close.setAttribute('aria-label', copy.close);
    close.textContent = '×';
    head.appendChild(title);
    head.appendChild(close);
    panel.appendChild(head);

    if (loc.address) {
      var addr = document.createElement('p');
      addr.className = 'location-panel-address';
      addr.textContent = loc.address;
      panel.appendChild(addr);
    }

    var body = document.createElement('div');
    body.className = 'location-panel-body';
    body.innerHTML = loc.html || '';
    panel.appendChild(body);

    var map = mapBlock(loc, copy);
    if (map) panel.appendChild(map);
    return panel;
  }

  var previews = null;
  var openPanel = null;
  var openLink = null;

  function closePanel() {
    if (openPanel && openPanel.parentNode) openPanel.parentNode.removeChild(openPanel);
    openPanel = null;
    if (openLink) {
      openLink.classList.remove('is-open');
      openLink.setAttribute('aria-expanded', 'false');
      openLink = null;
    }
  }

  function openFor(link) {
    if (previews === null) previews = loadPreviews();
    var href = link.getAttribute('href') || '';
    var loc = previews[href];
    if (!loc) return false;

    if (openLink === link) {
      closePanel();
      return true;
    }

    closePanel();
    var copy = labels();
    var panel = renderPanel(loc, copy);
    var block = link.closest('p, li, blockquote, h2, h3, h4, div') || link.parentNode;
    if (!block || !block.parentNode) return false;
    block.parentNode.insertBefore(panel, block.nextSibling);
    hydrateLocationMaps(panel);
    link.classList.add('is-open');
    link.setAttribute('aria-expanded', 'true');
    openPanel = panel;
    openLink = link;
    return true;
  }

  document.addEventListener('DOMContentLoaded', function() {
    hydrateLocationMaps(document);
  });

  document.addEventListener('click', function(event) {
    var closeBtn = event.target && event.target.closest ? event.target.closest('.location-panel-close') : null;
    if (closeBtn) {
      event.preventDefault();
      closePanel();
      return;
    }
    var link = closestLocationLink(event.target);
    if (!link) {
      if (openPanel && !openPanel.contains(event.target)) closePanel();
      return;
    }
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return;
    if (openFor(link)) event.preventDefault();
  });

  document.addEventListener('keydown', function(event) {
    if (event.key === 'Escape') closePanel();
  });
})();
`
