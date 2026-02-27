package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SiteConfig represents the complete site configuration
type SiteConfig struct {
	Site       Site         `yaml:"site"`
	Author     Author       `yaml:"author"`
	Build      Build        `yaml:"build"`
	Navigation Navigation   `yaml:"navigation"`
	Assets     AssetsConfig `yaml:"assets"`
}

// AssetsConfig contains asset processing settings
type AssetsConfig struct {
	Images ImagesConfig `yaml:"images"`
	CSS    CSSConfig    `yaml:"css"`
	JS     JSConfig     `yaml:"js"`
	Cache  CacheConfig  `yaml:"cache"`
}

// ImagesConfig contains image processing settings
type ImagesConfig struct {
	Enabled     bool           `yaml:"enabled"`
	Quality     int            `yaml:"quality"`
	Sizes       map[string]int `yaml:"sizes"`
	LazyLoading bool           `yaml:"lazyLoading"`
}

// CSSConfig contains CSS processing settings
type CSSConfig struct {
	Enabled bool `yaml:"enabled"`
	Minify  bool `yaml:"minify"`
}

// JSConfig contains JS processing settings
type JSConfig struct {
	Enabled bool `yaml:"enabled"`
	Minify  bool `yaml:"minify"`
}

// CacheConfig contains cache settings
type CacheConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Directory string `yaml:"directory"`
}

// Navigation contains navigation settings
type Navigation struct {
	Sidebar     SidebarConfig     `yaml:"sidebar"`
	TOC         TOCConfig         `yaml:"toc"`
	Breadcrumbs BreadcrumbsConfig `yaml:"breadcrumbs"`
	PrevNext    PrevNextConfig    `yaml:"prevNext"`
}

// SidebarConfig contains sidebar settings
type SidebarConfig struct {
	Collapsed    bool `yaml:"collapsed"`
	MaxDepth     int  `yaml:"maxDepth"`
	IncludeIndex bool `yaml:"includeIndex"`
}

// TOCConfig contains table of contents settings
type TOCConfig struct {
	MinDepth int  `yaml:"minDepth"`
	MaxDepth int  `yaml:"maxDepth"`
	Sticky   bool `yaml:"sticky"`
}

// BreadcrumbsConfig contains breadcrumb settings
type BreadcrumbsConfig struct {
	Enabled   bool   `yaml:"enabled"`
	HomeLabel string `yaml:"homeLabel"`
}

// PrevNextConfig contains prev/next navigation settings
type PrevNextConfig struct {
	Enabled          bool `yaml:"enabled"`
	SameCategoryOnly bool `yaml:"sameCategoryOnly"`
}

// Site contains site-wide settings
type Site struct {
	Title    string `yaml:"title"`
	Tagline  string `yaml:"tagline"`
	URL      string `yaml:"url"`
	BaseURL  string `yaml:"baseUrl"`
	Language string `yaml:"language"`
	Favicon  string `yaml:"favicon"`
}

// Author contains author information
type Author struct {
	Name   string            `yaml:"name"`
	Email  string            `yaml:"email"`
	Social map[string]string `yaml:"social"`
}

// Build contains build settings
type Build struct {
	ContentDir      string `yaml:"contentDir"`
	OutputDir       string `yaml:"outputDir"`
	CacheDir        string `yaml:"cacheDir"`
	ParallelWorkers int    `yaml:"parallelWorkers"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *SiteConfig {
	return &SiteConfig{
		Site: Site{
			BaseURL:  "/",
			Language: "en",
		},
		Build: Build{
			ContentDir:      "content",
			OutputDir:       "dist",
			CacheDir:        ".cache",
			ParallelWorkers: 4,
		},
		Navigation: Navigation{
			Sidebar: SidebarConfig{
				Collapsed:    false,
				MaxDepth:     3,
				IncludeIndex: true,
			},
			TOC: TOCConfig{
				MinDepth: 2,
				MaxDepth: 4,
				Sticky:   true,
			},
			Breadcrumbs: BreadcrumbsConfig{
				Enabled:   true,
				HomeLabel: "Home",
			},
			PrevNext: PrevNextConfig{
				Enabled:          true,
				SameCategoryOnly: false,
			},
		},
		Assets: AssetsConfig{
			Images: ImagesConfig{
				Enabled:     true,
				Quality:     85,
				Sizes:       map[string]int{"thumbnail": 150, "preview": 400, "full": 1200},
				LazyLoading: true,
			},
			CSS: CSSConfig{
				Enabled: true,
				Minify:  true,
			},
			JS: JSConfig{
				Enabled: true,
				Minify:  true,
			},
			Cache: CacheConfig{
				Enabled:   true,
				Directory: ".cache",
			},
		},
	}
}

// Load reads and parses the configuration file
func Load(path string) (*SiteConfig, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func Validate(cfg *SiteConfig) error {
	if cfg.Site.Title == "" {
		return fmt.Errorf("site.title is required")
	}
	if cfg.Site.URL == "" {
		return fmt.Errorf("site.url is required")
	}
	if cfg.Build.ContentDir == "" {
		cfg.Build.ContentDir = "content"
	}
	if cfg.Build.OutputDir == "" {
		cfg.Build.OutputDir = "dist"
	}
	if cfg.Build.ParallelWorkers <= 0 {
		cfg.Build.ParallelWorkers = 4
	}
	return nil
}
