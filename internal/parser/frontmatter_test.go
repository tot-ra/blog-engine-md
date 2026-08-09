package parser

import (
	"testing"
	"time"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		wantTitle        string
		wantNavTitle     string
		wantTags         []string
		wantTemplateHero bool
		wantRemaining    string
		wantErr          bool
	}{
		{
			name: "basic frontmatter",
			content: `---
title: Test Post
tags: [tag1, tag2]
---

Content here.`,
			wantTitle:     "Test Post",
			wantTags:      []string{"tag1", "tag2"},
			wantRemaining: "Content here.",
			wantErr:       false,
		},
		{
			name:          "no frontmatter",
			content:       "Just content",
			wantTitle:     "",
			wantTags:      nil,
			wantRemaining: "Just content",
			wantErr:       false,
		},
		{
			name: "with date",
			content: `---
title: Dated Post
date: 2025-01-15T10:00:00Z
---

Content.`,
			wantTitle:     "Dated Post",
			wantRemaining: "Content.",
			wantErr:       false,
		},
		{
			name: "draft post",
			content: `---
title: Draft Post
draft: true
---

Content.`,
			wantTitle:     "Draft Post",
			wantRemaining: "Content.",
			wantErr:       false,
		},
		{
			name: "with navigation title",
			content: `---
title: Hinnaplaanid
navTitle: Hinnad
---

Content.`,
			wantTitle:     "Hinnaplaanid",
			wantNavTitle:  "Hinnad",
			wantRemaining: "Content.",
			wantErr:       false,
		},
		{
			name: "with template hero opt-in",
			content: `---
title: Research
templateHero: true
---

Content.`,
			wantTitle:        "Research",
			wantTemplateHero: true,
			wantRemaining:    "Content.",
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, remaining, err := ParseFrontmatter(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if fm.Title != tt.wantTitle {
				t.Errorf("Expected title '%s', got '%s'", tt.wantTitle, fm.Title)
			}
			if fm.NavTitle != tt.wantNavTitle {
				t.Errorf("Expected navTitle '%s', got '%s'", tt.wantNavTitle, fm.NavTitle)
			}
			if fm.TemplateHero != tt.wantTemplateHero {
				t.Errorf("Expected templateHero %t, got %t", tt.wantTemplateHero, fm.TemplateHero)
			}

			if len(fm.Tags) != len(tt.wantTags) {
				t.Errorf("Expected %d tags, got %d", len(tt.wantTags), len(fm.Tags))
			}

			if remaining != tt.wantRemaining {
				t.Errorf("Expected remaining '%s', got '%s'", tt.wantRemaining, remaining)
			}
		})
	}
}

func TestParseFrontmatterSidebarPositionAlias(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantOrder int
	}{
		{
			name: "sidebar_position alone",
			content: `---
title: Exove
sidebar_position: 4
---

Content.`,
			wantOrder: 4,
		},
		{
			name: "order takes precedence over sidebar_position",
			content: `---
title: Mixed
order: 2
sidebar_position: 9
---

Content.`,
			wantOrder: 2,
		},
		{
			name: "plain order still works",
			content: `---
title: Native
order: 3
---

Content.`,
			wantOrder: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, _, err := ParseFrontmatter(tt.content)
			if err != nil {
				t.Fatalf("ParseFrontmatter failed: %v", err)
			}
			if fm.Order != tt.wantOrder {
				t.Fatalf("Order = %d, want %d", fm.Order, tt.wantOrder)
			}
		})
	}
}

func TestParseFrontmatterPreservesCustomParams(t *testing.T) {
	content := `---
title: Research Paper
year: "2025"
orgs:
  - University Lab
customField: custom value
---

Content.`

	fm, remaining, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter failed: %v", err)
	}
	if remaining != "Content." {
		t.Fatalf("remaining = %q, want %q", remaining, "Content.")
	}
	if got := fm.Params["year"]; got != "2025" {
		t.Fatalf("Params[year] = %#v, want 2025", got)
	}
	orgs, ok := fm.Params["orgs"].([]interface{})
	if !ok || len(orgs) != 1 || orgs[0] != "University Lab" {
		t.Fatalf("Params[orgs] = %#v, want single organization", fm.Params["orgs"])
	}
	if got := fm.Params["customField"]; got != "custom value" {
		t.Fatalf("Params[customField] = %#v, want custom value", got)
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Test Post 123", "test-post-123"},
		{"Привет мир", "privet-mir"},
		{"Шпаргалка по golang", "shpargalka-po-golang"},
		{"Special!@#Chars", "specialchars"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"", ""},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := GenerateSlug(tt.input)
			if got != tt.want {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTransliterate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"привет", "privet"},
		{"мир", "mir"},
		{"ёж", "yozh"},
		{"Привет Мир", "Privet Mir"},
		{"hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := transliterate(tt.input)
			if got != tt.want {
				t.Errorf("transliterate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFrontmatterDateParsing(t *testing.T) {
	content := `---
title: Test
date: 2025-01-15T10:30:00Z
---

Content.`

	fm, _, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter failed: %v", err)
	}

	expectedTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	if !fm.Date.Equal(expectedTime) {
		t.Errorf("Expected date %v, got %v", expectedTime, fm.Date)
	}
}
