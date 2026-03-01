package builder

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

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

func synthesizeWithEdge(binary, voice, rate, pitch, text, outputPath string) error {
	if strings.TrimSpace(binary) == "" {
		binary = "edge-tts"
	}
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

func synthesizeWithElevenLabs(
	baseURL, apiKey, voiceID, modelID, outputFormat string,
	stability, similarityBoost, style float64,
	speakerBoost bool,
	text, outputPath string,
) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.elevenlabs.io"
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("empty elevenlabs api key")
	}
	if strings.TrimSpace(voiceID) == "" {
		return fmt.Errorf("empty elevenlabs voice id")
	}
	if strings.TrimSpace(modelID) == "" {
		modelID = "eleven_multilingual_v2"
	}
	if strings.TrimSpace(outputFormat) == "" {
		outputFormat = "mp3_44100_128"
	}

	const elevenLabsMaxRunesPerRequest = 9500
	chunks := splitTextIntoChunks(text, elevenLabsMaxRunesPerRequest)
	if len(chunks) == 0 {
		return fmt.Errorf("empty text for elevenlabs synthesis")
	}

	endpoint := fmt.Sprintf(
		"%s/v1/text-to-speech/%s?output_format=%s",
		baseURL,
		neturl.PathEscape(strings.TrimSpace(voiceID)),
		neturl.QueryEscape(outputFormat),
	)

	type voiceSettings struct {
		Stability       float64 `json:"stability"`
		SimilarityBoost float64 `json:"similarity_boost"`
		Style           float64 `json:"style,omitempty"`
		SpeakerBoost    bool    `json:"use_speaker_boost"`
	}
	var merged bytes.Buffer
	for idx, chunk := range chunks {
		payload := map[string]any{
			"text":     chunk,
			"model_id": modelID,
			"voice_settings": voiceSettings{
				Stability:       clamp01(stability),
				SimilarityBoost: clamp01(similarityBoost),
				Style:           clamp01(style),
				SpeakerBoost:    speakerBoost,
			},
		}

		audioBytes, err := requestElevenLabsAudio(endpoint, apiKey, payload)
		if err != nil {
			return fmt.Errorf("chunk %d/%d: %w", idx+1, len(chunks), err)
		}
		if _, err := merged.Write(audioBytes); err != nil {
			return err
		}
	}

	if merged.Len() == 0 {
		return fmt.Errorf("empty audio response from elevenlabs")
	}
	return os.WriteFile(outputPath, merged.Bytes(), 0644)
}

func requestElevenLabsAudio(endpoint, apiKey string, payload map[string]any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("elevenlabs %s: %s", resp.Status, strings.TrimSpace(string(errBody)))
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(audioBytes) == 0 {
		return nil, fmt.Errorf("empty audio response from elevenlabs")
	}
	return audioBytes, nil
}

func splitTextIntoChunks(text string, maxRunes int) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	if maxRunes <= 0 {
		return []string{trimmed}
	}

	parts := splitIntoSentences(trimmed)
	chunks := make([]string, 0, len(parts))
	var current strings.Builder
	currentLen := 0

	flush := func() {
		if currentLen == 0 {
			return
		}
		chunks = append(chunks, strings.TrimSpace(current.String()))
		current.Reset()
		currentLen = 0
	}

	for _, part := range parts {
		segment := strings.TrimSpace(part)
		if segment == "" {
			continue
		}
		segmentRunes := []rune(segment)
		if len(segmentRunes) > maxRunes {
			flush()
			for len(segmentRunes) > maxRunes {
				chunks = append(chunks, strings.TrimSpace(string(segmentRunes[:maxRunes])))
				segmentRunes = segmentRunes[maxRunes:]
			}
			if len(segmentRunes) > 0 {
				current.WriteString(string(segmentRunes))
				currentLen = len(segmentRunes)
			}
			continue
		}

		additional := len(segmentRunes)
		if currentLen > 0 {
			additional++ // space
		}
		if currentLen+additional > maxRunes {
			flush()
		}
		if currentLen > 0 {
			current.WriteByte(' ')
			currentLen++
		}
		current.WriteString(segment)
		currentLen += len(segmentRunes)
	}

	flush()
	return chunks
}

func splitIntoSentences(s string) []string {
	runes := []rune(s)
	parts := make([]string, 0, 64)
	start := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		j := i + 1
		for j < len(runes) && unicode.IsSpace(runes[j]) {
			j++
		}
		if j > i+1 {
			part := strings.TrimSpace(string(runes[start:j]))
			if part != "" {
				parts = append(parts, part)
			}
			start = j
			i = j - 1
		}
	}
	if start < len(runes) {
		last := strings.TrimSpace(string(runes[start:]))
		if last != "" {
			parts = append(parts, last)
		}
	}
	return parts
}

func toSpeechText(markdown string) string {
	s := strings.TrimSpace(markdown)
	if s == "" {
		return ""
	}

	// Skip markdown tables entirely and insert stronger pauses around headings.
	s = stripMarkdownTables(s)
	s = addHeadingPauses(s)

	// Remove fenced code blocks and inline code first to avoid spelling code aloud.
	s = regexp.MustCompile("(?s)```.*?```").ReplaceAllString(s, " ")
	s = regexp.MustCompile("`[^`]*`").ReplaceAllString(s, " ")

	// Images/links/wiki links to readable text labels.
	s = regexp.MustCompile(`!\[([^\]]*)\]\([^\)]*\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\[([^\]]+)\]\([^\)]*\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\[\[([^\]|]+)\|([^\]]+)\]\]`).ReplaceAllString(s, "$2")
	s = regexp.MustCompile(`\[\[([^\]]+)\]\]`).ReplaceAllString(s, "$1")
	s = replaceURLsWithDomainSpeech(s)

	// Remove common markdown syntax tokens.
	s = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s*`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?m)^\s*([-*+]|\d+\.)\s+`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "~~", "")
	s = strings.ReplaceAll(s, "<!--truncate-->", " ")
	s = stripEmojiLikeRunes(s)

	// Remove any remaining HTML tags and collapse whitespace.
	s = regexp.MustCompile(`(?s)<[^>]*>`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func addHeadingPauses(markdown string) string {
	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}

		heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if heading == "" {
			lines[i] = ""
			continue
		}
		// Extra sentence boundaries create audible pauses before and after headings.
		lines[i] = "\n\n" + heading + ".\n\n"
	}
	return strings.Join(lines, "\n")
}

func stripMarkdownTables(markdown string) string {
	lines := strings.Split(markdown, "\n")
	out := make([]string, 0, len(lines))

	for i := 0; i < len(lines); i++ {
		if i+1 < len(lines) && looksLikeTableHeader(lines[i]) && isMarkdownTableSeparatorLine(lines[i+1]) {
			// Skip header + separator.
			i += 2
			// Skip table body rows.
			for i < len(lines) {
				trimmed := strings.TrimSpace(lines[i])
				if trimmed == "" || !strings.Contains(trimmed, "|") {
					i--
					break
				}
				i++
			}
			continue
		}
		out = append(out, lines[i])
	}

	return strings.Join(out, "\n")
}

func looksLikeTableHeader(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	return strings.Contains(trimmed, "|")
}

func isMarkdownTableSeparatorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.Contains(trimmed, "-") || !strings.Contains(trimmed, "|") {
		return false
	}

	parts := strings.Split(trimmed, "|")
	validCols := 0
	sepCell := regexp.MustCompile(`^:?-{3,}:?$`)
	for _, p := range parts {
		cell := strings.TrimSpace(p)
		if cell == "" {
			continue
		}
		if !sepCell.MatchString(cell) {
			return false
		}
		validCols++
	}
	return validCols >= 2
}

func replaceURLsWithDomainSpeech(s string) string {
	urlLike := regexp.MustCompile(`(?i)\b(?:https?|ftp)://[^\s<>()]+|\bwww\.[^\s<>()]+`)
	return urlLike.ReplaceAllStringFunc(s, func(raw string) string {
		trimmed := strings.TrimRight(raw, ".,;:!?)]}\"'")
		if trimmed == "" {
			return " "
		}

		toParse := trimmed
		if strings.HasPrefix(strings.ToLower(toParse), "www.") {
			toParse = "https://" + toParse
		}

		u, err := neturl.Parse(toParse)
		if err != nil || strings.TrimSpace(u.Hostname()) == "" {
			return " link "
		}

		domain := strings.ToLower(strings.TrimSpace(u.Hostname()))
		return " link to " + domain + " "
	})
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

func stripEmojiLikeRunes(s string) string {
	var b strings.Builder
	for _, r := range s {
		if isEmojiLikeRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isEmojiLikeRune(r rune) bool {
	if r == '\u200d' || r == '\ufe0f' {
		return true
	}
	if (r >= 0x1F300 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) {
		return true
	}
	if unicode.Is(unicode.So, r) || unicode.Is(unicode.Sk, r) {
		return true
	}
	return false
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
