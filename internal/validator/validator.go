package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// LinkStatus represents the status of a link check
type LinkStatus int

const (
	LinkOK LinkStatus = iota
	LinkBroken
	LinkWarning
	LinkSkipped
)

func (s LinkStatus) String() string {
	switch s {
	case LinkOK:
		return "OK"
	case LinkBroken:
		return "BROKEN"
	case LinkWarning:
		return "WARNING"
	case LinkSkipped:
		return "SKIPPED"
	default:
		return "UNKNOWN"
	}
}

// LinkCheck represents a single link check result
type LinkCheck struct {
	SourceFile string
	LinkText   string
	LinkURL    string
	Status     LinkStatus
	Message    string
	Line       int
}

// ValidationReport holds all validation results
type ValidationReport struct {
	Internal []LinkCheck
	External []LinkCheck
	Summary  ValidationSummary
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	Total    int
	OK       int
	Broken   int
	Warnings int
	Skipped  int
}

// PageContent holds page information for validation
type PageContent struct {
	SourcePath string
	URL        string
	RawContent string
	Anchors    []string // Heading IDs available on this page
}

// markdownLinkRegex matches [text](url) patterns
var markdownLinkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// ValidateLinks validates all internal links across pages
func ValidateLinks(pages []PageContent) *ValidationReport {
	report := &ValidationReport{
		Internal: make([]LinkCheck, 0),
		External: make([]LinkCheck, 0),
	}

	// Build lookup: URL -> page with anchors
	urlToPage := make(map[string]*PageContent)
	for i := range pages {
		p := &pages[i]
		urlToPage[p.URL] = p
		// Also map without trailing slash
		urlToPage[strings.TrimSuffix(p.URL, "/")] = p
	}

	for _, page := range pages {
		links := extractLinks(page.RawContent)

		for _, link := range links {
			check := LinkCheck{
				SourceFile: page.SourcePath,
				LinkText:   link.text,
				LinkURL:    link.url,
				Line:       link.line,
			}

			// Classify link
			if isExternalURL(link.url) {
				check.Status = LinkSkipped
				check.Message = "external link (use --external to check)"
				report.External = append(report.External, check)
				continue
			}

			if strings.HasPrefix(link.url, "#") {
				// Anchor-only link — validate on current page
				anchor := strings.TrimPrefix(link.url, "#")
				if containsAnchor(page.Anchors, anchor) {
					check.Status = LinkOK
				} else {
					check.Status = LinkBroken
					check.Message = fmt.Sprintf("anchor #%s not found on current page", anchor)
				}
				report.Internal = append(report.Internal, check)
				continue
			}

			// Internal link
			targetURL, anchor := splitAnchor(link.url)
			targetURL = normalizeURL(targetURL)

			targetPage, exists := urlToPage[targetURL]
			if !exists {
				// Also try with trailing slash
				targetPage, exists = urlToPage[targetURL+"/"]
			}

			if !exists {
				check.Status = LinkBroken
				check.Message = fmt.Sprintf("target page not found: %s", targetURL)
			} else if anchor != "" && !containsAnchor(targetPage.Anchors, anchor) {
				check.Status = LinkWarning
				check.Message = fmt.Sprintf("page exists but anchor #%s not found", anchor)
			} else {
				check.Status = LinkOK
			}

			report.Internal = append(report.Internal, check)
		}
	}

	// Compute summary
	all := append(report.Internal, report.External...)
	report.Summary = ValidationSummary{Total: len(all)}
	for _, c := range all {
		switch c.Status {
		case LinkOK:
			report.Summary.OK++
		case LinkBroken:
			report.Summary.Broken++
		case LinkWarning:
			report.Summary.Warnings++
		case LinkSkipped:
			report.Summary.Skipped++
		}
	}

	return report
}

// FormatReport formats a validation report as a human-readable string
func FormatReport(report *ValidationReport) string {
	var sb strings.Builder

	sb.WriteString("Link Validation Report\n")
	sb.WriteString("======================\n\n")

	// Broken links
	broken := filterByStatus(report.Internal, LinkBroken)
	if len(broken) > 0 {
		sb.WriteString(fmt.Sprintf("❌ Broken Links (%d):\n", len(broken)))
		for _, c := range broken {
			sb.WriteString(fmt.Sprintf("  %s:%d\n", c.SourceFile, c.Line))
			sb.WriteString(fmt.Sprintf("    [%s](%s)\n", c.LinkText, c.LinkURL))
			sb.WriteString(fmt.Sprintf("    → %s\n\n", c.Message))
		}
	}

	// Warnings
	warnings := filterByStatus(report.Internal, LinkWarning)
	if len(warnings) > 0 {
		sb.WriteString(fmt.Sprintf("⚠️ Warnings (%d):\n", len(warnings)))
		for _, c := range warnings {
			sb.WriteString(fmt.Sprintf("  %s:%d\n", c.SourceFile, c.Line))
			sb.WriteString(fmt.Sprintf("    [%s](%s)\n", c.LinkText, c.LinkURL))
			sb.WriteString(fmt.Sprintf("    → %s\n\n", c.Message))
		}
	}

	// Summary
	sb.WriteString(fmt.Sprintf("Summary: %d total, %d OK, %d broken, %d warnings, %d skipped\n",
		report.Summary.Total, report.Summary.OK, report.Summary.Broken, report.Summary.Warnings, report.Summary.Skipped))

	return sb.String()
}

type extractedLink struct {
	text string
	url  string
	line int
}

func extractLinks(content string) []extractedLink {
	var links []extractedLink
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		matches := markdownLinkRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			url := m[2]
			// Skip image references
			if strings.HasPrefix(strings.TrimSpace(line), "![") {
				continue
			}
			links = append(links, extractedLink{
				text: m[1],
				url:  url,
				line: i + 1,
			})
		}
	}

	return links
}

func isExternalURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "mailto:")
}

func splitAnchor(url string) (string, string) {
	parts := strings.SplitN(url, "#", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return url, ""
}

func normalizeURL(url string) string {
	url = strings.TrimSuffix(url, ".md")
	url = strings.TrimSuffix(url, "/index")
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return url
}

func containsAnchor(anchors []string, target string) bool {
	for _, a := range anchors {
		if strings.EqualFold(a, target) {
			return true
		}
	}
	return false
}

func filterByStatus(checks []LinkCheck, status LinkStatus) []LinkCheck {
	var result []LinkCheck
	for _, c := range checks {
		if c.Status == status {
			result = append(result, c)
		}
	}
	return result
}
