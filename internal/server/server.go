package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DevServer is a development server with live reload
type DevServer struct {
	host       string
	port       int
	outputDir  string
	liveReload bool
	clients    map[chan string]bool
	mu         sync.Mutex
}

// Config holds dev server configuration
type Config struct {
	Host       string
	Port       int
	OutputDir  string
	LiveReload bool
}

// New creates a new dev server
func New(cfg Config) *DevServer {
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == 0 {
		cfg.Port = 3000
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "dist"
	}

	return &DevServer{
		host:       cfg.Host,
		port:       cfg.Port,
		outputDir:  cfg.OutputDir,
		liveReload: cfg.LiveReload,
		clients:    make(map[chan string]bool),
	}
}

// Start starts the dev server
func (s *DevServer) Start() error {
	mux := http.NewServeMux()

	// Live reload SSE endpoint
	if s.liveReload {
		mux.HandleFunc("/__livereload", s.handleSSE)
	}

	// Static file server with HTML injection for live reload
	mux.HandleFunc("/", s.handleRequest)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	fmt.Printf("🚀 Dev server running at http://%s\n", addr)
	if s.liveReload {
		fmt.Println("   Live reload enabled")
	}
	fmt.Println("   Press Ctrl+C to stop")

	return http.ListenAndServe(addr, mux)
}

// NotifyReload sends a reload signal to all connected clients
func (s *DevServer) NotifyReload() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for ch := range s.clients {
		select {
		case ch <- "reload":
		default:
			// Client not ready, skip
		}
	}
}

func (s *DevServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 1)
	s.mu.Lock()
	s.clients[ch] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()

	// Send heartbeat to keep connection alive
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *DevServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	urlPath := strings.TrimPrefix(r.URL.Path, "/")
	if urlPath == "" {
		urlPath = "."
	}

	// Clean the path and ensure it stays inside outputDir.
	outputRoot, err := filepath.Abs(s.outputDir)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	filePath := filepath.Clean(filepath.Join(outputRoot, urlPath))
	if filePath != outputRoot && !strings.HasPrefix(filePath, outputRoot+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}

	// If path is a directory, look for index.html
	info, err := os.Stat(filePath)
	if err == nil && info.IsDir() {
		filePath = filepath.Join(filePath, "index.html")
	} else if err != nil {
		// Try adding .html
		if _, err2 := os.Stat(filePath + ".html"); err2 == nil {
			filePath = filePath + ".html"
		} else {
			// Try as directory with index.html
			filePath = filepath.Join(filePath, "index.html")
		}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Determine content type
	ext := filepath.Ext(filePath)
	contentType := mimeType(ext)
	w.Header().Set("Content-Type", contentType)

	// Inject live reload script into HTML
	if s.liveReload && strings.HasSuffix(filePath, ".html") {
		content := string(data)
		script := fmt.Sprintf(`<script>
(function() {
  var es = new EventSource('http://%s:%d/__livereload');
  es.onmessage = function(e) {
    if (e.data === 'reload') location.reload();
  };
  es.onerror = function() {
    setTimeout(function() { location.reload(); }, 2000);
  };
})();
</script>`, s.host, s.port)

		content = strings.Replace(content, "</body>", script+"\n</body>", 1)
		w.Write([]byte(content))
		return
	}

	w.Write(data)

	log.Printf("%s %s → %s", r.Method, r.URL.Path, filePath)
}

func mimeType(ext string) string {
	types := map[string]string{
		".html":  "text/html; charset=utf-8",
		".css":   "text/css; charset=utf-8",
		".js":    "application/javascript",
		".mjs":   "application/javascript",
		".json":  "application/json",
		".xml":   "application/xml",
		".pdf":   "application/pdf",
		".svg":   "image/svg+xml",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".webp":  "image/webp",
		".ico":   "image/x-icon",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
	}
	if ct, ok := types[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}
