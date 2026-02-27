package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// EmbedProvider represents a supported embed provider
type EmbedProvider string

const (
	EmbedYouTube  EmbedProvider = "youtube"
	EmbedVimeo    EmbedProvider = "vimeo"
	EmbedCodePen  EmbedProvider = "codepen"
	EmbedGist     EmbedProvider = "gist"
)

var (
	// Explicit embed syntax: ::youtube[VIDEO_ID]
	explicitEmbedRegex = regexp.MustCompile(`^::(youtube|vimeo)\[([^\]]+)\]\s*$`)

	// YouTube URL patterns
	youtubeURLRegex = regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtu\.be/|youtube\.com/embed/)([a-zA-Z0-9_-]{11})`)

	// Vimeo URL patterns
	vimeoURLRegex = regexp.MustCompile(`vimeo\.com/(\d+)`)
)

// TransformEmbeds processes embed syntax in markdown content
// Supports ::youtube[ID] and ::vimeo[ID] syntax
func TransformEmbeds(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check explicit embed syntax
		if match := explicitEmbedRegex.FindStringSubmatch(trimmed); match != nil {
			provider := EmbedProvider(match[1])
			id := match[2]
			result = append(result, renderEmbed(provider, id))
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func renderEmbed(provider EmbedProvider, id string) string {
	switch provider {
	case EmbedYouTube:
		return renderYouTubeEmbed(id)
	case EmbedVimeo:
		return renderVimeoEmbed(id)
	default:
		return fmt.Sprintf("<!-- unsupported embed: %s[%s] -->", provider, id)
	}
}

func renderYouTubeEmbed(videoID string) string {
	return fmt.Sprintf(`<div class="embed embed-youtube">
  <iframe
    src="https://www.youtube-nocookie.com/embed/%s"
    frameborder="0"
    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
    allowfullscreen
    loading="lazy">
  </iframe>
</div>`, videoID)
}

func renderVimeoEmbed(videoID string) string {
	return fmt.Sprintf(`<div class="embed embed-vimeo">
  <iframe
    src="https://player.vimeo.com/video/%s"
    frameborder="0"
    allow="autoplay; fullscreen; picture-in-picture"
    allowfullscreen
    loading="lazy">
  </iframe>
</div>`, videoID)
}
