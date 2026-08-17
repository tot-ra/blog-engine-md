package config

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
			Header: HeaderConfig{
				Enabled: true,
				Items: []HeaderItem{
					{Title: "Docs", Path: "docs"},
					{Title: "Blog", Path: "blog"},
				},
			},
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
				Sections:         map[string]PrevNextSectionConfig{},
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
				Enabled:          true,
				Quality:          85,
				Sizes:            map[string]int{"thumbnail": 150, "preview": 400, "full": 1200},
				LazyLoading:      true,
				ParallelWorkers:  0,
				MaxSourcePixels:  0,
				MaxVariantPixels: 0,
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
		Related: RelatedConfig{
			Enabled:       true,
			Provider:      "openai",
			Model:         "text-embedding-3-small",
			Dimensions:    512,
			APIKeyEnv:     "OPENAI_API_KEY",
			CachePath:     "content/embeddings.json",
			Sections:      []string{"blog"},
			Count:         4,
			MinScore:      0.3,
			Diversity:     0.7,
			CrossLanguage: false,
		},
		SEO: SEOConfig{
			Enabled: true,
		},
	}
}
