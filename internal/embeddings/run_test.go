package embeddings

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tot-ra/blog-engine/internal/config"
)

func TestCheckDetectsStaleCacheWithoutCallingNetwork(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(filepath.Join(contentDir, "ru", "blog"), 0755); err != nil {
		t.Fatal(err)
	}
	writeArticle := func(name, contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(contentDir, "ru", "blog", name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeArticle("post.md", "---\ntitle: Post\ndescription: Desc\ntags: [Go]\n---\nBody")
	writeArticle("draft.md", "---\ntitle: Draft\ndraft: true\n---\nHidden")
	writeArticle("index.md", "---\ntitle: Blog\n---\nIndex")

	cfg := testConfig(contentDir, filepath.Join(dir, "embeddings.json"))
	result, err := Run(context.Background(), cfg, RunOptions{Check: true})
	if !errors.Is(err, ErrCacheStale) {
		t.Fatalf("Run(check) error = %v, want ErrCacheStale", err)
	}
	if result.Articles != 1 || result.Sent != 1 {
		t.Fatalf("Run(check) result = %#v", result)
	}
}

func TestRunRemovesDeletedEntries(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(dir, "embeddings.json")
	cache := NewCache("text-embedding-3-small", 2)
	cache.Entries["deleted.md"] = Entry{Hash: "sha256:x", Vec: "AAA=", Scale: 1, Lang: "en", URL: "/deleted/"}
	if err := cache.Save(cachePath); err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(contentDir, cachePath)
	result, err := Run(context.Background(), cfg, RunOptions{Client: &fakeEmbedder{}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 {
		t.Fatalf("removed = %d", result.Removed)
	}
	loaded, err := Load(cachePath, cfg.Related.Model, cfg.Related.Dimensions)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Entries) != 0 {
		t.Fatalf("stale entries remain: %#v", loaded.Entries)
	}
}

type fakeEmbedder struct{}

func (*fakeEmbedder) Embed(context.Context, string, int, []string) ([][]float32, int, error) {
	return nil, 0, nil
}

func testConfig(contentDir, cachePath string) *config.SiteConfig {
	cfg := config.DefaultConfig()
	cfg.Site.Title = "Test"
	cfg.Site.URL = "https://example.com"
	cfg.Build.ContentDir = contentDir
	cfg.I18n.Default = "ru"
	cfg.I18n.Languages = []config.LanguageConfig{{Code: "ru"}, {Code: "en"}}
	cfg.Related.Dimensions = 2
	cfg.Related.CachePath = cachePath
	return cfg
}
