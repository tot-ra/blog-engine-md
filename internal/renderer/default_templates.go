package renderer

import "embed"

// defaultTemplateFS contains built-in templates split into logical partials.
//
//go:embed default_templates/*.html
var defaultTemplateFS embed.FS
