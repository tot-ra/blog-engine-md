package embeddings

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestCheckDetectsMissingFrontmatterEmbeddingWithoutCallingNetwork(t *testing.T) {
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
	writeArticle("redirect.md", "---\ntitle: Old URL\nredirectUrl: /ru/blog/post/\n---\nRedirect")
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

func TestRunWritesEmbeddingToArticleFrontmatter(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	articlePath := filepath.Join(contentDir, "en", "blog", "post.md")
	if err := os.MkdirAll(filepath.Dir(articlePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(articlePath, []byte("---\ntitle: Post\ncustom: keep-me\n---\nBody"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(contentDir, filepath.Join(dir, "embeddings.json"))
	cfg.I18n.Default = "en"
	result, err := Run(context.Background(), cfg, RunOptions{Client: &fakeEmbedder{vectors: [][]float32{{1, 0}}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Sent != 1 {
		t.Fatalf("result = %#v", result)
	}
	data, err := os.ReadFile(articlePath)
	if err != nil {
		t.Fatal(err)
	}
	fm, body, err := parser.ParseFrontmatter(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if fm.Embedding == nil || fm.Embedding.Model != cfg.Related.Model || fm.Embedding.Dimensions != 2 || fm.Embedding.Vector == "" || fm.Embedding.Hash == "" {
		t.Fatalf("embedding = %#v\n%s", fm.Embedding, data)
	}
	if body != "Body" || fm.Params["custom"] != "keep-me" {
		t.Fatalf("unrelated article data changed: body=%q params=%#v", body, fm.Params)
	}
	if _, err := os.Stat(cfg.Related.CachePath); !os.IsNotExist(err) {
		t.Fatalf("embed unexpectedly wrote central cache: %v", err)
	}

	result, err = Run(context.Background(), cfg, RunOptions{Check: true})
	if err != nil || result.Skipped != 1 || result.Sent != 0 {
		t.Fatalf("second check = %#v, %v", result, err)
	}
}

type fakeEmbedder struct {
	vectors [][]float32
}

func (f *fakeEmbedder) Embed(context.Context, string, int, []string) ([][]float32, int, error) {
	return f.vectors, 2, nil
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
