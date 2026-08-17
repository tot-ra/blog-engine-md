package config

// SiteConfig represents the complete site configuration
type SiteConfig struct {
	Site         Site                      `yaml:"site"`
	I18n         I18nConfig                `yaml:"i18n"`
	Author       Author                    `yaml:"author"`
	Build        Build                     `yaml:"build"`
	Navigation   Navigation                `yaml:"navigation"`
	Assets       AssetsConfig              `yaml:"assets"`
	Audio        AudioConfig               `yaml:"audio"`
	Related      RelatedConfig             `yaml:"related"`
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
	Enabled        bool               `yaml:"enabled"`
	Hero           HeroConfig         `yaml:"hero"`
	Chat           HomepageChatConfig `yaml:"chat"`
	BlogShowcase   BlogShowcaseConfig `yaml:"blogShowcase"`
	EventsShowcase BlogShowcaseConfig `yaml:"eventsShowcase"`
	HideProjects   bool               `yaml:"hideProjects"`
	Projects       []ProjectConfig    `yaml:"projects"`
	SocialLinks    []SocialLinkGroup  `yaml:"socialLinks"`
	CustomHTML     string             `yaml:"customHTML"` // Additional custom HTML/JS
}

// BlogShowcaseConfig controls the automatically generated latest-posts showcase.
// Reused for eventsShowcase (same enabled/limit/title knobs).
type BlogShowcaseConfig struct {
	Enabled bool   `yaml:"enabled"`
	Limit   int    `yaml:"limit"`
	Title   string `yaml:"title"`
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
	Enabled          bool           `yaml:"enabled"`
	Quality          int            `yaml:"quality"`
	Sizes            map[string]int `yaml:"sizes"`
	LazyLoading      bool           `yaml:"lazyLoading"`
	ParallelWorkers  int            `yaml:"parallelWorkers"`
	MaxSourcePixels  int64          `yaml:"maxSourcePixels"`
	MaxVariantPixels int64          `yaml:"maxVariantPixels"`
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

// RelatedConfig controls offline generation and later use of related-article embeddings.
type RelatedConfig struct {
	Enabled       bool     `yaml:"enabled"`
	Provider      string   `yaml:"provider"`
	Model         string   `yaml:"model"`
	Dimensions    int      `yaml:"dimensions"`
	APIKeyEnv     string   `yaml:"apiKeyEnv"`
	CachePath     string   `yaml:"cachePath"`
	Sections      []string `yaml:"sections"`
	Count         int      `yaml:"count"`
	MinScore      float64  `yaml:"minScore"`
	Diversity     float64  `yaml:"diversity"`
	CrossLanguage bool     `yaml:"crossLanguage"`
}

// Navigation contains navigation settings
type Navigation struct {
	Header      HeaderConfig      `yaml:"header"`
	Sidebar     SidebarConfig     `yaml:"sidebar"`
	TOC         TOCConfig         `yaml:"toc"`
	Breadcrumbs BreadcrumbsConfig `yaml:"breadcrumbs"`
	PrevNext    PrevNextConfig    `yaml:"prevNext"`
}

// HeaderConfig contains header navigation settings
type HeaderConfig struct {
	Enabled       bool                    `yaml:"enabled"`
	Items         []HeaderItem            `yaml:"items"`
	LanguageItems map[string][]HeaderItem `yaml:"languages"`
}

// HeaderItem contains one configurable header navigation item.
type HeaderItem struct {
	Title     string            `yaml:"title"`
	TitleI18n map[string]string `yaml:"titleI18n"`
	Languages []string          `yaml:"languages"`
	Path      string            `yaml:"path"`
	URL       string            `yaml:"url"`
	Class     string            `yaml:"class"`
}

// SidebarConfig contains sidebar settings
type SidebarConfig struct {
	Collapsed    bool                            `yaml:"collapsed"`
	MaxDepth     int                             `yaml:"maxDepth"`
	IncludeIndex bool                            `yaml:"includeIndex"`
	Sections     map[string]SidebarSectionConfig `yaml:"sections"`
	ExcludeRules []SidebarExcludeRule            `yaml:"excludeRules"`
}

// SidebarExcludeRule removes matching URLs from sidebar trees for pages that match configured paths.
type SidebarExcludeRule struct {
	MatchPaths   []string `yaml:"matchPaths"`
	ExcludePaths []string `yaml:"excludePaths"`
}

// SidebarSectionConfig contains per-section sidebar behavior settings.
type SidebarSectionConfig struct {
	DefaultMode string `yaml:"defaultMode"`
	EnableTime  bool   `yaml:"enableTime"`
	EnableGraph bool   `yaml:"enableGraph"`
	// EnableCategories toggles the folder/categories sidebar pane. Nil means enabled
	// (historical default). Set false for time-only sections such as events.
	EnableCategories *bool    `yaml:"enableCategories"`
	GraphPath        string   `yaml:"graphPath"`
	MatchPaths       []string `yaml:"matchPaths"`
	SidebarRoot      string   `yaml:"sidebarRoot"`
	// HideSidebar drops the left nav for matching pages when in-page links are enough
	// (e.g. about career/education cards) or the section is listed only in the header.
	HideSidebar      bool               `yaml:"hideSidebar"`
	ShowChildrenList *bool              `yaml:"showChildrenList"`
	RecentEmbeds     RecentEmbedsConfig `yaml:"recentEmbeds"`
}

// RecentEmbedsConfig controls a generic "latest embedded media" block for section index pages.
type RecentEmbedsConfig struct {
	Enabled   bool              `yaml:"enabled"`
	Provider  string            `yaml:"provider"`
	Limit     int               `yaml:"limit"`
	SortBy    string            `yaml:"sortBy"`
	Title     string            `yaml:"title"`
	TitleI18n map[string]string `yaml:"titleI18n"`
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
	Enabled          bool                             `yaml:"enabled"`
	SameCategoryOnly bool                             `yaml:"sameCategoryOnly"`
	Sections         map[string]PrevNextSectionConfig `yaml:"sections"`
}

// PrevNextSectionConfig contains path-scoped prev/next overrides.
// Pointer fields allow inheritance from broader/global settings.
type PrevNextSectionConfig struct {
	Enabled          *bool `yaml:"enabled"`
	SameCategoryOnly *bool `yaml:"sameCategoryOnly"`
}

// Site contains site-wide settings
type Site struct {
	Title    string `yaml:"title"`
	Tagline  string `yaml:"tagline"`
	URL      string `yaml:"url"`
	BaseURL  string `yaml:"baseUrl"`
	Language string `yaml:"language"`
	Logo     string `yaml:"logo"`
	Favicon  string `yaml:"favicon"`
}

// I18nConfig contains multilingual site settings.
type I18nConfig struct {
	Default         string                `yaml:"default"`
	BrowserRedirect BrowserRedirectConfig `yaml:"browserRedirect"`
	Languages       []LanguageConfig      `yaml:"languages"`
}

// LanguageConfig contains one available language option.
type LanguageConfig struct {
	Code      string   `yaml:"code"`
	Label     string   `yaml:"label"`
	Aliases   []string `yaml:"aliases"`
	Direction string   `yaml:"direction"`
}

// BrowserRedirectConfig controls language selection for the root URL.
type BrowserRedirectConfig struct {
	Enabled bool `yaml:"enabled"`
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
	PublishMarkdown bool   `yaml:"publishMarkdown"` // Publish source-backed pages as discoverable text/markdown alternatives
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
