package renderer

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
)

// Page represents page data for templates
type Page struct {
	ID           string
	URL          string
	Title        string
	Description  string
	Content      string
	Type         string
	ModifiedTime time.Time
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
	Site        config.SiteConfig
	Page        Page
	Frontmatter Frontmatter
	Content     template.HTML
	// Navigation fields (Phase 2)
	Sidebar     template.HTML
	TOC         template.HTML
	Breadcrumbs []BreadcrumbItem
	PrevNext    *PrevNextLinks
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
		"slugify": func(text string) string {
			return slugify(text)
		},
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"title":      strings.Title,
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

// RenderPage renders a page using the default page template
func (e *TemplateEngine) RenderPage(data PageData) (string, error) {
	return e.Render("page", data)
}

// slugify creates a URL-friendly slug
func slugify(text string) string {
	if text == "" {
		return ""
	}
	// Simple slugification
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, " ", "-")
	text = strings.ReplaceAll(text, "_", "-")
	return text
}

// defaultTemplates contains built-in templates
const defaultTemplates = `{{define "base"}}
<!DOCTYPE html>
<html lang="{{.Site.Site.Language}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Page.Title}}{{if .Site.Site.Title}} | {{.Site.Site.Title}}{{end}}</title>
    {{if .Page.Description}}
    <meta name="description" content="{{.Page.Description}}">
    {{end}}
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
        .site-header h1 { font-size: 1.2em; }
        .site-header h1 a { color: inherit; text-decoration: none; }
        .site-header p { margin: 0; color: var(--text-secondary); font-size: 0.9em; }
        /* Layout */
        .site-layout {
            display: flex;
            max-width: 1400px;
            margin: 0 auto;
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
        article .tags { margin: 8px 0 24px; }
        article .tag {
            display: inline-block;
            background: var(--nav-bg);
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 0.85em;
            color: var(--nav-active);
            margin-right: 4px;
        }
        article .content { margin-top: 16px; }
        article .content h2 { margin-top: 2em; margin-bottom: 0.5em; }
        article .content h3 { margin-top: 1.5em; margin-bottom: 0.4em; }
        article .content p { margin-bottom: 1em; }
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
        /* Responsive */
        @media (max-width: 1200px) {
            .toc { display: none; }
        }
        @media (max-width: 768px) {
            .sidebar { display: none; }
            .site-content { padding: 16px; }
        }
    </style>
</head>
<body>
    <header class="site-header">
        <div>
            <h1><a href="/">{{.Site.Site.Title}}</a></h1>
            {{if .Site.Site.Tagline}}<p>{{.Site.Site.Tagline}}</p>{{end}}
        </div>
    </header>
    <div class="site-layout">
        {{if .Sidebar}}{{.Sidebar}}{{end}}
        <main class="site-content">
            {{template "content" .}}
        </main>
        {{if .TOC}}{{.TOC}}{{end}}
    </div>
    <footer class="site-footer">
        {{if .Site.Author.Name}}&copy; {{.Site.Author.Name}}{{end}}
    </footer>
</body>
</html>
{{end}}

{{define "page"}}
{{template "base" .}}
{{end}}

{{define "content"}}
<article>
    {{if .Breadcrumbs}}
    <nav aria-label="Breadcrumb">
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
    {{if .Frontmatter.Date}}
    <time datetime="{{.Frontmatter.Date.Format "2006-01-02T15:04:05Z07:00"}}">
        {{.Frontmatter.Date.Format "January 2, 2006"}}
    </time>
    {{end}}
    {{if .Frontmatter.Tags}}
    <div class="tags">
        {{range .Frontmatter.Tags}}
        <span class="tag">#{{.}}</span>
        {{end}}
    </div>
    {{end}}
    <div class="content">
        {{.Content}}
    </div>
    {{if .PrevNext}}
    <nav class="prev-next" aria-label="Page navigation">
        {{if .PrevNext.Prev}}
        <a href="{{.PrevNext.Prev.URL}}" class="prev-link">
            <span class="prev-label">← Previous</span>
            <span class="prev-title">{{.PrevNext.Prev.Title}}</span>
        </a>
        {{end}}
        {{if .PrevNext.Next}}
        <a href="{{.PrevNext.Next.URL}}" class="next-link">
            <span class="next-label">Next →</span>
            <span class="next-title">{{.PrevNext.Next.Title}}</span>
        </a>
        {{end}}
    </nav>
    {{end}}
</article>
{{end}}
`
