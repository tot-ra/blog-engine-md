package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tot-ra/blog-engine/internal/builder"
	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "build":
		if err := runBuild(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "serve":
		if err := runServe(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Blog Engine MD - Static site generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  blog-engine build    Build the site")
	fmt.Println("  blog-engine serve    Run the dev server with live reload")
	fmt.Println("  blog-engine help     Show this help message")
}

func runBuild() error {
	if _, err := loadEnvFiles(); err != nil {
		return err
	}

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		// Try to create a default config if none exists
		if os.IsNotExist(err) {
			fmt.Println("No config.yaml found, using defaults")
			cfg = config.DefaultConfig()
			cfg.Site.Title = "My Blog"
			cfg.Site.URL = "http://localhost"
		} else {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Build site
	siteBuilder := builder.NewSiteBuilder(cfg)
	if err := siteBuilder.Build(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Println("Build completed successfully!")
	return nil
}

func runServe() error {
	envFiles, err := loadEnvFiles()
	if err != nil {
		return err
	}

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No config.yaml found, using defaults")
			cfg = config.DefaultConfig()
			cfg.Site.Title = "My Blog"
			cfg.Site.URL = "http://localhost"
		} else {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if cfg.Build.OutputDir == "" {
		cfg.Build.OutputDir = "public"
	}

	// Initial build
	fmt.Println("Building initial site...")
	siteBuilder := builder.NewSiteBuilder(cfg)
	if err := siteBuilder.Build(); err != nil {
		fmt.Printf("Initial build error: %v\n", err)
	}

	// Configure server
	srv := server.New(server.Config{
		Host:       "127.0.0.1",
		Port:       3000,
		OutputDir:  cfg.Build.OutputDir,
		LiveReload: true,
	})

	// Configure watcher
	watchPaths := []string{"content", "templates", "assets", "config.yaml"}
	watchPaths = append(watchPaths, envFiles...)
	watcher := server.NewWatcher(server.WatcherConfig{
		Paths:  watchPaths,
		Ignore: []string{".git", "node_modules", ".cache", cfg.Build.OutputDir},
		Server: srv,
		OnChange: func() error {
			if _, err := loadEnvFiles(); err != nil {
				return err
			}

			// Reload config if changed
			newCfg, err := config.Load("config.yaml")
			if err == nil {
				cfg = newCfg
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("config reload failed: %w", err)
			}
			b := builder.NewSiteBuilder(cfg)
			return b.Build()
		},
	})

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Start watcher (blocks)
	return watcher.Start()
}

func loadEnvFiles() ([]string, error) {
	var candidates []string
	exePath, err := os.Executable()
	if err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), ".env"))
	}
	candidates = append(candidates, ".env")

	seen := make(map[string]struct{}, len(candidates))
	loaded := make([]string, 0, len(candidates))
	for _, path := range candidates {
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		if err := config.LoadDotEnv(path); err != nil {
			return nil, fmt.Errorf("failed to load %q: %w", path, err)
		}
		loaded = append(loaded, path)
	}

	return loaded, nil
}
