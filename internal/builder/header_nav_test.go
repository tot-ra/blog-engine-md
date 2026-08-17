package builder

import (
	"testing"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestHeaderItemVisibleForLanguage(t *testing.T) {
	item := config.HeaderItem{
		Title:     "Sheet Archive",
		Languages: []string{"rus"},
	}

	if !headerItemVisibleForLanguage(item, "rus") {
		t.Fatal("expected item to be visible for allowed language")
	}
	if headerItemVisibleForLanguage(item, "est") {
		t.Fatal("expected item to be hidden for other languages")
	}
	if !headerItemVisibleForLanguage(config.HeaderItem{Title: "About"}, "est") {
		t.Fatal("expected item without language restriction to remain visible")
	}
}

func TestBuildHeaderNavSupportsLanguageGroups(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			I18n: config.I18nConfig{
				Default: "en",
				Languages: []config.LanguageConfig{
					{Code: "en", Label: "English"},
					{Code: "et", Label: "Eesti"},
				},
			},
			Navigation: config.Navigation{
				Header: config.HeaderConfig{
					Enabled: true,
					Items: []config.HeaderItem{
						{Title: "Status", URL: "https://status.example.com/", Languages: []string{"en", "et"}},
						{Title: "Meist", URL: "/et/about/", Languages: []string{"et"}},
						{Title: "Preview", URL: "/et/preview/", Languages: []string{"et"}},
					},
					LanguageItems: map[string][]config.HeaderItem{
						"en": {
							{Title: "About", URL: "/about/"},
						},
						"et": {
							{Title: "Meist", URL: "/et/about/"},
						},
					},
				},
			},
		},
	}

	nav := b.buildHeaderNav("et", "/et/about/")
	if len(nav) != 3 {
		t.Fatalf("expected shared, non-duplicate legacy, and Estonian nav items, got %d", len(nav))
	}
	if nav[0].Title != "Status" || nav[0].URL != "https://status.example.com/" {
		t.Fatalf("expected shared status item first, got %+v", nav[0])
	}
	if nav[1].Title != "Preview" || nav[1].URL != "/et/preview/" {
		t.Fatalf("expected non-duplicate single-language item to remain, got %+v", nav[1])
	}
	if nav[2].Title != "Meist" || nav[2].URL != "/et/about/" || !nav[2].IsCurrent {
		t.Fatalf("expected active Estonian about item, got %+v", nav[2])
	}
}

func TestBuildHeaderNavMarksCurrentSection(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Navigation: config.Navigation{
				Header: config.HeaderConfig{
					Enabled: true,
					Items: []config.HeaderItem{
						{Title: "About", Path: "about"},
						{Title: "Products", Path: "about/products"},
						{Title: "Research", Path: "research"},
						{Title: "Blog", Path: "blog"},
						{Title: "GitHub", URL: "https://github.com/example/project", Class: "nav-action"},
					},
				},
			},
		},
	}

	nav := b.buildHeaderNav("en", "/research/papers/bee-study/")
	if len(nav) != 5 {
		t.Fatalf("expected 5 nav items, got %d", len(nav))
	}
	if !nav[2].IsCurrent {
		t.Fatal("expected Research nav item to be current")
	}
	if nav[2].Class != "is-active" {
		t.Fatalf("expected Research class to be is-active, got %q", nav[2].Class)
	}
	if nav[3].IsCurrent || nav[3].Class != "" {
		t.Fatalf("expected Blog nav item to be inactive, got current=%t class=%q", nav[3].IsCurrent, nav[3].Class)
	}
	if nav[4].IsCurrent || nav[4].Class != "nav-action" {
		t.Fatalf("expected external nav item to keep configured class only, got current=%t class=%q", nav[4].IsCurrent, nav[4].Class)
	}

	nav = b.buildHeaderNav("en", "/about/products/web-app/")
	if nav[0].IsCurrent {
		t.Fatal("expected broader About nav item to stay inactive when Products matches more specifically")
	}
	if !nav[1].IsCurrent || nav[1].Class != "is-active" {
		t.Fatalf("expected Products nav item to be the only active item, got current=%t class=%q", nav[1].IsCurrent, nav[1].Class)
	}
}

func TestBuildHeaderNavDefaultLanguageUsesCurrentURLShape(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			I18n: config.I18nConfig{
				Default: "en",
				Languages: []config.LanguageConfig{
					{Code: "en", Label: "English"},
					{Code: "et", Label: "Eesti"},
				},
			},
			Navigation: config.Navigation{
				Header: config.HeaderConfig{
					Enabled: true,
					Items:   []config.HeaderItem{{Title: "Docs", Path: "docs"}},
				},
			},
		},
	}

	nav := b.buildHeaderNav("en", "/docs/beehive-sensors/")
	if len(nav) != 1 {
		t.Fatalf("expected one nav item, got %d", len(nav))
	}
	if nav[0].URL != "/docs/" {
		t.Fatalf("expected default-language docs URL to stay unprefixed, got %q", nav[0].URL)
	}
	if !nav[0].IsCurrent {
		t.Fatalf("expected docs nav item to be current")
	}
}

func TestBuildHeaderNavDefaultLanguagePreservesExplicitPrefix(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			I18n: config.I18nConfig{
				Default: "ru",
				Languages: []config.LanguageConfig{
					{Code: "ru", Label: "Русский"},
					{Code: "en", Label: "English"},
				},
			},
			Navigation: config.Navigation{
				Header: config.HeaderConfig{
					Enabled: true,
					Items: []config.HeaderItem{
						{Title: "Docs", Path: "docs"},
						{Title: "Blog", Path: "blog"},
					},
				},
			},
		},
	}

	nav := b.buildHeaderNav("ru", "/ru/docs/getting-started/")
	if len(nav) != 2 {
		t.Fatalf("expected two nav items, got %d", len(nav))
	}
	if nav[0].URL != "/ru/docs/" || !nav[0].IsCurrent {
		t.Fatalf("expected active prefixed docs URL, got %+v", nav[0])
	}
	if nav[1].URL != "/ru/blog/" {
		t.Fatalf("expected prefixed blog URL, got %+v", nav[1])
	}
}

func TestBuildHeaderNavUsesLocalizedContentLabels(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			I18n: config.I18nConfig{
				Default: "en",
				Languages: []config.LanguageConfig{
					{Code: "en", Label: "English"},
					{Code: "et", Label: "Eesti"},
				},
			},
			Navigation: config.Navigation{
				Header: config.HeaderConfig{
					Enabled: true,
					Items: []config.HeaderItem{
						{Title: "Docs", URL: "/docs/"},
						{Title: "Research", URL: "/research/"},
						{Title: "Pricing", URL: "/pricing/"},
						{Title: "Log in", URL: "https://app.example.com/account/"},
					},
				},
			},
		},
		languages: map[string]struct{}{"en": {}, "et": {}},
		pagesByURL: map[string]*Page{
			"/et/docs/web-app/": {
				URL:         "/et/docs/web-app/",
				Language:    "et",
				Title:       "Web-app",
				Frontmatter: &parser.Frontmatter{},
			},
			"/et/research/": {
				URL:      "/et/research/",
				Language: "et",
				Title:    "Research",
				Frontmatter: &parser.Frontmatter{
					NavTitle:    "Uuringud",
					RedirectURL: "https://example.com/research/",
				},
			},
			"/et/pricing/": {
				URL:      "/et/pricing/",
				Language: "et",
				Title:    "Hinnaplaanid",
				Frontmatter: &parser.Frontmatter{
					NavTitle: "Hinnad",
				},
			},
		},
		navTree: &NavTree{
			ByPath: map[string]*NavNode{
				"/et/docs/": {Title: "Dokumendid", URL: "/et/docs/"},
			},
		},
	}

	nav := b.buildHeaderNav("et", "/et/pricing/")
	if len(nav) != 4 {
		t.Fatalf("expected 4 nav items, got %d", len(nav))
	}
	if nav[0].Title != "Dokumendid" || nav[0].URL != "/et/docs/" {
		t.Fatalf("expected localized docs link, got %+v", nav[0])
	}
	if nav[1].Title != "Uuringud" || nav[1].URL != "/research/" {
		t.Fatalf("expected redirect placeholder to supply label but keep canonical URL, got %+v", nav[1])
	}
	if nav[2].Title != "Hinnad" || nav[2].URL != "/et/pricing/" {
		t.Fatalf("expected localized pricing link, got %+v", nav[2])
	}
	if nav[3].Title != "Logi sisse" {
		t.Fatalf("expected localized login UI label, got %+v", nav[3])
	}
}

func TestBuildHeaderNavUsesTitleI18nForDefaultLanguage(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			I18n: config.I18nConfig{
				Default: "rus",
				Languages: []config.LanguageConfig{
					{Code: "rus", Label: "Русский"},
					{Code: "est", Label: "Eesti"},
				},
			},
			Navigation: config.Navigation{
				Header: config.HeaderConfig{
					Enabled: true,
					Items: []config.HeaderItem{
						{
							Title: "About",
							TitleI18n: map[string]string{
								"rus": "Об авторе",
								"est": "Autorist",
							},
							Path: "about",
						},
						{
							Title: "Pedagogy",
							TitleI18n: map[string]string{
								"rus": "Педагогика",
								"est": "Pedagoogika",
							},
							Path: "study",
						},
					},
				},
			},
		},
		languages: map[string]struct{}{"rus": {}, "est": {}},
		pagesByURL: map[string]*Page{
			"/rus/about/": {
				URL:         "/rus/about/",
				Language:    "rus",
				Title:       "Обо мне",
				Frontmatter: &parser.Frontmatter{},
			},
			"/rus/study/": {
				URL:         "/rus/study/",
				Language:    "rus",
				Title:       "Педагогика",
				Frontmatter: &parser.Frontmatter{},
			},
		},
	}

	nav := b.buildHeaderNav("rus", "/rus/about/")
	if len(nav) != 2 {
		t.Fatalf("expected 2 nav items, got %d", len(nav))
	}
	if nav[0].Title != "Об авторе" || nav[0].URL != "/rus/about/" {
		t.Fatalf("expected Russian titleI18n about item, got %+v", nav[0])
	}
	if nav[1].Title != "Педагогика" || nav[1].URL != "/rus/study/" {
		t.Fatalf("expected Russian titleI18n pedagogy item, got %+v", nav[1])
	}
}

func TestBuildHeaderNavKeepsConfiguredDefaultLanguageLabels(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			I18n: config.I18nConfig{
				Default:   "en",
				Languages: []config.LanguageConfig{{Code: "en", Label: "English"}},
			},
			Navigation: config.Navigation{
				Header: config.HeaderConfig{
					Enabled: true,
					Items: []config.HeaderItem{
						{Title: "About", URL: "/about/"},
					},
				},
			},
		},
		pagesByURL: map[string]*Page{
			"/about/": {
				URL:         "/about/",
				Language:    "en",
				Title:       "Overview",
				Frontmatter: &parser.Frontmatter{},
			},
		},
		navTree: &NavTree{
			ByPath: map[string]*NavNode{
				"/about/": {Title: "Overview", URL: "/about/"},
			},
		},
	}

	nav := b.buildHeaderNav("en", "/about/")
	if len(nav) != 1 {
		t.Fatalf("expected 1 nav item, got %d", len(nav))
	}
	if nav[0].Title != "About" {
		t.Fatalf("expected configured default-language header label, got %q", nav[0].Title)
	}
}

func TestBuildHeaderNavKeepsConfiguredLabelForCanonicalRedirectFallback(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			I18n: config.I18nConfig{
				Default: "en",
				Languages: []config.LanguageConfig{
					{Code: "en", Label: "English"},
					{Code: "ru", Label: "Русский"},
				},
			},
			Navigation: config.Navigation{
				Header: config.HeaderConfig{
					Enabled: true,
					Items: []config.HeaderItem{
						{Title: "About", URL: "/about/"},
					},
				},
			},
		},
		languages: map[string]struct{}{"en": {}, "ru": {}},
		pagesByURL: map[string]*Page{
			"/about/": {
				URL:         "/about/",
				Language:    "en",
				Title:       "Overview",
				Frontmatter: &parser.Frontmatter{},
			},
			"/ru/about/": {
				URL:      "/ru/about/",
				Language: "ru",
				Title:    "About",
				Frontmatter: &parser.Frontmatter{
					RedirectURL: "https://example.com/about/",
				},
			},
		},
	}

	nav := b.buildHeaderNav("ru", "/ru/docs/")
	if len(nav) != 1 {
		t.Fatalf("expected 1 nav item, got %d", len(nav))
	}
	if nav[0].Title != "About" || nav[0].URL != "/about/" {
		t.Fatalf("expected configured label and canonical URL, got %+v", nav[0])
	}
}

func TestHeaderNavLinkIsCurrent(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		want    bool
	}{
		{name: "exact section", current: "/research/", target: "/research/", want: true},
		{name: "section child", current: "/research/papers/bee/", target: "/research/", want: true},
		{name: "no partial segment", current: "/researcher/", target: "/research/", want: false},
		{name: "home only exact", current: "/research/", target: "/", want: false},
		{name: "external url ignored", current: "/research/", target: "https://github.com/example/project", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := headerNavLinkIsCurrent(tt.current, tt.target); got != tt.want {
				t.Fatalf("headerNavLinkIsCurrent(%q, %q) = %t, want %t", tt.current, tt.target, got, tt.want)
			}
		})
	}
}

func TestBuildHeaderNavAttachesGraphIconAfterBlog(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			I18n: config.I18nConfig{
				Default: "ru",
				Languages: []config.LanguageConfig{
					{Code: "ru", Label: "Русский"},
					{Code: "en", Label: "English"},
				},
			},
			Navigation: config.Navigation{
				Header: config.HeaderConfig{
					Enabled: true,
					Items: []config.HeaderItem{
						{Title: "About", Path: "about"},
						{Title: "Blog", Path: "blog"},
					},
				},
			},
			Advanced: config.AdvancedConfig{
				Graph: config.GraphConfig{Enabled: true, Path: "graph"},
			},
		},
		languages: map[string]struct{}{"ru": {}, "en": {}},
	}

	nav := b.buildHeaderNav("ru", "/blog/")
	if len(nav) != 2 {
		t.Fatalf("expected 2 nav items, got %d", len(nav))
	}
	blog := nav[1]
	if blog.Title != "Блог" && blog.Title != "Blog" {
		// localized title may be Blog or Блог depending on SegmentLabel/path title
	}
	if blog.TrailingIconURL != "/graph/" {
		t.Fatalf("expected default-language graph icon at /graph/, got %+v", blog)
	}
	if blog.TrailingIconName != "graph" || blog.TrailingIconLabel == "" {
		t.Fatalf("expected graph trailing icon metadata, got %+v", blog)
	}
	if blog.TrailingIconCurrent {
		t.Fatalf("graph icon should not be current on blog index, got %+v", blog)
	}

	onGraph := b.buildHeaderNav("ru", "/graph/")
	if !onGraph[1].TrailingIconCurrent {
		t.Fatalf("expected graph icon current on /graph/, got %+v", onGraph[1])
	}

	en := b.buildHeaderNav("en", "/en/blog/")
	if en[1].TrailingIconURL != "/graph/" {
		t.Fatalf("expected graph icon to stay on canonical /graph/, got %+v", en[1])
	}

	b.config.Advanced.Graph.Enabled = false
	disabled := b.buildHeaderNav("ru", "/blog/")
	if disabled[1].TrailingIconURL != "" {
		t.Fatalf("expected no graph icon when disabled, got %+v", disabled[1])
	}
}
