package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Watcher watches for file changes and triggers rebuild
type Watcher struct {
	paths      []string
	ignore     []string
	onChange   func() error
	server     *DevServer
	fileState  map[string]time.Time
	interval   time.Duration
}

// WatcherConfig holds watcher configuration
type WatcherConfig struct {
	Paths    []string
	Ignore   []string
	OnChange func() error
	Server   *DevServer
	Interval time.Duration
}

// NewWatcher creates a file watcher
func NewWatcher(cfg WatcherConfig) *Watcher {
	if cfg.Interval == 0 {
		cfg.Interval = 500 * time.Millisecond
	}
	if len(cfg.Ignore) == 0 {
		cfg.Ignore = []string{".git", "node_modules", ".cache", "dist"}
	}

	return &Watcher{
		paths:     cfg.Paths,
		ignore:    cfg.Ignore,
		onChange:  cfg.OnChange,
		server:    cfg.Server,
		fileState: make(map[string]time.Time),
		interval:  cfg.Interval,
	}
}

// Start begins watching for changes (blocking)
func (w *Watcher) Start() error {
	// Build initial state
	if err := w.scanFiles(); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	fmt.Printf("👀 Watching %d files for changes...\n", len(w.fileState))

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for range ticker.C {
		changed, err := w.detectChanges()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)
			continue
		}

		if len(changed) > 0 {
			fmt.Printf("\n📝 Changed: %s\n", strings.Join(changed, ", "))
			fmt.Println("🔄 Rebuilding...")

			start := time.Now()
			if err := w.onChange(); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Rebuild error: %v\n", err)
			} else {
				fmt.Printf("✅ Rebuilt in %s\n", time.Since(start).Round(time.Millisecond))
				if w.server != nil {
					w.server.NotifyReload()
				}
			}
		}
	}

	return nil
}

func (w *Watcher) scanFiles() error {
	for _, root := range w.paths {
		if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if info.IsDir() {
				name := info.Name()
				for _, ig := range w.ignore {
					if name == ig {
						return filepath.SkipDir
					}
				}
				return nil
			}
			w.fileState[path] = info.ModTime()
			return nil
		}); err != nil {
			// Directory might not exist yet
			continue
		}
	}
	return nil
}

func (w *Watcher) detectChanges() ([]string, error) {
	var changed []string
	currentState := make(map[string]time.Time)

	for _, root := range w.paths {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				name := info.Name()
				for _, ig := range w.ignore {
					if name == ig {
						return filepath.SkipDir
					}
				}
				return nil
			}
			currentState[path] = info.ModTime()
			return nil
		})
	}

	// Check for new or modified files
	for path, modTime := range currentState {
		oldTime, exists := w.fileState[path]
		if !exists || !modTime.Equal(oldTime) {
			changed = append(changed, filepath.Base(path))
		}
	}

	// Check for deleted files
	for path := range w.fileState {
		if _, exists := currentState[path]; !exists {
			changed = append(changed, filepath.Base(path)+" (deleted)")
		}
	}

	// Update state
	w.fileState = currentState

	return changed, nil
}
