package related

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry contains all metadata needed to filter and rank one article offline.
type Entry struct {
	Path            string
	URL             string
	Title           string
	Language        string
	TranslationPath string
	Tags            []string
	Vector          []float32
}

// RelatedMatch is safe to pass directly to page templates.
type RelatedMatch struct {
	Path  string
	URL   string
	Title string
	Score float64
}

type Config struct {
	Count         int
	MinScore      float64
	Diversity     float64
	CrossLanguage bool
}

type Cache struct {
	Version int                   `json:"version"`
	Model   string                `json:"model"`
	Dims    int                   `json:"dims"`
	Entries map[string]CacheEntry `json:"entries"`
}

type CacheEntry struct {
	Hash  string  `json:"hash"`
	Vec   string  `json:"vec"`
	Scale float32 `json:"scale"`
	Lang  string  `json:"lang"`
	URL   string  `json:"url"`
}

func LoadCache(path string) (*Cache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parse embeddings cache: %w", err)
	}
	return &cache, nil
}

func DecodeVector(entry CacheEntry, dims int) ([]float32, error) {
	data, err := base64.StdEncoding.DecodeString(entry.Vec)
	if err != nil {
		return nil, err
	}
	if dims > 0 && len(data) != dims {
		return nil, fmt.Errorf("vector has %d dimensions, want %d", len(data), dims)
	}
	if entry.Scale <= 0 {
		return nil, fmt.Errorf("invalid vector scale")
	}
	vec := make([]float32, len(data))
	var norm float64
	for i, value := range data {
		vec[i] = float32(int8(value)) * entry.Scale
		norm += float64(vec[i]) * float64(vec[i])
	}
	if norm == 0 {
		return nil, fmt.Errorf("zero vector")
	}
	// Quantization slightly changes the norm, so normalize once while loading.
	inv := float32(1 / math.Sqrt(norm))
	for i := range vec {
		vec[i] *= inv
	}
	return vec, nil
}

// ComputeRelated computes every pair once, stores a flat symmetric matrix, then
// applies MMR using matrix lookups only. No allocations happen in ranking loops.
func ComputeRelated(entries []Entry, cfg Config) map[string][]RelatedMatch {
	result := make(map[string][]RelatedMatch, len(entries))
	if cfg.Count <= 0 || len(entries) < 2 {
		return result
	}
	if cfg.Diversity < 0 {
		cfg.Diversity = 0
	} else if cfg.Diversity > 1 {
		cfg.Diversity = 1
	}

	n := len(entries)
	similarities := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			score := dot(entries[i].Vector, entries[j].Vector)
			similarities[i*n+j] = score
			similarities[j*n+i] = score
		}
	}

	eligible := make([]bool, n)
	selected := make([]bool, n)
	maxSelectedSimilarity := make([]float64, n)
	for query := 0; query < n; query++ {
		clear(eligible)
		clear(selected)
		clear(maxSelectedSimilarity)
		eligibleCount := 0
		for candidate := 0; candidate < n; candidate++ {
			if candidate == query || !candidateAllowed(entries[query], entries[candidate], cfg) {
				continue
			}
			base := similarities[query*n+candidate]
			if base < cfg.MinScore {
				continue
			}
			eligible[candidate] = true
			eligibleCount++
		}
		limit := cfg.Count
		if eligibleCount < limit {
			limit = eligibleCount
		}
		matches := make([]RelatedMatch, 0, limit)
		for len(matches) < limit {
			best := -1
			bestMMR := math.Inf(-1)
			bestQueryScore := math.Inf(-1)
			for candidate := 0; candidate < n; candidate++ {
				if !eligible[candidate] || selected[candidate] {
					continue
				}
				queryScore := similarities[query*n+candidate] + tagBonus(entries[query].Tags, entries[candidate].Tags)
				mmr := queryScore
				if len(matches) > 0 {
					mmr = cfg.Diversity*queryScore - (1-cfg.Diversity)*maxSelectedSimilarity[candidate]
				}
				if mmr > bestMMR || (mmr == bestMMR && (queryScore > bestQueryScore || (queryScore == bestQueryScore && pathLess(entries[candidate].Path, best, entries)))) {
					best, bestMMR, bestQueryScore = candidate, mmr, queryScore
				}
			}
			if best < 0 {
				break
			}
			selected[best] = true
			matches = append(matches, RelatedMatch{Path: entries[best].Path, URL: entries[best].URL, Title: entries[best].Title, Score: similarities[query*n+best]})
			for candidate := 0; candidate < n; candidate++ {
				if eligible[candidate] && !selected[candidate] && similarities[best*n+candidate] > maxSelectedSimilarity[candidate] {
					maxSelectedSimilarity[candidate] = similarities[best*n+candidate]
				}
			}
		}
		if len(matches) > 0 {
			result[entries[query].Path] = matches
		}
	}
	return result
}

func candidateAllowed(query, candidate Entry, cfg Config) bool {
	if !cfg.CrossLanguage && !strings.EqualFold(query.Language, candidate.Language) {
		return false
	}
	return query.TranslationPath == "" || candidate.TranslationPath == "" || !strings.EqualFold(query.TranslationPath, candidate.TranslationPath)
}

func dot(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return -1
	}
	var sum float64
	for i, value := range a {
		sum += float64(value) * float64(b[i])
	}
	return sum
}

func tagBonus(a, b []string) float64 {
	for _, left := range a {
		for _, right := range b {
			if strings.EqualFold(left, right) {
				return 0.02
			}
		}
	}
	return 0
}

func pathLess(path string, best int, entries []Entry) bool {
	return best < 0 || path < entries[best].Path
}

// ResolveManual converts frontmatter references to matches. A URL, content path,
// extensionless path, or unique basename/slug may be used; unknown items are ignored.
func ResolveManual(refs []string, entries []Entry) []RelatedMatch {
	if len(refs) == 0 {
		return nil
	}
	aliases := make(map[string]int, len(entries)*4)
	ambiguous := make(map[string]bool)
	for i, entry := range entries {
		for _, alias := range entryAliases(entry) {
			if previous, ok := aliases[alias]; ok && previous != i {
				ambiguous[alias] = true
			} else {
				aliases[alias] = i
			}
		}
	}
	matches := make([]RelatedMatch, 0, len(refs))
	seen := make(map[int]struct{}, len(refs))
	for _, ref := range refs {
		key := normalizeRef(ref)
		i, ok := aliases[key]
		if !ok || ambiguous[key] {
			continue
		}
		if _, duplicate := seen[i]; duplicate {
			continue
		}
		seen[i] = struct{}{}
		entry := entries[i]
		matches = append(matches, RelatedMatch{Path: entry.Path, URL: entry.URL, Title: entry.Title, Score: 1})
	}
	return matches
}

func entryAliases(entry Entry) []string {
	path := normalizeRef(entry.Path)
	withoutExt := strings.TrimSuffix(path, filepath.Ext(path))
	base := filepath.Base(withoutExt)
	return []string{path, withoutExt, normalizeRef(entry.URL), normalizeRef(base)}
}

func normalizeRef(value string) string {
	value = strings.TrimSpace(filepath.ToSlash(value))
	value = strings.Trim(value, "/")
	return strings.ToLower(value)
}

// SortedEntries makes behavior deterministic when entries originate from maps.
func SortedEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
}
