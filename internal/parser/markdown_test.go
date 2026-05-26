package parser

import (
	"strings"
	"testing"
)

func TestMarkdownParser_Render(t *testing.T) {
	parser := NewMarkdownParser()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "heading",
			input:    "# Hello",
			contains: []string{"<h1", "Hello", "</h1>"},
		},
		{
			name:     "bold text",
			input:    "**bold**",
			contains: []string{"<strong>", "bold", "</strong>"},
		},
		{
			name:     "italic text",
			input:    "*italic*",
			contains: []string{"<em>", "italic", "</em>"},
		},
		{
			name:     "code block",
			input:    "```go\nfunc main() {}\n```",
			contains: []string{"<pre", "<code", "func main()"},
		},
		{
			name:     "inline code",
			input:    "`code`",
			contains: []string{"<code>", "code", "</code>"},
		},
		{
			name:     "link",
			input:    "[text](http://example.com)",
			contains: []string{"<a", "href=\"http://example.com\"", "text"},
		},
		{
			name:     "table",
			input:    "| A | B |\n|---|---|\n| 1 | 2 |",
			contains: []string{"<table", "<th", "<td"},
		},
		{
			name:     "strikethrough",
			input:    "~~deleted~~",
			contains: []string{"<del>", "deleted"},
		},
		{
			name:     "task list",
			input:    "- [x] done\n- [ ] todo",
			contains: []string{"<input", "checkbox", "checked"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := parser.Render(tt.input)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(html, want) {
					t.Errorf("Output doesn't contain %q:\n%s", want, html)
				}
			}
		})
	}
}

func TestMarkdownParser_RenderWithMeta(t *testing.T) {
	parser := NewMarkdownParser()
	content := `---
title: Test
---

# Hello`

	html, meta, err := parser.RenderWithMeta(content)
	if err != nil {
		t.Fatalf("RenderWithMeta failed: %v", err)
	}

	if !strings.Contains(html, "<h1") {
		t.Errorf("Output doesn't contain h1 tag")
	}

	if meta == nil {
		t.Error("Expected meta data, got nil")
	}

	if title, ok := meta["title"]; ok {
		if title != "Test" {
			t.Errorf("Expected title 'Test', got %v", title)
		}
	}
}

func TestMarkdownParser_RenderMDXStyleHTMLBlockImage(t *testing.T) {
	parser := NewMarkdownParser()
	input := `<div style={{ height:500, overflow:"hidden", verticalAlign:"middle", marginBottom:10, borderRadius:5 }}>
![](img/hero.webp)
</div>`

	html, err := parser.Render(input)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	for _, want := range []string{
		`<div style="border-radius:5px;height:500px;margin-bottom:10px;overflow:hidden;vertical-align:middle">`,
		`<img src="img/hero.webp" alt="">`,
		`</div>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("output doesn't contain %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, `![](img/hero.webp)`) {
		t.Fatalf("markdown image leaked into output:\n%s", html)
	}
}

func TestMarkdownParser_RenderNestedHTMLBlockImage(t *testing.T) {
	parser := NewMarkdownParser()
	input := `<div style={{ height:150 }}><div style={{ marginTop: "-10%" }}>
![](img/banner.jpg)
</div></div>`

	html, err := parser.Render(input)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, `<img src="img/banner.jpg" alt="">`) {
		t.Fatalf("expected markdown image inside HTML block to become img tag:\n%s", html)
	}
}

func TestMarkdownParser_DoesNotRewriteImagesInCodeFences(t *testing.T) {
	parser := NewMarkdownParser()
	input := "```md\n<div>\n![](img/hero.webp)\n</div>\n```"

	html, err := parser.Render(input)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if strings.Contains(html, `<img src="img/hero.webp"`) {
		t.Fatalf("image in code fence was rewritten:\n%s", html)
	}
	if !strings.Contains(html, `![](img/hero.webp)`) {
		t.Fatalf("expected markdown image text to remain in code fence:\n%s", html)
	}
}
