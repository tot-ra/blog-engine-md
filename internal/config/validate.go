package config

import (
	"fmt"
	"strings"
)

// Validate checks if the configuration is valid
func Validate(cfg *SiteConfig) error {
	if cfg.Site.Title == "" {
		return fmt.Errorf("site.title is required")
	}
	if cfg.Site.URL == "" {
		return fmt.Errorf("site.url is required")
	}
	if cfg.Site.Logo == "" {
		cfg.Site.Logo = "/assets/img/artjom.webp"
	}
	if cfg.Build.ContentDir == "" {
		cfg.Build.ContentDir = "content"
	}
	if cfg.Build.OutputDir == "" {
		cfg.Build.OutputDir = "dist"
	}
	if cfg.Build.ParallelWorkers <= 0 {
		cfg.Build.ParallelWorkers = 4
	}
	if cfg.Assets.Images.ParallelWorkers < 0 {
		cfg.Assets.Images.ParallelWorkers = 0
	}
	if cfg.Assets.Images.MaxSourcePixels < 0 {
		cfg.Assets.Images.MaxSourcePixels = 0
	}
	if cfg.Assets.Images.MaxVariantPixels < 0 {
		cfg.Assets.Images.MaxVariantPixels = 0
	}
	if cfg.Audio.Provider == "" {
		cfg.Audio.Provider = "elevenlabs"
	}
	if cfg.Audio.OutputDir == "" {
		cfg.Audio.OutputDir = "content/audio/posts"
	}
	if cfg.Audio.RecentPosts == 0 {
		cfg.Audio.RecentPosts = 10
	}
	if cfg.Audio.MaxChars <= 0 {
		cfg.Audio.MaxChars = 12000
	}
	if cfg.Audio.Edge.Binary == "" {
		cfg.Audio.Edge.Binary = "edge-tts"
	}
	if cfg.Audio.Edge.Rate == "" {
		cfg.Audio.Edge.Rate = "+0%"
	}
	if cfg.Audio.Edge.Pitch == "" {
		cfg.Audio.Edge.Pitch = "+0Hz"
	}
	if cfg.Audio.Edge.Voice == "" {
		cfg.Audio.Edge.Voice = "en-US-EmmaMultilingualNeural"
	}
	if cfg.Audio.ElevenLabs.APIKeyEnv == "" {
		cfg.Audio.ElevenLabs.APIKeyEnv = "ELEVENLABS_API_KEY"
	}
	if cfg.Audio.ElevenLabs.BaseURL == "" {
		cfg.Audio.ElevenLabs.BaseURL = "https://api.elevenlabs.io"
	}
	if cfg.Audio.ElevenLabs.ModelID == "" {
		cfg.Audio.ElevenLabs.ModelID = "eleven_multilingual_v2"
	}
	if cfg.Audio.ElevenLabs.OutputFormat == "" {
		cfg.Audio.ElevenLabs.OutputFormat = "mp3_44100_128"
	}
	if cfg.Audio.ElevenLabs.DefaultVoiceID == "" {
		cfg.Audio.ElevenLabs.DefaultVoiceID = "EXAVITQu4vr4xnSDxMaL"
	}
	if cfg.Audio.ElevenLabs.Stability <= 0 {
		cfg.Audio.ElevenLabs.Stability = 0.45
	}
	if cfg.Audio.ElevenLabs.SimilarityBoost <= 0 {
		cfg.Audio.ElevenLabs.SimilarityBoost = 0.75
	}
	if cfg.Audio.ElevenLabs.Style < 0 {
		cfg.Audio.ElevenLabs.Style = 0
	}
	if cfg.Audio.Voices == nil {
		cfg.Audio.Voices = map[string]string{
			"ru": cfg.Audio.ElevenLabs.DefaultVoiceID,
			"en": cfg.Audio.ElevenLabs.DefaultVoiceID,
		}
	}
	if cfg.Related.Provider == "" {
		cfg.Related.Provider = "openai"
	}
	if cfg.Related.Provider != "openai" {
		return fmt.Errorf("related.provider must be openai")
	}
	if cfg.Related.Model == "" {
		cfg.Related.Model = "text-embedding-3-small"
	}
	if cfg.Related.Dimensions == 0 {
		cfg.Related.Dimensions = 512
	}
	if cfg.Related.Dimensions < 0 || cfg.Related.Dimensions > 3072 {
		return fmt.Errorf("related.dimensions must be between 1 and 3072")
	}
	if cfg.Related.APIKeyEnv == "" {
		cfg.Related.APIKeyEnv = "OPENAI_API_KEY"
	}
	if cfg.Related.CachePath == "" {
		cfg.Related.CachePath = "content/embeddings.json"
	}
	if len(cfg.Related.Sections) == 0 {
		cfg.Related.Sections = []string{"blog"}
	}
	for i, section := range cfg.Related.Sections {
		section = strings.Trim(strings.TrimSpace(section), "/")
		if section == "" || strings.Contains(section, "..") {
			return fmt.Errorf("related.sections must contain safe non-empty section names")
		}
		cfg.Related.Sections[i] = section
	}
	if cfg.Related.Count <= 0 {
		cfg.Related.Count = 4
	}
	if cfg.Related.MinScore < 0 || cfg.Related.MinScore > 1 {
		return fmt.Errorf("related.minScore must be between 0 and 1")
	}
	if cfg.Related.Diversity < 0 || cfg.Related.Diversity > 1 {
		return fmt.Errorf("related.diversity must be between 0 and 1")
	}
	if cfg.Site.Language == "" {
		cfg.Site.Language = "en"
	}
	if cfg.I18n.Default == "" {
		cfg.I18n.Default = cfg.Site.Language
	}
	if len(cfg.I18n.Languages) == 0 {
		cfg.I18n.Languages = []LanguageConfig{
			{Code: cfg.I18n.Default, Label: strings.ToUpper(cfg.I18n.Default)},
		}
	}
	langs := make([]LanguageConfig, 0, len(cfg.I18n.Languages))
	seen := make(map[string]struct{}, len(cfg.I18n.Languages))
	for _, l := range cfg.I18n.Languages {
		code := strings.ToLower(strings.TrimSpace(l.Code))
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		label := strings.TrimSpace(l.Label)
		if label == "" {
			label = strings.ToUpper(code)
		}
		aliases := make([]string, 0, len(l.Aliases))
		aliasSeen := make(map[string]struct{}, len(l.Aliases)+1)
		for _, alias := range l.Aliases {
			normalized := strings.ToLower(strings.TrimSpace(alias))
			if normalized == "" {
				continue
			}
			if _, ok := aliasSeen[normalized]; ok {
				continue
			}
			aliasSeen[normalized] = struct{}{}
			aliases = append(aliases, normalized)
		}
		direction := strings.ToLower(strings.TrimSpace(l.Direction))
		if direction != "rtl" && direction != "ltr" {
			direction = defaultLanguageDirection(code)
		}
		langs = append(langs, LanguageConfig{
			Code:      code,
			Label:     label,
			Aliases:   aliases,
			Direction: direction,
		})
	}
	cfg.I18n.Languages = langs
	if len(cfg.I18n.Languages) == 0 {
		return fmt.Errorf("i18n.languages must contain at least one language")
	}
	cfg.I18n.Default = strings.ToLower(strings.TrimSpace(cfg.I18n.Default))
	if cfg.I18n.Default == "" {
		cfg.I18n.Default = cfg.I18n.Languages[0].Code
	}
	hasDefault := false
	for _, l := range cfg.I18n.Languages {
		if l.Code == cfg.I18n.Default {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		cfg.I18n.Languages = append([]LanguageConfig{{Code: cfg.I18n.Default, Label: strings.ToUpper(cfg.I18n.Default)}}, cfg.I18n.Languages...)
	}
	return nil
}
