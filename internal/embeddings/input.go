package embeddings

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/tot-ra/blog-engine/internal/related"
)

const (
	DefaultChunkChars   = 6000
	DefaultChunkOverlap = 200
)

// PrepareInput keeps the embeddings package API while sharing normalization with related matching.
func PrepareInput(title, description string, tags []string, body string) string {
	return related.PrepareInput(title, description, tags, body)
}

func HashInput(text, model string, dims int) string {
	return related.HashInput(text, model, dims)
}

// ChunkText splits on nearby whitespace where possible and keeps a rune overlap.
func ChunkText(text string, maxChars, overlap int) []string {
	if maxChars <= 0 {
		maxChars = DefaultChunkChars
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxChars {
		overlap = maxChars / 10
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}
	if len(runes) <= maxChars {
		return []string{string(runes)}
	}

	chunks := make([]string, 0, len(runes)/maxChars+1)
	for start := 0; start < len(runes); {
		end := start + maxChars
		if end >= len(runes) {
			chunks = append(chunks, strings.TrimSpace(string(runes[start:])))
			break
		}
		// Avoid cutting words while retaining most of the requested chunk size.
		for candidate := end; candidate > start+maxChars/2; candidate-- {
			if runes[candidate-1] == ' ' || runes[candidate-1] == '\n' || runes[candidate-1] == '\t' {
				end = candidate
				break
			}
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		next := end - overlap
		if next <= start {
			next = end
		} else {
			// Move to the beginning of the overlapping word rather than emitting
			// fragments such as "mbedding" at the next chunk boundary.
			for next > start && !isSpaceRune(runes[next-1]) {
				next--
			}
		}
		start = next
	}
	return chunks
}

// MergeChunks averages chunk embeddings and L2-normalizes the article vector.
func MergeChunks(vectors [][]float32) ([]float32, error) {
	if len(vectors) == 0 {
		return nil, fmt.Errorf("no chunk vectors")
	}
	dims := len(vectors[0])
	if dims == 0 {
		return nil, fmt.Errorf("empty chunk vector")
	}
	merged := make([]float32, dims)
	for _, vec := range vectors {
		if len(vec) != dims {
			return nil, fmt.Errorf("inconsistent chunk dimensions: got %d, want %d", len(vec), dims)
		}
		for i, value := range vec {
			merged[i] += value
		}
	}
	inv := float32(1) / float32(len(vectors))
	for i := range merged {
		merged[i] *= inv
	}
	return Normalize(merged), nil
}

func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	// OpenAI does not expose a tokenizer in the API. Four UTF-8 bytes per token
	// is a practical language-neutral estimate for progress and cost previews.
	return (len([]byte(text)) + 3) / 4
}

func truncateRunes(value string, max int) string {
	if max <= 0 || utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}

func isSpaceRune(value rune) bool {
	return value == ' ' || value == '\n' || value == '\t' || value == '\r'
}
