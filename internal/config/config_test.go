package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
site:
  title: "Test Site"
  url: "https://example.com"
  language: "ru"

author:
  name: "Test Author"

build:
  contentDir: "content"
  outputDir: "output"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Site.Title != "Test Site" {
		t.Errorf("Expected title 'Test Site', got '%s'", cfg.Site.Title)
	}

	if cfg.Site.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", cfg.Site.URL)
	}

	if cfg.Site.Language != "ru" {
		t.Errorf("Expected language 'ru', got '%s'", cfg.Site.Language)
	}

	if cfg.Author.Name != "Test Author" {
		t.Errorf("Expected author name 'Test Author', got '%s'", cfg.Author.Name)
	}

	if cfg.Build.ContentDir != "content" {
		t.Errorf("Expected content dir 'content', got '%s'", cfg.Build.ContentDir)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *SiteConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &SiteConfig{
				Site: Site{Title: "Test", URL: "https://example.com"},
			},
			wantErr: false,
		},
		{
			name: "missing title",
			cfg: &SiteConfig{
				Site: Site{URL: "https://example.com"},
			},
			wantErr: true,
		},
		{
			name: "missing URL",
			cfg: &SiteConfig{
				Site: Site{Title: "Test"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Site.BaseURL != "/" {
		t.Errorf("Expected base URL '/', got '%s'", cfg.Site.BaseURL)
	}

	if cfg.Site.Language != "en" {
		t.Errorf("Expected language 'en', got '%s'", cfg.Site.Language)
	}

	if cfg.Build.ContentDir != "content" {
		t.Errorf("Expected content dir 'content', got '%s'", cfg.Build.ContentDir)
	}

	if cfg.Build.OutputDir != "dist" {
		t.Errorf("Expected output dir 'dist', got '%s'", cfg.Build.OutputDir)
	}

	if cfg.Build.ParallelWorkers != 4 {
		t.Errorf("Expected 4 parallel workers, got %d", cfg.Build.ParallelWorkers)
	}
}

func TestLoadNormalizesLanguageAliasesAndDirections(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
site:
  title: "Test Site"
  url: "https://example.com"

i18n:
  default: "rus"
  browserRedirect:
    enabled: true
  languages:
    - code: "RUS"
      label: "Русский"
      aliases: ["ru", "RU-ru", "ru"]
    - code: "EST"
      label: "Eesti"
      aliases: ["et", "ET-EE"]
      direction: "rtl"
    - code: "HE"
      label: "עברית"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !cfg.I18n.BrowserRedirect.Enabled {
		t.Fatalf("expected browser redirect to be enabled")
	}
	if got := cfg.I18n.Languages[0].Aliases; len(got) != 2 || got[0] != "ru" || got[1] != "ru-ru" {
		t.Fatalf("unexpected normalized aliases for rus: %#v", got)
	}
	if got := cfg.I18n.Languages[0].Direction; got != "ltr" {
		t.Fatalf("expected Russian direction to default to ltr, got %q", got)
	}
	if got := cfg.I18n.Languages[1].Aliases; len(got) != 2 || got[0] != "et" || got[1] != "et-ee" {
		t.Fatalf("unexpected normalized aliases for est: %#v", got)
	}
	if got := cfg.I18n.Languages[1].Direction; got != "rtl" {
		t.Fatalf("expected explicit Estonian test direction to stay rtl, got %q", got)
	}
	if got := cfg.I18n.Languages[2].Direction; got != "rtl" {
		t.Fatalf("expected Hebrew direction to default to rtl, got %q", got)
	}
}

func TestLanguageDirection(t *testing.T) {
	languages := []LanguageConfig{
		{Code: "ar", Direction: "ltr"},
		{Code: "fa"},
	}

	if got := LanguageDirection("ar", languages); got != "ltr" {
		t.Fatalf("expected explicit Arabic direction override to ltr, got %q", got)
	}
	if got := LanguageDirection("fa", languages); got != "rtl" {
		t.Fatalf("expected Persian to default to rtl, got %q", got)
	}
	if got := LanguageDirection("en", languages); got != "ltr" {
		t.Fatalf("expected English to default to ltr, got %q", got)
	}
}

func TestLoadHeaderLanguageItems(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
site:
  title: "Test Site"
  url: "https://example.com"

navigation:
  header:
    enabled: true
    items:
      - title: "Status"
        url: "https://status.example.com/"
        languages: ["en", "et"]
    languages:
      en:
        - title: "About"
          url: "/about/"
      et:
        - title: "Meist"
          url: "/et/about/"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Navigation.Header.Items) != 1 {
		t.Fatalf("expected one shared header item, got %d", len(cfg.Navigation.Header.Items))
	}
	if got := cfg.Navigation.Header.Items[0].Languages; len(got) != 2 || got[0] != "en" || got[1] != "et" {
		t.Fatalf("unexpected shared header item languages: %#v", got)
	}
	if got := cfg.Navigation.Header.LanguageItems["et"]; len(got) != 1 || got[0].Title != "Meist" || got[0].URL != "/et/about/" {
		t.Fatalf("unexpected Estonian header items: %#v", got)
	}
}
