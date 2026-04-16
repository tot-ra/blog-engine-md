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
	Date time.Time
	Tags []string
}

// BreadcrumbItem represents a breadcrumb navigation entry for templates
type BreadcrumbItem struct {
	Title     string
	URL       string
	IsCurrent bool
}

// NavLink represents a navigation link for templates
type NavLink struct {
	Title string
	URL   string
	Type  string
}

// PrevNextLinks holds previous and next navigation links for templates
type PrevNextLinks struct {
	Prev *NavLink
	Next *NavLink
}

// PageData holds data for template rendering
type PageData struct {
	Site            config.SiteConfig
	Homepage        config.HomepageConfig
	Page            Page
	CanonicalURL    string
	SocialImageURL  string
	OpenGraphType   string
	SocialCard      string
	MetaDescription string
	UI              i18n.UIStrings
	Frontmatter     Frontmatter
	Content         template.HTML
	CSSPath         string
	JSPath          string
	HeaderNav       []NavLink
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
	}

	return &TemplateEngine{
		funcs: funcs,
	}
}

// LoadTemplates loads templates from a directory
func (e *TemplateEngine) LoadTemplates(dir string) error {
	// Check if templates directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Use embedded default templates
		e.templates = template.Must(template.New("").Funcs(e.funcs).Parse(defaultTemplates))
		return nil
	}

	// Load from directory
	patterns := []string{
		filepath.Join(dir, "*.html"),
		filepath.Join(dir, "**", "*.html"),
	}

	tmpl := template.New("").Funcs(e.funcs)
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

	// Check if any templates were loaded
	templatesLoaded := false
	for range tmpl.Templates() {
		templatesLoaded = true
		break
	}
	if !templatesLoaded {
		// Use default templates
		tmpl = template.Must(template.New("").Funcs(e.funcs).Parse(defaultTemplates))
	}

	e.templates = tmpl
	return nil
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

// defaultTemplates contains built-in templates
const defaultTemplates = `{{define "base"}}
<!DOCTYPE html>
<html lang="{{.Site.Site.Language}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Page.Title}}{{if .Site.Site.Title}} | {{.Site.Site.Title}}{{end}}</title>
    {{if .MetaDescription}}
    <meta name="description" content="{{.MetaDescription}}">
    {{end}}
    {{if .Site.SEO.Enabled}}
    {{if .CanonicalURL}}<link rel="canonical" href="{{.CanonicalURL}}">{{end}}
    <meta property="og:title" content="{{.Page.Title}}">
    {{if .MetaDescription}}<meta property="og:description" content="{{.MetaDescription}}">{{end}}
    <meta property="og:type" content="{{.OpenGraphType}}">
    {{if .CanonicalURL}}<meta property="og:url" content="{{.CanonicalURL}}">{{end}}
    {{if .SocialImageURL}}<meta property="og:image" content="{{.SocialImageURL}}">{{end}}
    {{if .Site.Site.Title}}<meta property="og:site_name" content="{{.Site.Site.Title}}">{{end}}
    <meta name="twitter:card" content="{{.SocialCard}}">
    <meta name="twitter:title" content="{{.Page.Title}}">
    {{if .MetaDescription}}<meta name="twitter:description" content="{{.MetaDescription}}">{{end}}
    {{if .SocialImageURL}}<meta name="twitter:image" content="{{.SocialImageURL}}">{{end}}
    {{if .Site.SEO.Twitter.Site}}<meta name="twitter:site" content="{{.Site.SEO.Twitter.Site}}">{{end}}
    {{if .Site.SEO.Twitter.Creator}}<meta name="twitter:creator" content="{{.Site.SEO.Twitter.Creator}}">{{end}}
    {{end}}
    {{if .Site.Site.Favicon}}<link rel="icon" href="{{.Site.Site.Favicon}}">{{end}}
    <style>
        :root {
            --sidebar-width: 280px;
            --toc-width: 240px;
            --nav-bg: #f8f9fa;
            --nav-border: #e0e0e0;
            --nav-active: #0066cc;
            --bg-primary: #ffffff;
            --text-primary: #333333;
            --text-secondary: #666666;
            --quote-bg: #f5f9ff;
            --quote-border: #8cb8ee;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            line-height: 1.6;
            color: var(--text-primary);
            background: var(--bg-primary);
        }
        /* Header */
        .site-header {
            border-bottom: 1px solid var(--nav-border);
            padding: 15px 20px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .header-left {
            display: flex;
            align-items: center;
            gap: 14px;
        }
        .navbar-logo {
            display: inline-flex;
            align-items: center;
        }
        .navbar-logo img {
            width: 40px;
            height: 40px;
            border-radius: 20px;
            object-fit: cover;
        }
        .site-nav {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .header-right {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .lang-switcher {
            display: inline-flex;
            gap: 4px;
            padding: 3px;
            border: 1px solid var(--nav-border);
            border-radius: 999px;
            background: var(--nav-bg);
        }
        .lang-link {
            border: 0;
            border-radius: 999px;
            padding: 6px 11px;
            font-size: 0.8em;
            color: var(--text-secondary);
            text-decoration: none;
            font-weight: 600;
        }
        .lang-link.is-active {
            color: #fff;
            background: var(--nav-active);
        }
        .site-nav a {
            color: var(--text-primary);
            text-decoration: none;
            font-weight: 600;
        }
        .site-nav a:hover { color: var(--nav-active); }
        .theme-toggle {
            border: 1px solid var(--nav-border);
            background: transparent;
            width: 36px;
            height: 36px;
            border-radius: 18px;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            color: var(--text-primary);
        }
        .theme-toggle svg {
            width: 18px;
            height: 18px;
            display: block;
        }
        .theme-toggle .icon-moon { display: none; }
        html[data-theme='dark'] .theme-toggle .icon-sun { display: none; }
        html[data-theme='dark'] .theme-toggle .icon-moon { display: block; }
        html[data-theme='dark'] {
            --nav-bg: #1d1f22;
            --nav-border: #383c42;
            --nav-active: #6aa4ff;
            --bg-primary: #111315;
            --text-primary: #eceef1;
            --text-secondary: #a9adb5;
            --quote-bg: #18212d;
            --quote-border: #4b7ebd;
        }
        /* Layout */
        .site-layout {
            display: flex;
            width: 100%;
            max-width: none;
            margin: 0;
            min-height: calc(100vh - 120px);
        }
        /* Sidebar */
        .sidebar {
            width: var(--sidebar-width);
            padding: 20px 16px;
            border-right: 1px solid var(--nav-border);
            background: var(--nav-bg);
            flex-shrink: 0;
            overflow-y: auto;
            max-height: calc(100vh - 60px);
            position: sticky;
            top: 0;
        }
        .blog-sidebar {
            display: flex;
            flex-direction: column;
            gap: 12px;
            overflow: hidden;
        }
        .sidebar-mode-switch {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 4px;
            border: 1px solid var(--nav-border);
            background: var(--bg-primary);
            border-radius: 999px;
            padding: 4px;
        }
        .sidebar-mode-btn {
            border: 0;
            background: transparent;
            color: var(--text-secondary);
            border-radius: 999px;
            font-size: 0.78em;
            font-weight: 600;
            padding: 7px 6px;
            cursor: pointer;
        }
        .sidebar-mode-btn.is-active {
            color: #fff;
            background: var(--nav-active);
        }
        .sidebar-mode-pane {
            flex: 1;
            min-height: 0;
            overflow-y: auto;
        }
        .sidebar-timeline {
            display: flex;
            flex-direction: column;
            gap: 10px;
            max-height: none;
            overflow: visible;
            padding-right: 2px;
        }
        .timeline-year h4 {
            font-size: 0.86em;
            color: var(--text-secondary);
            margin: 4px 0;
        }
        .timeline-list {
            list-style: none;
            margin: 0;
            padding: 0;
            margin-bottom: 8px;
        }
        .timeline-list li a {
            display: block;
            padding: 4px 8px;
            border-radius: 4px;
            color: var(--text-primary);
            text-decoration: none;
            font-size: 0.86em;
        }
        .timeline-list li a time {
            color: var(--text-secondary);
            margin-right: 6px;
            font-size: 0.9em;
        }
        .timeline-list li.active > a {
            color: var(--nav-active);
            font-weight: 600;
            background: rgba(0, 102, 204, 0.08);
        }
        .sidebar-graph-pane {
            height: calc(100vh - 130px);
        }
        .sidebar-graph-frame {
            width: 100%;
            height: 100%;
            border: 1px solid var(--nav-border);
            border-radius: 8px;
            background: var(--bg-primary);
        }
        .site-layout.sidebar-graph-mode .sidebar {
            width: 40%;
        }
        .site-layout.sidebar-graph-mode .site-content {
            max-width: none;
        }
        .sidebar-menu, .sidebar-submenu {
            list-style: none;
            padding: 0;
            margin: 0;
        }
        .sidebar-submenu {
            padding-left: 16px;
            border-left: 1px solid var(--nav-border);
            margin-left: 8px;
        }
        .sidebar-item a {
            display: block;
            padding: 4px 8px;
            color: var(--text-primary);
            text-decoration: none;
            border-radius: 4px;
            font-size: 0.9em;
        }
        .sidebar-section-head {
            display: flex;
            align-items: center;
            gap: 4px;
        }
        .sidebar-section-head a {
            flex: 1;
        }
        .sidebar-toggle {
            appearance: none;
            border: 0;
            background: transparent;
            color: var(--text-secondary);
            width: 20px;
            height: 20px;
            cursor: pointer;
            padding: 0;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            border-radius: 4px;
        }
        .sidebar-toggle:hover {
            background: rgba(0, 102, 204, 0.08);
            color: var(--nav-active);
        }
        .sidebar-toggle::before {
            content: "▸";
            font-size: 0.8em;
            line-height: 1;
            transition: transform 0.15s ease;
        }
        .sidebar-section.expanded > .sidebar-section-head .sidebar-toggle::before {
            transform: rotate(90deg);
        }
        .sidebar-section > .sidebar-submenu {
            display: none;
        }
        .sidebar-section.expanded > .sidebar-submenu {
            display: block;
        }
        .sidebar-item a:hover {
            background: rgba(0, 102, 204, 0.08);
        }
        .sidebar-item.active > a {
            color: var(--nav-active);
            font-weight: 600;
            background: rgba(0, 102, 204, 0.08);
        }
        .sidebar-section > a {
            font-weight: 600;
            text-transform: uppercase;
            font-size: 0.8em;
            letter-spacing: 0.05em;
            color: var(--text-secondary);
            margin-top: 12px;
        }
        /* Main content */
        .site-content {
            flex: 1;
            min-width: 0;
            padding: 24px 40px;
            max-width: 800px;
        }
        .embed {
            width: 100%;
            max-width: 800px;
            margin: 24px auto;
        }
        .embed iframe {
            display: block;
            width: min(100%, 800px);
            aspect-ratio: 16 / 9;
            height: auto;
            border: 0;
            border-radius: 12px;
        }
        /* Breadcrumbs */
        .breadcrumbs {
            list-style: none;
            display: flex;
            flex-wrap: wrap;
            gap: 4px;
            padding: 0;
            margin-bottom: 20px;
            font-size: 0.85em;
            color: var(--text-secondary);
        }
        .breadcrumbs li:not(:last-child)::after {
            content: "›";
            margin-left: 4px;
            color: var(--text-secondary);
        }
        .breadcrumbs a { color: var(--nav-active); text-decoration: none; }
        .breadcrumbs a:hover { text-decoration: underline; }
        /* TOC */
        .toc {
            width: var(--toc-width);
            padding: 20px 16px;
            flex-shrink: 0;
            position: sticky;
            top: 0;
            max-height: 100vh;
            overflow-y: auto;
            font-size: 0.85em;
        }
        .toc-title {
            font-size: 0.8em;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--text-secondary);
            margin-bottom: 8px;
            font-weight: 600;
        }
        .toc-list, .toc-sublist {
            list-style: none;
            padding: 0;
            margin: 0;
        }
        .toc-sublist { padding-left: 12px; }
        .toc-item a {
            display: block;
            padding: 3px 0;
            color: var(--text-secondary);
            text-decoration: none;
            border-left: 2px solid transparent;
            padding-left: 8px;
        }
        .toc-item a:hover {
            color: var(--nav-active);
            border-left-color: var(--nav-active);
        }
        /* Prev/Next */
        .prev-next {
            display: flex;
            justify-content: space-between;
            margin-top: 48px;
            padding-top: 24px;
            border-top: 1px solid var(--nav-border);
            gap: 20px;
        }
        .prev-next a {
            display: flex;
            flex-direction: column;
            text-decoration: none;
            padding: 12px 16px;
            border: 1px solid var(--nav-border);
            border-radius: 8px;
            flex: 1;
            transition: border-color 0.2s;
        }
        .prev-next a:hover { border-color: var(--nav-active); }
        .prev-next .prev-link { text-align: left; }
        .prev-next .next-link { text-align: right; margin-left: auto; }
        .prev-label, .next-label {
            font-size: 0.8em;
            color: var(--text-secondary);
            text-transform: uppercase;
        }
        .prev-title, .next-title {
            color: var(--nav-active);
            font-weight: 500;
        }
        /* Article */
        article h1 { font-size: 2em; margin-bottom: 8px; }
        article time { color: var(--text-secondary); font-size: 0.9em; }
        .article-meta {
            display: flex;
            align-items: center;
            justify-content: flex-start;
            gap: 4px;
            margin: 0 0 8px;
            flex-wrap: nowrap;
        }
        .article-audio {
            display: flex;
            align-items: center;
            gap: 8px;
            margin: 0 0 0 2px;
        }
        .article-audio button {
            border: 1px solid var(--nav-active);
            border-radius: 999px;
            background: var(--nav-active);
            color: #fff;
            width: 36px;
            height: 36px;
            padding: 0;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
        }
        .article-audio button:hover {
            filter: brightness(0.92);
        }
        .article-audio button svg {
            width: 14px;
            height: 14px;
            fill: currentColor;
        }
        .article-audio button .icon-stop { display: none; }
        .article-audio button.is-playing .icon-play { display: none; }
        .article-audio button.is-playing .icon-stop { display: block; }
        .article-audio .audio-wave {
            width: 88px;
            height: 28px;
            border: 0;
            border-radius: 0;
            background: transparent;
            opacity: 0.95;
        }
        .article-audio button:disabled {
            opacity: 0.55;
            cursor: default;
        }
        article .tags { margin: 22px 0 24px; }
        article .tag {
            display: inline-block;
            background: var(--nav-bg);
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 0.85em;
            color: var(--nav-active);
            margin-right: 4px;
            text-decoration: none;
        }
        article .tag:hover { text-decoration: underline; }
        article .content { margin-top: 16px; }
        article .content h2 { margin-top: 2em; margin-bottom: 0.5em; }
        article .content h3 { margin-top: 1.5em; margin-bottom: 0.4em; }
        article .content p { margin-bottom: 1em; }
        article .content blockquote {
            margin: 1.4em 0;
            padding: 0.8em 1em;
            border-left: 4px solid var(--quote-border);
            background: var(--quote-bg);
            color: var(--text-secondary);
            font-style: italic;
        }
        article .content blockquote p:last-child {
            margin-bottom: 0;
        }
        /* Footer */
        .site-footer {
            border-top: 1px solid var(--nav-border);
            padding: 20px;
            color: var(--text-secondary);
            font-size: 0.9em;
            text-align: center;
        }
        /* Standard elements */
        a { color: var(--nav-active); text-decoration: none; }
        a:hover { text-decoration: underline; }
        img { max-width: 100%; height: auto; }
        pre {
            background: #f4f4f4;
            padding: 15px;
            overflow-x: auto;
            border-radius: 4px;
        }
        code {
            background: #f4f4f4;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: "Monaco", "Menlo", monospace;
        }
        pre code { padding: 0; background: none; }
        table { border-collapse: collapse; width: 100%; margin: 1em 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background: #f4f4f4; }
        .section-index { list-style: none; padding: 0; }
        .section-index li { padding: 6px 0; border-bottom: 1px solid var(--nav-border); }
        .section-index li a { font-weight: 500; }
        .section-article-list {
            display: flex;
            flex-direction: column;
            gap: 14px;
        }
        .section-article-preview {
            border-bottom: 1px solid var(--nav-border);
            padding-bottom: 12px;
        }
        .section-article-preview h2 {
            margin: 0 0 4px;
            font-size: 1.05em;
        }
        .section-article-preview time {
            display: block;
            color: var(--text-secondary);
            font-size: 0.83em;
            margin-bottom: 6px;
        }
        .section-article-preview p {
            margin: 0;
            color: var(--text-secondary);
        }
        .projects-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 24px;
            margin: 32px 0 0;
        }
        .project-card {
            background: var(--bg-secondary);
            border-radius: 12px;
            overflow: hidden;
            border: 1px solid var(--border);
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .project-card:hover {
            transform: translateY(-4px);
            box-shadow: 0 8px 24px var(--shadow);
        }
        .project-image {
            width: 100%;
            height: 180px;
            object-fit: cover;
            background: var(--border);
            display: block;
        }
        .project-content {
            padding: 20px;
        }
        .project-content h3 {
            margin: 0 0 8px;
            font-size: 1.25em;
        }
        .project-content p {
            margin: 0 0 16px;
            color: var(--text-secondary);
            font-size: 0.95em;
        }
        .project-tags {
            display: flex;
            gap: 8px;
            flex-wrap: wrap;
        }
        .project-tag {
            font-size: 0.8em;
            padding: 4px 10px;
            background: var(--bg-primary);
            border-radius: 20px;
            color: var(--text-secondary);
        }
        /* Responsive */
        @media (max-width: 1200px) {
            .toc { display: none; }
        }
        @media (max-width: 768px) {
            .sidebar { display: none; }
            .site-content { padding: 16px; }
            .projects-grid { grid-template-columns: 1fr; }
        }
    </style>
    {{if .CSSPath}}<link rel="stylesheet" href="{{.CSSPath}}">{{end}}
</head>
<body class="{{if .Page.Layout}}page-layout-{{.Page.Layout}}{{else}}page-layout-page{{end}}">
    <header class="site-header">
        <div class="header-left">
            <a class="navbar-logo" href="/{{.Page.Language}}/">
                <img src="{{.Site.Site.Logo}}" alt="{{.Site.Site.Title}}">
            </a>
            <nav class="site-nav">
                {{range .HeaderNav}}<a href="{{.URL}}">{{.Title}}</a>{{end}}
            </nav>
        </div>
        <div class="header-right">
            <nav class="lang-switcher" aria-label="{{.UI.Languages}}">
                {{range .Languages}}<a class="lang-link{{if .Active}} is-active{{end}}" href="{{.URL}}" hreflang="{{.Code}}">{{.Label}}</a>{{end}}
            </nav>
            <button id="theme-toggle" class="theme-toggle" aria-label="{{.UI.ToggleTheme}}" title="{{.UI.ToggleTheme}}">
                <svg class="icon-sun" viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="4"></circle>
                    <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"></path>
                </svg>
                <svg class="icon-moon" viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 12.79A9 9 0 1 1 11.21 3c0 0 0 0 0 0A7 7 0 0 0 21 12.79z"></path>
                </svg>
            </button>
        </div>
    </header>
    <div class="site-layout">
        {{if .Sidebar}}{{.Sidebar}}{{end}}
        <main class="site-content">
            {{template "content" .}}
        </main>
        {{if .TOC}}{{.TOC}}{{end}}
    </div>
    <script>
    (function() {
        var players = document.querySelectorAll("[data-audio-player]");
        players.forEach(function(container) {
            var audio = container.querySelector("[data-audio-element]");
            var toggleBtn = container.querySelector("[data-audio-toggle]");
            var wave = container.querySelector("[data-audio-wave]");
            if (!audio || !toggleBtn || !wave) {
                return;
            }
            var audioCtx = null;
            var sourceNode = null;
            var analyser = null;
            var waveData = null;
            var rafId = 0;

            function syncToggleState() {
                var isPlaying = !audio.paused && !audio.ended;
                toggleBtn.classList.toggle("is-playing", isPlaying);
                var label = isPlaying ? (toggleBtn.dataset.stopLabel || "Stop") : (toggleBtn.dataset.playLabel || "Listen");
                toggleBtn.setAttribute("aria-label", label);
                toggleBtn.setAttribute("title", label);
            }

            function ensureVisualizer() {
                if (analyser) {
                    return;
                }
                var AudioCtx = window.AudioContext || window.webkitAudioContext;
                if (!AudioCtx) {
                    return;
                }
                audioCtx = new AudioCtx();
                sourceNode = audioCtx.createMediaElementSource(audio);
                analyser = audioCtx.createAnalyser();
                analyser.fftSize = 1024;
                waveData = new Uint8Array(analyser.fftSize);
                sourceNode.connect(analyser);
                analyser.connect(audioCtx.destination);
            }

            function stopWave() {
                if (rafId) {
                    cancelAnimationFrame(rafId);
                    rafId = 0;
                }
            }

            function drawWave() {
                if (!analyser || !wave) {
                    return;
                }
                var dpr = window.devicePixelRatio || 1;
                var w = Math.max(88, wave.clientWidth || 88);
                var h = Math.max(28, wave.clientHeight || 28);
                if (wave.width !== Math.floor(w * dpr) || wave.height !== Math.floor(h * dpr)) {
                    wave.width = Math.floor(w * dpr);
                    wave.height = Math.floor(h * dpr);
                }
                var ctx = wave.getContext("2d");
                if (!ctx) {
                    return;
                }
                ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
                ctx.clearRect(0, 0, w, h);
                ctx.strokeStyle = "#0066cc";
                ctx.lineWidth = 1.2;

                analyser.getByteTimeDomainData(waveData);
                ctx.beginPath();
                var slice = waveData.length / w;
                for (var x = 0; x < w; x++) {
                    var idx = Math.floor(x * slice);
                    var y = ((waveData[idx] - 128) / 128) * (h * 0.36) + (h / 2);
                    if (x === 0) {
                        ctx.moveTo(x, y);
                    } else {
                        ctx.lineTo(x, y);
                    }
                }
                ctx.stroke();

                if (!audio.paused && !audio.ended) {
                    rafId = requestAnimationFrame(drawWave);
                } else {
                    stopWave();
                }
            }

            toggleBtn.addEventListener("click", function() {
                if (!audio.paused && !audio.ended) {
                    audio.pause();
                    audio.currentTime = 0;
                    syncToggleState();
                    stopWave();
                    return;
                }
                ensureVisualizer();
                if (audioCtx && audioCtx.state === "suspended") {
                    audioCtx.resume().catch(function() {});
                }
                var started = audio.play();
                if (started && typeof started.then === "function") {
                    started.then(function() {
                        syncToggleState();
                        drawWave();
                    }).catch(function() {});
                    return;
                }
                syncToggleState();
                drawWave();
            });
            audio.addEventListener("pause", syncToggleState);
            audio.addEventListener("play", syncToggleState);
            audio.addEventListener("ended", function() {
                syncToggleState();
                stopWave();
            });

            syncToggleState();
        });
    })();
    </script>
    {{if .JSPath}}<script defer src="{{.JSPath}}"></script>{{end}}
</body>
</html>
{{end}}

{{define "page"}}
{{template "base" .}}
{{end}}

{{define "projects"}}
{{template "base" .}}
{{end}}

{{define "content"}}
<article>
    {{if .Breadcrumbs}}
    <nav aria-label="{{.UI.Breadcrumb}}">
        <ol class="breadcrumbs">
            {{range .Breadcrumbs}}
            <li>
                {{if .IsCurrent}}
                    <span aria-current="page">{{.Title}}</span>
                {{else}}
                    <a href="{{.URL}}">{{.Title}}</a>
                {{end}}
            </li>
            {{end}}
        </ol>
    </nav>
    {{end}}
    <h1>{{.Page.Title}}</h1>
    {{if or (hasDate .Frontmatter.Date) .Page.AudioURL}}
    <div class="article-meta">
        {{if hasDate .Frontmatter.Date}}
        <time datetime="{{.Frontmatter.Date.Format "2006-01-02T15:04:05Z07:00"}}">
            {{formatDateLocalized .Frontmatter.Date .Page.Language}}
        </time>
        {{else}}<span></span>{{end}}
        {{if .Page.AudioURL}}
        <div class="article-audio" data-audio-player>
            <audio preload="metadata" src="{{.Page.AudioURL}}" data-audio-element></audio>
            <button type="button" data-audio-toggle data-play-label="{{.UI.Listen}}" data-stop-label="{{.UI.Stop}}" aria-label="{{.UI.Listen}}" title="{{.UI.Listen}}">
                <svg class="icon-play" viewBox="0 0 24 24" aria-hidden="true">
                    <polygon points="8,5 19,12 8,19"></polygon>
                </svg>
                <svg class="icon-stop" viewBox="0 0 24 24" aria-hidden="true">
                    <rect x="7" y="7" width="10" height="10"></rect>
                </svg>
            </button>
            <canvas class="audio-wave" data-audio-wave></canvas>
        </div>
        {{end}}
    </div>
    {{end}}
    <div class="content">
        {{.Content}}
    </div>
    {{if and (eq .Page.Layout "projects") .Homepage.Projects}}
    <section class="projects-grid" aria-label="{{.UI.Projects}}">
        {{range .Homepage.Projects}}
        <div class="project-card">
            {{if .Image}}<img src="{{.Image}}" alt="{{.Title}}" class="project-image">{{end}}
            <div class="project-content">
                <h3>
                    {{if .URL}}
                    <a href="{{.URL}}">{{.Title}}</a>
                    {{else if .GitHub}}
                    <a href="{{.GitHub}}">{{.Title}}</a>
                    {{else}}
                    {{.Title}}
                    {{end}}
                </h3>
                <p>{{.Description}}</p>
                {{if .Tags}}
                <div class="project-tags">
                    {{range .Tags}}<span class="project-tag">{{.}}</span>{{end}}
                </div>
                {{end}}
            </div>
        </div>
        {{end}}
    </section>
    {{end}}
    {{if .Frontmatter.Tags}}
    <div class="tags">
        {{range .Frontmatter.Tags}}
        <a class="tag" href="/{{$.Page.Language}}/tags/{{slugify .}}/">#{{.}}</a>
        {{end}}
    </div>
    {{end}}
    {{if .PrevNext}}
    <nav class="prev-next" aria-label="{{.UI.PageNavigation}}">
        {{if .PrevNext.Prev}}
        <a href="{{.PrevNext.Prev.URL}}" class="prev-link">
            <span class="prev-label">← {{.UI.Previous}}</span>
            <span class="prev-title">{{.PrevNext.Prev.Title}}</span>
        </a>
        {{end}}
        {{if .PrevNext.Next}}
        <a href="{{.PrevNext.Next.URL}}" class="next-link">
            <span class="next-label">{{.UI.Next}} →</span>
            <span class="next-title">{{.PrevNext.Next.Title}}</span>
        </a>
        {{end}}
    </nav>
    {{end}}
</article>
{{end}}

{{define "homepage"}}
<!DOCTYPE html>
<html lang="{{.Site.Site.Language}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Page.Title}}{{if .Site.Site.Title}} — {{.Site.Site.Title}}{{end}}</title>
    {{if .MetaDescription}}<meta name="description" content="{{.MetaDescription}}">{{end}}
    {{if .Site.SEO.Enabled}}
    {{if .CanonicalURL}}<link rel="canonical" href="{{.CanonicalURL}}">{{end}}
    <meta property="og:title" content="{{.Page.Title}}">
    {{if .MetaDescription}}<meta property="og:description" content="{{.MetaDescription}}">{{end}}
    <meta property="og:type" content="{{.OpenGraphType}}">
    {{if .CanonicalURL}}<meta property="og:url" content="{{.CanonicalURL}}">{{end}}
    {{if .SocialImageURL}}<meta property="og:image" content="{{.SocialImageURL}}">{{end}}
    {{if .Site.Site.Title}}<meta property="og:site_name" content="{{.Site.Site.Title}}">{{end}}
    <meta name="twitter:card" content="{{.SocialCard}}">
    <meta name="twitter:title" content="{{.Page.Title}}">
    {{if .MetaDescription}}<meta name="twitter:description" content="{{.MetaDescription}}">{{end}}
    {{if .SocialImageURL}}<meta name="twitter:image" content="{{.SocialImageURL}}">{{end}}
    {{if .Site.SEO.Twitter.Site}}<meta name="twitter:site" content="{{.Site.SEO.Twitter.Site}}">{{end}}
    {{if .Site.SEO.Twitter.Creator}}<meta name="twitter:creator" content="{{.Site.SEO.Twitter.Creator}}">{{end}}
    {{end}}
    {{if .Site.Site.Favicon}}<link rel="icon" href="{{.Site.Site.Favicon}}">{{end}}
    <style>
        :root {
            --bg-primary: #ffffff;
            --bg-secondary: #f8f9fa;
            --text-primary: #1a1a1a;
            --text-secondary: #666666;
            --accent: #0066cc;
            --accent-hover: #0052a3;
            --border: #e0e0e0;
            --shadow: rgba(0,0,0,0.1);
        }
        * { box-sizing: border-box; }
        body {
            margin: 0;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.6;
        }
        .site-header {
            border-bottom: 1px solid var(--border);
            padding: 15px 20px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            background: var(--bg-primary);
        }
        .header-left {
            display: flex;
            align-items: center;
            gap: 14px;
        }
        .navbar-logo {
            display: inline-flex;
            align-items: center;
        }
        .navbar-logo img {
            width: 40px;
            height: 40px;
            border-radius: 20px;
            object-fit: cover;
        }
        .site-nav {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .header-right {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .lang-switcher {
            display: inline-flex;
            gap: 4px;
            padding: 3px;
            border: 1px solid var(--border);
            border-radius: 999px;
            background: var(--bg-secondary);
        }
        .lang-link {
            border: 0;
            border-radius: 999px;
            padding: 6px 11px;
            font-size: 0.8em;
            color: var(--text-secondary);
            text-decoration: none;
            font-weight: 600;
        }
        .lang-link.is-active {
            color: #fff;
            background: var(--accent);
        }
        .site-nav a {
            color: var(--text-primary);
            text-decoration: none;
            font-weight: 600;
        }
        .site-nav a:hover {
            color: var(--accent);
        }
        .theme-toggle {
            border: 1px solid var(--border);
            background: transparent;
            width: 36px;
            height: 36px;
            border-radius: 18px;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            color: var(--text-primary);
        }
        .theme-toggle svg {
            width: 18px;
            height: 18px;
            display: block;
        }
        .theme-toggle .icon-moon { display: none; }
        html[data-theme='dark'] .theme-toggle .icon-sun { display: none; }
        html[data-theme='dark'] .theme-toggle .icon-moon { display: block; }
        html[data-theme='dark'] {
            --bg-primary: #1a1a1a;
            --bg-secondary: #2d2d2d;
            --text-primary: #e0e0e0;
            --text-secondary: #a0a0a0;
            --accent: #4d9fff;
            --accent-hover: #66b3ff;
            --border: #404040;
            --shadow: rgba(0,0,0,0.3);
        }
        /* Hero Section */
        .hero {
            position: relative;
            min-height: 42vh;
            display: flex;
            align-items: center;
            justify-content: center;
            text-align: left;
            padding: 0;
            margin-bottom: 32px;
            {{if .Homepage.Hero.Background}}
            background: url('{{.Homepage.Hero.Background}}') #20241f no-repeat center center;
            background-size: cover;
            color: white;
            {{else}}
            background: var(--bg-secondary);
            {{end}}
        }
        .hero-content {
            width: 100%;
            min-height: 42vh;
            display: flex;
            flex-direction: column;
            align-items: stretch;
            justify-content: center;
            padding: 3.9rem 1.25rem;
            {{if .Homepage.Hero.Background}}
            backdrop-filter: blur(6px);
            background-color: rgba(0, 0, 0, 0.15);
            {{end}}
        }
        .hero-layout {
            width: min(1200px, 100%);
            margin: 0 auto;
            display: grid;
            grid-template-columns: 1.1fr 0.9fr;
            gap: 24px;
            align-items: center;
        }
        .hero-copy {
            max-width: 680px;
        }
        .hero h1 {
            font-size: 3em;
            margin: 0 0 16px;
            font-weight: 700;
        }
        .hero .subtitle {
            font-size: 1.5em;
            margin: 0 0 24px;
            {{if .Homepage.Hero.Background}}
            color: rgba(255,255,255,0.9);
            {{else}}
            color: var(--text-secondary);
            {{end}}
        }
        .hero .description {
            font-size: 1.1em;
            margin: 0 0 32px;
            {{if .Homepage.Hero.Background}}
            color: rgba(255,255,255,0.8);
            {{else}}
            color: var(--text-secondary);
            {{end}}
        }
        .hero-cta {
            display: flex;
            gap: 16px;
            justify-content: flex-start;
            flex-wrap: wrap;
        }
        .hero-media {
            width: 100%;
            max-width: 560px;
            justify-self: end;
        }
        .hero-media iframe {
            width: 100%;
            aspect-ratio: 16 / 9;
            height: auto;
            border: 0;
            border-radius: 10px;
            box-shadow: 0 8px 30px rgba(0, 0, 0, 0.25);
            background: rgba(0, 0, 0, 0.35);
        }
        .btn {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            padding: 12px 24px;
            border-radius: 8px;
            text-decoration: none;
            font-weight: 500;
            transition: all 0.2s;
        }
        .btn-primary {
            background: var(--accent);
            color: white;
        }
        .btn-primary:hover {
            background: var(--accent-hover);
        }
        .btn-secondary {
            background: transparent;
            color: {{if .Homepage.Hero.Background}}white{{else}}var(--text-primary){{end}};
            border: 2px solid {{if .Homepage.Hero.Background}}white{{else}}var(--border){{end}};
        }
        .btn-secondary:hover {
            border-color: var(--accent);
            color: var(--accent);
        }
        .hero-chat-panel {
            overflow: visible;
        }
        .hero-chat-panel-inner {
            max-width: 800px;
            margin: 0 auto;
            padding: 18px 20px 28px;
            width: 100%;
        }
        .triangle-chat-root {
            display: flex;
            justify-content: center;
            width: 100%;
        }
        .triangle-chat-root > div {
            width: 100% !important;
            max-width: 800px !important;
        }
        /* Content Section */
        .content-section {
            max-width: 1200px;
            margin: 0 auto;
            padding: 60px 20px;
        }
        .content-section h2 {
            text-align: center;
            font-size: 2em;
            margin: 0 0 40px;
        }
        /* Projects Grid */
        .projects-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 24px;
            margin-bottom: 60px;
        }
        .project-card {
            background: var(--bg-secondary);
            border-radius: 12px;
            overflow: hidden;
            border: 1px solid var(--border);
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .project-card:hover {
            transform: translateY(-4px);
            box-shadow: 0 8px 24px var(--shadow);
        }
        .project-card .md-image {
            margin: 0;
        }
        .project-card .md-image img {
            width: 100%;
            height: 180px;
            object-fit: cover;
            background: var(--border);
            display: block;
        }
        .project-image {
            width: 100%;
            height: 180px;
            object-fit: cover;
            background: var(--border);
        }
        .project-content {
            padding: 20px;
        }
        .project-content h3 {
            margin: 0 0 8px;
            font-size: 1.25em;
        }
        .project-content p {
            margin: 0 0 16px;
            color: var(--text-secondary);
            font-size: 0.95em;
        }
        .project-tags {
            display: flex;
            gap: 8px;
            flex-wrap: wrap;
            margin-bottom: 16px;
        }
        .project-tag {
            font-size: 0.8em;
            padding: 4px 10px;
            background: var(--bg-primary);
            border-radius: 20px;
            color: var(--text-secondary);
        }
        .project-links {
            display: flex;
            gap: 16px;
        }
        .project-links a {
            color: var(--accent);
            text-decoration: none;
            font-size: 0.9em;
            font-weight: 500;
        }
        .project-links a:hover {
            text-decoration: underline;
        }
        /* Social Links */
        .social-section {
            background: var(--bg-secondary);
            padding: 60px 20px;
        }
        .social-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 40px;
            max-width: 1200px;
            margin: 0 auto;
        }
        .social-group h3 {
            margin: 0 0 16px;
            font-size: 1.1em;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        .social-links {
            list-style: none;
            padding: 0;
            margin: 0;
        }
        .social-links li {
            margin-bottom: 8px;
        }
        .social-links a {
            color: var(--text-primary);
            text-decoration: none;
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 8px 0;
        }
        .social-links a:hover {
            color: var(--accent);
        }
        /* Main Content Area */
        .main-content {
            max-width: 800px;
            margin: 0 auto;
            padding: 40px 20px;
        }
        .embed {
            width: 100%;
            max-width: 800px;
            margin: 24px auto;
        }
        .embed iframe {
            display: block;
            width: min(100%, 800px);
            aspect-ratio: 16 / 9;
            height: auto;
            border: 0;
            border-radius: 12px;
        }
        .main-content h2 {
            margin-top: 40px;
        }
        /* Footer */
        .site-footer {
            text-align: center;
            padding: 40px 20px;
            border-top: 1px solid var(--border);
            color: var(--text-secondary);
        }
        /* Responsive */
        @media (max-width: 768px) {
            .hero h1 { font-size: 2em; }
            .hero .subtitle { font-size: 1.1em; }
            .hero-layout {
                grid-template-columns: 1fr;
                gap: 16px;
            }
            .hero-content {
                min-height: 34vh;
                padding: 2.4rem 1rem;
            }
            .hero-copy {
                max-width: none;
            }
            .hero-media {
                max-width: none;
                justify-self: stretch;
            }
            .hero-content {
                align-items: stretch;
            }
            .hero-chat-panel-inner {
                padding: 14px 12px 20px;
            }
            .projects-grid { grid-template-columns: 1fr; }
        }
    </style>
    {{if .CSSPath}}<link rel="stylesheet" href="{{.CSSPath}}">{{end}}
</head>
<body>
    <header class="site-header">
        <div class="header-left">
            <a class="navbar-logo" href="/{{.Page.Language}}/">
                <img src="{{.Site.Site.Logo}}" alt="{{.Site.Site.Title}}">
            </a>
            <nav class="site-nav">
                {{range .HeaderNav}}<a href="{{.URL}}">{{.Title}}</a>{{end}}
            </nav>
        </div>
        <div class="header-right">
            <nav class="lang-switcher" aria-label="{{.UI.Languages}}">
                {{range .Languages}}<a class="lang-link{{if .Active}} is-active{{end}}" href="{{.URL}}" hreflang="{{.Code}}">{{.Label}}</a>{{end}}
            </nav>
            <button id="theme-toggle" class="theme-toggle" aria-label="{{.UI.ToggleTheme}}" title="{{.UI.ToggleTheme}}">
                <svg class="icon-sun" viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="4"></circle>
                    <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"></path>
                </svg>
                <svg class="icon-moon" viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 12.79A9 9 0 1 1 11.21 3c0 0 0 0 0 0A7 7 0 0 0 21 12.79z"></path>
                </svg>
            </button>
        </div>
    </header>

    <!-- Hero Section -->
    <section class="hero">
        <div class="hero-content">
            <div class="hero-layout">
                <div class="hero-copy">
                    <h1>{{if .Homepage.Hero.Title}}{{.Homepage.Hero.Title}}{{else}}{{.Page.Title}}{{end}}</h1>
                    {{if .Homepage.Hero.Subtitle}}<p class="subtitle">{{.Homepage.Hero.Subtitle}}</p>{{end}}
                    {{if .Homepage.Hero.Description}}<p class="description">{{.Homepage.Hero.Description}}</p>{{end}}
                    {{if .Homepage.Hero.CTAButtons}}
                    <div class="hero-cta">
                        {{range .Homepage.Hero.CTAButtons}}
                        <a href="{{.URL}}" class="btn {{if eq .Icon "primary"}}btn-primary{{else}}btn-secondary{{end}}">
                            {{.Label}}
                        </a>
                        {{end}}
                    </div>
                    {{end}}
                </div>
                {{if .Homepage.Hero.VideoEmbed}}
                <div class="hero-media">
                    <iframe
                        src="{{.Homepage.Hero.VideoEmbed}}"
                        title="{{.UI.HeroVideo}}"
                        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                        loading="lazy"
                        referrerpolicy="strict-origin-when-cross-origin"
                        allowfullscreen>
                    </iframe>
                </div>
                {{end}}
            </div>
        </div>
    </section>
    {{if .Homepage.Chat.Enabled}}
    <section id="hero-chat-panel" class="hero-chat-panel">
        <div class="hero-chat-panel-inner">
            <div
                id="triangle-chat-root"
                class="triangle-chat-root"
                data-base-url="{{.Homepage.Chat.BaseURL}}"
                data-recipient-agent-id="{{.Homepage.Chat.RecipientAgentID}}"
                data-title="{{if .Homepage.Chat.Title}}{{.Homepage.Chat.Title}}{{else}}{{.UI.Chat}}{{end}}">
            </div>
        </div>
    </section>
    {{end}}

    <!-- Main Content from Markdown -->
    {{if .Content}}
    <div class="main-content">
        {{.Content}}
    </div>
    {{end}}

    <!-- Projects Section -->
    {{if and (not .Homepage.HideProjects) .Homepage.Projects}}
    <section class="content-section">
        <h2>{{.UI.Projects}}</h2>
        <div class="projects-grid">
            {{range .Homepage.Projects}}
            <div class="project-card">
                {{if .Image}}<img src="{{.Image}}" alt="{{.Title}}" class="project-image">{{end}}
                <div class="project-content">
                    <h3>
                        {{if .URL}}
                        <a href="{{.URL}}">{{.Title}}</a>
                        {{else if .GitHub}}
                        <a href="{{.GitHub}}">{{.Title}}</a>
                        {{else}}
                        {{.Title}}
                        {{end}}
                    </h3>
                    <p>{{.Description}}</p>
                    {{if .Tags}}
                    <div class="project-tags">
                        {{range .Tags}}<span class="project-tag">{{.}}</span>{{end}}
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>
    </section>
    {{end}}

    <!-- Social Links Section -->
    {{if .Homepage.SocialLinks}}
    <section class="social-section">
        <div class="social-grid">
            {{range .Homepage.SocialLinks}}
            <div class="social-group">
                <h3>{{.Title}}</h3>
                <ul class="social-links">
                    {{range .Links}}
                    <li><a href="{{.URL}}">{{.Label}}</a></li>
                    {{end}}
                </ul>
            </div>
            {{end}}
        </div>
    </section>
    {{end}}

    {{if .Homepage.CustomHTML}}{{.Homepage.CustomHTML}}{{end}}
    {{if .Homepage.Chat.Enabled}}
    <script type="module">
        const chatRoot = document.getElementById("triangle-chat-root");
        let triangleWidget = null;

        async function ensureTriangleWidget() {
            if (triangleWidget || !chatRoot) {
                return;
            }
            const mod = await import("/assets/triangle/embed.js");
            const baseUrl = chatRoot.dataset.baseUrl || window.location.origin;
            const recipientAgentId = chatRoot.dataset.recipientAgentId || "";
            const title = chatRoot.dataset.title || "{{.UI.Chat}}";
            triangleWidget = mod.createTriangleWidget({
                baseUrl: baseUrl,
                recipientAgentId: recipientAgentId,
                title: title,
                mount: chatRoot
            });
        }

        void ensureTriangleWidget();
    </script>
    {{end}}
    {{if .JSPath}}<script defer src="{{.JSPath}}"></script>{{end}}
</body>
</html>
{{end}}
`
