package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SiteConfig represents the complete site configuration
type SiteConfig struct {
	Site       Site             `yaml:"site"`
	Author     Author          `yaml:"author"`
	Build      Build           `yaml:"build"`
	Navigation Navigation      `yaml:"navigation"`
	Assets     AssetsConfig    `yaml:"assets"`
	Feeds      FeedsConfig     `yaml:"feeds"`
	Sitemap    SitemapConfig   `yaml:"sitemap"`
	Tags       TagsConfig      `yaml:"tags"`
	Archive    ArchiveConfig   `yaml:"archive"`
	Pagination PaginationConfig `yaml:"pagination"`
	Advanced   AdvancedConfig  `yaml:"advanced"`
	SEO        SEOConfig       `yaml:"seo"`
	Homepage   HomepageConfig  `yaml:"homepage"` // Homepage customization
}

// HomepageConfig contains homepage-specific settings
type HomepageConfig struct {
	Enabled     bool                `yaml:"enabled"`
	Hero        HeroConfig          `yaml:"hero"`
	Projects    []ProjectConfig     `yaml:"projects"`
	SocialLinks []SocialLinkGroup   `yaml:"socialLinks"`
	CustomHTML  string              `yaml:"customHTML"` // Additional custom HTML/JS
}

// HeroConfig contains hero section settings
type HeroConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Background  string `yaml:"background"`  // Background image URL
	Title       string `yaml:"title"`       // Override page title
	Subtitle    string `yaml:"subtitle"`    // Subtitle/tagline
	Description string `yaml:"description"` // Description text
	CTAButtons  []CTAButton `yaml:"ctaButtons"` // Call-to-action buttons
}

// CTAButton represents a call-to-action button
type CTAButton struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
	Icon  string `yaml:"icon"` // Optional icon class/name
}

// ProjectConfig represents a project showcase item
type ProjectConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Image       string `yaml:"image"`    // Project image URL
	URL         string `yaml:"url"`      // Link to project
	GitHub      string `yaml:"github"`   // GitHub repo URL
	Tags        []string `yaml:"tags"`   // Technology tags
}

// SocialLinkGroup represents a group of social links (e.g., "Business", "Community")
type SocialLinkGroup struct {
	Title string       `yaml:"title"`
	Links []SocialLink `yaml:"links"`
}

// SocialLink represents a single social media/link entry
type SocialLink struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
	Icon  string `yaml:"icon"`
}

// FeedsConfig contains feed generation settings
type FeedsConfig struct {
	RSS  RSSConfig  `yaml:"rss"`
	Atom AtomConfig `yaml:"atom"`
}

// RSSConfig contains RSS feed settings
type RSSConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Path        string `yaml:"path"`
	Items       int    `yaml:"items"`
	FullContent bool   `yaml:"fullContent"`
}

// AtomConfig contains Atom feed settings
type AtomConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
	Items   int    `yaml:"items"`
}

// SitemapConfig contains sitemap generation settings
type SitemapConfig struct {
	Enabled bool `yaml:"enabled"`
}

// TagsConfig contains tag system settings
type TagsConfig struct {
	Enabled bool `yaml:"enabled"`
}

// ArchiveConfig contains archive page settings
type ArchiveConfig struct {
	Enabled bool `yaml:"enabled"`
}

// PaginationConfig contains pagination settings
type PaginationConfig struct {
	Enabled  bool `yaml:"enabled"`
	PageSize int  `yaml:"pageSize"`
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
	Profile         bool   `yaml:"profile"`
}

// AdvancedConfig contains Phase 5 advanced feature settings
type AdvancedConfig struct {
	Mermaid     MermaidConfig     `yaml:"mermaid"`
	Graph       GraphConfig       `yaml:"graph"`
	Theme       ThemeConfig       `yaml:"theme"`
	Embeds      EmbedsConfig      `yaml:"embeds"`
	Admonitions AdmonitionsConfig `yaml:"admonitions"`
	ScrollSpy   ScrollSpyConfig   `yaml:"scrollSpy"`
	CodeCopy    CodeCopyConfig    `yaml:"codeCopy"`
}

// MermaidConfig contains Mermaid diagram settings
type MermaidConfig struct {
	Enabled bool   `yaml:"enabled"`
	Theme   string `yaml:"theme"` // default, dark, forest, neutral
}

// GraphConfig contains graph view settings
type GraphConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// ThemeConfig contains theme toggle settings
type ThemeConfig struct {
	Default     string `yaml:"default"` // light, dark, auto
	AllowToggle bool   `yaml:"allowToggle"`
}

// EmbedsConfig contains embed support settings
type EmbedsConfig struct {
	Enabled bool `yaml:"enabled"`
}

// AdmonitionsConfig contains admonition block settings
type AdmonitionsConfig struct {
	Enabled bool `yaml:"enabled"`
}

// ScrollSpyConfig contains scroll spy settings
type ScrollSpyConfig struct {
	Enabled bool `yaml:"enabled"`
}

// CodeCopyConfig contains code copy button settings
type CodeCopyConfig struct {
	Enabled bool `yaml:"enabled"`
}

// SEOConfig contains SEO optimization settings
type SEOConfig struct {
	Enabled       bool          `yaml:"enabled"`
	TitleTemplate string        `yaml:"titleTemplate"`
	DefaultDesc   string        `yaml:"defaultDescription"`
	DefaultImage  string        `yaml:"defaultImage"`
	Twitter       TwitterConfig `yaml:"twitter"`
}

// TwitterConfig contains Twitter card settings
type TwitterConfig struct {
	Site    string `yaml:"site"`
	Creator string `yaml:"creator"`
}

// DevServerConfig contains dev server settings
type DevServerConfig struct {
	Port       int    `yaml:"port"`
	Host       string `yaml:"host"`
	LiveReload bool   `yaml:"liveReload"`
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
		Feeds: FeedsConfig{
			RSS: RSSConfig{
				Enabled:     true,
				Path:        "rss.xml",
				Items:       20,
				FullContent: false,
			},
			Atom: AtomConfig{
				Enabled: true,
				Path:    "atom.xml",
				Items:   20,
			},
		},
		Sitemap: SitemapConfig{
			Enabled: true,
		},
		Tags: TagsConfig{
			Enabled: true,
		},
		Archive: ArchiveConfig{
			Enabled: true,
		},
		Pagination: PaginationConfig{
			Enabled:  true,
			PageSize: 10,
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
