package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tot-ra/blog-engine/internal/embeddings"
)

func TestParseServeConfig(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		port    int
		wantErr bool
	}{
		{name: "default", port: 3000},
		{name: "custom port", args: []string{"--port", "3001"}, port: 3001},
		{name: "invalid low port", args: []string{"--port", "0"}, wantErr: true},
		{name: "invalid high port", args: []string{"--port", "65536"}, wantErr: true},
		{name: "invalid value", args: []string{"--port", "not-a-port"}, wantErr: true},
		{name: "unexpected argument", args: []string{"extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseServeConfig(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseServeConfig() error = %v", err)
			}
			if cfg.port != tt.port {
				t.Fatalf("parseServeConfig() port = %d, want %d", cfg.port, tt.port)
			}
		})
	}
}

func TestRunEmbedCheckReturnsStaleWithoutAPIKey(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "content", "en", "blog"), 0755); err != nil {
		t.Fatal(err)
	}
	article := "---\ntitle: Test\n---\nBody"
	if err := os.WriteFile(filepath.Join(dir, "content", "en", "blog", "test.md"), []byte(article), 0644); err != nil {
		t.Fatal(err)
	}
	configYAML := "site:\n  title: Test\n  url: https://example.com\nbuild:\n  contentDir: content\nrelated:\n  enabled: true\n  dimensions: 2\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	if err := runEmbed(embedConfig{check: true}); !errors.Is(err, embeddings.ErrCacheStale) {
		t.Fatalf("runEmbed(check) error = %v, want stale", err)
	}
}
