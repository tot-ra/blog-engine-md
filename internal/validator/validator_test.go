package validator

import (
	"testing"
)

func TestValidateLinks_AllOK(t *testing.T) {
	pages := []PageContent{
		{
			SourcePath: "content/blog/hello.md",
			URL:        "/blog/hello/",
			RawContent: "See [about](/docs/about/) for more.",
			Anchors:    []string{},
		},
		{
			SourcePath: "content/docs/about.md",
			URL:        "/docs/about/",
			RawContent: "Back to [hello](/blog/hello/).",
			Anchors:    []string{"introduction"},
		},
	}

	report := ValidateLinks(pages)

	if report.Summary.Broken != 0 {
		t.Errorf("expected 0 broken links, got %d", report.Summary.Broken)
	}
	if report.Summary.OK != 2 {
		t.Errorf("expected 2 OK links, got %d", report.Summary.OK)
	}
}

func TestValidateLinks_BrokenLink(t *testing.T) {
	pages := []PageContent{
		{
			SourcePath: "content/blog/hello.md",
			URL:        "/blog/hello/",
			RawContent: "See [missing](/does/not/exist/) page.",
		},
	}

	report := ValidateLinks(pages)

	if report.Summary.Broken != 1 {
		t.Errorf("expected 1 broken link, got %d", report.Summary.Broken)
	}
	if report.Internal[0].Status != LinkBroken {
		t.Error("expected broken status")
	}
}

func TestValidateLinks_BrokenAnchor(t *testing.T) {
	pages := []PageContent{
		{
			SourcePath: "content/blog/hello.md",
			URL:        "/blog/hello/",
			RawContent: "See [section](#nonexistent).",
			Anchors:    []string{"introduction", "conclusion"},
		},
	}

	report := ValidateLinks(pages)

	if report.Summary.Broken != 1 {
		t.Errorf("expected 1 broken anchor, got %d", report.Summary.Broken)
	}
}

func TestValidateLinks_ValidAnchor(t *testing.T) {
	pages := []PageContent{
		{
			SourcePath: "content/blog/hello.md",
			URL:        "/blog/hello/",
			RawContent: "See [intro](#introduction).",
			Anchors:    []string{"introduction"},
		},
	}

	report := ValidateLinks(pages)

	if report.Summary.OK != 1 {
		t.Errorf("expected 1 OK link, got %d", report.Summary.OK)
	}
}

func TestValidateLinks_ExternalSkipped(t *testing.T) {
	pages := []PageContent{
		{
			SourcePath: "content/blog/hello.md",
			URL:        "/blog/hello/",
			RawContent: "Visit [Google](https://google.com).",
		},
	}

	report := ValidateLinks(pages)

	if report.Summary.Skipped != 1 {
		t.Errorf("expected 1 skipped link, got %d", report.Summary.Skipped)
	}
}

func TestValidateLinks_CrossPageAnchor(t *testing.T) {
	pages := []PageContent{
		{
			SourcePath: "content/blog/hello.md",
			URL:        "/blog/hello/",
			RawContent: "See [section](/docs/about/#missing-anchor).",
		},
		{
			SourcePath: "content/docs/about.md",
			URL:        "/docs/about/",
			RawContent: "Content here.",
			Anchors:    []string{"introduction"},
		},
	}

	report := ValidateLinks(pages)

	if report.Summary.Warnings != 1 {
		t.Errorf("expected 1 warning for missing anchor, got %d", report.Summary.Warnings)
	}
}

func TestFormatReport(t *testing.T) {
	report := &ValidationReport{
		Internal: []LinkCheck{
			{
				SourceFile: "test.md",
				LinkText:   "broken",
				LinkURL:    "/missing/",
				Status:     LinkBroken,
				Message:    "target page not found",
				Line:       5,
			},
		},
		Summary: ValidationSummary{
			Total:  1,
			Broken: 1,
		},
	}

	output := FormatReport(report)

	if output == "" {
		t.Error("expected non-empty report")
	}
	if len(output) < 50 {
		t.Error("expected detailed report output")
	}
}
