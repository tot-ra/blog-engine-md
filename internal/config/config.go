package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SiteConfig represents the complete site configuration
type SiteConfig struct {
	Site         Site                      `yaml:"site"`
	I18n         I18nConfig                `yaml:"i18n"`
	Author       Author                    `yaml:"author"`
	Build        Build                     `yaml:"build"`
	Navigation   Navigation                `yaml:"navigation"`
	Assets       AssetsConfig              `yaml:"assets"`
	Audio        AudioConfig               `yaml:"audio"`
	Feeds        FeedsConfig               `yaml:"feeds"`
	Sitemap      SitemapConfig             `yaml:"sitemap"`
	Tags         TagsConfig                `yaml:"tags"`
	Archive      ArchiveConfig             `yaml:"archive"`
	Pagination   PaginationConfig          `yaml:"pagination"`
	Advanced     AdvancedConfig            `yaml:"advanced"`
	SEO          SEOConfig                 `yaml:"seo"`
	Homepage     HomepageConfig            `yaml:"homepage"` // Homepage customization
	HomepageI18n map[string]HomepageConfig `yaml:"homepageI18n"`
}

// HomepageConfig contains homepage-specific settings
type HomepageConfig struct {
	Enabled     bool               `yaml:"enabled"`
	Hero        HeroConfig         `yaml:"hero"`
	Chat        HomepageChatConfig `yaml:"chat"`
	Projects    []ProjectConfig    `yaml:"projects"`
	SocialLinks []SocialLinkGroup  `yaml:"socialLinks"`
	CustomHTML  string             `yaml:"customHTML"` // Additional custom HTML/JS
}

// HeroConfig contains hero section settings
type HeroConfig struct {
	Enabled     bool        `yaml:"enabled"`
	Background  string      `yaml:"background"`  // Background image URL
	Title       string      `yaml:"title"`       // Override page title
	Subtitle    string      `yaml:"subtitle"`    // Subtitle/tagline
	Description string      `yaml:"description"` // Description text
	VideoEmbed  string      `yaml:"videoEmbed"`  // Optional iframe src for hero video
	CTAButtons  []CTAButton `yaml:"ctaButtons"`  // Call-to-action buttons
}

// HomepageChatConfig contains homepage chat widget settings
type HomepageChatConfig struct {
	Enabled          bool   `yaml:"enabled"`
	BaseURL          string `yaml:"baseUrl"`
	RecipientAgentID string `yaml:"recipientAgentId"`
	Title            string `yaml:"title"`
}

// CTAButton represents a call-to-action button
type CTAButton struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
	Icon  string `yaml:"icon"` // Optional icon class/name
}

// ProjectConfig represents a project showcase item
type ProjectConfig struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Image       string   `yaml:"image"`  // Project image URL
	URL         string   `yaml:"url"`    // Link to project
	GitHub      string   `yaml:"github"` // GitHub repo URL
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

// AudioConfig contains article text-to-speech generation settings.
type AudioConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Provider    string            `yaml:"provider"`    // elevenlabs, edge
	OutputDir   string            `yaml:"outputDir"`   // Path inside content dir to store generated audio
	RecentPosts int               `yaml:"recentPosts"` // Generate for N latest blog posts (<=0 means all)
	MaxChars    int               `yaml:"maxChars"`    // Character cap per article for synthesis request
	Edge        EdgeAudioConfig   `yaml:"edge"`
	ElevenLabs  ElevenLabsConfig  `yaml:"elevenlabs"`
	Voices      map[string]string `yaml:"voices"` // Language code to voice mapping, e.g. ru: ru-RU-SvetlanaNeural
}

// EdgeAudioConfig contains settings for edge-tts provider.
type EdgeAudioConfig struct {
	Binary string `yaml:"binary"` // edge-tts executable path
	Rate   string `yaml:"rate"`   // +0%
	Pitch  string `yaml:"pitch"`  // +0Hz
	Voice  string `yaml:"voice"`  // Fallback voice
}

// ElevenLabsConfig contains settings for ElevenLabs text-to-speech provider.
type ElevenLabsConfig struct {
	APIKeyEnv       string  `yaml:"apiKeyEnv"`       // Environment variable name with API key
	BaseURL         string  `yaml:"baseUrl"`         // API base URL
	ModelID         string  `yaml:"modelId"`         // e.g. eleven_multilingual_v2
	OutputFormat    string  `yaml:"outputFormat"`    // e.g. mp3_44100_128
	DefaultVoiceID  string  `yaml:"defaultVoiceId"`  // Fallback voice id
	Stability       float64 `yaml:"stability"`       // 0..1
	SimilarityBoost float64 `yaml:"similarityBoost"` // 0..1
	Style           float64 `yaml:"style"`           // 0..1
	SpeakerBoost    bool    `yaml:"speakerBoost"`    // Enhance speaker similarity
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

// I18nConfig contains multilingual site settings.
type I18nConfig struct {
	Default   string           `yaml:"default"`
	Languages []LanguageConfig `yaml:"languages"`
}

// LanguageConfig contains one available language option.
type LanguageConfig struct {
	Code  string `yaml:"code"`
	Label string `yaml:"label"`
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
		I18n: I18nConfig{
			Default: "en",
			Languages: []LanguageConfig{
				{Code: "en", Label: "English"},
				{Code: "ru", Label: "Русский"},
			},
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
		Audio: AudioConfig{
			Enabled:     false,
			Provider:    "elevenlabs",
			OutputDir:   "content/audio/posts",
			RecentPosts: 10,
			MaxChars:    12000,
			Edge: EdgeAudioConfig{
				Binary: "edge-tts",
				Rate:   "+0%",
				Pitch:  "+0Hz",
				Voice:  "en-US-EmmaMultilingualNeural",
			},
			ElevenLabs: ElevenLabsConfig{
				APIKeyEnv:       "ELEVENLABS_API_KEY",
				BaseURL:         "https://api.elevenlabs.io",
				ModelID:         "eleven_multilingual_v2",
				OutputFormat:    "mp3_44100_128",
				DefaultVoiceID:  "EXAVITQu4vr4xnSDxMaL",
				Stability:       0.45,
				SimilarityBoost: 0.75,
				Style:           0.2,
				SpeakerBoost:    true,
			},
			Voices: map[string]string{
				"ru": "EXAVITQu4vr4xnSDxMaL",
				"en": "EXAVITQu4vr4xnSDxMaL",
			},
		},
		SEO: SEOConfig{
			Enabled: true,
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
	if cfg.Audio.Provider == "" {
		cfg.Audio.Provider = "elevenlabs"
	}
	if cfg.Audio.OutputDir == "" {
		cfg.Audio.OutputDir = "content/audio/posts"
	}
	if cfg.Audio.RecentPosts == 0 {
		cfg.Audio.RecentPosts = 10
	}
	if cfg.Audio.MaxChars <= 0 {
		cfg.Audio.MaxChars = 12000
	}
	if cfg.Audio.Edge.Binary == "" {
		cfg.Audio.Edge.Binary = "edge-tts"
	}
	if cfg.Audio.Edge.Rate == "" {
		cfg.Audio.Edge.Rate = "+0%"
	}
	if cfg.Audio.Edge.Pitch == "" {
		cfg.Audio.Edge.Pitch = "+0Hz"
	}
	if cfg.Audio.Edge.Voice == "" {
		cfg.Audio.Edge.Voice = "en-US-EmmaMultilingualNeural"
	}
	if cfg.Audio.ElevenLabs.APIKeyEnv == "" {
		cfg.Audio.ElevenLabs.APIKeyEnv = "ELEVENLABS_API_KEY"
	}
	if cfg.Audio.ElevenLabs.BaseURL == "" {
		cfg.Audio.ElevenLabs.BaseURL = "https://api.elevenlabs.io"
	}
	if cfg.Audio.ElevenLabs.ModelID == "" {
		cfg.Audio.ElevenLabs.ModelID = "eleven_multilingual_v2"
	}
	if cfg.Audio.ElevenLabs.OutputFormat == "" {
		cfg.Audio.ElevenLabs.OutputFormat = "mp3_44100_128"
	}
	if cfg.Audio.ElevenLabs.DefaultVoiceID == "" {
		cfg.Audio.ElevenLabs.DefaultVoiceID = "EXAVITQu4vr4xnSDxMaL"
	}
	if cfg.Audio.ElevenLabs.Stability <= 0 {
		cfg.Audio.ElevenLabs.Stability = 0.45
	}
	if cfg.Audio.ElevenLabs.SimilarityBoost <= 0 {
		cfg.Audio.ElevenLabs.SimilarityBoost = 0.75
	}
	if cfg.Audio.ElevenLabs.Style < 0 {
		cfg.Audio.ElevenLabs.Style = 0
	}
	if cfg.Audio.Voices == nil {
		cfg.Audio.Voices = map[string]string{
			"ru": cfg.Audio.ElevenLabs.DefaultVoiceID,
			"en": cfg.Audio.ElevenLabs.DefaultVoiceID,
		}
	}
	if cfg.Site.Language == "" {
		cfg.Site.Language = "en"
	}
	if cfg.I18n.Default == "" {
		cfg.I18n.Default = cfg.Site.Language
	}
	if len(cfg.I18n.Languages) == 0 {
		cfg.I18n.Languages = []LanguageConfig{
			{Code: cfg.I18n.Default, Label: strings.ToUpper(cfg.I18n.Default)},
		}
	}
	langs := make([]LanguageConfig, 0, len(cfg.I18n.Languages))
	seen := make(map[string]struct{}, len(cfg.I18n.Languages))
	for _, l := range cfg.I18n.Languages {
		code := strings.ToLower(strings.TrimSpace(l.Code))
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		label := strings.TrimSpace(l.Label)
		if label == "" {
			label = strings.ToUpper(code)
		}
		langs = append(langs, LanguageConfig{
			Code:  code,
			Label: label,
		})
	}
	cfg.I18n.Languages = langs
	if len(cfg.I18n.Languages) == 0 {
		return fmt.Errorf("i18n.languages must contain at least one language")
	}
	cfg.I18n.Default = strings.ToLower(strings.TrimSpace(cfg.I18n.Default))
	if cfg.I18n.Default == "" {
		cfg.I18n.Default = cfg.I18n.Languages[0].Code
	}
	hasDefault := false
	for _, l := range cfg.I18n.Languages {
		if l.Code == cfg.I18n.Default {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		cfg.I18n.Languages = append([]LanguageConfig{{Code: cfg.I18n.Default, Label: strings.ToUpper(cfg.I18n.Default)}}, cfg.I18n.Languages...)
	}
	return nil
}
