package builder

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tot-ra/blog-engine/internal/i18n"
	"github.com/tot-ra/blog-engine/internal/parser"
)

func (b *SiteBuilder) prepareBlogAudio(index *ContentIndex) error {
	if b.config == nil || !b.config.Audio.Enabled {
		return nil
	}

	if strings.ToLower(strings.TrimSpace(b.config.Audio.Provider)) != "edge" {
		return fmt.Errorf("unsupported audio provider %q", b.config.Audio.Provider)
	}

	edgeBin := strings.TrimSpace(b.config.Audio.Edge.Binary)
	if edgeBin == "" {
		edgeBin = "edge-tts"
	}
	if _, err := exec.LookPath(edgeBin); err != nil {
		fmt.Fprintf(os.Stderr, "Audio generation skipped: %q not found in PATH\n", edgeBin)
		return nil
	}

	posts := b.collectBlogPosts()
	posts = filterRenderableBlogPosts(posts)
	if len(posts) == 0 {
		return nil
	}

	// Populate URLs for already-generated files for all blog posts.
	for _, post := range posts {
		absPath, url, err := b.audioFilePathAndURL(post)
		if err == nil && url != "" && fileExistsNonEmpty(absPath) {
			post.AudioURL = url
		}
	}

	limit := b.config.Audio.RecentPosts
	if limit <= 0 || limit > len(posts) {
		limit = len(posts)
	}
	targets := posts[:limit]
	fmt.Printf("Audio generation enabled (%s), checking %d recent blog posts\n", b.config.Audio.Provider, len(targets))
	generated := make([]string, 0, len(targets))

	for _, post := range targets {
		absPath, url, err := b.audioFilePathAndURL(post)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Audio path error for %s: %v\n", post.URL, err)
			continue
		}
		if fileExistsNonEmpty(absPath) {
			post.AudioURL = url
			continue
		}

		text := toSpeechText(post.RawContent)
		text = trimToRunes(text, b.config.Audio.MaxChars)
		if text == "" {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Audio mkdir error for %s: %v\n", post.URL, err)
			continue
		}

		voice := b.voiceForLanguage(post.Language)
		if err := synthesizeWithEdge(edgeBin, voice, b.config.Audio.Edge.Rate, b.config.Audio.Edge.Pitch, text, absPath); err != nil {
			fmt.Fprintf(os.Stderr, "Audio generation failed for %s: %v\n", post.URL, err)
			_ = os.Remove(absPath)
			continue
		}

		generated = append(generated, absPath)
		post.AudioURL = url
		fmt.Printf("Generated audio: %s -> %s\n", post.URL, absPath)
	}

	if len(generated) > 0 {
		b.appendAssetFiles(index, generated)
	}

	return nil
}

func filterRenderableBlogPosts(posts []*Page) []*Page {
	out := make([]*Page, 0, len(posts))
	for _, p := range posts {
		if p == nil || p.Type != TypeBlog || p.Frontmatter == nil {
			continue
		}
		if p.Frontmatter.Date.IsZero() || strings.TrimSpace(p.RawContent) == "" {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Frontmatter.Date.After(out[j].Frontmatter.Date)
	})
	return out
}

func (b *SiteBuilder) audioFilePathAndURL(page *Page) (string, string, error) {
	if page == nil {
		return "", "", fmt.Errorf("nil page")
	}
	baseDir := strings.TrimSpace(b.config.Audio.OutputDir)
	if baseDir == "" {
		baseDir = "content/assets/audio/posts"
	}

	absBaseDir := b.resolvePath(baseDir)
	lang := i18n.NormalizeLanguage(page.Language)
	slug := parser.GenerateSlug(strings.TrimSpace(page.Title))
	if slug == "" {
		slug = "post"
	}

	sum := sha1.Sum([]byte(page.URL))
	hash := hex.EncodeToString(sum[:])[:10]
	filename := fmt.Sprintf("%s-%s.mp3", slug, hash)
	absPath := filepath.Join(absBaseDir, lang, filename)

	contentAbs := b.resolvePath(b.config.Build.ContentDir)
	relToContent, err := filepath.Rel(contentAbs, absPath)
	if err != nil {
		return absPath, "", err
	}
	if strings.HasPrefix(relToContent, "..") {
		return absPath, "", fmt.Errorf("audio output dir %q must be inside content dir %q", absBaseDir, contentAbs)
	}

	assetRel := filepath.ToSlash(relToContent)
	url := "/assets/" + strings.TrimPrefix(assetRel, "/")
	return absPath, url, nil
}

func (b *SiteBuilder) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	wd, err := os.Getwd()
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(wd, path))
}

func (b *SiteBuilder) voiceForLanguage(lang string) string {
	code := i18n.NormalizeLanguage(lang)
	if v, ok := b.config.Audio.Voices[code]; ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if strings.TrimSpace(b.config.Audio.Edge.Voice) != "" {
		return strings.TrimSpace(b.config.Audio.Edge.Voice)
	}
	return "en-US-EmmaMultilingualNeural"
}

func synthesizeWithEdge(binary, voice, rate, pitch, text, outputPath string) error {
	tmp, err := os.CreateTemp("", "blog-audio-*.txt")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(text); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if strings.TrimSpace(rate) == "" {
		rate = "+0%"
	}
	if strings.TrimSpace(pitch) == "" {
		pitch = "+0Hz"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := []string{
		"--voice", voice,
		"--rate", rate,
		"--pitch", pitch,
		"--file", tmpPath,
		"--write-media", outputPath,
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func toSpeechText(markdown string) string {
	s := strings.TrimSpace(markdown)
	if s == "" {
		return ""
	}

	// Remove fenced code blocks and inline code first to avoid spelling code aloud.
	s = regexp.MustCompile("(?s)```.*?```").ReplaceAllString(s, " ")
	s = regexp.MustCompile("`[^`]*`").ReplaceAllString(s, " ")

	// Images/links/wiki links to readable text labels.
	s = regexp.MustCompile(`!\[([^\]]*)\]\([^\)]*\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\[([^\]]+)\]\([^\)]*\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\[\[([^\]|]+)\|([^\]]+)\]\]`).ReplaceAllString(s, "$2")
	s = regexp.MustCompile(`\[\[([^\]]+)\]\]`).ReplaceAllString(s, "$1")

	// Remove common markdown syntax tokens.
	s = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s*`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?m)^\s*([-*+]|\d+\.)\s+`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "~~", "")
	s = strings.ReplaceAll(s, "<!--truncate-->", " ")

	// Remove any remaining HTML tags and collapse whitespace.
	s = regexp.MustCompile(`(?s)<[^>]*>`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func trimToRunes(s string, limit int) string {
	if limit <= 0 {
		return strings.TrimSpace(s)
	}
	r := []rune(strings.TrimSpace(s))
	if len(r) <= limit {
		return string(r)
	}
	return strings.TrimSpace(string(r[:limit]))
}

func (b *SiteBuilder) appendAssetFiles(index *ContentIndex, absPaths []string) {
	if index == nil || len(absPaths) == 0 {
		return
	}

	existing := make(map[string]struct{}, len(index.AssetFiles))
	for _, f := range index.AssetFiles {
		existing[filepath.Clean(f.Path)] = struct{}{}
	}

	contentAbs := b.resolvePath(b.config.Build.ContentDir)
	for _, p := range absPaths {
		abs := filepath.Clean(p)
		if _, ok := existing[abs]; ok {
			continue
		}

		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(contentAbs, abs)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}

		index.AssetFiles = append(index.AssetFiles, ContentFile{
			Path:         abs,
			RelativePath: rel,
			ContentType:  TypeAsset,
			ModifiedTime: info.ModTime().Unix(),
			Size:         info.Size(),
		})
		existing[abs] = struct{}{}
	}
}

func fileExistsNonEmpty(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}
