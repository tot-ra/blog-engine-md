package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewAppliesDefaults(t *testing.T) {
	s := New(Config{})

	if s.host != "localhost" {
		t.Fatalf("expected default host localhost, got %q", s.host)
	}
	if s.port != 3000 {
		t.Fatalf("expected default port 3000, got %d", s.port)
	}
	if s.outputDir != "dist" {
		t.Fatalf("expected default output dir dist, got %q", s.outputDir)
	}
	if s.liveReload {
		t.Fatal("expected live reload to default to false")
	}
	if s.clients == nil {
		t.Fatal("expected clients map to be initialized")
	}
}

func TestNotifyReloadSendsToReadyClientsOnly(t *testing.T) {
	s := New(Config{})
	ready := make(chan string, 1)
	blocked := make(chan string)
	s.clients[ready] = true
	s.clients[blocked] = true

	s.NotifyReload()

	select {
	case got := <-ready:
		if got != "reload" {
			t.Fatalf("expected reload message, got %q", got)
		}
	default:
		t.Fatal("expected ready client to receive reload message")
	}

	select {
	case got := <-blocked:
		t.Fatalf("blocked client should have been skipped, got %q", got)
	default:
	}
}

func TestHandleRequestServesStaticFiles(t *testing.T) {
	outputDir := t.TempDir()
	writeFile(t, filepath.Join(outputDir, "index.html"), "<html><body>home</body></html>")
	writeFile(t, filepath.Join(outputDir, "about.html"), "about page")
	writeFile(t, filepath.Join(outputDir, "blog", "index.html"), "blog index")
	writeFile(t, filepath.Join(outputDir, "assets", "style.css"), "body { color: red; }")

	s := New(Config{OutputDir: outputDir})

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantBody    string
		contentType string
	}{
		{
			name:        "root index",
			path:        "/",
			wantStatus:  http.StatusOK,
			wantBody:    "home",
			contentType: "text/html; charset=utf-8",
		},
		{
			name:        "html extension fallback",
			path:        "/about",
			wantStatus:  http.StatusOK,
			wantBody:    "about page",
			contentType: "text/html; charset=utf-8",
		},
		{
			name:        "directory index fallback",
			path:        "/blog",
			wantStatus:  http.StatusOK,
			wantBody:    "blog index",
			contentType: "text/html; charset=utf-8",
		},
		{
			name:        "asset content type",
			path:        "/assets/style.css",
			wantStatus:  http.StatusOK,
			wantBody:    "color: red",
			contentType: "text/css; charset=utf-8",
		},
		{
			name:       "missing file",
			path:       "/missing",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			s.handleRequest(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}
			if tt.wantBody != "" && !strings.Contains(rr.Body.String(), tt.wantBody) {
				t.Fatalf("expected body to contain %q, got %q", tt.wantBody, rr.Body.String())
			}
			if tt.contentType != "" && rr.Header().Get("Content-Type") != tt.contentType {
				t.Fatalf("expected content type %q, got %q", tt.contentType, rr.Header().Get("Content-Type"))
			}
		})
	}
}

func TestHandleRequestInjectsLiveReloadScriptIntoHTML(t *testing.T) {
	outputDir := t.TempDir()
	writeFile(t, filepath.Join(outputDir, "index.html"), "<html><body>home</body></html>")

	s := New(Config{Host: "127.0.0.1", Port: 4567, OutputDir: outputDir, LiveReload: true})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	s.handleRequest(rr, req)

	body := rr.Body.String()
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status OK, got %d", rr.Code)
	}
	if !strings.Contains(body, "new EventSource('http://127.0.0.1:4567/__livereload')") {
		t.Fatalf("expected live reload EventSource script, got %q", body)
	}
	if !strings.Contains(body, "</body>") {
		t.Fatalf("expected closing body tag to remain, got %q", body)
	}
}

func TestHandleRequestPreventsDirectoryTraversal(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "dist")
	writeFile(t, filepath.Join(outputDir, "index.html"), "public")
	writeFile(t, filepath.Join(root, "secret.html"), "secret")

	s := New(Config{OutputDir: outputDir})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/../secret", nil)

	s.handleRequest(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected traversal attempt to be hidden as not found, got %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "secret") {
		t.Fatalf("directory traversal leaked file contents: %q", rr.Body.String())
	}
}

func TestMimeType(t *testing.T) {
	tests := map[string]string{
		".html": "text/html; charset=utf-8",
		".css":  "text/css; charset=utf-8",
		".js":   "application/javascript",
		".png":  "image/png",
		".woff": "font/woff",
		".bin":  "application/octet-stream",
	}

	for ext, want := range tests {
		if got := mimeType(ext); got != want {
			t.Fatalf("mimeType(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestWatcherScansAndDetectsChanges(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "post.md"), "first")
	writeFile(t, filepath.Join(root, "dist", "ignored.html"), "ignored")

	w := NewWatcher(WatcherConfig{Paths: []string{root}})
	if w.interval != 500*time.Millisecond {
		t.Fatalf("expected default interval, got %s", w.interval)
	}

	if err := w.scanFiles(); err != nil {
		t.Fatalf("scanFiles failed: %v", err)
	}
	if _, ok := w.fileState[filepath.Join(root, "post.md")]; !ok {
		t.Fatal("expected scan to track post.md")
	}
	if _, ok := w.fileState[filepath.Join(root, "dist", "ignored.html")]; ok {
		t.Fatal("expected scan to ignore default dist directory")
	}

	newFile := filepath.Join(root, "new.md")
	writeFile(t, newFile, "new")
	modified := filepath.Join(root, "post.md")
	writeFile(t, modified, "second")
	removed := filepath.Join(root, "removed.md")
	w.fileState[removed] = time.Now()

	changed, err := w.detectChanges()
	if err != nil {
		t.Fatalf("detectChanges failed: %v", err)
	}
	assertContains(t, changed, "new.md")
	assertContains(t, changed, "post.md")
	assertContains(t, changed, "removed.md (deleted)")
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %q in %v", want, values)
}
