package main

import (
	"fmt"
	"os"

	"github.com/tot-ra/blog-engine/internal/builder"
	"github.com/tot-ra/blog-engine/internal/config"
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
	fmt.Println("  blog-engine help     Show this help message")
}

func runBuild() error {
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
