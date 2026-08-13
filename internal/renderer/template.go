package renderer

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/i18n"
	"github.com/tot-ra/blog-engine/internal/parser"
)

// Page represents page data for templates
type Page struct {
	ID           string
	URL          string
	Language     string
	Direction    string
	Title        string
	Description  string
	Content      string
	AudioURL     string
	Type         string
	ModifiedTime time.Time
	Layout       string // Custom layout template name
}

// Frontmatter represents page frontmatter for templates
type Frontmatter struct {
	Date         time.Time
	Tags         []string
	TemplateHero bool
	Params       map[string]interface{}
}

// BreadcrumbItem represents a breadcrumb navigation entry for templates
type BreadcrumbItem struct {
	Title     string
	URL       string
	IsCurrent bool
}

// NavLink represents a navigation link for templates
type NavLink struct {
	Title     string
	URL       string
	Type      string
	Class     string
	IsCurrent bool
}

// HeaderSocialLink is an icon link rendered in the site header.
type HeaderSocialLink struct {
	Label string
	URL   string
	Icon  string
}

// PrevNextLinks holds previous and next navigation links for templates
type PrevNextLinks struct {
	Prev *NavLink
	Next *NavLink
}

// BlogShowcasePost represents one automatically selected homepage blog/event card.
type BlogShowcasePost struct {
	Title       string
	URL         string
	Description string
	ImageHTML   template.HTML
	Date        time.Time
}

// PageData holds data for template rendering
type PageData struct {
	Site            config.SiteConfig
	Homepage        config.HomepageConfig
	BlogShowcase    []BlogShowcasePost
	EventsShowcase  []BlogShowcasePost
	Page            Page
	HomeURL         string
	TagURL          func(string) string
	CanonicalURL    string
	SocialImageURL  string
	OpenGraphType   string
	SocialCard      string
	MetaDescription string
	MarkdownURL     string
	UI              i18n.UIStrings
	Frontmatter     Frontmatter
	Content         template.HTML
	CSSPath         string
	JSPath          string
	HeaderNav       []NavLink
	HeaderSocial    []HeaderSocialLink
	Languages       []LanguageOption
	// Navigation fields (Phase 2)
	Sidebar     template.HTML
	TOC         template.HTML
	Breadcrumbs []BreadcrumbItem
	PrevNext    *PrevNextLinks
}

// LanguageOption represents one language item in the language switcher.
type LanguageOption struct {
	Code   string
	Label  string
	URL    string
	Active bool
}

// TemplateEngine handles template loading and rendering
type TemplateEngine struct {
	templates *template.Template
	funcs     template.FuncMap
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() *TemplateEngine {
	funcs := template.FuncMap{
		"formatDate": func(t time.Time, layout string) string {
			if layout == "" {
				layout = "2006-01-02"
			}
			return t.Format(layout)
		},
		"formatDateLocalized": func(t time.Time, lang string) string {
			return i18n.FormatDateLong(t, lang)
		},
		"slugify": func(text string) string {
			return slugify(text)
		},
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"title":      strings.Title,
		"hasDate": func(t time.Time) bool {
			return !t.IsZero()
		},
		"split":    strings.Split,
		"contains": strings.Contains,
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	return &TemplateEngine{
		funcs: funcs,
	}
}

// LoadTemplates loads embedded defaults and overlays site-specific templates.
func (e *TemplateEngine) LoadTemplates(dir string) error {
	tmpl, err := e.loadDefaultTemplates()
	if err != nil {
		return err
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		e.templates = tmpl
		return nil
	}

	patterns := []string{
		filepath.Join(dir, "*.html"),
		filepath.Join(dir, "**", "*.html"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				return fmt.Errorf("failed to read template %s: %w", match, err)
			}
			name := strings.TrimSuffix(filepath.Base(match), ".html")
			_, err = tmpl.New(name).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", match, err)
			}
		}
	}

	e.templates = tmpl
	return nil
}

func (e *TemplateEngine) loadDefaultTemplates() (*template.Template, error) {
	return template.New("").Funcs(e.funcs).ParseFS(defaultTemplateFS, "default_templates/*.html")
}

// Render renders a template with data
func (e *TemplateEngine) Render(templateName string, data PageData) (string, error) {
	if e.templates == nil {
		return "", fmt.Errorf("templates not loaded")
	}

	var buf strings.Builder
	if err := e.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", templateName, err)
	}

	return buf.String(), nil
}

// RenderPage renders a page using the specified or default page template
func (e *TemplateEngine) RenderPage(data PageData) (string, error) {
	// Use custom layout if specified, otherwise use default "page" template
	templateName := data.Page.Layout
	if templateName == "" {
		templateName = "page"
	}
	return e.Render(templateName, data)
}

// slugify creates a URL-friendly slug
func slugify(text string) string {
	return parser.GenerateSlug(text)
}
