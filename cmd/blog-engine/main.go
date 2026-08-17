package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tot-ra/blog-engine/internal/builder"
	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/embeddings"
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
		serveConfig, err := parseServeConfig(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := runServe(serveConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "embed":
		embedConfig, err := parseEmbedConfig(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := runEmbed(embedConfig); err != nil {
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
	fmt.Println("  blog-engine serve [--port 3000]    Run the dev server with live reload")
	fmt.Println("  blog-engine embed [--check|--force|--dry-run]    Generate article frontmatter embeddings")
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

type embedConfig struct {
	check  bool
	force  bool
	dryRun bool
}

func parseEmbedConfig(args []string) (embedConfig, error) {
	flags := flag.NewFlagSet("embed", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	check := flags.Bool("check", false, "verify article frontmatter embeddings without network access")
	force := flags.Bool("force", false, "regenerate all embeddings")
	dryRun := flags.Bool("dry-run", false, "show planned work and estimated cost")
	if err := flags.Parse(args); err != nil {
		return embedConfig{}, err
	}
	if flags.NArg() != 0 {
		return embedConfig{}, fmt.Errorf("unexpected embed arguments: %v", flags.Args())
	}
	if *check && (*force || *dryRun) {
		return embedConfig{}, fmt.Errorf("--check cannot be combined with --force or --dry-run")
	}
	return embedConfig{check: *check, force: *force, dryRun: *dryRun}, nil
}

func runEmbed(embedCfg embedConfig) error {
	if _, err := loadEnvFiles(); err != nil {
		return err
	}
	cfg, err := config.Load("config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var client embeddings.Embedder
	if !embedCfg.check && !embedCfg.dryRun {
		apiKey := os.Getenv(cfg.Related.APIKeyEnv)
		if apiKey == "" {
			return fmt.Errorf("environment variable %s is required", cfg.Related.APIKeyEnv)
		}
		client = &embeddings.OpenAIClient{APIKey: apiKey}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	_, err = embeddings.Run(ctx, cfg, embeddings.RunOptions{
		Check: embedCfg.check, Force: embedCfg.force, DryRun: embedCfg.dryRun,
		Output: os.Stdout, Client: client,
	})
	if errors.Is(err, embeddings.ErrCacheStale) {
		return embeddings.ErrCacheStale
	}
	return err
}

type serveConfig struct {
	port int
}

func parseServeConfig(args []string) (serveConfig, error) {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	port := flags.Int("port", 3000, "port to serve on")
	if err := flags.Parse(args); err != nil {
		return serveConfig{}, err
	}
	if flags.NArg() != 0 {
		return serveConfig{}, fmt.Errorf("unexpected serve arguments: %v", flags.Args())
	}
	if *port < 1 || *port > 65535 {
		return serveConfig{}, fmt.Errorf("port must be between 1 and 65535")
	}
	return serveConfig{port: *port}, nil
}

func runServe(serveCfg serveConfig) error {
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
		Port:       serveCfg.port,
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
