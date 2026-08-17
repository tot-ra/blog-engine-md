package embeddings

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tot-ra/blog-engine/internal/builder"
	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
)

var ErrCacheStale = errors.New("embeddings cache is stale or incomplete")

type Embedder interface {
	Embed(ctx context.Context, model string, dims int, inputs []string) ([][]float32, int, error)
}

type RunOptions struct {
	Check  bool
	Force  bool
	DryRun bool
	Output io.Writer
	Client Embedder
}

type Result struct {
	Articles      int
	Skipped       int
	Sent          int
	Chunks        int
	Tokens        int
	EstimatedCost float64
	Removed       int
}

type article struct {
	path      string
	hash      string
	lang      string
	url       string
	text      string
	chunks    []string
	embedding *parser.FrontmatterEmbedding
}

func Run(ctx context.Context, cfg *config.SiteConfig, opts RunOptions) (Result, error) {
	out := opts.Output
	if out == nil {
		out = io.Discard
	}
	if !cfg.Related.Enabled {
		return Result{}, fmt.Errorf("related embeddings are disabled")
	}

	articles, err := discoverArticles(cfg)
	if err != nil {
		return Result{}, err
	}
	result := Result{Articles: len(articles)}
	pending := make([]article, 0, len(articles))
	for _, item := range articles {
		if !opts.Force && validFrontmatterEmbedding(item.embedding, item.hash, cfg.Related.Model, cfg.Related.Dimensions) {
			result.Skipped++
			continue
		}
		item.chunks = ChunkText(item.text, DefaultChunkChars, DefaultChunkOverlap)
		for _, chunk := range item.chunks {
			result.Tokens += EstimateTokens(chunk)
		}
		result.Chunks += len(item.chunks)
		pending = append(pending, item)
	}
	result.Sent = len(pending)
	result.EstimatedCost = float64(result.Tokens) / 1_000_000 * 0.02

	fmt.Fprintf(out, "Articles: %d, embedded in frontmatter: %d, to embed: %d, chunks: %d, estimated tokens: %d\n", result.Articles, result.Skipped, result.Sent, result.Chunks, result.Tokens)
	if opts.DryRun {
		fmt.Fprintf(out, "Estimated OpenAI cost: $%.4f (text-embedding-3-small list price approximation)\n", result.EstimatedCost)
		for _, item := range pending {
			fmt.Fprintf(out, "  %s -> %s [%s]\n", item.path, item.url, item.lang)
		}
		return result, nil
	}
	if opts.Check {
		if result.Sent > 0 {
			return result, ErrCacheStale
		}
		fmt.Fprintln(out, "Article frontmatter embeddings are up to date.")
		return result, nil
	}
	if len(pending) == 0 {
		fmt.Fprintln(out, "Article frontmatter embeddings are up to date.")
		return result, nil
	}
	if opts.Client == nil {
		return result, fmt.Errorf("OpenAI embeddings client is not configured")
	}

	inputs := make([]string, 0, result.Chunks)
	for _, item := range pending {
		inputs = append(inputs, item.chunks...)
	}
	vectors, actualTokens, err := opts.Client.Embed(ctx, cfg.Related.Model, cfg.Related.Dimensions, inputs)
	if err != nil {
		return result, err
	}
	if len(vectors) != len(inputs) {
		return result, fmt.Errorf("embedder returned %d vectors for %d chunks", len(vectors), len(inputs))
	}
	if actualTokens > 0 {
		result.Tokens = actualTokens
	}

	vectorOffset := 0
	for _, item := range pending {
		count := len(item.chunks)
		merged, err := MergeChunks(vectors[vectorOffset : vectorOffset+count])
		if err != nil {
			return result, fmt.Errorf("merge chunks for %s: %w", item.path, err)
		}
		vectorOffset += count
		encoded, scale, err := Quantize(merged)
		if err != nil {
			return result, fmt.Errorf("quantize %s: %w", item.path, err)
		}
		embedding := parser.FrontmatterEmbedding{
			Version: CacheVersion, Model: cfg.Related.Model, Dimensions: cfg.Related.Dimensions,
			Hash: item.hash, Vector: encoded, Scale: scale,
		}
		if err := WriteFrontmatterEmbedding(filepath.Join(cfg.Build.ContentDir, filepath.FromSlash(item.path)), embedding); err != nil {
			return result, fmt.Errorf("write embedding for %s: %w", item.path, err)
		}
	}
	fmt.Fprintf(out, "Embedded %d articles into Markdown frontmatter, tokens: %d\n", result.Sent, result.Tokens)
	return result, nil
}

func discoverArticles(cfg *config.SiteConfig) ([]article, error) {
	index, err := builder.Discover(cfg.Build.ContentDir)
	if err != nil {
		return nil, fmt.Errorf("discover content: %w", err)
	}
	sections := make(map[string]struct{}, len(cfg.Related.Sections))
	for _, section := range cfg.Related.Sections {
		sections[strings.ToLower(strings.Trim(strings.TrimSpace(section), "/"))] = struct{}{}
	}
	languages := make(map[string]struct{}, len(cfg.I18n.Languages))
	for _, language := range cfg.I18n.Languages {
		languages[strings.ToLower(strings.TrimSpace(language.Code))] = struct{}{}
	}
	urlGenerator := builder.NewURLGenerator(cfg.Site.URL)
	explicitIndexDirs := make(map[string]struct{})
	for _, file := range index.MarkdownFiles {
		name := strings.TrimSuffix(filepath.Base(file.RelativePath), filepath.Ext(file.RelativePath))
		if strings.EqualFold(name, "index") || strings.EqualFold(name, "README") {
			explicitIndexDirs[filepath.ToSlash(filepath.Dir(file.RelativePath))] = struct{}{}
		}
	}
	urlGenerator.SetExplicitIndexDirs(explicitIndexDirs)
	articles := make([]article, 0)
	for _, file := range index.MarkdownFiles {
		rel := filepath.ToSlash(file.RelativePath)
		lang, contentPath := detectLanguage(rel, cfg.I18n.Default, languages)
		parts := strings.Split(strings.Trim(contentPath, "/"), "/")
		if len(parts) < 2 {
			continue
		}
		if _, ok := sections[strings.ToLower(parts[0])]; !ok {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(contentPath), filepath.Ext(contentPath))
		if strings.EqualFold(name, "index") || strings.EqualFold(name, "README") {
			continue
		}
		data, err := os.ReadFile(file.Path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file.Path, err)
		}
		fm, body, err := parser.ParseFrontmatter(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse frontmatter %s: %w", file.Path, err)
		}
		if fm.Draft || strings.TrimSpace(fm.RedirectURL) != "" || (strings.TrimSpace(fm.Slug) == "" && isSectionFile(contentPath, name)) {
			continue
		}
		title := fm.Title
		if strings.TrimSpace(title) == "" {
			title = strings.ReplaceAll(strings.ReplaceAll(name, "-", " "), "_", " ")
		}
		text := PrepareInput(title, fm.Description, fm.Tags, body)
		if text == "" {
			continue
		}
		articles = append(articles, article{
			path:      rel,
			hash:      HashInput(text, cfg.Related.Model, cfg.Related.Dimensions),
			lang:      lang,
			url:       urlGenerator.Generate(rel, fm),
			text:      text,
			embedding: fm.Embedding,
		})
	}
	sort.Slice(articles, func(i, j int) bool { return articles[i].path < articles[j].path })
	return articles, nil
}

func detectLanguage(relPath, defaultLang string, languages map[string]struct{}) (string, string) {
	clean := strings.Trim(filepath.ToSlash(relPath), "/")
	parts := strings.Split(clean, "/")
	if len(parts) > 1 {
		first := strings.ToLower(parts[0])
		if _, ok := languages[first]; ok {
			return first, strings.Join(parts[1:], "/")
		}
	}
	return defaultLang, clean
}

func isSectionFile(contentPath, name string) bool {
	dir := filepath.Dir(contentPath)
	if dir == "." {
		return false
	}
	parent := filepath.Base(dir)
	return strings.EqualFold(parent, name) || strings.EqualFold(parser.GenerateSlug(parent), parser.GenerateSlug(name))
}

func validEntry(entry Entry, dims int) bool {
	if entry.Hash == "" || entry.Scale <= 0 || entry.Vec == "" {
		return false
	}
	vec, err := Dequantize(entry.Vec, entry.Scale)
	return err == nil && len(vec) == dims
}

func validFrontmatterEmbedding(embedding *parser.FrontmatterEmbedding, hash, model string, dims int) bool {
	if embedding == nil || embedding.Version != CacheVersion || embedding.Model != model || embedding.Dimensions != dims || embedding.Hash != hash {
		return false
	}
	return validEntry(Entry{Hash: embedding.Hash, Vec: embedding.Vector, Scale: embedding.Scale}, dims)
}
