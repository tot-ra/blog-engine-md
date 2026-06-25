package builder

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tot-ra/blog-engine/internal/i18n"
	"github.com/tot-ra/blog-engine/internal/parser"
)

func (b *SiteBuilder) prepareBlogAudio(index *ContentIndex) error {
	if b.config == nil || !b.config.Audio.Enabled {
		return nil
	}

	provider := strings.ToLower(strings.TrimSpace(b.config.Audio.Provider))
	switch provider {
	case "edge":
		edgeBin := strings.TrimSpace(b.config.Audio.Edge.Binary)
		if edgeBin == "" {
			edgeBin = "edge-tts"
		}
		if _, err := exec.LookPath(edgeBin); err != nil {
			fmt.Fprintf(os.Stderr, "Audio generation skipped: %q not found in PATH\n", edgeBin)
			return nil
		}
	case "elevenlabs":
		apiKey := strings.TrimSpace(os.Getenv(b.config.Audio.ElevenLabs.APIKeyEnv))
		if apiKey == "" {
			fmt.Fprintf(os.Stderr, "Audio generation skipped: env %q is not set\n", b.config.Audio.ElevenLabs.APIKeyEnv)
			return nil
		}
	default:
		return fmt.Errorf("unsupported audio provider %q", b.config.Audio.Provider)
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
	quotaPaused := false

	for _, post := range targets {
		if quotaPaused {
			break
		}

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
		var synthErr error
		switch provider {
		case "edge":
			synthErr = synthesizeWithEdge(
				strings.TrimSpace(b.config.Audio.Edge.Binary),
				voice,
				b.config.Audio.Edge.Rate,
				b.config.Audio.Edge.Pitch,
				text,
				absPath,
			)
		case "elevenlabs":
			synthErr = synthesizeWithElevenLabs(
				b.config.Audio.ElevenLabs.BaseURL,
				os.Getenv(b.config.Audio.ElevenLabs.APIKeyEnv),
				voice,
				b.config.Audio.ElevenLabs.ModelID,
				b.config.Audio.ElevenLabs.OutputFormat,
				b.config.Audio.ElevenLabs.Stability,
				b.config.Audio.ElevenLabs.SimilarityBoost,
				b.config.Audio.ElevenLabs.Style,
				b.config.Audio.ElevenLabs.SpeakerBoost,
				text,
				absPath,
			)
		}
		if synthErr != nil {
			_ = os.Remove(absPath)
			if provider == "elevenlabs" && isElevenLabsQuotaExceeded(synthErr) {
				fmt.Fprintf(
					os.Stderr,
					"Audio generation paused: ElevenLabs quota exceeded while processing %s. Existing audio is kept; add credits and rebuild to continue.\n",
					post.URL,
				)
				quotaPaused = true
				continue
			}
			fmt.Fprintf(os.Stderr, "Audio generation failed for %s: %v\n", post.URL, synthErr)
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

func isElevenLabsQuotaExceeded(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "quota_exceeded") || strings.Contains(msg, "exceeds your quota")
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

	hashInput := fmt.Sprintf("%s|%s", page.URL, page.RawContent)
	sum := sha1.Sum([]byte(hashInput))
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
	if strings.ToLower(strings.TrimSpace(b.config.Audio.Provider)) == "elevenlabs" {
		if strings.TrimSpace(b.config.Audio.ElevenLabs.DefaultVoiceID) != "" {
			return strings.TrimSpace(b.config.Audio.ElevenLabs.DefaultVoiceID)
		}
	}
	if strings.TrimSpace(b.config.Audio.Edge.Voice) != "" {
		return strings.TrimSpace(b.config.Audio.Edge.Voice)
	}
	return "en-US-EmmaMultilingualNeural"
}
