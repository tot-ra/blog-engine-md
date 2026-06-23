package builder

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestFilterRenderableBlogPostsSortsRecentPosts(t *testing.T) {
	older := &Page{
		Title:       "Older",
		Type:        TypeBlog,
		RawContent:  "Older content",
		Frontmatter: &parser.Frontmatter{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
	}
	newer := &Page{
		Title:       "Newer",
		Type:        TypeBlog,
		RawContent:  "Newer content",
		Frontmatter: &parser.Frontmatter{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)},
	}

	got := filterRenderableBlogPosts([]*Page{
		nil,
		{Title: "Doc", Type: TypeDoc, RawContent: "Doc", Frontmatter: &parser.Frontmatter{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC)}},
		{Title: "No date", Type: TypeBlog, RawContent: "Content", Frontmatter: &parser.Frontmatter{}},
		{Title: "No content", Type: TypeBlog, RawContent: "  ", Frontmatter: &parser.Frontmatter{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)}},
		older,
		newer,
	})

	if !reflect.DeepEqual(got, []*Page{newer, older}) {
		t.Fatalf("filterRenderableBlogPosts() = %#v, want newest renderable blog posts only", got)
	}
}

func TestAudioFilePathAndURLUsesStableContentScopedAssetURL(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir)

	builder := NewSiteBuilder(&config.SiteConfig{})
	builder.config.Build.ContentDir = "content"
	builder.config.Audio.OutputDir = "content/assets/audio/posts"

	post := &Page{
		Title:      "Hello, Bees!",
		Language:   "ru",
		URL:        "/ru/blog/hello-bees/",
		RawContent: "Same content keeps the generated file stable.",
	}

	absPath, url, err := builder.audioFilePathAndURL(post)
	if err != nil {
		t.Fatalf("audioFilePathAndURL() error = %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(cwd, "content", "assets", "audio", "posts", "ru")
	if filepath.Dir(absPath) != wantDir {
		t.Fatalf("audio path dir = %q, want %q", filepath.Dir(absPath), wantDir)
	}
	if !strings.HasPrefix(filepath.Base(absPath), "hello-bees-") || !strings.HasSuffix(absPath, ".mp3") {
		t.Fatalf("audio path = %q, want slugged mp3 filename", absPath)
	}
	if !strings.HasPrefix(url, "/assets/assets/audio/posts/ru/hello-bees-") || !strings.HasSuffix(url, ".mp3") {
		t.Fatalf("audio URL = %q, want content-relative asset URL", url)
	}

	absPathAgain, urlAgain, err := builder.audioFilePathAndURL(post)
	if err != nil {
		t.Fatalf("second audioFilePathAndURL() error = %v", err)
	}
	if absPathAgain != absPath || urlAgain != url {
		t.Fatalf("audioFilePathAndURL() is not stable: (%q, %q) then (%q, %q)", absPath, url, absPathAgain, urlAgain)
	}
}

func TestAudioFilePathAndURLRejectsOutputOutsideContent(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir)

	builder := NewSiteBuilder(&config.SiteConfig{})
	builder.config.Build.ContentDir = "content"
	builder.config.Audio.OutputDir = "generated/audio"

	_, _, err := builder.audioFilePathAndURL(&Page{Title: "Post", URL: "/blog/post/", RawContent: "Body"})
	if err == nil {
		t.Fatal("audioFilePathAndURL() error = nil, want output outside content to be rejected")
	}
	if !strings.Contains(err.Error(), "must be inside content dir") {
		t.Fatalf("audioFilePathAndURL() error = %q, want content-dir validation", err)
	}
}

func TestVoiceForLanguageChoosesLanguageProviderAndDefaultFallbacks(t *testing.T) {
	builder := NewSiteBuilder(&config.SiteConfig{})
	builder.config.Audio.Provider = "elevenlabs"
	builder.config.Audio.ElevenLabs.DefaultVoiceID = "eleven-default"
	builder.config.Audio.Edge.Voice = "edge-default"
	builder.config.Audio.Voices = map[string]string{"ru": " ru-voice "}

	if got := builder.voiceForLanguage("ru-RU"); got != "ru-voice" {
		t.Fatalf("voiceForLanguage(ru-RU) = %q, want configured normalized language voice", got)
	}
	if got := builder.voiceForLanguage("de"); got != "eleven-default" {
		t.Fatalf("voiceForLanguage(de) = %q, want elevenlabs default voice", got)
	}

	builder.config.Audio.Provider = "edge"
	if got := builder.voiceForLanguage("de"); got != "edge-default" {
		t.Fatalf("voiceForLanguage(de) = %q, want edge fallback voice", got)
	}

	builder.config.Audio.Edge.Voice = ""
	if got := builder.voiceForLanguage("de"); got != "en-US-EmmaMultilingualNeural" {
		t.Fatalf("voiceForLanguage(de) = %q, want hard-coded multilingual fallback", got)
	}
}

func TestToSpeechTextRemovesNonNarrativeMarkdown(t *testing.T) {
	markdown := `# 🐝 Intro

A paragraph with [a paper](https://example.com/paper.pdf), [[Bee|honey bee]], ` + "`code`" + `, and https://gratheon.com/research.

| Metric | Value |
| --- | --- |
| waggle | 42 |

` + "```go\nfmt.Println(\"skip me\")\n```" + `

![Alt image](image.png)
<!--truncate-->
<strong>HTML text</strong>`

	got := toSpeechText(markdown)

	for _, unwanted := range []string{"Metric", "waggle", "fmt.Println", "code", "🐝", "<strong>", "<!--truncate-->"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("toSpeechText() = %q, should not contain %q", got, unwanted)
		}
	}
	for _, wanted := range []string{"Intro.", "a paper", "honey bee", "link to gratheon.com", "Alt image", "HTML text"} {
		if !strings.Contains(got, wanted) {
			t.Fatalf("toSpeechText() = %q, want to contain %q", got, wanted)
		}
	}
}

func TestSplitTextIntoChunksPrefersSentenceBoundariesAndSplitsLongSegments(t *testing.T) {
	got := splitTextIntoChunks("First sentence. Second sentence is longer. abcdefghij", 16)
	want := []string{"First sentence.", "Second sentence", "is longer.", "abcdefghij"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitTextIntoChunks() = %#v, want %#v", got, want)
	}

	if got := splitTextIntoChunks("  unsplit text  ", 0); !reflect.DeepEqual(got, []string{"unsplit text"}) {
		t.Fatalf("splitTextIntoChunks(max=0) = %#v", got)
	}
	if got := splitTextIntoChunks("   ", 10); got != nil {
		t.Fatalf("splitTextIntoChunks(blank) = %#v, want nil", got)
	}
}

func TestAppendAssetFilesAddsExistingContentFilesOnce(t *testing.T) {
	tmpDir := t.TempDir()
	withWorkingDir(t, tmpDir)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	contentDir := filepath.Join(cwd, "content")
	inside := filepath.Join(contentDir, "assets", "audio", "post.mp3")
	outside := filepath.Join(tmpDir, "other", "ignored.mp3")
	if err := writeTestFile(inside, "audio"); err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(outside, "audio"); err != nil {
		t.Fatal(err)
	}

	builder := NewSiteBuilder(&config.SiteConfig{})
	builder.config.Build.ContentDir = "content"
	index := &ContentIndex{}

	builder.appendAssetFiles(index, []string{inside, inside, outside, filepath.Join(contentDir, "missing.mp3")})

	if len(index.AssetFiles) != 1 {
		t.Fatalf("appendAssetFiles added %d files, want 1: %#v", len(index.AssetFiles), index.AssetFiles)
	}
	if index.AssetFiles[0].Path != filepath.Clean(inside) {
		t.Fatalf("asset path = %q, want %q", index.AssetFiles[0].Path, filepath.Clean(inside))
	}
	if index.AssetFiles[0].RelativePath != filepath.Join("assets", "audio", "post.mp3") {
		t.Fatalf("asset relative path = %q", index.AssetFiles[0].RelativePath)
	}
}

func TestAudioSmallHelpers(t *testing.T) {
	if !isElevenLabsQuotaExceeded(errString("quota_exceeded: limit reached")) {
		t.Fatal("expected quota_exceeded errors to be detected")
	}
	if !isElevenLabsQuotaExceeded(errString("This request exceeds your quota")) {
		t.Fatal("expected exceeds-your-quota errors to be detected")
	}
	if isElevenLabsQuotaExceeded(nil) || isElevenLabsQuotaExceeded(errString("network timeout")) {
		t.Fatal("expected non-quota errors to be ignored")
	}

	if got := trimToRunes("  abcdef  ", 4); got != "abcd" {
		t.Fatalf("trimToRunes() = %q, want abcd", got)
	}
	if got := trimToRunes("  abc  ", 0); got != "abc" {
		t.Fatalf("trimToRunes(limit=0) = %q, want abc", got)
	}

	file := filepath.Join(t.TempDir(), "audio.mp3")
	if fileExistsNonEmpty(file) {
		t.Fatal("missing file should not exist")
	}
	if err := os.WriteFile(file, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if fileExistsNonEmpty(file) {
		t.Fatal("empty file should not count as non-empty")
	}
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if !fileExistsNonEmpty(file) {
		t.Fatal("non-empty file should exist")
	}

	if got := clamp01(-0.5); got != 0 {
		t.Fatalf("clamp01(-0.5) = %v, want 0", got)
	}
	if got := clamp01(0.5); got != 0.5 {
		t.Fatalf("clamp01(0.5) = %v, want 0.5", got)
	}
	if got := clamp01(1.5); got != 1 {
		t.Fatalf("clamp01(1.5) = %v, want 1", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore working dir: %v", err)
		}
	})
}

func writeTestFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
