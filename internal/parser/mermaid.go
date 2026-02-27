package parser

import (
	"regexp"
	"strings"
)

// mermaidFenceRegex matches ```mermaid code blocks
var mermaidFenceRegex = regexp.MustCompile("(?m)^```mermaid\\s*$")
var closingFenceRegex = regexp.MustCompile("(?m)^```\\s*$")

// TransformMermaid converts ```mermaid code blocks to <pre class="mermaid"> blocks
// for client-side rendering by Mermaid.js
func TransformMermaid(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	var result []string
	var inMermaid bool
	var mermaidContent []string
	hasMermaid := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inMermaid {
			if mermaidFenceRegex.MatchString(trimmed) {
				inMermaid = true
				mermaidContent = nil
				hasMermaid = true
				continue
			}
			result = append(result, line)
		} else {
			if closingFenceRegex.MatchString(trimmed) {
				// Output mermaid block as HTML
				result = append(result, `<pre class="mermaid">`)
				result = append(result, mermaidContent...)
				result = append(result, `</pre>`)
				inMermaid = false
				continue
			}
			mermaidContent = append(mermaidContent, line)
		}
	}

	// If never closed, output raw
	if inMermaid {
		result = append(result, "```mermaid")
		result = append(result, mermaidContent...)
	}

	return strings.Join(result, "\n"), hasMermaid
}
