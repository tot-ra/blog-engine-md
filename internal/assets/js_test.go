package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinScriptsScopeSidebarModeStorageToConfiguredDefault(t *testing.T) {
	outDir := t.TempDir()
	bundle, err := NewJSProcessor(false).Process(nil, outDir)
	if err != nil {
		t.Fatalf("process js: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "assets", "js", "main.js")); err != nil {
		t.Fatalf("expected main.js to be written: %v", err)
	}

	if !strings.Contains(bundle.Content, `var defaultMode = sidebar.getAttribute('data-sidebar-default-mode') || 'categories';`) {
		t.Fatalf("expected sidebar default mode to be read from markup")
	}
	if !strings.Contains(bundle.Content, `var storageKey = 'blog-sidebar-mode:' + idx + ':' + defaultMode;`) {
		t.Fatalf("expected saved sidebar mode key to include configured default mode")
	}
	if strings.Contains(bundle.Content, `var storageKey = 'blog-sidebar-mode:' + idx;`) {
		t.Fatalf("expected old sidebar mode storage key not to be used")
	}
}
